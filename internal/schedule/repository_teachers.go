package schedule

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// Teachers

func (r *Repository) ListTeachers() ([]Teacher, error) {
	var rows []Teacher
	err := r.db.Where("deleted_at IS NULL").Order("id asc").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListTeachersPaged(limit, offset *int) ([]Teacher, error) {
	var rows []Teacher
	q := r.db.Where("deleted_at IS NULL").Order("id asc")
	q = applyLimitOffset(q, limit, offset)
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) GetTeacher(id int) (*Teacher, error) {
	var row Teacher
	if err := r.db.Where("deleted_at IS NULL").First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) CreateTeacher(t *Teacher) error {
	if t == nil {
		return fmt.Errorf("teacher is nil")
	}
	if t.Name == "" {
		return fmt.Errorf("name required")
	}
	// Upsert by name_key and auto-restore if it was soft-deleted.
	var out struct {
		ID int `gorm:"column:id"`
	}
	err := r.db.Raw(
		"INSERT INTO teachers (name) VALUES (?) ON CONFLICT (name_key) DO UPDATE SET name = EXCLUDED.name, deleted_at = NULL RETURNING id",
		t.Name,
	).Scan(&out).Error
	if err != nil {
		return err
	}
	t.ID = out.ID
	return nil
}

func (r *Repository) UpdateTeacher(id int, patch *Teacher) (*Teacher, error) {
	if patch == nil {
		return nil, fmt.Errorf("teacher patch is nil")
	}
	if patch.Name == "" {
		return nil, fmt.Errorf("name required")
	}
	// Restore if soft-deleted.
	if err := r.db.Exec("UPDATE teachers SET name = ?, deleted_at = NULL WHERE id = ?", patch.Name, id).Error; err != nil {
		return nil, err
	}
	var row Teacher
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteTeacher(id int) error {
	res := r.db.Exec("UPDATE teachers SET deleted_at = now() WHERE id = ? AND deleted_at IS NULL", id)
	return res.Error
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

func (r *Repository) ListTeacherSubjectsPaged(filters TeacherSubjectFilters, limit, offset *int) ([]TeacherSubject, error) {
	q := r.db.Model(&TeacherSubject{}).Order("teacher_id asc, subject_id asc")
	if filters.TeacherID != nil {
		q = q.Where("teacher_id = ?", *filters.TeacherID)
	}
	if filters.SubjectID != nil {
		q = q.Where("subject_id = ?", *filters.SubjectID)
	}
	q = applyLimitOffset(q, limit, offset)
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
	CampusID  *int
	IsFlow    *bool
	Status    *EntityStatus
}

type CourseAssignmentTeacherView struct {
	ID          int64  `gorm:"column:id"`
	Semester    int16  `gorm:"column:semester"`
	SubjectID   int    `gorm:"column:subject_id"`
	Subgroup    *int16 `gorm:"column:subgroup"`
	TeacherName *string
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
	if filters.CampusID != nil {
		q = q.Where("campus_id = ?", *filters.CampusID)
	}
	if filters.IsFlow != nil {
		q = q.Where("is_flow = ?", *filters.IsFlow)
	}
	if filters.Status != nil {
		q = q.Where("status = ?", *filters.Status)
	} else {
		q = q.Where("status = ?", StatusPublished)
	}
	var rows []CourseAssignment
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) ListCourseAssignmentsPaged(filters CourseAssignmentFilters, limit, offset *int) ([]CourseAssignment, error) {
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
	if filters.CampusID != nil {
		q = q.Where("campus_id = ?", *filters.CampusID)
	}
	if filters.IsFlow != nil {
		q = q.Where("is_flow = ?", *filters.IsFlow)
	}
	if filters.Status != nil {
		q = q.Where("status = ?", *filters.Status)
	} else {
		q = q.Where("status = ?", StatusPublished)
	}
	q = applyLimitOffset(q, limit, offset)
	var rows []CourseAssignment
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateCourseAssignment(a *CourseAssignment) error {
	if a == nil {
		return fmt.Errorf("assignment is nil")
	}
	if a.Status == "" {
		a.Status = StatusPublished
	}
	if a.Status != StatusDraft && a.Status != StatusPublished {
		return fmt.Errorf("invalid status: %s", a.Status)
	}
	return r.db.Create(a).Error
}

func (r *Repository) UpdateCourseAssignment(id int64, patch *CourseAssignment) (*CourseAssignment, error) {
	var row CourseAssignment
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	if patch.Status == "" {
		patch.Status = row.Status
	}
	if patch.Status != StatusDraft && patch.Status != StatusPublished {
		return nil, fmt.Errorf("invalid status: %s", patch.Status)
	}
	row.GroupID = patch.GroupID
	row.Semester = patch.Semester
	row.SubjectID = patch.SubjectID
	row.Status = patch.Status
	row.TeacherID = patch.TeacherID
	row.CampusID = patch.CampusID
	row.IsFlow = patch.IsFlow
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

func (r *Repository) GetCourseAssignmentByID(id int64) (*CourseAssignment, error) {
	var row CourseAssignment
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
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

func (r *Repository) PublishDraftCourseAssignments(groupID int, semester *int16) (int64, error) {
	if groupID <= 0 {
		return 0, fmt.Errorf("group_id required")
	}
	var moved int64
	err := r.db.Transaction(func(tx *gorm.DB) error {
		whereSem := ""
		args := []any{groupID}
		if semester != nil {
			whereSem = " AND d.semester = ?"
			args = append(args, *semester)
		}

		// Delete matching published keys for each draft.
		if err := tx.Exec(`
			DELETE FROM course_assignments p
			USING course_assignments d
			WHERE d.group_id = ?
			  AND d.status = 'draft'
			`+whereSem+`
			  AND p.group_id = d.group_id
			  AND p.semester = d.semester
			  AND p.subject_id = d.subject_id
			  AND COALESCE(p.subgroup, 0) = COALESCE(d.subgroup, 0)
			  AND p.status = 'published'
		`, args...).Error; err != nil {
			return err
		}

		res := tx.Exec(`
			INSERT INTO course_assignments (group_id, semester, subject_id, status, teacher_id, campus_id, is_flow, curriculum_item_id, subgroup, notes, created_at, updated_at)
			SELECT group_id, semester, subject_id, 'published', teacher_id, campus_id, is_flow, curriculum_item_id, subgroup, notes, now(), now()
			FROM course_assignments
			WHERE group_id = ? AND status = 'draft'`+func() string {
			if semester != nil {
				return " AND semester = ?"
			}
			return ""
		}()+`
		`, args...)
		if res.Error != nil {
			return res.Error
		}
		moved = res.RowsAffected

		// Remove drafts after publish.
		delArgs := []any{groupID}
		delWhere := ""
		if semester != nil {
			delWhere = " AND semester = ?"
			delArgs = append(delArgs, *semester)
		}
		if err := tx.Exec(`DELETE FROM course_assignments WHERE group_id = ? AND status = 'draft'`+delWhere, delArgs...).Error; err != nil {
			return err
		}
		return nil
	})
	return moved, err
}

func (r *Repository) DiscardDraftCourseAssignments(groupID int, semester *int16) (int64, error) {
	if groupID <= 0 {
		return 0, fmt.Errorf("group_id required")
	}
	q := r.db.Table("course_assignments").Where("group_id = ? AND status = 'draft'", groupID)
	if semester != nil {
		q = q.Where("semester = ?", *semester)
	}
	res := q.Delete(&CourseAssignment{})
	return res.RowsAffected, res.Error
}

func (r *Repository) ListCourseAssignmentTeachersForGroup(groupID int) ([]CourseAssignmentTeacherView, error) {
	if groupID <= 0 {
		return nil, fmt.Errorf("group_id required")
	}

	var rows []CourseAssignmentTeacherView
	err := r.db.Table("course_assignments ca").
		Select("ca.id, ca.semester, ca.subject_id, ca.subgroup, t.name as teacher_name").
		Joins("LEFT JOIN teachers t ON t.id = ca.teacher_id").
		Where("ca.group_id = ? AND ca.status = ?", groupID, StatusPublished).
		Order("ca.subject_id asc, COALESCE(ca.subgroup, 0) asc, ca.semester desc, ca.id desc").
		Scan(&rows).Error
	return rows, err
}
