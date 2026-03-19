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

func TestRepositoryTeachers_GetOrCreateTeacherIDSemantics(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	name := fmt.Sprintf("teacher-upsert-%d", time.Now().UnixNano())

	t1 := &schedule.Teacher{Name: name}
	require.NoError(t, repo.CreateTeacher(t1))
	require.NotZero(t, t1.ID)
	t.Cleanup(func() { _ = db.Exec("UPDATE teachers SET deleted_at = now() WHERE id = ?", t1.ID).Error })

	t2 := &schedule.Teacher{Name: name}
	require.NoError(t, repo.CreateTeacher(t2))
	require.Equal(t, t1.ID, t2.ID)
}

func TestRepositoryTeachers_TeacherSubjectsCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	teacher := &schedule.Teacher{Name: fmt.Sprintf("teacher-subj-%d", time.Now().UnixNano())}
	subject := &schedule.Subject{Name: fmt.Sprintf("teacher-subj-name-%d", time.Now().UnixNano())}
	require.NoError(t, db.Create(teacher).Error)
	require.NoError(t, db.Create(subject).Error)
	t.Cleanup(func() {
		_ = db.Where("teacher_id = ? AND subject_id = ?", teacher.ID, subject.ID).Delete(&schedule.TeacherSubject{}).Error
		_ = db.Exec("UPDATE teachers SET deleted_at = now() WHERE id = ?", teacher.ID).Error
		_ = db.Exec("UPDATE subjects SET deleted_at = now() WHERE id = ?", subject.ID).Error
	})

	require.NoError(t, repo.CreateTeacherSubject(&schedule.TeacherSubject{TeacherID: teacher.ID, SubjectID: subject.ID}))

	rows, err := repo.ListTeacherSubjects(schedule.TeacherSubjectFilters{TeacherID: &teacher.ID})
	require.NoError(t, err)
	require.NotEmpty(t, rows)
	assert.Equal(t, teacher.ID, rows[0].TeacherID)
	assert.Equal(t, subject.ID, rows[0].SubjectID)

	require.NoError(t, repo.DeleteTeacherSubject(teacher.ID, subject.ID))
	after, err := repo.ListTeacherSubjects(schedule.TeacherSubjectFilters{TeacherID: &teacher.ID})
	require.NoError(t, err)
	assert.Empty(t, after)
}
