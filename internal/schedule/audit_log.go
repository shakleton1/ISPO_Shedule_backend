package schedule

import "time"

// AuditLog stores a minimal audit trail for admin-side changes.
// It is intentionally append-only.
//
// DB schema is created via goose migration: db/migrations/20260226131000_audit_logs.sql
type AuditLog struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	ActorType  string  `gorm:"type:text;not null" json:"actor_type"`
	ActorID    *int64  `gorm:"" json:"actor_id"`
	ActorLogin *string `gorm:"size:50" json:"actor_login"`
	ActorRole  *string `gorm:"type:text" json:"actor_role"`

	Method string `gorm:"size:10;not null" json:"method"`
	Path   string `gorm:"size:200;not null" json:"path"`

	RequestID *string `gorm:"type:text" json:"request_id"`
	IP        *string `gorm:"type:text" json:"ip"`
	UserAgent *string `gorm:"type:text" json:"user_agent"`

	Action     string `gorm:"type:text;not null" json:"action"`
	EntityType string `gorm:"type:text;not null" json:"entity_type"`
	EntityID   string `gorm:"type:text;not null" json:"entity_id"`

	Payload []byte `gorm:"type:jsonb" json:"payload"`
}

func (AuditLog) TableName() string { return "audit_logs" }
