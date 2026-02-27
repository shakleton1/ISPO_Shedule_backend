package schedule

import (
	"fmt"
	"time"
)

type ExplainDecision struct {
	SemesterInferred *int16 `json:"semester_inferred"`
	NonTeaching      bool   `json:"non_teaching"`
}

type ExplainSlotResponse struct {
	GroupID     int        `json:"group_id"`
	Date        string     `json:"date"`
	DayOfWeek   int16      `json:"day_of_week"`
	WeekParity  WeekParity `json:"week_parity"`
	DataVersion string     `json:"data_version"`

	PairNumber int16  `json:"pair_number"`
	Subgroup   *int16 `json:"subgroup"`

	TemplatesPublished []TemplateView `json:"templates_published"`
	TemplatesDraft     []TemplateView `json:"templates_draft"`

	OverridesRaw        []OverrideView `json:"overrides_raw"`
	OverridesNormalized []OverrideView `json:"overrides_normalized"`

	Lessons []Lesson `json:"lessons"`

	Decision ExplainDecision `json:"decision"`
}

func (s *Service) ExplainSlot(groupID int, date time.Time, pairNumber int16, subgroup *int16) (*ExplainSlotResponse, error) {
	if groupID <= 0 {
		return nil, fmt.Errorf("group_id required")
	}
	if pairNumber <= 0 {
		return nil, fmt.Errorf("pair_number required")
	}

	group, err := s.repo.GetGroup(groupID)
	if err != nil {
		return nil, err
	}

	state, err := s.repo.GetSystemState()
	if err != nil {
		return nil, err
	}

	// Calendar exceptions for works-as day-of-week.
	exceptions, err := s.repo.ListCalendarExceptionsBetween(date, date)
	if err != nil {
		return nil, err
	}
	worksAs := map[string]int16{}
	for _, e := range exceptions {
		worksAs[e.TargetDate.Format("2006-01-02")] = e.WorksAsDay
	}

	dayOfWeek := dayOfWeekForDate(date, worksAs)
	parity := s.weekParityForDate(date)

	// Teaching weeks (if linked).
	nonTeaching := false
	teachingWeeks, err := s.repo.ListTeachingWeeksForGroupBetween(groupID, date, date)
	if err != nil {
		return nil, err
	}
	weekKey := mondayOfWeek(date).Format("2006-01-02")
	if v, ok := teachingWeeks[weekKey]; ok && !v {
		nonTeaching = true
	}

	var tplsPub, tplsDraft []TemplateView
	if !nonTeaching {
		tplsPub, err = s.repo.ListTemplatesForStatus(groupID, dayOfWeek, parity, StatusPublished)
		if err != nil {
			return nil, err
		}
		tplsDraft, err = s.repo.ListTemplatesForStatus(groupID, dayOfWeek, parity, StatusDraft)
		if err != nil {
			return nil, err
		}
	}

	ovrsRaw, err := s.repo.ListOverridesForDate(groupID, date)
	if err != nil {
		return nil, err
	}
	ovrsNorm := normalizeOverrides(ovrsRaw)

	// Build schedule lessons from published templates.
	var merged []Lesson
	if !nonTeaching {
		merged, err = mergeLessons(tplsPub, ovrsRaw)
		if err != nil {
			return nil, err
		}
	} else {
		merged = []Lesson{}
	}

	// Auto-fill teacher names from published course assignments (same logic as in buildDays).
	assignmentRows, err := s.repo.ListCourseAssignmentTeachersForGroup(groupID)
	if err != nil {
		return nil, err
	}
	assignmentsBySemester := map[int16]map[assignmentKey]string{}
	latestAssignments := map[assignmentKey]string{}
	for _, a := range assignmentRows {
		if a.TeacherName == nil || *a.TeacherName == "" {
			continue
		}
		k := assignmentKey{SubjectID: a.SubjectID, Subgroup: subgroupKey(a.Subgroup)}
		m, ok := assignmentsBySemester[a.Semester]
		if !ok {
			m = map[assignmentKey]string{}
			assignmentsBySemester[a.Semester] = m
		}
		if _, exists := m[k]; !exists {
			m[k] = *a.TeacherName
		}
		if _, exists := latestAssignments[k]; !exists {
			latestAssignments[k] = *a.TeacherName
		}
	}

	semester := inferSemesterForDate(date, group.Course)
	var semesterAssignments map[assignmentKey]string
	if semester != nil {
		semesterAssignments = assignmentsBySemester[*semester]
	}
	for i := range merged {
		if merged[i].TeacherName != "" {
			continue
		}
		if merged[i].SubjectID == nil {
			continue
		}

		var candidates []assignmentKey
		if merged[i].Subgroup == nil {
			candidates = []assignmentKey{{SubjectID: *merged[i].SubjectID, Subgroup: 0}}
		} else {
			candidates = []assignmentKey{
				{SubjectID: *merged[i].SubjectID, Subgroup: subgroupKey(merged[i].Subgroup)},
				{SubjectID: *merged[i].SubjectID, Subgroup: 0},
			}
		}

		var resolved string
		if semesterAssignments != nil {
			for _, k := range candidates {
				if v, ok := semesterAssignments[k]; ok {
					resolved = v
					break
				}
			}
		}
		if resolved == "" {
			for _, k := range candidates {
				if v, ok := latestAssignments[k]; ok {
					resolved = v
					break
				}
			}
		}
		if resolved != "" {
			merged[i].TeacherName = resolved
		}
	}

	// Filter templates/overrides/lessons to requested slot.
	filterTpls := func(in []TemplateView) []TemplateView {
		out := make([]TemplateView, 0)
		for _, t := range in {
			if t.PairNumber != pairNumber {
				continue
			}
			if subgroup != nil {
				if t.Subgroup != nil && *t.Subgroup != *subgroup {
					continue
				}
				if t.Subgroup == nil {
					// show whole-group template as relevant for any subgroup
				}
			}
			out = append(out, t)
		}
		return out
	}
	filterOvrs := func(in []OverrideView) []OverrideView {
		out := make([]OverrideView, 0)
		for _, o := range in {
			if o.PairNumber != pairNumber {
				continue
			}
			if subgroup != nil {
				if o.Subgroup != nil && *o.Subgroup != *subgroup {
					continue
				}
				if o.Subgroup == nil {
					// relevant to all
				}
			}
			out = append(out, o)
		}
		return out
	}
	filterLessons := func(in []Lesson) []Lesson {
		out := make([]Lesson, 0)
		for _, l := range in {
			if l.PairNumber != pairNumber {
				continue
			}
			if subgroup != nil {
				if l.Subgroup != nil && *l.Subgroup != *subgroup {
					continue
				}
				if l.Subgroup == nil {
					// keep whole-group lesson as relevant
				}
			}
			out = append(out, l)
		}
		return out
	}

	return &ExplainSlotResponse{
		GroupID:             groupID,
		Date:                dateOnly(date).Format("2006-01-02"),
		DayOfWeek:           dayOfWeek,
		WeekParity:          parity,
		DataVersion:         state.ScheduleVersion.UTC().Format(time.RFC3339),
		PairNumber:          pairNumber,
		Subgroup:            subgroup,
		TemplatesPublished:  filterTpls(tplsPub),
		TemplatesDraft:      filterTpls(tplsDraft),
		OverridesRaw:        filterOvrs(ovrsRaw),
		OverridesNormalized: filterOvrs(ovrsNorm),
		Lessons:             filterLessons(merged),
		Decision:            ExplainDecision{SemesterInferred: semester, NonTeaching: nonTeaching},
	}, nil
}
