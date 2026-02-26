package schedule

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// Groups

func (r *Repository) ListGroups() ([]Group, error) {
	var rows []Group
	err := r.db.Order("id asc").Find(&rows).Error
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
	err := r.db.Order("id asc").Find(&rows).Error
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
	return &row, nil
}

func (r *Repository) DeleteSubject(id int) error {
	return r.db.Delete(&Subject{}, id).Error
}

// Locations

func (r *Repository) ListLocations() ([]Location, error) {
	var rows []Location
	err := r.db.Order("id asc").Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateLocation(l *Location) error {
	return r.db.Create(l).Error
}

func (r *Repository) UpdateLocation(id int, patch *Location) (*Location, error) {
	var row Location
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.Name = patch.Name
	row.IsVirtual = patch.IsVirtual
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
	GroupID   *int
	DayOfWeek *int16
	WeekParity *WeekParity
}

func (r *Repository) ListTemplates(filters TemplateFilters) ([]ScheduleTemplate, error) {
	q := r.db.Model(&ScheduleTemplate{}).Order("group_id asc, day_of_week asc, pair_number asc")
	if filters.GroupID != nil {
		q = q.Where("group_id = ?", *filters.GroupID)
	}
	if filters.DayOfWeek != nil {
		q = q.Where("day_of_week = ?", *filters.DayOfWeek)
	}
	if filters.WeekParity != nil {
		q = q.Where("week_parity = ?", *filters.WeekParity)
	}
	var rows []ScheduleTemplate
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateTemplate(tpl *ScheduleTemplate) error {
	return r.db.Create(tpl).Error
}

func (r *Repository) UpdateTemplate(id int64, patch *ScheduleTemplate) (*ScheduleTemplate, error) {
	var row ScheduleTemplate
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.GroupID = patch.GroupID
	row.DayOfWeek = patch.DayOfWeek
	row.WeekParity = patch.WeekParity
	row.PairNumber = patch.PairNumber
	row.SubjectID = patch.SubjectID
	row.LocationID = patch.LocationID
	row.TeacherName = patch.TeacherName
	row.Subgroup = patch.Subgroup
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteTemplate(id int64) error {
	return r.db.Delete(&ScheduleTemplate{}, id).Error
}

// Overrides

type OverrideFilters struct {
	GroupID    *int
	TargetDate *time.Time
}

func (r *Repository) ListOverrides(filters OverrideFilters) ([]ScheduleOverride, error) {
	q := r.db.Model(&ScheduleOverride{}).Order("target_date asc, pair_number asc")
	if filters.GroupID != nil {
		q = q.Where("group_id = ?", *filters.GroupID)
	}
	if filters.TargetDate != nil {
		q = q.Where("target_date = ?", dateOnly(*filters.TargetDate))
	}
	var rows []ScheduleOverride
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateOverride(o *ScheduleOverride) error {
	o.TargetDate = dateOnly(o.TargetDate)
	return r.db.Create(o).Error
}

func (r *Repository) UpdateOverride(id int64, patch *ScheduleOverride) (*ScheduleOverride, error) {
	var row ScheduleOverride
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.TargetDate = dateOnly(patch.TargetDate)
	row.GroupID = patch.GroupID
	row.PairNumber = patch.PairNumber
	row.ActionType = patch.ActionType
	row.NewSubjectID = patch.NewSubjectID
	row.NewLocationID = patch.NewLocationID
	row.NewTeacherName = patch.NewTeacherName
	row.Comment = patch.Comment
	row.Subgroup = patch.Subgroup
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteOverride(id int64) error {
	return r.db.Delete(&ScheduleOverride{}, id).Error
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
