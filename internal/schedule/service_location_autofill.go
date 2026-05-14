package schedule

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type LocationAutofillRequest struct {
	GroupID        int       `json:"group_id"`
	StartDate      time.Time `json:"-"`
	EndDate        time.Time `json:"-"`
	CampusName     *string   `json:"campus,omitempty"`
	LocationType   *string   `json:"location_type_code,omitempty"`
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
	ScheduleLessonID int64  `json:"schedule_lesson_id"`
	Date             string `json:"date"`
	PairNumber       int16  `json:"pair_number"`
	Subgroup         *int16 `json:"subgroup"`
	SubjectID        *int   `json:"subject_id"`
	SubjectName      string `json:"subject_name"`
	LocationID       *int   `json:"location_id"`
	LocationName     string `json:"location_name"`
	Status           string `json:"status"`
	Reason           string `json:"reason,omitempty"`
	AssignmentAction string `json:"assignment_action,omitempty"`
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

	rows, err := s.repo.ListScheduleLessonViewsBetween([]int{req.GroupID}, startDate, endDate, false)
	if err != nil {
		return nil, err
	}
	locationMeta, err := s.locationMetaForLessonViews(rows)
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
		Assignments: []LocationAutofillAssignment{},
	}
	state, err := s.repo.GetSystemState()
	if err != nil {
		return nil, err
	}
	resp.DataVersion = state.ScheduleVersion.UTC().Format(time.RFC3339)

	candidatesByWeek := map[string][]Location{}
	getCandidates := func(day time.Time) ([]Location, error) {
		week := mondayOfWeek(day)
		key := week.Format("2006-01-02")
		if rows, ok := candidatesByWeek[key]; ok {
			return rows, nil
		}
		rows, err := s.repo.ListAvailableLocationsForWeek(week, req.CampusName, req.LocationType)
		if err != nil {
			return nil, err
		}
		candidatesByWeek[key] = rows
		return rows, nil
	}

	for _, lesson := range rows {
		dayDate := dateOnly(lesson.LessonDate)
		dayKey := dayDate.Format("2006-01-02")
		if !lessonViewNeedsAutofill(lesson, locationMeta, req.ReplaceVirtual) {
			continue
		}
		if lesson.SubjectID == nil {
			resp.Skipped++
			resp.Assignments = append(resp.Assignments, LocationAutofillAssignment{
				ScheduleLessonID: lesson.ID,
				Date:             dayKey,
				PairNumber:       lesson.PairNumber,
				Subgroup:         lesson.Subgroup,
				Status:           "skipped",
				Reason:           "lesson_without_subject",
			})
			continue
		}

		candidates, err := getCandidates(dayDate)
		if err != nil {
			return nil, err
		}
		chosen := chooseFreeLocation(dayKey, lesson.PairNumber, candidates, occupied)
		if chosen == nil {
			resp.Skipped++
			resp.Assignments = append(resp.Assignments, LocationAutofillAssignment{
				ScheduleLessonID: lesson.ID,
				Date:             dayKey,
				PairNumber:       lesson.PairNumber,
				Subgroup:         lesson.Subgroup,
				SubjectID:        lesson.SubjectID,
				SubjectName:      lesson.SubjectName,
				Status:           "skipped",
				Reason:           "no_available_location",
			})
			continue
		}

		locationID := chosen.ID
		occupied[locationAutofillSlot{Date: dayKey, PairNumber: lesson.PairNumber, LocationID: locationID}] = true
		resp.Assigned++
		resp.Assignments = append(resp.Assignments, LocationAutofillAssignment{
			ScheduleLessonID: lesson.ID,
			Date:             dayKey,
			PairNumber:       lesson.PairNumber,
			Subgroup:         lesson.Subgroup,
			SubjectID:        lesson.SubjectID,
			SubjectName:      lesson.SubjectName,
			LocationID:       &locationID,
			LocationName:     chosen.Name,
			Status:           "assigned",
		})
	}

	if req.DryRun || resp.Assigned == 0 {
		return resp, nil
	}

	if err := s.repo.DB().Transaction(func(tx *gorm.DB) error {
		for i := range resp.Assignments {
			a := &resp.Assignments[i]
			if a.Status != "assigned" || a.LocationID == nil {
				continue
			}
			existing, err := getRoomAssignmentForLesson(tx, a.ScheduleLessonID)
			if err != nil {
				return err
			}
			if _, err := upsertRoomAssignmentForLesson(tx, a.ScheduleLessonID, *a.LocationID, "auto"); err != nil {
				return err
			}
			if existing == nil {
				resp.Created++
				a.AssignmentAction = "created"
			} else {
				resp.Updated++
				a.AssignmentAction = "updated"
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

func lessonViewNeedsAutofill(lesson ScheduleLessonView, locationMeta map[int]LocationMeta, replaceVirtual bool) bool {
	if lesson.LocationID == nil || *lesson.LocationID <= 0 {
		return true
	}
	if !replaceVirtual {
		return false
	}
	meta := locationMeta[*lesson.LocationID]
	return meta.IsVirtual()
}

func locationNeedsAutofill(lesson Lesson, locationMeta map[int]LocationMeta, replaceVirtual bool) bool {
	view := ScheduleLessonView{LocationID: lesson.LocationID}
	return lessonViewNeedsAutofill(view, locationMeta, replaceVirtual)
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
	out := map[locationAutofillSlot]bool{}
	var rows []struct {
		LessonDate time.Time `gorm:"column:lesson_date"`
		PairNumber int16     `gorm:"column:pair_number"`
		LocationID int       `gorm:"column:location_id"`
	}
	err := s.repo.DB().Table("schedule_lessons sl").
		Select("sl.lesson_date, sl.pair_number, ra.location_id").
		Joins("JOIN room_assignments ra ON ra.schedule_lesson_id = sl.id AND ra.status = ?", StatusPublished).
		Where("sl.lesson_date BETWEEN ? AND ? AND sl.status <> ?", dateOnly(startDate), dateOnly(endDate), StatusCancelled).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[locationAutofillSlot{Date: dateOnly(row.LessonDate).Format("2006-01-02"), PairNumber: row.PairNumber, LocationID: row.LocationID}] = true
	}
	return out, nil
}

func (s *Service) locationMetaForLessonViews(rows []ScheduleLessonView) (map[int]LocationMeta, error) {
	seen := map[int]bool{}
	ids := make([]int, 0)
	for _, lesson := range rows {
		if lesson.LocationID == nil || *lesson.LocationID <= 0 {
			continue
		}
		if seen[*lesson.LocationID] {
			continue
		}
		seen[*lesson.LocationID] = true
		ids = append(ids, *lesson.LocationID)
	}
	return s.repo.ListLocationMetaByIDs(ids)
}
