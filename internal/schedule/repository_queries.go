package schedule

import (
	"time"
)

func (r *Repository) ListTemplatesForWeekStatus(groupID int, parity WeekParity, status EntityStatus) ([]TemplateDayView, error) {
	var rows []TemplateDayView
	err := r.db.Table("schedule_templates st").
		Select(`st.day_of_week, st.pair_number, st.subject_id, s.name AS subject_name, st.location_id, l.name AS location_name, COALESCE(t.name, '') AS teacher_name, st.teacher_manual, st.location_manual, st.subgroup`).
		Joins("JOIN subjects s ON s.id = st.subject_id").
		Joins("JOIN locations l ON l.id = st.location_id").
		Joins("LEFT JOIN teachers t ON t.id = st.teacher_id").
		Where("st.group_id = ? AND st.day_of_week BETWEEN 0 AND 5 AND st.week_parity IN (?, ?) AND st.status = ?", groupID, parity, WeekParityBoth, status).
		Order("st.day_of_week asc, st.pair_number asc, COALESCE(st.subgroup, 0) asc, st.id asc").
		Scan(&rows).Error
	return rows, err
}

func (r *Repository) ListOverridesBetween(groupID int, startDate, endDate time.Time) ([]OverrideDateView, error) {
	var rows []OverrideDateView
	err := r.db.Table("schedule_overrides so").
		Select(`so.target_date, so.id, so.pair_number, so.action_type, so.new_subject_id, COALESCE(s.name, '') AS new_subject_name, so.new_location_id, COALESCE(l.name, '') AS new_location_name, so.new_teacher_manual, t.name AS new_teacher_name, so.comment, so.subgroup, so.updated_at`).
		Joins("LEFT JOIN subjects s ON s.id = so.new_subject_id").
		Joins("LEFT JOIN locations l ON l.id = so.new_location_id").
		Joins("LEFT JOIN teachers t ON t.id = so.new_teacher_id").
		Where("so.group_id = ? AND so.target_date BETWEEN ? AND ?", groupID, dateOnly(startDate), dateOnly(endDate)).
		Order("so.target_date asc, so.pair_number asc, COALESCE(so.subgroup, 0) asc, so.id asc").
		Scan(&rows).Error
	return rows, err
}

func (r *Repository) ListTemplatesFor(groupID int, dayOfWeek int16, parity WeekParity) ([]TemplateView, error) {
	return r.ListTemplatesForStatus(groupID, dayOfWeek, parity, StatusPublished)
}

func (r *Repository) ListTemplatesForStatus(groupID int, dayOfWeek int16, parity WeekParity, status EntityStatus) ([]TemplateView, error) {
	var rows []TemplateView
	err := r.db.Table("schedule_templates st").
		Select(`st.pair_number, st.subject_id, s.name AS subject_name, st.location_id, l.name AS location_name, COALESCE(t.name, '') AS teacher_name, st.teacher_manual, st.location_manual, st.subgroup`).
		Joins("JOIN subjects s ON s.id = st.subject_id").
		Joins("JOIN locations l ON l.id = st.location_id").
		Joins("LEFT JOIN teachers t ON t.id = st.teacher_id").
		Where("st.group_id = ? AND st.day_of_week = ? AND st.week_parity IN (?, ?) AND st.status = ?", groupID, dayOfWeek, parity, WeekParityBoth, status).
		Scan(&rows).Error
	return rows, err
}

func (r *Repository) ListOverridesForDate(groupID int, date time.Time) ([]OverrideView, error) {
	var rows []OverrideView
	err := r.db.Table("schedule_overrides so").
		Select(`so.id, so.pair_number, so.action_type, so.new_subject_id, COALESCE(s.name, '') AS new_subject_name, so.new_location_id, COALESCE(l.name, '') AS new_location_name, so.new_teacher_manual, t.name AS new_teacher_name, so.comment, so.subgroup, so.updated_at`).
		Joins("LEFT JOIN subjects s ON s.id = so.new_subject_id").
		Joins("LEFT JOIN locations l ON l.id = so.new_location_id").
		Joins("LEFT JOIN teachers t ON t.id = so.new_teacher_id").
		Where("so.group_id = ? AND so.target_date = ?", groupID, dateOnly(date)).
		Order(`so.pair_number asc,
			(CASE so.action_type WHEN 'CANCEL' THEN 0 WHEN 'REPLACE' THEN 1 WHEN 'ADD' THEN 2 ELSE 3 END) asc,
			so.updated_at desc,
			so.id desc`).
		Scan(&rows).Error
	return rows, err
}

func (r *Repository) ListOverlaysBetween(groupID int, startDate, endDate time.Time) ([]ScheduleDayOverlay, error) {
	var rows []ScheduleDayOverlay
	err := r.db.Where("group_id = ? AND target_date BETWEEN ? AND ?", groupID, dateOnly(startDate), dateOnly(endDate)).
		Find(&rows).Error
	return rows, err
}

func (r *Repository) ListCalendarExceptionsBetween(startDate, endDate time.Time) ([]CalendarException, error) {
	var rows []CalendarException
	err := r.db.Where("target_date BETWEEN ? AND ?", dateOnly(startDate), dateOnly(endDate)).
		Find(&rows).Error
	return rows, err
}
