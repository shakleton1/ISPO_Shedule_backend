package schedule

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type LocationAutofillRequest struct {
	GroupID        int       `json:"group_id"`
	StartDate      time.Time `json:"-"`
	EndDate        time.Time `json:"-"`
	Campus         *string   `json:"campus,omitempty"`
	LocationKind   *string   `json:"location_kind,omitempty"`
	ReplaceVirtual bool      `json:"replace_virtual"`
	DryRun         bool      `json:"dry_run"`
	Comment        *string   `json:"comment,omitempty"`
}

type LocationAutofillResponse struct {
	GroupID     int                          `json:"group_id"`
	DateStart   string                       `json:"date_start"`
	DateEnd     string                       `json:"date_end"`
	DryRun      bool                         `json:"dry_run"`
	Assigned    int                          `json:"assigned"`
	Skipped     int                          `json:"skipped"`
	Created     int                          `json:"created"`
	Updated     int                          `json:"updated"`
	DataVersion string                       `json:"data_version"`
	Assignments []LocationAutofillAssignment `json:"assignments"`
}

type LocationAutofillAssignment struct {
	Date           string `json:"date"`
	PairNumber     int16  `json:"pair_number"`
	Subgroup       *int16 `json:"subgroup"`
	SubjectID      *int   `json:"subject_id"`
	SubjectName    string `json:"subject_name"`
	LocationID     *int   `json:"location_id"`
	LocationName   string `json:"location_name"`
	Status         string `json:"status"`
	Reason         string `json:"reason,omitempty"`
	OverrideAction string `json:"override_action,omitempty"`
}

type locationAutofillSlot struct {
	Date       string
	PairNumber int16
	LocationID int
}

func (s *Service) AutofillLocations(req LocationAutofillRequest) (*LocationAutofillResponse, error) {
	if req.GroupID <= 0 {
		return nil, fmt.Errorf("group_id required")
	}
	startDate := dateOnly(req.StartDate)
	endDate := dateOnly(req.EndDate)
	if startDate.IsZero() || endDate.IsZero() {
		return nil, fmt.Errorf("start_date and end_date required")
	}
	if endDate.Before(startDate) {
		return nil, fmt.Errorf("date_end before date_start")
	}

	target, err := s.GetRange(req.GroupID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	locationMeta, err := s.locationMetaForDays(target.Days)
	if err != nil {
		return nil, err
	}
	occupied, err := s.occupiedLocationSlots(startDate, endDate)
	if err != nil {
		return nil, err
	}

	resp := &LocationAutofillResponse{
		GroupID:     req.GroupID,
		DateStart:   startDate.Format("2006-01-02"),
		DateEnd:     endDate.Format("2006-01-02"),
		DryRun:      req.DryRun,
		DataVersion: target.DataVersion,
		Assignments: []LocationAutofillAssignment{},
	}

	candidatesByWeek := map[string][]Location{}
	getCandidates := func(day time.Time) ([]Location, error) {
		week := mondayOfWeek(day)
		key := week.Format("2006-01-02")
		if rows, ok := candidatesByWeek[key]; ok {
			return rows, nil
		}
		rows, err := s.repo.ListAvailableLocationsForWeek(week, req.Campus, req.LocationKind)
		if err != nil {
			return nil, err
		}
		candidatesByWeek[key] = rows
		return rows, nil
	}

	for _, day := range target.Days {
		dayDate, err := time.Parse("2006-01-02", day.Date)
		if err != nil {
			return nil, err
		}
		for _, lesson := range day.Lessons {
			if !locationNeedsAutofill(lesson, locationMeta, req.ReplaceVirtual) {
				continue
			}
			if lesson.SubjectID == nil {
				resp.Skipped++
				resp.Assignments = append(resp.Assignments, LocationAutofillAssignment{
					Date:       day.Date,
					PairNumber: lesson.PairNumber,
					Subgroup:   lesson.Subgroup,
					Status:     "skipped",
					Reason:     "lesson_without_subject",
				})
				continue
			}

			candidates, err := getCandidates(dayDate)
			if err != nil {
				return nil, err
			}
			chosen := chooseFreeLocation(day.Date, lesson.PairNumber, candidates, occupied)
			if chosen == nil {
				resp.Skipped++
				resp.Assignments = append(resp.Assignments, LocationAutofillAssignment{
					Date:        day.Date,
					PairNumber:  lesson.PairNumber,
					Subgroup:    lesson.Subgroup,
					SubjectID:   lesson.SubjectID,
					SubjectName: lesson.SubjectName,
					Status:      "skipped",
					Reason:      "no_available_location",
				})
				continue
			}

			locationID := chosen.ID
			occupied[locationAutofillSlot{Date: day.Date, PairNumber: lesson.PairNumber, LocationID: locationID}] = true
			resp.Assigned++
			resp.Assignments = append(resp.Assignments, LocationAutofillAssignment{
				Date:         day.Date,
				PairNumber:   lesson.PairNumber,
				Subgroup:     lesson.Subgroup,
				SubjectID:    lesson.SubjectID,
				SubjectName:  lesson.SubjectName,
				LocationID:   &locationID,
				LocationName: chosen.Name,
				Status:       "assigned",
			})
		}
	}

	if req.DryRun || resp.Assigned == 0 {
		return resp, nil
	}

	if err := s.repo.DB().Transaction(func(tx *gorm.DB) error {
		txRepo := NewRepository(tx)
		for i := range resp.Assignments {
			a := &resp.Assignments[i]
			if a.Status != "assigned" || a.LocationID == nil {
				continue
			}
			day, err := time.Parse("2006-01-02", a.Date)
			if err != nil {
				return err
			}
			mode, err := txRepo.UpsertLocationOverrideForSlot(req.GroupID, day, a.PairNumber, a.Subgroup, *a.LocationID, req.Comment)
			if err != nil {
				return err
			}
			switch mode {
			case "created":
				resp.Created++
				a.OverrideAction = "created"
			case "updated":
				resp.Updated++
				a.OverrideAction = "updated"
			case "blocked_cancel":
				resp.Assigned--
				resp.Skipped++
				a.Status = "skipped"
				a.Reason = "cancel_override_exists"
			default:
				return fmt.Errorf("unknown location override mode %q", mode)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if resp.Created > 0 || resp.Updated > 0 {
		if err := s.repo.BumpScheduleVersion(); err != nil {
			return nil, err
		}
		state, err := s.repo.GetSystemState()
		if err != nil {
			return nil, err
		}
		resp.DataVersion = state.ScheduleVersion.UTC().Format(time.RFC3339)
	}

	return resp, nil
}

func locationNeedsAutofill(lesson Lesson, locationMeta map[int]LocationMeta, replaceVirtual bool) bool {
	if lesson.LocationID == nil || *lesson.LocationID <= 0 {
		return true
	}
	if !replaceVirtual {
		return false
	}
	meta := locationMeta[*lesson.LocationID]
	return strings.EqualFold(strings.TrimSpace(meta.LocationKind), "virtual")
}

func chooseFreeLocation(date string, pairNumber int16, candidates []Location, occupied map[locationAutofillSlot]bool) *Location {
	for i := range candidates {
		slot := locationAutofillSlot{Date: date, PairNumber: pairNumber, LocationID: candidates[i].ID}
		if occupied[slot] {
			continue
		}
		return &candidates[i]
	}
	return nil
}

func (s *Service) occupiedLocationSlots(startDate, endDate time.Time) (map[locationAutofillSlot]bool, error) {
	groups, err := s.repo.ListGroups()
	if err != nil {
		return nil, err
	}
	out := map[locationAutofillSlot]bool{}
	for _, group := range groups {
		week, err := s.GetRange(group.ID, startDate, endDate)
		if err != nil {
			return nil, err
		}
		for _, day := range week.Days {
			for _, lesson := range day.Lessons {
				if lesson.LocationID == nil || *lesson.LocationID <= 0 || lesson.PairNumber <= 0 {
					continue
				}
				out[locationAutofillSlot{Date: day.Date, PairNumber: lesson.PairNumber, LocationID: *lesson.LocationID}] = true
			}
		}
	}
	return out, nil
}
