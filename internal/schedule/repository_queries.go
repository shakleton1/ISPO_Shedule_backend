package schedule

import "time"

func (r *Repository) ListOverlaysBetween(groupID int, startDate, endDate time.Time) ([]ScheduleDayOverlay, error) {
	var rows []ScheduleDayOverlay
	err := r.db.Where("group_id = ? AND target_date BETWEEN ? AND ?", groupID, dateOnly(startDate), dateOnly(endDate)).
		Find(&rows).Error
	return rows, err
}
