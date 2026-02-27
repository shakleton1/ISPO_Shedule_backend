package schedule

import "gorm.io/gorm"

func applyLimitOffset(q *gorm.DB, limit, offset *int) *gorm.DB {
	if limit == nil {
		return q
	}
	q = q.Limit(*limit)
	if offset != nil {
		q = q.Offset(*offset)
	}
	return q
}
