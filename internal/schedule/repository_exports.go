package schedule

import "time"

type ScheduleOverrideReportRow struct {
	ID                      int64          `gorm:"column:id"`
	ScheduleLessonID        *int64         `gorm:"column:schedule_lesson_id"`
	GroupID                 int            `gorm:"column:group_id"`
	GroupName               string         `gorm:"column:group_name"`
	LessonDate              time.Time      `gorm:"column:lesson_date"`
	PairNumber              int16          `gorm:"column:pair_number"`
	Subgroup                *int16         `gorm:"column:subgroup"`
	ActionType              OverrideAction `gorm:"column:action_type"`
	SourceSubjectID         *int           `gorm:"column:source_subject_id"`
	SourceSubjectName       string         `gorm:"column:source_subject_name"`
	SourceTeacherID         *int           `gorm:"column:source_teacher_id"`
	SourceTeacherName       string         `gorm:"column:source_teacher_name"`
	SourceLocationID        *int           `gorm:"column:source_location_id"`
	SourceLocationName      string         `gorm:"column:source_location_name"`
	SourceLessonFormat      *string        `gorm:"column:source_lesson_format"`
	ReplacementSubjectID    *int           `gorm:"column:replacement_subject_id"`
	ReplacementSubjectName  string         `gorm:"column:replacement_subject_name"`
	ReplacementTeacherID    *int           `gorm:"column:replacement_teacher_id"`
	ReplacementTeacherName  string         `gorm:"column:replacement_teacher_name"`
	ReplacementLocationID   *int           `gorm:"column:replacement_location_id"`
	ReplacementLocationName string         `gorm:"column:replacement_location_name"`
	ReplacementLessonFormat *string        `gorm:"column:replacement_lesson_format"`
	Reason                  *string        `gorm:"column:reason"`
	Status                  string         `gorm:"column:status"`
	ExpectedLessonVersion   *int           `gorm:"column:expected_lesson_version"`
	AppliedLessonVersion    *int           `gorm:"column:applied_lesson_version"`
	CreatedBy               *int           `gorm:"column:created_by"`
	CreatedAt               time.Time      `gorm:"column:created_at"`
	AppliedAt               *time.Time     `gorm:"column:applied_at"`
}

func (r *Repository) ListScheduleOverrideReportRows(filters ScheduleOverrideFilters) ([]ScheduleOverrideReportRow, error) {
	q := r.db.Table("schedule_overrides so").
		Select(`
so.id,
so.schedule_lesson_id,
so.group_id,
COALESCE(g.name, '') AS group_name,
so.lesson_date,
so.pair_number,
so.subgroup,
so.action_type,
so.source_subject_id,
COALESCE(src_subject.name, '') AS source_subject_name,
so.source_teacher_id,
COALESCE(src_teacher.name, '') AS source_teacher_name,
so.source_location_id,
COALESCE(src_location.name, '') AS source_location_name,
so.source_lesson_format,
so.replacement_subject_id,
COALESCE(repl_subject.name, '') AS replacement_subject_name,
so.replacement_teacher_id,
COALESCE(repl_teacher.name, '') AS replacement_teacher_name,
so.replacement_location_id,
COALESCE(repl_location.name, '') AS replacement_location_name,
so.replacement_lesson_format,
so.reason,
so.status,
so.expected_lesson_version,
so.applied_lesson_version,
so.created_by,
so.created_at,
so.applied_at`).
		Joins("LEFT JOIN groups g ON g.id = so.group_id").
		Joins("LEFT JOIN subjects src_subject ON src_subject.id = so.source_subject_id").
		Joins("LEFT JOIN teachers src_teacher ON src_teacher.id = so.source_teacher_id").
		Joins("LEFT JOIN locations src_location ON src_location.id = so.source_location_id").
		Joins("LEFT JOIN subjects repl_subject ON repl_subject.id = so.replacement_subject_id").
		Joins("LEFT JOIN teachers repl_teacher ON repl_teacher.id = so.replacement_teacher_id").
		Joins("LEFT JOIN locations repl_location ON repl_location.id = so.replacement_location_id").
		Order("so.lesson_date asc, so.pair_number asc, COALESCE(so.subgroup, 0) asc, so.id asc")
	q = applyScheduleOverrideFilters(q, filters)

	var rows []ScheduleOverrideReportRow
	err := q.Scan(&rows).Error
	return rows, err
}
