//go:build integration

package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ispo-schedule/internal/schedule"
)

func TestServiceGetRange_StudyDayStateAcademicWeekAndDayOverride(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	svc := schedule.NewService(schedule.ServiceDeps{Repo: repo, SemesterStartDate: "2026-09-01", Now: time.Now})

	spec := &schedule.Specialty{Code: shortCurriculaSpecialtyCode("SDS"), Name: "Study Day State Spec"}
	require.NoError(t, db.Create(spec).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", spec.ID).Delete(&schedule.Specialty{}).Error })

	curr := &schedule.Curriculum{SpecialtyID: spec.ID, AdmissionYear: 2026, Variant: "state", Title: "Study Day State", IsActive: true}
	require.NoError(t, db.Create(curr).Error)
	t.Cleanup(func() { _ = db.Exec("DELETE FROM curricula WHERE id = ?", curr.ID).Error })

	ac := &schedule.AcademicCalendar{CurriculumID: curr.ID, AcademicYearStart: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), WeeksTotal: 52}
	require.NoError(t, repo.CreateAcademicCalendar(ac))

	group := &schedule.Group{Name: fmt.Sprintf("study-day-state-%d", time.Now().UnixNano()), Course: 1, CurriculumID: &curr.ID}
	require.NoError(t, db.Create(group).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", group.ID).Delete(&schedule.Group{}).Error })

	_, err := repo.UpsertAcademicCalendarWeeks(ac.ID, []schedule.AcademicCalendarWeek{{
		CourseNumber:  1,
		WeekNumber:    1,
		WeekStartDate: time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
		ActivityCode:  "PRACTICE",
		ActivityName:  ptrString("Учебная практика"),
		IsTeaching:    false,
	}})
	require.NoError(t, err)
	require.NoError(t, repo.CreateAcademicCalendarDayOverride(&schedule.AcademicCalendarDayOverride{
		CalendarID:   ac.ID,
		CourseNumber: 1,
		WeekNumber:   1,
		DayOfWeek:    4,
		ActivityCode: "TEACHING",
		ActivityName: ptrString("Учебные занятия"),
		IsTeaching:   true,
	}))

	resp, err := svc.GetRange(group.ID, time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC), time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, resp.Days, 2)

	require.NotNil(t, resp.Days[0].StudyDayState)
	assert.Equal(t, "PRACTICE", resp.Days[0].StudyDayState.ActivityCode)
	assert.False(t, resp.Days[0].StudyDayState.IsTeaching)
	assert.Equal(t, "academic_week", resp.Days[0].StudyDayState.Source)

	require.NotNil(t, resp.Days[1].StudyDayState)
	assert.Equal(t, "TEACHING", resp.Days[1].StudyDayState.ActivityCode)
	assert.True(t, resp.Days[1].StudyDayState.IsTeaching)
	assert.Equal(t, "academic_day_override", resp.Days[1].StudyDayState.Source)
}
