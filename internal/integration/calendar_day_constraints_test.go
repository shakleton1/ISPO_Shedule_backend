//go:build integration

package integration

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ispo-schedule/internal/schedule"
)

func TestRepositoryCalendarDayConstraints_CRUDAndValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	targetDate := uniqueCalendarConstraintDate(0)
	t.Cleanup(func() {
		_ = db.Where("target_date BETWEEN ? AND ?", targetDate.AddDate(0, 0, -1), targetDate.AddDate(0, 0, 2)).Delete(&schedule.CalendarDayConstraint{}).Error
	})

	row := &schedule.CalendarDayConstraint{
		TargetDate:     targetDate,
		Title:          " Holiday ",
		Reason:         ptrString(" Seed reason "),
		ConstraintType: "blocked",
		AffectsLessons: true,
	}
	require.NoError(t, repo.CreateCalendarDayConstraint(row))
	assert.NotZero(t, row.ID)
	assert.Equal(t, "Holiday", row.Title)
	assert.Equal(t, "danger", row.StylePreset)
	require.NotNil(t, row.Reason)
	assert.Equal(t, "Seed reason", *row.Reason)

	duplicate := &schedule.CalendarDayConstraint{
		TargetDate:     targetDate,
		Title:          "Duplicate",
		ConstraintType: "info",
	}
	require.Error(t, repo.CreateCalendarDayConstraint(duplicate))

	got, err := repo.GetCalendarDayConstraintByDate(targetDate)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, row.ID, got.ID)

	updated, err := repo.UpdateCalendarDayConstraint(row.ID, &schedule.CalendarDayConstraint{
		TargetDate:           targetDate.AddDate(0, 0, 1),
		Title:                "Event",
		ConstraintType:       "warning",
		AffectsLessons:       true,
		RequiresConfirmation: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "warning", updated.ConstraintType)
	assert.Equal(t, "warning", updated.StylePreset)
	assert.True(t, updated.RequiresConfirmation)

	rows, err := repo.ListCalendarDayConstraintsBetween(targetDate, targetDate.AddDate(0, 0, 2))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, updated.ID, rows[0].ID)

	require.Error(t, repo.CreateCalendarDayConstraint(&schedule.CalendarDayConstraint{
		TargetDate:     targetDate.AddDate(0, 0, 2),
		Title:          "Bad type",
		ConstraintType: "bad",
	}))
	require.Error(t, repo.CreateCalendarDayConstraint(&schedule.CalendarDayConstraint{
		TargetDate:     targetDate.AddDate(0, 0, 2),
		Title:          "Bad style",
		ConstraintType: "info",
		StylePreset:    "purple",
	}))

	require.NoError(t, repo.DeleteCalendarDayConstraint(updated.ID))
	missing, err := repo.GetCalendarDayConstraintByDate(updated.TargetDate)
	require.NoError(t, err)
	assert.Nil(t, missing)
}

func TestScheduleRange_IncludesGlobalDayConstraintAndKeepsLessons(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	svc := schedule.NewService(schedule.ServiceDeps{Repo: repo, SemesterStartDate: "2026-09-01", Now: time.Now})

	targetDate := uniqueCalendarConstraintDate(10)
	group := &schedule.Group{Name: fmt.Sprintf("global-day-group-%d", time.Now().UnixNano()), Course: 1}
	subject := &schedule.Subject{Name: fmt.Sprintf("global-day-subject-%d", time.Now().UnixNano())}
	require.NoError(t, db.Create(group).Error)
	require.NoError(t, db.Create(subject).Error)
	t.Cleanup(func() {
		_ = db.Where("group_id = ?", group.ID).Delete(&schedule.ScheduleLesson{}).Error
		_ = db.Where("target_date = ?", targetDate).Delete(&schedule.CalendarDayConstraint{}).Error
		_ = db.Where("id = ?", group.ID).Delete(&schedule.Group{}).Error
		_ = db.Exec("DELETE FROM subjects WHERE id = ?", subject.ID).Error
	})

	require.NoError(t, repo.CreateCalendarDayConstraint(&schedule.CalendarDayConstraint{
		TargetDate:           targetDate,
		Title:                "Blocked day",
		Reason:               ptrString("Visible banner"),
		ConstraintType:       "blocked",
		AffectsLessons:       true,
		RequiresConfirmation: false,
		StylePreset:          "danger",
	}))
	require.NoError(t, repo.CreateScheduleLesson(&schedule.ScheduleLesson{
		GroupID:      group.ID,
		LessonDate:   targetDate,
		PairNumber:   1,
		SubjectID:    &subject.ID,
		LessonFormat: "offline",
		Status:       schedule.StatusPublished,
		Source:       "manual",
	}))

	resp, err := svc.GetRange(group.ID, targetDate, targetDate)
	require.NoError(t, err)
	require.Len(t, resp.Days, 1)
	require.NotNil(t, resp.Days[0].GlobalDayConstraint)
	assert.Equal(t, "Blocked day", resp.Days[0].GlobalDayConstraint.Title)
	assert.Equal(t, "danger", resp.Days[0].GlobalDayConstraint.StylePreset)
	require.Len(t, resp.Days[0].Lessons, 1)
	assert.Equal(t, subject.Name, resp.Days[0].Lessons[0].SubjectName)
}

func TestServiceCreateScheduleLesson_GlobalDayConstraints(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	svc := schedule.NewService(schedule.ServiceDeps{Repo: repo, SemesterStartDate: "2026-09-01", Now: time.Now})

	blockedDate := uniqueCalendarConstraintDate(20)
	warningDate := blockedDate.AddDate(0, 0, 1)
	group := &schedule.Group{Name: fmt.Sprintf("global-create-group-%d", time.Now().UnixNano()), Course: 1}
	subject := &schedule.Subject{Name: fmt.Sprintf("global-create-subject-%d", time.Now().UnixNano())}
	require.NoError(t, db.Create(group).Error)
	require.NoError(t, db.Create(subject).Error)
	t.Cleanup(func() {
		_ = db.Where("group_id = ?", group.ID).Delete(&schedule.ScheduleLesson{}).Error
		_ = db.Where("target_date IN ?", []time.Time{blockedDate, warningDate}).Delete(&schedule.CalendarDayConstraint{}).Error
		_ = db.Where("id = ?", group.ID).Delete(&schedule.Group{}).Error
		_ = db.Exec("DELETE FROM subjects WHERE id = ?", subject.ID).Error
	})

	require.NoError(t, repo.CreateCalendarDayConstraint(&schedule.CalendarDayConstraint{
		TargetDate:     blockedDate,
		Title:          "Closed",
		ConstraintType: "blocked",
		AffectsLessons: true,
		StylePreset:    "danger",
	}))
	require.NoError(t, repo.CreateCalendarDayConstraint(&schedule.CalendarDayConstraint{
		TargetDate:           warningDate,
		Title:                "Event",
		ConstraintType:       "warning",
		AffectsLessons:       true,
		RequiresConfirmation: true,
		StylePreset:          "warning",
	}))

	err := svc.CreateScheduleLesson(&schedule.ScheduleLesson{
		GroupID:      group.ID,
		LessonDate:   blockedDate,
		PairNumber:   1,
		SubjectID:    &subject.ID,
		LessonFormat: "offline",
	}, false, false)
	require.Error(t, err)
	var blocked *schedule.GlobalDayBlockedError
	assert.True(t, errors.As(err, &blocked))

	err = svc.CreateScheduleLesson(&schedule.ScheduleLesson{
		GroupID:      group.ID,
		LessonDate:   warningDate,
		PairNumber:   1,
		SubjectID:    &subject.ID,
		LessonFormat: "offline",
	}, false, false)
	require.Error(t, err)
	var confirm *schedule.GlobalDayConstraintConfirmationRequiredError
	assert.True(t, errors.As(err, &confirm))

	err = svc.CreateScheduleLesson(&schedule.ScheduleLesson{
		GroupID:      group.ID,
		LessonDate:   warningDate,
		PairNumber:   1,
		SubjectID:    &subject.ID,
		LessonFormat: "offline",
	}, false, true)
	require.NoError(t, err)
}

func TestApplyScheduleOverride_GlobalDayConstraints(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	svc := schedule.NewService(schedule.ServiceDeps{Repo: repo, SemesterStartDate: "2026-09-01", Now: time.Now})

	blockedDate := uniqueCalendarConstraintDate(30)
	warningDate := blockedDate.AddDate(0, 0, 1)
	group := &schedule.Group{Name: fmt.Sprintf("global-override-group-%d", time.Now().UnixNano()), Course: 1}
	oldSubject := &schedule.Subject{Name: fmt.Sprintf("global-override-old-%d", time.Now().UnixNano())}
	newSubject := &schedule.Subject{Name: fmt.Sprintf("global-override-new-%d", time.Now().UnixNano())}
	require.NoError(t, db.Create(group).Error)
	require.NoError(t, db.Create(oldSubject).Error)
	require.NoError(t, db.Create(newSubject).Error)
	t.Cleanup(func() {
		_ = db.Where("group_id = ?", group.ID).Delete(&schedule.ScheduleLesson{}).Error
		_ = db.Where("group_id = ?", group.ID).Delete(&schedule.ScheduleOverride{}).Error
		_ = db.Where("target_date IN ?", []time.Time{blockedDate, warningDate}).Delete(&schedule.CalendarDayConstraint{}).Error
		_ = db.Where("id = ?", group.ID).Delete(&schedule.Group{}).Error
		_ = db.Exec("DELETE FROM subjects WHERE id IN ?", []int{oldSubject.ID, newSubject.ID}).Error
	})

	require.NoError(t, repo.CreateCalendarDayConstraint(&schedule.CalendarDayConstraint{
		TargetDate:     blockedDate,
		Title:          "Closed",
		ConstraintType: "blocked",
		AffectsLessons: true,
		StylePreset:    "danger",
	}))
	require.NoError(t, repo.CreateCalendarDayConstraint(&schedule.CalendarDayConstraint{
		TargetDate:           warningDate,
		Title:                "Event",
		ConstraintType:       "warning",
		AffectsLessons:       true,
		RequiresConfirmation: true,
		StylePreset:          "warning",
	}))
	require.NoError(t, repo.CreateScheduleLesson(&schedule.ScheduleLesson{
		GroupID:      group.ID,
		LessonDate:   blockedDate,
		PairNumber:   1,
		SubjectID:    &oldSubject.ID,
		LessonFormat: "offline",
		Status:       schedule.StatusPublished,
		Source:       "manual",
	}))
	lesson, err := repo.ListScheduleLessons(schedule.ScheduleLessonFilters{GroupID: &group.ID, LessonDate: &blockedDate})
	require.NoError(t, err)
	require.Len(t, lesson, 1)

	_, err = svc.ApplyScheduleOverride(schedule.ApplyScheduleOverrideRequest{
		ScheduleLessonID:     &lesson[0].ID,
		GroupID:              group.ID,
		LessonDate:           blockedDate,
		PairNumber:           1,
		ActionType:           string(schedule.OverrideReplace),
		ReplacementSubjectID: &newSubject.ID,
	})
	require.Error(t, err)
	var blocked *schedule.GlobalDayBlockedError
	assert.True(t, errors.As(err, &blocked))

	_, err = svc.ApplyScheduleOverride(schedule.ApplyScheduleOverrideRequest{
		GroupID:              group.ID,
		LessonDate:           blockedDate,
		PairNumber:           2,
		ActionType:           string(schedule.OverrideAdd),
		ReplacementSubjectID: &newSubject.ID,
	})
	require.Error(t, err)
	assert.True(t, errors.As(err, &blocked))

	_, err = svc.ApplyScheduleOverride(schedule.ApplyScheduleOverrideRequest{
		GroupID:              group.ID,
		LessonDate:           warningDate,
		PairNumber:           1,
		ActionType:           string(schedule.OverrideAdd),
		ReplacementSubjectID: &newSubject.ID,
	})
	require.Error(t, err)
	var confirm *schedule.GlobalDayConstraintConfirmationRequiredError
	assert.True(t, errors.As(err, &confirm))

	_, err = svc.ApplyScheduleOverride(schedule.ApplyScheduleOverrideRequest{
		GroupID:                    group.ID,
		LessonDate:                 warningDate,
		PairNumber:                 1,
		ActionType:                 string(schedule.OverrideAdd),
		ReplacementSubjectID:       &newSubject.ID,
		ConfirmGlobalDayConstraint: true,
	})
	require.NoError(t, err)
}

func uniqueCalendarConstraintDate(offset int) time.Time {
	base := time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC)
	days := int(time.Now().UnixNano()%2000) + offset
	return base.AddDate(0, 0, days)
}
