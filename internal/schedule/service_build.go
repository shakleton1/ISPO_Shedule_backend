package schedule

import (
	"sort"
	"time"
)

func (s *Service) buildDays(groupID int, startDate, endDate time.Time) ([]DaySchedule, error) {
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

	var out []DaySchedule
	for d := dateOnly(startDate); !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dayKey := d.Format("2006-01-02")
		dayOfWeek := dayOfWeekForDate(d, worksAs)

		parity := s.weekParityForDate(d)

		tpls, err := s.repo.ListTemplatesFor(groupID, dayOfWeek, parity)
		if err != nil {
			return nil, err
		}

		ovrs, err := s.repo.ListOverridesForDate(groupID, d)
		if err != nil {
			return nil, err
		}

		lessons, err := mergeLessons(tpls, ovrs)
		if err != nil {
			return nil, err
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

		out = append(out, DaySchedule{
			Date:        dayKey,
			DayOfWeek:   dayOfWeek,
			WeekParity:  parity,
			OverlayText: overlayPtr,
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
