package schedule

import "time"

type Group struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:50;uniqueIndex;not null" json:"name"`
	Course    int       `gorm:"not null" json:"course"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Subject struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	ShortName string    `gorm:"size:30" json:"short_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Location struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:50;not null" json:"name"`
	IsVirtual bool      `gorm:"not null;default:false" json:"is_virtual"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ScheduleTemplate struct {
	ID        int64      `gorm:"primaryKey" json:"id"`
	GroupID   int        `gorm:"not null;index:idx_tpl_query,priority:1" json:"group_id"`
	DayOfWeek int16      `gorm:"not null;index:idx_tpl_query,priority:2" json:"day_of_week"`
	WeekParity WeekParity `gorm:"type:text;not null;index:idx_tpl_query,priority:3" json:"week_parity"`
	PairNumber int16     `gorm:"not null;index:idx_tpl_query,priority:4" json:"pair_number"`
	SubjectID int        `gorm:"not null" json:"subject_id"`
	LocationID int       `gorm:"not null" json:"location_id"`
	TeacherName string   `gorm:"size:100;not null" json:"teacher_name"`
	Subgroup  *int16     `gorm:"" json:"subgroup"` // nil = вся группа
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type ScheduleOverride struct {
	ID            int64          `gorm:"primaryKey" json:"id"`
	TargetDate    time.Time      `gorm:"type:date;not null;index:idx_ovr_query,priority:2" json:"target_date"`
	GroupID       int            `gorm:"not null;index:idx_ovr_query,priority:1" json:"group_id"`
	PairNumber    int16          `gorm:"not null" json:"pair_number"`
	ActionType    OverrideAction `gorm:"type:text;not null" json:"action_type"`
	NewSubjectID  *int           `json:"new_subject_id"`
	NewLocationID *int           `json:"new_location_id"`
	NewTeacherName *string       `gorm:"size:100" json:"new_teacher_name"`
	Comment       *string        `json:"comment"`
	Subgroup      *int16         `json:"subgroup"` // nil = для всех
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type ScheduleDayOverlay struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	TargetDate time.Time `gorm:"type:date;not null;uniqueIndex:uidx_overlay" json:"target_date"`
	GroupID    int       `gorm:"not null;uniqueIndex:uidx_overlay" json:"group_id"`
	Text       string    `gorm:"size:255;not null" json:"text"`
	StylePreset string   `gorm:"size:30;not null;default:standard" json:"style_preset"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CalendarException struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	TargetDate time.Time `gorm:"type:date;not null;uniqueIndex" json:"target_date"`
	WorksAsDay int16     `gorm:"not null" json:"works_as_day"`
	Comment    *string   `json:"comment"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type SystemState struct {
	ID              int16     `gorm:"primaryKey" json:"id"`
	ScheduleVersion time.Time `gorm:"not null" json:"schedule_version"`
}

func (SystemState) TableName() string { return "system_state" }

