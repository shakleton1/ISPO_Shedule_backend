package auth

import "time"

type Role string

const (
	RoleStudent    Role = "student"
	RoleDispatcher Role = "dispatcher"
	RoleAdmin      Role = "admin"
)

// User is stored in the existing `students` table (historical naming from spec).
type User struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	Login        string    `gorm:"size:50;uniqueIndex;not null" json:"login"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Role         Role      `gorm:"type:text;not null;default:student" json:"role"`
	GroupID      *int      `json:"group_id"`
	Subgroup     *int16    `json:"subgroup"` // 1/2 or null
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (User) TableName() string { return "students" }
