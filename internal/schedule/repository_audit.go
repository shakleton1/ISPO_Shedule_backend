package schedule

func (r *Repository) CreateAuditLog(l *AuditLog) error {
	return r.db.Create(l).Error
}
