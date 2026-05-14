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

func TestApplyScheduleOverride_ReplaceUpdatesLessonRoomAndPublicSchedule(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	svc := schedule.NewService(schedule.ServiceDeps{Repo: repo, SemesterStartDate: "2026-09-01", Now: time.Now})

	group := &schedule.Group{Name: fmt.Sprintf("apply-replace-group-%d", time.Now().UnixNano()), Course: 1}
	oldSubject := &schedule.Subject{Name: fmt.Sprintf("apply-old-subject-%d", time.Now().UnixNano())}
	newSubject := &schedule.Subject{Name: fmt.Sprintf("apply-new-subject-%d", time.Now().UnixNano())}
	oldTeacher := &schedule.Teacher{Name: fmt.Sprintf("apply-old-teacher-%d", time.Now().UnixNano())}
	newTeacher := &schedule.Teacher{Name: fmt.Sprintf("apply-new-teacher-%d", time.Now().UnixNano())}
	oldRoom := &schedule.Location{Name: fmt.Sprintf("apply-old-room-%d", time.Now().UnixNano()), Kind: "physical", IsActive: true}
	newRoom := &schedule.Location{Name: fmt.Sprintf("apply-new-room-%d", time.Now().UnixNano()), Kind: "physical", IsActive: true}
	require.NoError(t, db.Create(group).Error)
	require.NoError(t, db.Create(oldSubject).Error)
	require.NoError(t, db.Create(newSubject).Error)
	require.NoError(t, db.Create(oldTeacher).Error)
	require.NoError(t, db.Create(newTeacher).Error)
	require.NoError(t, db.Create(oldRoom).Error)
	require.NoError(t, db.Create(newRoom).Error)
	t.Cleanup(func() {
		_ = db.Where("group_id = ?", group.ID).Delete(&schedule.ScheduleLesson{}).Error
		_ = db.Where("group_id = ?", group.ID).Delete(&schedule.ScheduleOverride{}).Error
		_ = db.Where("id = ?", group.ID).Delete(&schedule.Group{}).Error
		_ = db.Exec("UPDATE subjects SET deleted_at = now() WHERE id IN ?", []int{oldSubject.ID, newSubject.ID}).Error
		_ = db.Exec("UPDATE teachers SET deleted_at = now() WHERE id IN ?", []int{oldTeacher.ID, newTeacher.ID}).Error
		_ = db.Where("id IN ?", []int{oldRoom.ID, newRoom.ID}).Delete(&schedule.Location{}).Error
	})

	oldSubjectID := oldSubject.ID
	oldTeacherID := oldTeacher.ID
	lessonDate := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	lesson := &schedule.ScheduleLesson{
		GroupID:      group.ID,
		LessonDate:   lessonDate,
		PairNumber:   2,
		SubjectID:    &oldSubjectID,
		TeacherID:    &oldTeacherID,
		LessonFormat: "offline",
		Status:       schedule.StatusPublished,
		Source:       "manual",
	}
	require.NoError(t, repo.CreateScheduleLesson(lesson))
	require.NoError(t, repo.CreateRoomAssignment(&schedule.RoomAssignment{
		ScheduleLessonID: lesson.ID,
		LocationID:       oldRoom.ID,
		Source:           "manual",
		Status:           schedule.StatusPublished,
	}))

	expectedVersion := 1
	hybrid := "hybrid"
	reason := "replacement test"
	applied, err := svc.ApplyScheduleOverride(schedule.ApplyScheduleOverrideRequest{
		ScheduleLessonID:        &lesson.ID,
		GroupID:                 group.ID,
		LessonDate:              lessonDate,
		PairNumber:              2,
		ActionType:              string(schedule.OverrideReplace),
		ReplacementSubjectID:    &newSubject.ID,
		ReplacementTeacherID:    &newTeacher.ID,
		ReplacementLocationID:   &newRoom.ID,
		ReplacementLessonFormat: &hybrid,
		Reason:                  &reason,
		ExpectedLessonVersion:   &expectedVersion,
		ConfirmConstraints:      true,
	})
	require.NoError(t, err)
	assert.Equal(t, schedule.OverrideReplace, applied.ActionType)
	require.NotNil(t, applied.SourceSubjectID)
	assert.Equal(t, oldSubject.ID, *applied.SourceSubjectID)
	require.NotNil(t, applied.ReplacementSubjectID)
	assert.Equal(t, newSubject.ID, *applied.ReplacementSubjectID)

	updated, err := repo.GetScheduleLesson(lesson.ID)
	require.NoError(t, err)
	require.NotNil(t, updated.SubjectID)
	assert.Equal(t, newSubject.ID, *updated.SubjectID)
	require.NotNil(t, updated.TeacherID)
	assert.Equal(t, newTeacher.ID, *updated.TeacherID)
	assert.Equal(t, "hybrid", updated.LessonFormat)
	assert.Equal(t, 2, updated.Version)

	assignments, err := repo.ListRoomAssignments(schedule.RoomAssignmentFilters{ScheduleLessonID: &lesson.ID})
	require.NoError(t, err)
	require.Len(t, assignments, 1)
	assert.Equal(t, newRoom.ID, assignments[0].LocationID)

	rangeResp, err := svc.GetRange(group.ID, lessonDate, lessonDate)
	require.NoError(t, err)
	require.Len(t, rangeResp.Days, 1)
	require.Len(t, rangeResp.Days[0].Lessons, 1)
	assert.Equal(t, newSubject.Name, rangeResp.Days[0].Lessons[0].SubjectName)
	assert.True(t, rangeResp.Days[0].Lessons[0].IsChanged)

	_, err = svc.ApplyScheduleOverride(schedule.ApplyScheduleOverrideRequest{
		ScheduleLessonID:      &lesson.ID,
		GroupID:               group.ID,
		LessonDate:            lessonDate,
		PairNumber:            2,
		ActionType:            string(schedule.OverrideReplace),
		ReplacementSubjectID:  &oldSubject.ID,
		ExpectedLessonVersion: &expectedVersion,
		ConfirmConstraints:    true,
	})
	require.Error(t, err)
	assert.True(t, schedule.IsLessonVersionConflict(err))
}

func TestApplyScheduleOverride_CancelThenAddReusesSlotAndRoom(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	svc := schedule.NewService(schedule.ServiceDeps{Repo: repo, SemesterStartDate: "2026-09-01", Now: time.Now})

	group := &schedule.Group{Name: fmt.Sprintf("apply-cancel-group-%d", time.Now().UnixNano()), Course: 1}
	oldSubject := &schedule.Subject{Name: fmt.Sprintf("cancel-old-subject-%d", time.Now().UnixNano())}
	newSubject := &schedule.Subject{Name: fmt.Sprintf("cancel-new-subject-%d", time.Now().UnixNano())}
	room := &schedule.Location{Name: fmt.Sprintf("cancel-room-%d", time.Now().UnixNano()), Kind: "physical", IsActive: true}
	require.NoError(t, db.Create(group).Error)
	require.NoError(t, db.Create(oldSubject).Error)
	require.NoError(t, db.Create(newSubject).Error)
	require.NoError(t, db.Create(room).Error)
	t.Cleanup(func() {
		_ = db.Where("group_id = ?", group.ID).Delete(&schedule.ScheduleLesson{}).Error
		_ = db.Where("group_id = ?", group.ID).Delete(&schedule.ScheduleOverride{}).Error
		_ = db.Where("id = ?", group.ID).Delete(&schedule.Group{}).Error
		_ = db.Exec("UPDATE subjects SET deleted_at = now() WHERE id IN ?", []int{oldSubject.ID, newSubject.ID}).Error
		_ = db.Where("id = ?", room.ID).Delete(&schedule.Location{}).Error
	})

	oldSubjectID := oldSubject.ID
	lessonDate := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	lesson := &schedule.ScheduleLesson{
		GroupID:      group.ID,
		LessonDate:   lessonDate,
		PairNumber:   1,
		SubjectID:    &oldSubjectID,
		LessonFormat: "offline",
		Status:       schedule.StatusPublished,
		Source:       "manual",
	}
	require.NoError(t, repo.CreateScheduleLesson(lesson))
	require.NoError(t, repo.CreateRoomAssignment(&schedule.RoomAssignment{
		ScheduleLessonID: lesson.ID,
		LocationID:       room.ID,
		Source:           "manual",
		Status:           schedule.StatusPublished,
	}))

	expectedVersion := 1
	cancelled, err := svc.ApplyScheduleOverride(schedule.ApplyScheduleOverrideRequest{
		ScheduleLessonID:      &lesson.ID,
		GroupID:               group.ID,
		LessonDate:            lessonDate,
		PairNumber:            1,
		ActionType:            string(schedule.OverrideCancel),
		ExpectedLessonVersion: &expectedVersion,
	})
	require.NoError(t, err)
	assert.Equal(t, schedule.OverrideCancel, cancelled.ActionType)

	cancelledLesson, err := repo.GetScheduleLesson(lesson.ID)
	require.NoError(t, err)
	assert.Equal(t, schedule.StatusCancelled, cancelledLesson.Status)

	offline := "offline"
	added, err := svc.ApplyScheduleOverride(schedule.ApplyScheduleOverrideRequest{
		GroupID:                 group.ID,
		LessonDate:              lessonDate,
		PairNumber:              1,
		ActionType:              string(schedule.OverrideAdd),
		ReplacementSubjectID:    &newSubject.ID,
		ReplacementLocationID:   &room.ID,
		ReplacementLessonFormat: &offline,
	})
	require.NoError(t, err)
	assert.Equal(t, schedule.OverrideAdd, added.ActionType)
	require.NotNil(t, added.ScheduleLessonID)

	addedLesson, err := repo.GetScheduleLesson(*added.ScheduleLessonID)
	require.NoError(t, err)
	assert.Equal(t, schedule.StatusPublished, addedLesson.Status)
	require.NotNil(t, addedLesson.SubjectID)
	assert.Equal(t, newSubject.ID, *addedLesson.SubjectID)

	assignments, err := repo.ListRoomAssignments(schedule.RoomAssignmentFilters{ScheduleLessonID: added.ScheduleLessonID})
	require.NoError(t, err)
	require.Len(t, assignments, 1)
	assert.Equal(t, room.ID, assignments[0].LocationID)
}

func TestApplyScheduleOverride_TeacherWarningRequiresConfirmation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	svc := schedule.NewService(schedule.ServiceDeps{Repo: repo, SemesterStartDate: "2026-09-01", Now: time.Now})

	group := &schedule.Group{Name: fmt.Sprintf("apply-constraint-group-%d", time.Now().UnixNano()), Course: 1}
	subject := &schedule.Subject{Name: fmt.Sprintf("apply-constraint-subject-%d", time.Now().UnixNano())}
	teacher := &schedule.Teacher{Name: fmt.Sprintf("apply-constraint-teacher-%d", time.Now().UnixNano())}
	require.NoError(t, db.Create(group).Error)
	require.NoError(t, db.Create(subject).Error)
	require.NoError(t, db.Create(teacher).Error)
	t.Cleanup(func() {
		_ = db.Where("group_id = ?", group.ID).Delete(&schedule.ScheduleLesson{}).Error
		_ = db.Where("group_id = ?", group.ID).Delete(&schedule.ScheduleOverride{}).Error
		_ = db.Where("teacher_id = ?", teacher.ID).Delete(&schedule.TeacherDayConstraint{}).Error
		_ = db.Where("id = ?", group.ID).Delete(&schedule.Group{}).Error
		_ = db.Exec("UPDATE subjects SET deleted_at = now() WHERE id = ?", subject.ID).Error
		_ = db.Exec("UPDATE teachers SET deleted_at = now() WHERE id = ?", teacher.ID).Error
	})

	lessonDate := time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)
	require.NoError(t, repo.CreateTeacherDayConstraint(&schedule.TeacherDayConstraint{
		TeacherID:            teacher.ID,
		TargetDate:           lessonDate,
		Reason:               "method day",
		ConstraintLevel:      "warning",
		RequiresConfirmation: true,
	}))

	_, err := svc.ApplyScheduleOverride(schedule.ApplyScheduleOverrideRequest{
		GroupID:              group.ID,
		LessonDate:           lessonDate,
		PairNumber:           1,
		ActionType:           string(schedule.OverrideAdd),
		ReplacementSubjectID: &subject.ID,
		ReplacementTeacherID: &teacher.ID,
	})
	require.Error(t, err)
	var confirm *schedule.TeacherConstraintConfirmationRequiredError
	assert.True(t, errors.As(err, &confirm))

	_, err = svc.ApplyScheduleOverride(schedule.ApplyScheduleOverrideRequest{
		GroupID:              group.ID,
		LessonDate:           lessonDate,
		PairNumber:           1,
		ActionType:           string(schedule.OverrideAdd),
		ReplacementSubjectID: &subject.ID,
		ReplacementTeacherID: &teacher.ID,
		ConfirmConstraints:   true,
	})
	require.NoError(t, err)
}
