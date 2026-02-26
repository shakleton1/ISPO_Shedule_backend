package schedule

import (
	"testing"
	"time"
)

func TestWeekParityForDate_SpecEvenIsDenominator(t *testing.T) {
	repo := &Repository{}                                                                        // not used by weekParityForDate
	svc := &Service{repo: repo, semesterStartDate: time.Date(2026, 2, 23, 0, 0, 0, 0, time.UTC)} // Monday

	// week_diff=0 => denominator
	got := svc.weekParityForDate(time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC))
	if got != WeekParityDenominator {
		t.Fatalf("expected %s, got %s", WeekParityDenominator, got)
	}

	// Within the same week should be same parity
	got = svc.weekParityForDate(time.Date(2026, 2, 26, 12, 0, 0, 0, time.UTC))
	if got != WeekParityDenominator {
		t.Fatalf("expected %s, got %s", WeekParityDenominator, got)
	}

	// Next week => numerator
	got = svc.weekParityForDate(time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC))
	if got != WeekParityNumerator {
		t.Fatalf("expected %s, got %s", WeekParityNumerator, got)
	}
}

func TestWeekParityForDate_ZeroSemesterStartMeansBoth(t *testing.T) {
	svc := &Service{semesterStartDate: time.Time{}}
	got := svc.weekParityForDate(time.Date(2026, 2, 26, 0, 0, 0, 0, time.UTC))
	if got != WeekParityBoth {
		t.Fatalf("expected %s, got %s", WeekParityBoth, got)
	}
}

func TestMondayOfWeek(t *testing.T) {
	// Sunday should map to previous Monday
	sun := time.Date(2026, 3, 1, 15, 0, 0, 0, time.UTC) // Sunday
	mon := mondayOfWeek(sun)
	if mon.Weekday() != time.Monday {
		t.Fatalf("expected Monday, got %s", mon.Weekday())
	}
	if mon.Format("2006-01-02") != "2026-02-23" {
		t.Fatalf("expected 2026-02-23, got %s", mon.Format("2006-01-02"))
	}
	if mon.Hour() != 0 || mon.Minute() != 0 || mon.Second() != 0 {
		t.Fatalf("expected dateOnly time, got %s", mon)
	}
}

func TestDayOfWeekForDate_WithCalendarException(t *testing.T) {
	// 2026-02-26 is Thursday => Mon=0 => Thu=3
	d := time.Date(2026, 2, 26, 12, 0, 0, 0, time.UTC)
	got := dayOfWeekForDate(d, map[string]int16{})
	if got != 3 {
		t.Fatalf("expected Thu=3, got %d", got)
	}

	worksAs := map[string]int16{"2026-02-26": 0} // works as Monday
	got = dayOfWeekForDate(d, worksAs)
	if got != 0 {
		t.Fatalf("expected works_as Monday=0, got %d", got)
	}
}
