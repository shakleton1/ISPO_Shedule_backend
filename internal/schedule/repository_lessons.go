package schedule

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ScheduleLessonFilters struct {
	GroupID    *int
	TeacherID  *int
	LessonDate *time.Time
	StartDate  *time.Time
	EndDate    *time.Time
	Status     *EntityStatus
}

type ScheduleOverrideFilters struct {
	GroupID    *int
	TeacherID  *int
	StartDate  *time.Time
	EndDate    *time.Time
	ActionType *OverrideAction
}

func (r *Repository) ListScheduleLessons(filters ScheduleLessonFilters) ([]ScheduleLesson, error) {
	q := r.db.Model(&ScheduleLesson{}).
		Order("lesson_date asc, group_id asc, pair_number asc, COALESCE(subgroup, 0) asc, id asc")
	q = applyScheduleLessonFilters(q, filters)
	var rows []ScheduleLesson
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) ListScheduleLessonsPaged(filters ScheduleLessonFilters, limit, offset *int) ([]ScheduleLesson, error) {
	q := r.db.Model(&ScheduleLesson{}).
		Order("lesson_date asc, group_id asc, pair_number asc, COALESCE(subgroup, 0) asc, id asc")
	q = applyScheduleLessonFilters(q, filters)
	q = applyLimitOffset(q, limit, offset)
	var rows []ScheduleLesson
	err := q.Find(&rows).Error
	return rows, err
}

func applyScheduleLessonFilters(q *gorm.DB, filters ScheduleLessonFilters) *gorm.DB {
	if filters.GroupID != nil {
		q = q.Where("group_id = ?", *filters.GroupID)
	}
	if filters.TeacherID != nil {
		q = q.Where("teacher_id = ?", *filters.TeacherID)
	}
	if filters.LessonDate != nil {
		q = q.Where("lesson_date = ?", dateOnly(*filters.LessonDate))
	}
	if filters.StartDate != nil {
		q = q.Where("lesson_date >= ?", dateOnly(*filters.StartDate))
	}
	if filters.EndDate != nil {
		q = q.Where("lesson_date <= ?", dateOnly(*filters.EndDate))
	}
	if filters.Status != nil {
		q = q.Where("status = ?", *filters.Status)
	}
	return q
}

func (r *Repository) CreateScheduleLesson(row *ScheduleLesson) error {
	if err := normalizeScheduleLesson(row); err != nil {
		return err
	}
	return r.db.Create(row).Error
}

func (r *Repository) UpdateScheduleLesson(id int64, patch *ScheduleLesson, expectedVersion *int) (*ScheduleLesson, error) {
	if patch == nil {
		return nil, fmt.Errorf("schedule lesson patch is nil")
	}
	var out ScheduleLesson
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var row ScheduleLesson
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, id).Error; err != nil {
			return err
		}
		if expectedVersion != nil && row.Version != *expectedVersion {
			return ErrLessonVersionConflict
		}
		row.GroupID = patch.GroupID
		row.LessonDate = dateOnly(patch.LessonDate)
		row.PairNumber = patch.PairNumber
		row.Subgroup = patch.Subgroup
		row.SubjectID = patch.SubjectID
		row.TeacherID = patch.TeacherID
		row.LessonFormat = normalizeLessonFormat(patch.LessonFormat)
		row.Status = patch.Status
		row.Source = normalizeLessonSource(patch.Source)
		row.FlowKey = patch.FlowKey
		row.Comment = patch.Comment
		row.Version++
		if err := normalizeScheduleLesson(&row); err != nil {
			return err
		}
		if err := tx.Save(&row).Error; err != nil {
			return err
		}
		out = row
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *Repository) CancelScheduleLesson(id int64, expectedVersion *int) (*ScheduleLesson, error) {
	var out ScheduleLesson
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var row ScheduleLesson
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, id).Error; err != nil {
			return err
		}
		if expectedVersion != nil && row.Version != *expectedVersion {
			return ErrLessonVersionConflict
		}
		row.Status = StatusCancelled
		row.Version++
		if err := tx.Save(&row).Error; err != nil {
			return err
		}
		out = row
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *Repository) DeleteScheduleLesson(id int64) error {
	return r.db.Delete(&ScheduleLesson{}, id).Error
}

func (r *Repository) GetScheduleLesson(id int64) (*ScheduleLesson, error) {
	var row ScheduleLesson
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) ListScheduleLessonViewsBetween(groupIDs []int, startDate, endDate time.Time, includeCancelled bool) ([]ScheduleLessonView, error) {
	if len(groupIDs) == 0 {
		return []ScheduleLessonView{}, nil
	}
	q := r.db.Table("schedule_lessons sl").
		Select(`
sl.id,
sl.group_id,
sl.lesson_date,
sl.pair_number,
sl.subgroup,
sl.subject_id,
COALESCE(s.name, '') AS subject_name,
sl.teacher_id,
COALESCE(t.name, '') AS teacher_name,
ra.id AS room_assignment_id,
ra.location_id,
COALESCE(l.name, '') AS location_name,
sl.lesson_format,
sl.status,
sl.source,
sl.flow_key,
sl.comment,
sl.version`).
		Joins("LEFT JOIN subjects s ON s.id = sl.subject_id").
		Joins("LEFT JOIN teachers t ON t.id = sl.teacher_id").
		Joins("LEFT JOIN room_assignments ra ON ra.schedule_lesson_id = sl.id AND ra.status = ?", StatusPublished).
		Joins("LEFT JOIN locations l ON l.id = ra.location_id").
		Where("sl.group_id IN ? AND sl.lesson_date BETWEEN ? AND ?", groupIDs, dateOnly(startDate), dateOnly(endDate)).
		Order("sl.lesson_date asc, sl.pair_number asc, COALESCE(sl.subgroup, 0) asc, sl.group_id asc, sl.id asc")
	if !includeCancelled {
		q = q.Where("sl.status <> ?", StatusCancelled)
	}
	var rows []ScheduleLessonView
	err := q.Scan(&rows).Error
	return rows, err
}

func (r *Repository) ListScheduleOverrides(filters ScheduleOverrideFilters) ([]ScheduleOverride, error) {
	q := r.db.Model(&ScheduleOverride{}).Order("lesson_date asc, pair_number asc, id asc")
	q = applyScheduleOverrideFilters(q, filters)
	var rows []ScheduleOverride
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) ListScheduleOverridesPaged(filters ScheduleOverrideFilters, limit, offset *int) ([]ScheduleOverride, error) {
	q := r.db.Model(&ScheduleOverride{}).Order("lesson_date asc, pair_number asc, id asc")
	q = applyScheduleOverrideFilters(q, filters)
	q = applyLimitOffset(q, limit, offset)
	var rows []ScheduleOverride
	err := q.Find(&rows).Error
	return rows, err
}

func applyScheduleOverrideFilters(q *gorm.DB, filters ScheduleOverrideFilters) *gorm.DB {
	if filters.GroupID != nil {
		q = q.Where("group_id = ?", *filters.GroupID)
	}
	if filters.TeacherID != nil {
		q = q.Where("(source_teacher_id = ? OR replacement_teacher_id = ?)", *filters.TeacherID, *filters.TeacherID)
	}
	if filters.StartDate != nil {
		q = q.Where("lesson_date >= ?", dateOnly(*filters.StartDate))
	}
	if filters.EndDate != nil {
		q = q.Where("lesson_date <= ?", dateOnly(*filters.EndDate))
	}
	if filters.ActionType != nil {
		q = q.Where("action_type = ?", *filters.ActionType)
	}
	return q
}

func normalizeScheduleLesson(row *ScheduleLesson) error {
	if row == nil {
		return fmt.Errorf("schedule lesson is nil")
	}
	if row.GroupID <= 0 {
		return fmt.Errorf("group_id required")
	}
	row.LessonDate = dateOnly(row.LessonDate)
	if row.LessonDate.IsZero() {
		return fmt.Errorf("lesson_date required")
	}
	if row.PairNumber < 1 || row.PairNumber > 8 {
		return fmt.Errorf("pair_number must be 1..8")
	}
	if row.Subgroup != nil && (*row.Subgroup < 1 || *row.Subgroup > 2) {
		return fmt.Errorf("subgroup must be 1 or 2")
	}
	row.LessonFormat = normalizeLessonFormat(row.LessonFormat)
	if row.Status == "" {
		row.Status = StatusPublished
	}
	switch row.Status {
	case StatusDraft, StatusPublished, StatusCancelled:
	default:
		return fmt.Errorf("invalid status: %s", row.Status)
	}
	row.Source = normalizeLessonSource(row.Source)
	if row.Version <= 0 {
		row.Version = 1
	}
	if row.FlowKey != nil {
		v := strings.TrimSpace(*row.FlowKey)
		if v == "" {
			row.FlowKey = nil
		} else {
			row.FlowKey = &v
		}
	}
	return nil
}

func normalizeLessonSource(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "imported", "auto", "generated", "replacement":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return "manual"
	}
}

func normalizeOverrideAction(v OverrideAction) OverrideAction {
	switch OverrideAction(strings.ToLower(strings.TrimSpace(string(v)))) {
	case OverrideAdd:
		return OverrideAdd
	case OverrideReplace:
		return OverrideReplace
	case OverrideCancel:
		return OverrideCancel
	case OverrideRestore:
		return OverrideRestore
	default:
		return ""
	}
}

func normalizeOverrideStatus(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "draft", "cancelled":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return "applied"
	}
}

var ErrLessonVersionConflict = errors.New("lesson version conflict")
