package schedule

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// Teachers

func (r *Repository) ListTeachers() ([]Teacher, error) {
	var rows []Teacher
	err := r.db.Order("id asc").Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateTeacher(t *Teacher) error {
	return r.db.Create(t).Error
}

func (r *Repository) UpdateTeacher(id int, patch *Teacher) (*Teacher, error) {
	var row Teacher
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.Name = patch.Name
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteTeacher(id int) error {
	return r.db.Delete(&Teacher{}, id).Error
}

// Teacher subjects

type TeacherSubjectFilters struct {
	TeacherID *int
	SubjectID *int
}

func (r *Repository) ListTeacherSubjects(filters TeacherSubjectFilters) ([]TeacherSubject, error) {
	q := r.db.Model(&TeacherSubject{}).Order("teacher_id asc, subject_id asc")
	if filters.TeacherID != nil {
		q = q.Where("teacher_id = ?", *filters.TeacherID)
	}
	if filters.SubjectID != nil {
		q = q.Where("subject_id = ?", *filters.SubjectID)
	}
	var rows []TeacherSubject
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateTeacherSubject(ts *TeacherSubject) error {
	if ts == nil {
		return fmt.Errorf("teacher_subject is nil")
	}
	// idempotent insert
	return r.db.Exec(
		"INSERT INTO teacher_subjects (teacher_id, subject_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
		ts.TeacherID, ts.SubjectID,
	).Error
}

func (r *Repository) DeleteTeacherSubject(teacherID, subjectID int) error {
	return r.db.Where("teacher_id = ? AND subject_id = ?", teacherID, subjectID).Delete(&TeacherSubject{}).Error
}

// Course assignments

type CourseAssignmentFilters struct {
	GroupID   *int
	Semester  *int16
	SubjectID *int
	TeacherID *int
}

func (r *Repository) ListCourseAssignments(filters CourseAssignmentFilters) ([]CourseAssignment, error) {
	q := r.db.Model(&CourseAssignment{}).Order("group_id asc, semester asc, subject_id asc, COALESCE(subgroup, 0) asc, id asc")
	if filters.GroupID != nil {
		q = q.Where("group_id = ?", *filters.GroupID)
	}
	if filters.Semester != nil {
		q = q.Where("semester = ?", *filters.Semester)
	}
	if filters.SubjectID != nil {
		q = q.Where("subject_id = ?", *filters.SubjectID)
	}
	if filters.TeacherID != nil {
		q = q.Where("teacher_id = ?", *filters.TeacherID)
	}
	var rows []CourseAssignment
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateCourseAssignment(a *CourseAssignment) error {
	if a == nil {
		return fmt.Errorf("assignment is nil")
	}
	return r.db.Create(a).Error
}

func (r *Repository) UpdateCourseAssignment(id int64, patch *CourseAssignment) (*CourseAssignment, error) {
	var row CourseAssignment
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.GroupID = patch.GroupID
	row.Semester = patch.Semester
	row.SubjectID = patch.SubjectID
	row.TeacherID = patch.TeacherID
	row.LocationID = patch.LocationID
	row.CurriculumItemID = patch.CurriculumItemID
	row.Subgroup = patch.Subgroup
	row.Notes = patch.Notes
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteCourseAssignment(id int64) error {
	return r.db.Delete(&CourseAssignment{}, id).Error
}

func (r *Repository) EnsureTeacherSubjectAllowed(teacherID int, subjectID int) error {
	var row TeacherSubject
	err := r.db.Where("teacher_id = ? AND subject_id = ?", teacherID, subjectID).First(&row).Error
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("teacher %d is not allowed for subject %d", teacherID, subjectID)
	}
	return err
}
