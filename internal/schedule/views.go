package schedule

import "time"

// View structs with joined names for response building.

type TemplateDayView struct {
	DayOfWeek      int16   `gorm:"column:day_of_week"`
	PairNumber     int16   `gorm:"column:pair_number"`
	SubjectID      int     `gorm:"column:subject_id"`
	SubjectName    string  `gorm:"column:subject_name"`
	LocationID     int     `gorm:"column:location_id"`
	LocationName   string  `gorm:"column:location_name"`
	TeacherName    string  `gorm:"column:teacher_name"`
	TeacherManual  bool    `gorm:"column:teacher_manual"`
	LocationManual bool    `gorm:"column:location_manual"`
	Subgroup       *int16  `gorm:"column:subgroup"`
	FlowKey        *string `gorm:"column:flow_key"`
}

type TemplateView struct {
	PairNumber     int16   `json:"pair_number"`
	SubjectID      int     `json:"subject_id"`
	SubjectName    string  `json:"subject_name"`
	LocationID     int     `json:"location_id"`
	LocationName   string  `json:"location_name"`
	TeacherName    string  `json:"teacher_name"`
	TeacherManual  bool    `json:"teacher_manual"`
	LocationManual bool    `json:"location_manual"`
	Subgroup       *int16  `json:"subgroup"`
	FlowKey        *string `json:"flow_key,omitempty"`
}

type OverrideView struct {
	ID               int64          `json:"id"`
	PairNumber       int16          `json:"pair_number"`
	ActionType       OverrideAction `json:"action_type"`
	NewSubjectID     *int           `json:"new_subject_id"`
	NewSubjectName   string         `json:"new_subject_name"`
	NewLocationID    *int           `json:"new_location_id"`
	NewLocationName  string         `json:"new_location_name"`
	NewTeacherManual bool           `json:"new_teacher_manual"`
	NewTeacherName   *string        `json:"new_teacher_name"`
	Comment          *string        `json:"comment"`
	Subgroup         *int16         `json:"subgroup"`
	FlowKey          *string        `json:"flow_key,omitempty"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type OverrideDateView struct {
	TargetDate       time.Time      `gorm:"column:target_date"`
	ID               int64          `gorm:"column:id"`
	PairNumber       int16          `gorm:"column:pair_number"`
	ActionType       OverrideAction `gorm:"column:action_type"`
	NewSubjectID     *int           `gorm:"column:new_subject_id"`
	NewSubjectName   string         `gorm:"column:new_subject_name"`
	NewLocationID    *int           `gorm:"column:new_location_id"`
	NewLocationName  string         `gorm:"column:new_location_name"`
	NewTeacherManual bool           `gorm:"column:new_teacher_manual"`
	NewTeacherName   *string        `gorm:"column:new_teacher_name"`
	Comment          *string        `gorm:"column:comment"`
	Subgroup         *int16         `gorm:"column:subgroup"`
	FlowKey          *string        `gorm:"column:flow_key"`
	UpdatedAt        time.Time      `gorm:"column:updated_at"`
}
