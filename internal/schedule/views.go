package schedule

import "time"

// View structs with joined names for response building.

type TemplateView struct {
	PairNumber   int16
	SubjectID    int
	SubjectName  string
	LocationID   int
	LocationName string
	TeacherName  string
	Subgroup     *int16
}

type OverrideView struct {
	ID              int64
	PairNumber      int16
	ActionType      OverrideAction
	NewSubjectID    *int
	NewSubjectName  string
	NewLocationID   *int
	NewLocationName string
	NewTeacherName  *string
	Comment         *string
	Subgroup        *int16
	UpdatedAt       time.Time
}
