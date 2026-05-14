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
	TeacherName string
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
	if _, err := s.repo.GetGroup(groupID); err != nil {
		return nil, err
	}
	chainIDs, err := s.scheduleInheritanceChainIDs(groupID)
	if err != nil {
		return nil, err
	}

	startDate = dateOnly(startDate)
	endDate = dateOnly(endDate)

	overlays, err := s.repo.ListOverlaysBetween(groupID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	overlayText := map[string]string{}
	for _, o := range overlays {
		overlayText[o.TargetDate.Format("2006-01-02")] = o.Text
	}

	globalConstraints, err := s.repo.ListCalendarDayConstraintsBetween(startDate, endDate)
	if err != nil {
		return nil, err
	}
	globalConstraintsByDay := map[string]CalendarDayConstraintView{}
	for _, c := range globalConstraints {
		globalConstraintsByDay[c.TargetDate.Format("2006-01-02")] = CalendarDayConstraintView{
			ID:             c.ID,
			Title:          c.Title,
			Reason:         c.Reason,
			ConstraintType: c.ConstraintType,
			StylePreset:    c.StylePreset,
		}
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

	if _, err := s.repo.ListTeachingWeeksForGroupBetween(groupID, startDate, endDate); err != nil {
		return nil, err
	}

	parityByDay := map[string]WeekParity{}
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		k := d.Format("2006-01-02")
		parityByDay[k] = s.weekParityForDate(d)
	}

	lessonsByDay := map[string]map[slotKey]ScheduleLessonView{}
	for _, gid := range chainIDs {
		rows, err := s.repo.ListScheduleLessonViewsBetween([]int{gid}, startDate, endDate, false)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			dayKey := dateOnly(r.LessonDate).Format("2006-01-02")
			m, ok := lessonsByDay[dayKey]
			if !ok {
				m = map[slotKey]ScheduleLessonView{}
				lessonsByDay[dayKey] = m
			}
			m[slotKey{PairNumber: r.PairNumber, Subgroup: subgroupKey(r.Subgroup)}] = r
		}
	}

	var out []DaySchedule
	for d := dateOnly(startDate); !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dayKey := d.Format("2006-01-02")
		dayOfWeek := dayOfWeekForDate(d, nil)

		parity := parityByDay[dayKey]

		lessons := make([]Lesson, 0, len(lessonsByDay[dayKey]))
		for _, row := range lessonsByDay[dayKey] {
			lessons = append(lessons, lessonFromScheduleLessonView(row))
		}

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

		var globalConstraint *CalendarDayConstraintView
		if c, ok := globalConstraintsByDay[dayKey]; ok {
			row := c
			globalConstraint = &row
		}

		out = append(out, DaySchedule{
			Date:                dayKey,
			DayOfWeek:           dayOfWeek,
			WeekParity:          parity,
			OverlayText:         overlayPtr,
			GlobalDayConstraint: globalConstraint,
			Events:              dEvents,
			Lessons:             lessons,
		})
	}

	return out, nil
}

func lessonFromScheduleLessonView(row ScheduleLessonView) Lesson {
	return Lesson{
		PairNumber:   row.PairNumber,
		SubjectID:    row.SubjectID,
		SubjectName:  row.SubjectName,
		LocationID:   row.LocationID,
		LocationName: row.LocationName,
		LessonFormat: normalizeLessonFormat(row.LessonFormat),
		TeacherName:  row.TeacherName,
		Subgroup:     row.Subgroup,
		FlowKey:      row.FlowKey,
		IsChanged:    row.Source == "replacement",
		Comment:      row.Comment,
	}
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
