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
	chainIDs, err := s.scheduleInheritanceChainIDs(groupID)
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
		// Published templates: inherit from schedule source chain, leaf wins.
		tplByKey := map[slotKey]TemplateView{}
		for _, gid := range chainIDs {
			rows, err := s.repo.ListTemplatesForStatus(gid, dayOfWeek, parity, StatusPublished)
			if err != nil {
				return nil, err
			}
			for _, t := range rows {
				tplByKey[slotKey{PairNumber: t.PairNumber, Subgroup: subgroupKey(t.Subgroup)}] = t
			}
		}
		tplsPub = make([]TemplateView, 0, len(tplByKey))
		for _, v := range tplByKey {
			tplsPub = append(tplsPub, v)
		}

		// Draft templates: only for this group.
		tplsDraft, err = s.repo.ListTemplatesForStatus(groupID, dayOfWeek, parity, StatusDraft)
		if err != nil {
			return nil, err
		}
	}

	ovrByKey := map[overrideKey]OverrideView{}
	for _, gid := range chainIDs {
		rows, err := s.repo.ListOverridesForDate(gid, date)
		if err != nil {
			return nil, err
		}
		for _, o := range rows {
			k := overrideKey{PairNumber: o.PairNumber, Subgroup: -1}
			if o.Subgroup != nil {
				k.Subgroup = *o.Subgroup
			}
			ovrByKey[k] = o
		}
	}
	ovrsRaw := make([]OverrideView, 0, len(ovrByKey))
	for _, v := range ovrByKey {
		ovrsRaw = append(ovrsRaw, v)
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
	assignmentRows := make([]CourseAssignmentTeacherView, 0)
	for i := len(chainIDs) - 1; i >= 0; i-- {
		rows, err := s.repo.ListCourseAssignmentTeachersForGroup(chainIDs[i])
		if err != nil {
			return nil, err
		}
		assignmentRows = append(assignmentRows, rows...)
	}
	assignmentsBySemester := map[int16]map[assignmentKey]assignmentResolve{}
	latestAssignments := map[assignmentKey]assignmentResolve{}
	for _, a := range assignmentRows {
		if (a.TeacherName == nil || *a.TeacherName == "") && a.LocationID == nil {
			continue
		}
		k := assignmentKey{SubjectID: a.SubjectID, Subgroup: subgroupKey(a.Subgroup)}
		m, ok := assignmentsBySemester[a.Semester]
		if !ok {
			m = map[assignmentKey]assignmentResolve{}
			assignmentsBySemester[a.Semester] = m
		}
		res := assignmentResolve{LocationID: a.LocationID}
		if a.TeacherName != nil {
			res.TeacherName = *a.TeacherName
		}
		if a.LocationName != nil {
			res.LocationName = *a.LocationName
		}
		if _, exists := m[k]; !exists {
			m[k] = res
		}
		if _, exists := latestAssignments[k]; !exists {
			latestAssignments[k] = res
		}
	}

	// Manual-empty suppression flags for teacher auto-resolve.
	tplManualEmpty := map[slotKey]bool{}
	for _, t := range tplsPub {
		if t.TeacherManual && t.TeacherName == "" {
			tplManualEmpty[slotKey{PairNumber: t.PairNumber, Subgroup: subgroupKey(t.Subgroup)}] = true
		}
	}
	ovrManualEmptyAll := map[int16]bool{}
	ovrManualEmpty := map[slotKey]bool{}
	for _, o := range ovrsNorm {
		if !o.NewTeacherManual {
			continue
		}
		if o.NewTeacherName != nil {
			continue
		}
		if o.Subgroup == nil {
			ovrManualEmptyAll[o.PairNumber] = true
			continue
		}
		ovrManualEmpty[slotKey{PairNumber: o.PairNumber, Subgroup: subgroupKey(o.Subgroup)}] = true
	}

	semester := inferSemesterForDate(date, group.Course)
	var semesterAssignments map[assignmentKey]assignmentResolve
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
		if ovrManualEmptyAll[merged[i].PairNumber] {
			continue
		}
		sgKey := subgroupKey(merged[i].Subgroup)
		if ovrManualEmpty[slotKey{PairNumber: merged[i].PairNumber, Subgroup: sgKey}] {
			continue
		}
		if tplManualEmpty[slotKey{PairNumber: merged[i].PairNumber, Subgroup: sgKey}] {
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
					resolved = v.TeacherName
					break
				}
			}
		}
		if resolved == "" {
			for _, k := range candidates {
				if v, ok := latestAssignments[k]; ok {
					resolved = v.TeacherName
					break
				}
			}
		}
		if resolved != "" {
			merged[i].TeacherName = resolved
		}
	}

	for i := range merged {
		if merged[i].LocationID != nil {
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

		var resolved *int
		var resolvedName string
		if semesterAssignments != nil {
			for _, k := range candidates {
				if v, ok := semesterAssignments[k]; ok && v.LocationID != nil {
					resolved = v.LocationID
					resolvedName = v.LocationName
					break
				}
			}
		}
		if resolved == nil {
			for _, k := range candidates {
				if v, ok := latestAssignments[k]; ok && v.LocationID != nil {
					resolved = v.LocationID
					resolvedName = v.LocationName
					break
				}
			}
		}
		if resolved != nil {
			merged[i].LocationID = resolved
			if resolvedName != "" {
				merged[i].LocationName = resolvedName
			}
		}
	}

	// Filter templates/overrides/lessons to requested slot.
	filterTpls := func(in []TemplateView) []TemplateView {
		out := make([]TemplateView, 0)
		for _, t := range in {
			if t.PairNumber != pairNumber {
				continue
			}
			// If a specific subgroup is requested, keep both:
			// - subgroup-specific entries for this subgroup
			// - whole-group entries (Subgroup == nil)
			if subgroup != nil && t.Subgroup != nil && *t.Subgroup != *subgroup {
				continue
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
			// Whole-group overrides (Subgroup == nil) are relevant for any subgroup.
			if subgroup != nil && o.Subgroup != nil && *o.Subgroup != *subgroup {
				continue
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
			// Whole-group lessons (Subgroup == nil) are relevant for any subgroup.
			if subgroup != nil && l.Subgroup != nil && *l.Subgroup != *subgroup {
				continue
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
