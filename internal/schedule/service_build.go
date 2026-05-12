package schedule

import (
	"sort"
	"time"
)

type assignmentKey struct {
	SubjectID int
	Subgroup  int16 // 0 means NULL
}

type assignmentResolve struct {
	TeacherName  string
	LocationID   *int
	LocationName string
}

type slotKey struct {
	PairNumber int16
	Subgroup   int16 // 0 means NULL
}

func subgroupKey(subgroup *int16) int16 {
	if subgroup == nil {
		return 0
	}
	return *subgroup
}

func inferSemesterForDate(d time.Time, course int) *int16 {
	if course <= 0 || course > 6 {
		return nil
	}

	var sem int16
	switch d.Month() {
	case time.September, time.October, time.November, time.December:
		sem = int16((course-1)*2 + 1)
	case time.January, time.February, time.March, time.April, time.May, time.June:
		sem = int16((course-1)*2 + 2)
	default:
		return nil
	}
	if sem < 1 || sem > 12 {
		return nil
	}
	return &sem
}

func (s *Service) buildDays(groupID int, startDate, endDate time.Time) ([]DaySchedule, error) {
	group, err := s.repo.GetGroup(groupID)
	if err != nil {
		return nil, err
	}
	chainIDs, err := s.scheduleInheritanceChainIDs(groupID)
	if err != nil {
		return nil, err
	}

	startDate = dateOnly(startDate)
	endDate = dateOnly(endDate)

	assignmentRows := make([]CourseAssignmentTeacherView, 0)
	// Process leaf->base so this group's assignments win.
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

	// Preload calendar exceptions for range
	exceptions, err := s.repo.ListCalendarExceptionsBetween(startDate, endDate)
	if err != nil {
		return nil, err
	}
	worksAs := map[string]int16{}
	for _, e := range exceptions {
		worksAs[e.TargetDate.Format("2006-01-02")] = e.WorksAsDay
	}

	overlays, err := s.repo.ListOverlaysBetween(groupID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	overlayText := map[string]string{}
	for _, o := range overlays {
		overlayText[o.TargetDate.Format("2006-01-02")] = o.Text
	}

	events := make([]dayEventViewRow, 0)
	for _, gid := range chainIDs {
		rows, err := s.repo.ListDayEventsBetween(gid, startDate, endDate)
		if err != nil {
			return nil, err
		}
		events = append(events, rows...)
	}
	eventsByDay := map[string][]DayEvent{}
	for _, e := range events {
		dayKey := e.TargetDate.Format("2006-01-02")
		locationName := e.LocationName
		var locNamePtr *string
		if locationName != "" {
			locNamePtr = &locationName
		}
		eventsByDay[dayKey] = append(eventsByDay[dayKey], DayEvent{
			ID:           e.ID,
			EventType:    e.EventType,
			Title:        e.Title,
			Details:      e.Details,
			LocationID:   e.LocationID,
			LocationName: locNamePtr,
		})
	}

	// Preload academic calendar weeks (if group is linked to a curriculum).
	teachingWeeks, err := s.repo.ListTeachingWeeksForGroupBetween(groupID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Precompute parity per date and set of parities used in the range.
	parityByDay := map[string]WeekParity{}
	paritySet := map[WeekParity]bool{}
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		k := d.Format("2006-01-02")
		p := s.weekParityForDate(d)
		parityByDay[k] = p
		paritySet[p] = true
	}
	parities := make([]WeekParity, 0, len(paritySet))
	for p := range paritySet {
		parities = append(parities, p)
	}

	// Batch load templates (per group in chain, per parity used).
	tplByGroupParityDay := map[int]map[WeekParity]map[int16][]TemplateView{}
	for _, gid := range chainIDs {
		pm := map[WeekParity]map[int16][]TemplateView{}
		for _, p := range parities {
			rows, err := s.repo.ListTemplatesForWeekStatus(gid, p, StatusPublished)
			if err != nil {
				return nil, err
			}
			byDay := map[int16][]TemplateView{}
			for _, r := range rows {
				byDay[r.DayOfWeek] = append(byDay[r.DayOfWeek], TemplateView{
					PairNumber:     r.PairNumber,
					SubjectID:      r.SubjectID,
					SubjectName:    r.SubjectName,
					LocationID:     r.LocationID,
					LocationName:   r.LocationName,
					TeacherName:    r.TeacherName,
					TeacherManual:  r.TeacherManual,
					LocationManual: r.LocationManual,
					Subgroup:       r.Subgroup,
					FlowKey:        r.FlowKey,
				})
			}
			pm[p] = byDay
		}
		tplByGroupParityDay[gid] = pm
	}

	// Batch load overrides (per group in chain, whole range), then resolve inheritance base->leaf.
	overridesByDay := map[string]map[overrideKey]OverrideView{}
	for _, gid := range chainIDs {
		rows, err := s.repo.ListOverridesBetween(gid, startDate, endDate)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			dayKey := dateOnly(r.TargetDate).Format("2006-01-02")
			m, ok := overridesByDay[dayKey]
			if !ok {
				m = map[overrideKey]OverrideView{}
				overridesByDay[dayKey] = m
			}
			k := overrideKey{PairNumber: r.PairNumber, Subgroup: -1}
			if r.Subgroup != nil {
				k.Subgroup = *r.Subgroup
			}
			m[k] = OverrideView{
				ID:               r.ID,
				PairNumber:       r.PairNumber,
				ActionType:       r.ActionType,
				NewSubjectID:     r.NewSubjectID,
				NewSubjectName:   r.NewSubjectName,
				NewLocationID:    r.NewLocationID,
				NewLocationName:  r.NewLocationName,
				NewTeacherManual: r.NewTeacherManual,
				NewTeacherName:   r.NewTeacherName,
				Comment:          r.Comment,
				Subgroup:         r.Subgroup,
				FlowKey:          r.FlowKey,
				UpdatedAt:        r.UpdatedAt,
			}
		}
	}

	var out []DaySchedule
	for d := dateOnly(startDate); !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dayKey := d.Format("2006-01-02")
		dayOfWeek := dayOfWeekForDate(d, worksAs)

		weekKey := mondayOfWeek(d).Format("2006-01-02")
		nonTeaching := false
		if v, ok := teachingWeeks[weekKey]; ok && !v {
			nonTeaching = true
		}

		parity := parityByDay[dayKey]

		var tpls []TemplateView
		if !nonTeaching {
			tplByKey := map[slotKey]TemplateView{}
			for _, gid := range chainIDs {
				byParity := tplByGroupParityDay[gid]
				byDay := byParity[parity]
				rows := byDay[dayOfWeek]
				for _, t := range rows {
					tplByKey[slotKey{PairNumber: t.PairNumber, Subgroup: subgroupKey(t.Subgroup)}] = t
				}
			}
			tpls = make([]TemplateView, 0, len(tplByKey))
			for _, v := range tplByKey {
				tpls = append(tpls, v)
			}
		}

		ovrByKey := overridesByDay[dayKey]
		ovrs := make([]OverrideView, 0, len(ovrByKey))
		for _, v := range ovrByKey {
			ovrs = append(ovrs, v)
		}

		lessons, err := mergeLessons(tpls, ovrs)
		if err != nil {
			return nil, err
		}

		ovrsNorm := normalizeOverrides(ovrs)

		// Manual-empty suppression flags for teacher auto-resolve.
		tplManualEmpty := map[slotKey]bool{}
		for _, t := range tpls {
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
			// If manual flag is set but new teacher is NULL => dispatcher explicitly cleared teacher.
			if o.NewTeacherName != nil {
				continue
			}
			if o.Subgroup == nil {
				ovrManualEmptyAll[o.PairNumber] = true
				continue
			}
			ovrManualEmpty[slotKey{PairNumber: o.PairNumber, Subgroup: subgroupKey(o.Subgroup)}] = true
		}

		semester := inferSemesterForDate(d, group.Course)
		var semesterAssignments map[assignmentKey]assignmentResolve
		if semester != nil {
			semesterAssignments = assignmentsBySemester[*semester]
		}
		for i := range lessons {
			if lessons[i].TeacherName != "" {
				continue
			}
			if lessons[i].SubjectID == nil {
				continue
			}
			if ovrManualEmptyAll[lessons[i].PairNumber] {
				continue
			}
			sgKey := subgroupKey(lessons[i].Subgroup)
			if ovrManualEmpty[slotKey{PairNumber: lessons[i].PairNumber, Subgroup: sgKey}] {
				continue
			}
			if tplManualEmpty[slotKey{PairNumber: lessons[i].PairNumber, Subgroup: sgKey}] {
				continue
			}

			// Match subgroup strictly for whole-group lessons; for subgroup lessons allow fallback to whole-group assignment.
			var candidates []assignmentKey
			if lessons[i].Subgroup == nil {
				candidates = []assignmentKey{{SubjectID: *lessons[i].SubjectID, Subgroup: 0}}
			} else {
				candidates = []assignmentKey{
					{SubjectID: *lessons[i].SubjectID, Subgroup: subgroupKey(lessons[i].Subgroup)},
					{SubjectID: *lessons[i].SubjectID, Subgroup: 0},
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
				lessons[i].TeacherName = resolved
			}
		}

		// Auto-fill location from course_assignments when missing (e.g. ADD override without new_location_id).
		for i := range lessons {
			if lessons[i].LocationID != nil {
				continue
			}
			if lessons[i].SubjectID == nil {
				continue
			}

			var candidates []assignmentKey
			if lessons[i].Subgroup == nil {
				candidates = []assignmentKey{{SubjectID: *lessons[i].SubjectID, Subgroup: 0}}
			} else {
				candidates = []assignmentKey{
					{SubjectID: *lessons[i].SubjectID, Subgroup: subgroupKey(lessons[i].Subgroup)},
					{SubjectID: *lessons[i].SubjectID, Subgroup: 0},
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
				lessons[i].LocationID = resolved
				if resolvedName != "" {
					lessons[i].LocationName = resolvedName
				}
			}
		}

		// Sort lessons by pair_number then subgroup
		sort.SliceStable(lessons, func(i, j int) bool {
			if lessons[i].PairNumber != lessons[j].PairNumber {
				return lessons[i].PairNumber < lessons[j].PairNumber
			}
			// nil subgroup first
			var a, b int16 = -1, -1
			if lessons[i].Subgroup != nil {
				a = *lessons[i].Subgroup
			}
			if lessons[j].Subgroup != nil {
				b = *lessons[j].Subgroup
			}
			return a < b
		})

		var overlayPtr *string
		if txt, ok := overlayText[dayKey]; ok {
			t := txt
			overlayPtr = &t
		}

		dEvents := eventsByDay[dayKey]
		if dEvents == nil {
			dEvents = []DayEvent{}
		}

		out = append(out, DaySchedule{
			Date:        dayKey,
			DayOfWeek:   dayOfWeek,
			WeekParity:  parity,
			OverlayText: overlayPtr,
			Events:      dEvents,
			Lessons:     lessons,
		})
	}

	return out, nil
}

func dayOfWeekForDate(d time.Time, worksAs map[string]int16) int16 {
	dayKey := dateOnly(d).Format("2006-01-02")
	dayOfWeek := int16((int(d.Weekday()) + 6) % 7) // Go: Sun=0 => convert to Mon=0
	if v, ok := worksAs[dayKey]; ok {
		return v
	}
	return dayOfWeek
}

func (s *Service) weekParityForDate(date time.Time) WeekParity {
	if s.semesterStartDate.IsZero() {
		return WeekParityBoth
	}
	weekStart := mondayOfWeek(date)
	semStart := mondayOfWeek(s.semesterStartDate)
	weeks := int(weekStart.Sub(semStart).Hours() / 24 / 7)
	// По спецификации: четное -> знаменатель, нечетное -> числитель
	if weeks%2 == 0 {
		return WeekParityDenominator
	}
	return WeekParityNumerator
}

func mondayOfWeek(t time.Time) time.Time {
	d := dateOnly(t)
	wd := int(d.Weekday())
	// Monday=1 ... Sunday=0
	offset := (wd + 6) % 7
	return d.AddDate(0, 0, -offset)
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
