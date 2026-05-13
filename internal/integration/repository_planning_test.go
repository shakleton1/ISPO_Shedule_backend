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

func TestRepositoryPlanning_StudyCalendarTeacherConstraintsAndReplacements(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	group := &schedule.Group{Name: "PLAN-" + fmt.Sprint(time.Now().UnixNano()), Course: 1}
	require.NoError(t, db.Create(group).Error)
	t.Cleanup(func() { _ = db.Where("id = ?", group.ID).Delete(&schedule.Group{}).Error })

	activity := &schedule.StudyActivity{
		Code:          "PR-" + fmt.Sprint(time.Now().UnixNano()),
		Name:          "Производственная практика",
		ActivityKind:  "PRACTICE",
		AllowsLessons: false,
	}
	require.NoError(t, repo.CreateStudyActivity(activity))
	t.Cleanup(func() { _ = db.Where("id = ?", activity.ID).Delete(&schedule.StudyActivity{}).Error })

	start := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	rows, err := repo.UpsertStudyCalendarWeeks(group.ID, []schedule.StudyCalendarWeek{{
		WeekNumber:    1,
		WeekStartDate: &start,
		ActivityID:    &activity.ID,
		AllowsLessons: false,
	}})
	require.NoError(t, err)
	require.NotEmpty(t, rows)
	t.Cleanup(func() { _ = db.Where("group_id = ?", group.ID).Delete(&schedule.StudyCalendarWeek{}).Error })

	teaching, err := repo.ListTeachingWeeksForGroupBetween(group.ID, start, start.AddDate(0, 0, 5))
	require.NoError(t, err)
	assert.False(t, teaching["2026-08-31"])

	teacher := &schedule.Teacher{Name: "Planning Teacher " + fmt.Sprint(time.Now().UnixNano())}
	require.NoError(t, repo.CreateTeacher(teacher))
	t.Cleanup(func() { _ = db.Exec("UPDATE teachers SET deleted_at = now() WHERE id = ?", teacher.ID).Error })

	constraint := &schedule.TeacherDayConstraint{
		TeacherID:            teacher.ID,
		TargetDate:           time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC),
		Reason:               "Больничный",
		ConstraintLevel:      "warning",
		RequiresConfirmation: true,
	}
	require.NoError(t, repo.CreateTeacherDayConstraint(constraint))
	t.Cleanup(func() { _ = db.Where("id = ?", constraint.ID).Delete(&schedule.TeacherDayConstraint{}).Error })

	blocked, err := repo.ListBlockingTeacherConstraintsBetween(constraint.TargetDate, constraint.TargetDate)
	require.NoError(t, err)
	require.NotEmpty(t, blocked)
	assert.Equal(t, teacher.ID, blocked[0].TeacherID)

	rep := &schedule.ScheduleReplacement{
		TargetDate: time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC),
		GroupID:    group.ID,
		PairNumber: 2,
		Reason:     ptrString("Замена преподавателя"),
	}
	require.NoError(t, repo.CreateScheduleReplacement(rep))
	t.Cleanup(func() { _ = db.Where("id = ?", rep.ID).Delete(&schedule.ScheduleReplacement{}).Error })

	reps, err := repo.ListScheduleReplacements(schedule.ScheduleReplacementFilters{GroupID: &group.ID})
	require.NoError(t, err)
	require.NotEmpty(t, reps)
	assert.Equal(t, int16(2), reps[0].PairNumber)
}
