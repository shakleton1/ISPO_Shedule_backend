package schedule

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type CalendarDayConstraintFilters struct {
	DateFrom       *time.Time
	DateTo         *time.Time
	ConstraintType *string
	AffectsLessons *bool
}

func (r *Repository) ListCalendarDayConstraints(filters CalendarDayConstraintFilters) ([]CalendarDayConstraint, error) {
	q := r.db.Model(&CalendarDayConstraint{}).Order("target_date asc, id asc")
	q = applyCalendarDayConstraintFilters(q, filters)
	var rows []CalendarDayConstraint
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) ListCalendarDayConstraintsPaged(filters CalendarDayConstraintFilters, limit, offset *int) ([]CalendarDayConstraint, error) {
	q := r.db.Model(&CalendarDayConstraint{}).Order("target_date asc, id asc")
	q = applyCalendarDayConstraintFilters(q, filters)
	q = applyLimitOffset(q, limit, offset)
	var rows []CalendarDayConstraint
	err := q.Find(&rows).Error
	return rows, err
}

func applyCalendarDayConstraintFilters(q *gorm.DB, filters CalendarDayConstraintFilters) *gorm.DB {
	if filters.DateFrom != nil {
		q = q.Where("target_date >= ?", dateOnly(*filters.DateFrom))
	}
	if filters.DateTo != nil {
		q = q.Where("target_date <= ?", dateOnly(*filters.DateTo))
	}
	if filters.ConstraintType != nil && strings.TrimSpace(*filters.ConstraintType) != "" {
		q = q.Where("constraint_type = ?", strings.ToLower(strings.TrimSpace(*filters.ConstraintType)))
	}
	if filters.AffectsLessons != nil {
		q = q.Where("affects_lessons = ?", *filters.AffectsLessons)
	}
	return q
}

func (r *Repository) GetCalendarDayConstraintByDate(targetDate time.Time) (*CalendarDayConstraint, error) {
	var row CalendarDayConstraint
	err := r.db.
		Where("target_date = ?", dateOnly(targetDate)).
		First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) ListCalendarDayConstraintsBetween(startDate, endDate time.Time) ([]CalendarDayConstraint, error) {
	var rows []CalendarDayConstraint
	err := r.db.
		Where("target_date BETWEEN ? AND ?", dateOnly(startDate), dateOnly(endDate)).
		Order("target_date asc, id asc").
		Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateCalendarDayConstraint(row *CalendarDayConstraint) error {
	if err := normalizeCalendarDayConstraint(row); err != nil {
		return err
	}
	return r.db.Create(row).Error
}

func (r *Repository) UpdateCalendarDayConstraint(id int64, patch *CalendarDayConstraint) (*CalendarDayConstraint, error) {
	if patch == nil {
		return nil, fmt.Errorf("calendar day constraint patch is nil")
	}
	if err := normalizeCalendarDayConstraint(patch); err != nil {
		return nil, err
	}
	var row CalendarDayConstraint
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.TargetDate = patch.TargetDate
	row.Title = patch.Title
	row.Reason = patch.Reason
	row.ConstraintType = patch.ConstraintType
	row.AffectsLessons = patch.AffectsLessons
	row.RequiresConfirmation = patch.RequiresConfirmation
	row.StylePreset = patch.StylePreset
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteCalendarDayConstraint(id int64) error {
	return r.db.Delete(&CalendarDayConstraint{}, id).Error
}

func normalizeCalendarDayConstraint(row *CalendarDayConstraint) error {
	if row == nil {
		return fmt.Errorf("calendar day constraint is nil")
	}

	row.Title = strings.TrimSpace(row.Title)
	if row.Title == "" {
		return fmt.Errorf("title required")
	}
	if row.Reason != nil {
		v := strings.TrimSpace(*row.Reason)
		if v == "" {
			row.Reason = nil
		} else {
			row.Reason = &v
		}
	}

	row.ConstraintType = strings.ToLower(strings.TrimSpace(row.ConstraintType))
	if row.ConstraintType == "" {
		row.ConstraintType = "blocked"
	}
	switch row.ConstraintType {
	case "blocked", "warning", "info":
	default:
		return fmt.Errorf("invalid constraint_type: %s", row.ConstraintType)
	}

	row.StylePreset = strings.ToLower(strings.TrimSpace(row.StylePreset))
	if row.StylePreset == "" {
		switch row.ConstraintType {
		case "blocked":
			row.StylePreset = "danger"
		case "info":
			row.StylePreset = "info"
		default:
			row.StylePreset = "warning"
		}
	}
	switch row.StylePreset {
	case "standard", "warning", "danger", "info":
	default:
		return fmt.Errorf("invalid style_preset: %s", row.StylePreset)
	}

	row.TargetDate = dateOnly(row.TargetDate)
	if row.TargetDate.IsZero() {
		return fmt.Errorf("target_date required")
	}
	return nil
}
