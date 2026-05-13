package schedule

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ispo-schedule/internal/auth"

	"gorm.io/gorm"
)

func (r *Repository) HasAnyOverrideForSlot(groupID int, date time.Time, pairNumber int16, subgroup *int16) (bool, error) {
	q := r.db.Table("schedule_overrides").
		Where("group_id = ? AND target_date = ? AND pair_number = ?", groupID, dateOnly(date), pairNumber)
	if subgroup == nil {
		q = q.Where("subgroup IS NULL")
	} else {
		q = q.Where("subgroup = ?", *subgroup)
	}
	var cnt int64
	if err := q.Count(&cnt).Error; err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func normalizeTeacherName(name string) string {
	return strings.TrimSpace(name)
}

func ensureTeacherID(tx *gorm.DB, name string) (*int, error) {
	name = normalizeTeacherName(name)
	if name == "" {
		return nil, nil
	}
	var out struct {
		ID int `gorm:"column:id"`
	}
	err := tx.Raw(
		"INSERT INTO teachers (name) VALUES (?) ON CONFLICT (name_key) DO UPDATE SET name = EXCLUDED.name, deleted_at = NULL RETURNING id",
		name,
	).Scan(&out).Error
	if err != nil {
		return nil, err
	}
	return &out.ID, nil
}

// Groups

func (r *Repository) ListGroups() ([]Group, error) {
	var rows []Group
	err := r.db.Order("id asc").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListGroupsPaged(limit, offset *int) ([]Group, error) {
	var rows []Group
	q := r.db.Order("id asc")
	q = applyLimitOffset(q, limit, offset)
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) GetGroup(id int) (*Group, error) {
	var row Group
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) CreateGroup(g *Group) error {
	return r.db.Create(g).Error
}

func (r *Repository) UpdateGroup(id int, patch *Group) (*Group, error) {
	var row Group
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.Name = patch.Name
	row.Course = patch.Course
	if patch.ScheduleSourceGroupID != nil {
		if *patch.ScheduleSourceGroupID == row.ID {
			return nil, fmt.Errorf("schedule_source_group_id must not point to self")
		}
		// Cycle check: follow schedule_source_group_id pointers.
		seen := map[int]bool{row.ID: true}
		next := *patch.ScheduleSourceGroupID
		for depth := 0; depth < 10; depth++ {
			if next <= 0 {
				break
			}
			if seen[next] {
				return nil, fmt.Errorf("schedule_source_group_id cycle detected")
			}
			seen[next] = true
			var g Group
			if err := r.db.First(&g, next).Error; err != nil {
				return nil, err
			}
			if g.ScheduleSourceGroupID == nil {
				break
			}
			next = *g.ScheduleSourceGroupID
		}
		row.ScheduleSourceGroupID = patch.ScheduleSourceGroupID
	}
	if patch.CurriculumID != nil {
		row.CurriculumID = patch.CurriculumID
	}
	if patch.AdmissionYear != nil {
		row.AdmissionYear = patch.AdmissionYear
	}
	if patch.SpecialtyID != nil {
		row.SpecialtyID = patch.SpecialtyID
	}
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteGroup(id int) error {
	return r.db.Delete(&Group{}, id).Error
}

// Subjects

func (r *Repository) ListSubjects() ([]Subject, error) {
	var rows []Subject
	err := r.db.Where("deleted_at IS NULL").Order("id asc").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListSubjectsPaged(limit, offset *int) ([]Subject, error) {
	var rows []Subject
	q := r.db.Where("deleted_at IS NULL").Order("id asc")
	q = applyLimitOffset(q, limit, offset)
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateSubject(s *Subject) error {
	return r.db.Create(s).Error
}

func (r *Repository) UpdateSubject(id int, patch *Subject) (*Subject, error) {
	var row Subject
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.Name = patch.Name
	row.ShortName = patch.ShortName
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	// If it was soft-deleted earlier, restore.
	if err := r.db.Exec("UPDATE subjects SET deleted_at = NULL WHERE id = ?", id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteSubject(id int) error {
	res := r.db.Exec("UPDATE subjects SET deleted_at = now() WHERE id = ? AND deleted_at IS NULL", id)
	return res.Error
}

// Locations

func (r *Repository) ListLocations() ([]Location, error) {
	var rows []Location
	err := r.db.Order("id asc").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListLocationsPaged(limit, offset *int) ([]Location, error) {
	var rows []Location
	q := r.db.Order("id asc")
	q = applyLimitOffset(q, limit, offset)
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) GetLocation(id int) (*Location, error) {
	var row Location
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) CreateLocation(l *Location) error {
	if err := normalizeLocation(l); err != nil {
		return err
	}
	return r.db.Create(l).Error
}

func (r *Repository) UpdateLocation(id int, patch *Location) (*Location, error) {
	if err := normalizeLocation(patch); err != nil {
		return nil, err
	}
	var row Location
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.Name = patch.Name
	row.CampusID = patch.CampusID
	row.Kind = patch.Kind
	row.Capacity = patch.Capacity
	row.IsActive = patch.IsActive
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteLocation(id int) error {
	return r.db.Delete(&Location{}, id).Error
}

// Templates

type TemplateFilters struct {
	GroupID    *int
	DayOfWeek  *int16
	WeekParity *WeekParity
	Status     *EntityStatus
}

func (r *Repository) ListTemplates(filters TemplateFilters) ([]ScheduleTemplate, error) {
	q := r.db.Table("schedule_templates st").
		Select("st.id, st.group_id, st.day_of_week, st.week_parity, st.pair_number, st.subject_id, st.location_id, st.lesson_format, st.status, st.teacher_manual, st.location_manual, COALESCE(t.name, '') AS teacher_name, st.subgroup, st.flow_key, st.created_at, st.updated_at").
		Joins("LEFT JOIN teachers t ON t.id = st.teacher_id").
		Order("st.group_id asc, st.day_of_week asc, st.week_parity asc, st.pair_number asc, st.subgroup asc")
	if filters.GroupID != nil {
		q = q.Where("st.group_id = ?", *filters.GroupID)
	}
	if filters.DayOfWeek != nil {
		q = q.Where("st.day_of_week = ?", *filters.DayOfWeek)
	}
	if filters.WeekParity != nil {
		q = q.Where("st.week_parity = ?", *filters.WeekParity)
	}
	if filters.Status != nil {
		q = q.Where("st.status = ?", *filters.Status)
	} else {
		q = q.Where("st.status = ?", StatusPublished)
	}
	var rows []ScheduleTemplate
	err := q.Scan(&rows).Error
	return rows, err
}

func (r *Repository) ListTemplatesPaged(filters TemplateFilters, limit, offset *int) ([]ScheduleTemplate, error) {
	q := r.db.Table("schedule_templates st").
		Select("st.id, st.group_id, st.day_of_week, st.week_parity, st.pair_number, st.subject_id, st.location_id, st.lesson_format, st.status, st.teacher_manual, st.location_manual, COALESCE(t.name, '') AS teacher_name, st.subgroup, st.flow_key, st.created_at, st.updated_at").
		Joins("LEFT JOIN teachers t ON t.id = st.teacher_id").
		Order("st.group_id asc, st.day_of_week asc, st.week_parity asc, st.pair_number asc, st.subgroup asc")
	if filters.GroupID != nil {
		q = q.Where("st.group_id = ?", *filters.GroupID)
	}
	if filters.DayOfWeek != nil {
		q = q.Where("st.day_of_week = ?", *filters.DayOfWeek)
	}
	if filters.WeekParity != nil {
		q = q.Where("st.week_parity = ?", *filters.WeekParity)
	}
	if filters.Status != nil {
		q = q.Where("st.status = ?", *filters.Status)
	} else {
		q = q.Where("st.status = ?", StatusPublished)
	}
	q = applyLimitOffset(q, limit, offset)
	var rows []ScheduleTemplate
	err := q.Scan(&rows).Error
	return rows, err
}

func (r *Repository) CreateTemplate(tpl *ScheduleTemplate) error {
	if tpl == nil {
		return fmt.Errorf("template is nil")
	}
	if tpl.Status == "" {
		tpl.Status = StatusPublished
	}
	if tpl.Status != StatusDraft && tpl.Status != StatusPublished {
		return fmt.Errorf("invalid status: %s", tpl.Status)
	}
	tpl.LessonFormat = normalizeLessonFormat(tpl.LessonFormat)
	teacherID, err := ensureTeacherID(r.db, tpl.TeacherName)
	if err != nil {
		return err
	}
	tpl.TeacherID = teacherID
	return r.db.Omit("TeacherName").Create(tpl).Error
}

func (r *Repository) UpdateTemplate(id int64, patch *ScheduleTemplate) (*ScheduleTemplate, error) {
	var row ScheduleTemplate
	if err := r.db.Table("schedule_templates").First(&row, id).Error; err != nil {
		return nil, err
	}
	teacherID, err := ensureTeacherID(r.db, patch.TeacherName)
	if err != nil {
		return nil, err
	}
	if patch.Status == "" {
		patch.Status = row.Status
	}
	if patch.Status != StatusDraft && patch.Status != StatusPublished {
		return nil, fmt.Errorf("invalid status: %s", patch.Status)
	}
	patch.LessonFormat = normalizeLessonFormat(patch.LessonFormat)
	row.GroupID = patch.GroupID
	row.DayOfWeek = patch.DayOfWeek
	row.WeekParity = patch.WeekParity
	row.PairNumber = patch.PairNumber
	row.SubjectID = patch.SubjectID
	row.LocationID = patch.LocationID
	row.LessonFormat = patch.LessonFormat
	row.Status = patch.Status
	row.TeacherManual = patch.TeacherManual
	row.LocationManual = patch.LocationManual
	row.TeacherID = teacherID
	row.Subgroup = patch.Subgroup
	row.FlowKey = patch.FlowKey
	if err := r.db.Omit("TeacherName").Save(&row).Error; err != nil {
		return nil, err
	}
	return r.GetTemplateByID(id)
}

func (r *Repository) DeleteTemplate(id int64) error {
	return r.db.Delete(&ScheduleTemplate{}, id).Error
}

func (r *Repository) GetTemplateByID(id int64) (*ScheduleTemplate, error) {
	var row ScheduleTemplate
	if err := r.db.Table("schedule_templates st").
		Select("st.id, st.group_id, st.day_of_week, st.week_parity, st.pair_number, st.subject_id, st.location_id, st.lesson_format, st.status, st.teacher_manual, st.location_manual, COALESCE(t.name, '') AS teacher_name, st.subgroup, st.flow_key, st.created_at, st.updated_at").
		Joins("LEFT JOIN teachers t ON t.id = st.teacher_id").
		Where("st.id = ?", id).
		Scan(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) PublishDraftTemplates(groupID int) (int64, error) {
	if groupID <= 0 {
		return 0, fmt.Errorf("group_id required")
	}
	var moved int64
	err := r.db.Transaction(func(tx *gorm.DB) error {
		// Replace matching published rows for each draft slot.
		if err := tx.Exec(`
			DELETE FROM schedule_templates p
			USING schedule_templates d
			WHERE d.group_id = ?
			  AND d.status = 'draft'
			  AND p.group_id = d.group_id
			  AND p.day_of_week = d.day_of_week
			  AND p.week_parity = d.week_parity
			  AND p.pair_number = d.pair_number
			  AND COALESCE(p.subgroup, 0) = COALESCE(d.subgroup, 0)
			  AND p.status = 'published'
		`, groupID).Error; err != nil {
			return err
		}

		res := tx.Exec(`
			INSERT INTO schedule_templates (group_id, day_of_week, week_parity, pair_number, subject_id, location_id, lesson_format, status, teacher_manual, location_manual, teacher_id, subgroup, flow_key, created_at, updated_at)
			SELECT group_id, day_of_week, week_parity, pair_number, subject_id, location_id, lesson_format, 'published', teacher_manual, location_manual, teacher_id, subgroup, flow_key, now(), now()
			FROM schedule_templates
			WHERE group_id = ? AND status = 'draft'
		`, groupID)
		if res.Error != nil {
			return res.Error
		}
		moved = res.RowsAffected

		// Remove drafts after publish.
		if err := tx.Exec(`DELETE FROM schedule_templates WHERE group_id = ? AND status = 'draft'`, groupID).Error; err != nil {
			return err
		}
		return nil
	})
	return moved, err
}

func (r *Repository) DiscardDraftTemplates(groupID int) (int64, error) {
	if groupID <= 0 {
		return 0, fmt.Errorf("group_id required")
	}
	res := r.db.Exec(`DELETE FROM schedule_templates WHERE group_id = ? AND status = 'draft'`, groupID)
	return res.RowsAffected, res.Error
}

// Overrides

type OverrideFilters struct {
	GroupID    *int
	TargetDate *time.Time
}

func (r *Repository) ListOverrides(filters OverrideFilters) ([]ScheduleOverride, error) {
	q := r.db.Table("schedule_overrides so").
		Select("so.id, so.target_date, so.group_id, so.pair_number, so.action_type, so.new_subject_id, so.new_location_id, so.new_lesson_format, so.new_teacher_manual, t.name AS new_teacher_name, so.comment, so.subgroup, so.flow_key, so.created_at, so.updated_at").
		Joins("LEFT JOIN teachers t ON t.id = so.new_teacher_id").
		Order("so.target_date asc, so.pair_number asc")
	if filters.GroupID != nil {
		q = q.Where("so.group_id = ?", *filters.GroupID)
	}
	if filters.TargetDate != nil {
		q = q.Where("so.target_date = ?", dateOnly(*filters.TargetDate))
	}
	var rows []ScheduleOverride
	err := q.Scan(&rows).Error
	return rows, err
}

func (r *Repository) ListOverridesPaged(filters OverrideFilters, limit, offset *int) ([]ScheduleOverride, error) {
	q := r.db.Table("schedule_overrides so").
		Select("so.id, so.target_date, so.group_id, so.pair_number, so.action_type, so.new_subject_id, so.new_location_id, so.new_lesson_format, so.new_teacher_manual, t.name AS new_teacher_name, so.comment, so.subgroup, so.flow_key, so.created_at, so.updated_at").
		Joins("LEFT JOIN teachers t ON t.id = so.new_teacher_id").
		Order("so.target_date asc, so.pair_number asc")
	if filters.GroupID != nil {
		q = q.Where("so.group_id = ?", *filters.GroupID)
	}
	if filters.TargetDate != nil {
		q = q.Where("so.target_date = ?", dateOnly(*filters.TargetDate))
	}
	q = applyLimitOffset(q, limit, offset)
	var rows []ScheduleOverride
	err := q.Scan(&rows).Error
	return rows, err
}

func (r *Repository) CreateOverride(o *ScheduleOverride) error {
	if err := validateOverrideForWrite(o); err != nil {
		return err
	}
	if o.NewTeacherManual {
		// Explicit teacher override. Empty name means "clear teacher".
		if o.NewTeacherName == nil || strings.TrimSpace(*o.NewTeacherName) == "" {
			o.NewTeacherID = nil
		} else {
			teacherID, err := ensureTeacherID(r.db, *o.NewTeacherName)
			if err != nil {
				return err
			}
			o.NewTeacherID = teacherID
		}
	} else if o.NewTeacherName != nil {
		// Backward-compatible: if name is provided, treat as explicit override.
		o.NewTeacherManual = true
		teacherID, err := ensureTeacherID(r.db, *o.NewTeacherName)
		if err != nil {
			return err
		}
		o.NewTeacherID = teacherID
	}
	o.TargetDate = dateOnly(o.TargetDate)
	return r.db.Omit("NewTeacherName").Create(o).Error
}

func (r *Repository) UpdateOverride(id int64, patch *ScheduleOverride) (*ScheduleOverride, error) {
	if err := validateOverrideForWrite(patch); err != nil {
		return nil, err
	}
	var row ScheduleOverride
	if err := r.db.Table("schedule_overrides").First(&row, id).Error; err != nil {
		return nil, err
	}
	if patch.NewTeacherManual {
		row.NewTeacherManual = true
		if patch.NewTeacherName == nil || strings.TrimSpace(*patch.NewTeacherName) == "" {
			row.NewTeacherID = nil
		} else {
			teacherID, err := ensureTeacherID(r.db, *patch.NewTeacherName)
			if err != nil {
				return nil, err
			}
			row.NewTeacherID = teacherID
		}
	} else if patch.NewTeacherName != nil {
		row.NewTeacherManual = true
		teacherID, err := ensureTeacherID(r.db, *patch.NewTeacherName)
		if err != nil {
			return nil, err
		}
		row.NewTeacherID = teacherID
	}
	row.TargetDate = dateOnly(patch.TargetDate)
	row.GroupID = patch.GroupID
	row.PairNumber = patch.PairNumber
	row.ActionType = patch.ActionType
	row.NewSubjectID = patch.NewSubjectID
	row.NewLocationID = patch.NewLocationID
	row.NewLessonFormat = patch.NewLessonFormat
	row.Comment = patch.Comment
	row.Subgroup = patch.Subgroup
	row.FlowKey = patch.FlowKey
	if err := r.db.Omit("NewTeacherName").Save(&row).Error; err != nil {
		return nil, err
	}
	return r.GetOverrideByID(id)
}

func validateOverrideForWrite(o *ScheduleOverride) error {
	if o == nil {
		return fmt.Errorf("override is nil")
	}
	if o.GroupID <= 0 {
		return fmt.Errorf("group_id required")
	}
	if o.PairNumber < 1 || o.PairNumber > 8 {
		return fmt.Errorf("pair_number must be 1..8")
	}
	if o.Subgroup != nil && (*o.Subgroup < 1 || *o.Subgroup > 2) {
		return fmt.Errorf("subgroup must be 1 or 2")
	}
	switch o.ActionType {
	case OverrideCancel:
		if o.NewSubjectID != nil || o.NewLocationID != nil || o.NewLessonFormat != nil || o.NewTeacherName != nil || o.NewTeacherManual || o.FlowKey != nil {
			return fmt.Errorf("CANCEL must not set new_* fields")
		}
	case OverrideAdd:
		if o.NewSubjectID == nil {
			return fmt.Errorf("ADD requires new_subject_id")
		}
	case OverrideReplace:
		if o.NewSubjectID == nil && o.NewLocationID == nil && o.NewLessonFormat == nil && o.NewTeacherName == nil && o.Comment == nil && !o.NewTeacherManual && o.FlowKey == nil {
			return fmt.Errorf("REPLACE requires at least one change field")
		}
	default:
		return fmt.Errorf("invalid action_type")
	}
	return nil
}

func (r *Repository) DeleteOverride(id int64) error {
	return r.db.Delete(&ScheduleOverride{}, id).Error
}

func (r *Repository) GetOverrideByID(id int64) (*ScheduleOverride, error) {
	var row ScheduleOverride
	if err := r.db.Table("schedule_overrides so").
		Select("so.id, so.target_date, so.group_id, so.pair_number, so.action_type, so.new_subject_id, so.new_location_id, so.new_lesson_format, so.new_teacher_manual, t.name AS new_teacher_name, so.comment, so.subgroup, so.flow_key, so.created_at, so.updated_at").
		Joins("LEFT JOIN teachers t ON t.id = so.new_teacher_id").
		Where("so.id = ?", id).
		Scan(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// Overlay

func (r *Repository) UpsertOverlay(groupID int, date time.Time, text string, stylePreset string) (*ScheduleDayOverlay, error) {
	date = dateOnly(date)
	if stylePreset == "" {
		stylePreset = "standard"
	}

	var row ScheduleDayOverlay
	err := r.db.Where("group_id = ? AND target_date = ?", groupID, date).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row = ScheduleDayOverlay{GroupID: groupID, TargetDate: date, Text: text, StylePreset: stylePreset}
			if err2 := r.db.Create(&row).Error; err2 != nil {
				return nil, err2
			}
			return &row, nil
		}
		return nil, err
	}
	row.Text = text
	row.StylePreset = stylePreset
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// Calendar exceptions

func (r *Repository) ListCalendarExceptions() ([]CalendarException, error) {
	var rows []CalendarException
	err := r.db.Order("target_date asc").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListCalendarExceptionsPaged(limit, offset *int) ([]CalendarException, error) {
	var rows []CalendarException
	q := r.db.Order("target_date asc")
	q = applyLimitOffset(q, limit, offset)
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) UpsertCalendarException(date time.Time, worksAsDay int16, comment *string) (*CalendarException, error) {
	date = dateOnly(date)
	var row CalendarException
	err := r.db.Where("target_date = ?", date).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row = CalendarException{TargetDate: date, WorksAsDay: worksAsDay, Comment: comment}
			if err2 := r.db.Create(&row).Error; err2 != nil {
				return nil, err2
			}
			return &row, nil
		}
		return nil, err
	}
	row.WorksAsDay = worksAsDay
	row.Comment = comment
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteCalendarExceptionByDate(dateStr string) error {
	d, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return fmt.Errorf("invalid date: %w", err)
	}
	return r.db.Where("target_date = ?", dateOnly(d)).Delete(&CalendarException{}).Error
}

func ParseInt64Param(v string) (int64, error) {
	return strconv.ParseInt(v, 10, 64)
}

// Users (auth)

func (r *Repository) GetUserByLogin(login string) (*auth.User, error) {
	var u auth.User
	if err := r.db.Where("login = ?", login).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) GetUserByID(id int64) (*auth.User, error) {
	var u auth.User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) CreateUser(u *auth.User) error {
	return r.db.Create(u).Error
}

func (r *Repository) CountAdmins() (int64, error) {
	var cnt int64
	err := r.db.Model(&auth.User{}).Where("role = ?", auth.RoleAdmin).Count(&cnt).Error
	return cnt, err
}
