package schedule

import "time"

// View structs with joined names for response building.

type TemplateView struct {
	PairNumber     int16  `json:"pair_number"`
	SubjectID      int    `json:"subject_id"`
	SubjectName    string `json:"subject_name"`
	LocationID     int    `json:"location_id"`
	LocationName   string `json:"location_name"`
	TeacherName    string `json:"teacher_name"`
	TeacherManual  bool   `json:"teacher_manual"`
	LocationManual bool   `json:"location_manual"`
	Subgroup       *int16 `json:"subgroup"`
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
	UpdatedAt        time.Time      `json:"updated_at"`
}
