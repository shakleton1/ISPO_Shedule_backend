package schedule

import "time"

type DeviceToken struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	GroupID   int       `gorm:"not null;index" json:"group_id"`
	Token     string    `gorm:"not null;uniqueIndex" json:"token"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (DeviceToken) TableName() string { return "device_tokens" }
