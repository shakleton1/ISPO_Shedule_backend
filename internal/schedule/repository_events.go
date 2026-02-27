package schedule

import "time"

type dayEventViewRow struct {
	ID           int64     `gorm:"column:id"`
	TargetDate   time.Time `gorm:"column:target_date"`
	EventType    string    `gorm:"column:event_type"`
	Title        string    `gorm:"column:title"`
	Details      *string   `gorm:"column:details"`
	LocationID   *int      `gorm:"column:location_id"`
	LocationName string    `gorm:"column:location_name"`
}

func (r *Repository) ListDayEventsBetween(groupID int, startDate, endDate time.Time) ([]dayEventViewRow, error) {
	var rows []dayEventViewRow
	err := r.db.Table("schedule_day_events e").
		Select("e.id, e.target_date, e.event_type, e.title, e.details, e.location_id, COALESCE(l.name,'') AS location_name").
		Joins("LEFT JOIN locations l ON l.id = e.location_id").
		Where("e.group_id = ? AND e.target_date BETWEEN ? AND ?", groupID, dateOnly(startDate), dateOnly(endDate)).
		Order("e.target_date asc, e.id asc").
		Scan(&rows).Error
	return rows, err
}

type DayEventFilters struct {
	GroupID    *int
	TargetDate *time.Time
}

func (r *Repository) ListDayEvents(filters DayEventFilters) ([]ScheduleDayEvent, error) {
	q := r.db.Table("schedule_day_events").Order("target_date asc, id asc")
	if filters.GroupID != nil {
		q = q.Where("group_id = ?", *filters.GroupID)
	}
	if filters.TargetDate != nil {
		q = q.Where("target_date = ?", dateOnly(*filters.TargetDate))
	}
	var rows []ScheduleDayEvent
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) ListDayEventsPaged(filters DayEventFilters, limit, offset *int) ([]ScheduleDayEvent, error) {
	q := r.db.Table("schedule_day_events").Order("target_date asc, id asc")
	if filters.GroupID != nil {
		q = q.Where("group_id = ?", *filters.GroupID)
	}
	if filters.TargetDate != nil {
		q = q.Where("target_date = ?", dateOnly(*filters.TargetDate))
	}
	q = applyLimitOffset(q, limit, offset)
	var rows []ScheduleDayEvent
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateDayEvent(e *ScheduleDayEvent) error {
	e.TargetDate = dateOnly(e.TargetDate)
	return r.db.Create(e).Error
}

func (r *Repository) UpdateDayEvent(id int64, patch *ScheduleDayEvent) (*ScheduleDayEvent, error) {
	var row ScheduleDayEvent
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.TargetDate = dateOnly(patch.TargetDate)
	row.GroupID = patch.GroupID
	row.EventType = patch.EventType
	row.Title = patch.Title
	row.Details = patch.Details
	row.LocationID = patch.LocationID
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteDayEvent(id int64) error {
	return r.db.Delete(&ScheduleDayEvent{}, id).Error
}
