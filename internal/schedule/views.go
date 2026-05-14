package schedule

import "time"

// View structs with joined names for response building.

type ScheduleLessonView struct {
	ID               int64        `gorm:"column:id" json:"id"`
	GroupID          int          `gorm:"column:group_id" json:"group_id"`
	LessonDate       time.Time    `gorm:"column:lesson_date" json:"lesson_date"`
	PairNumber       int16        `gorm:"column:pair_number" json:"pair_number"`
	Subgroup         *int16       `gorm:"column:subgroup" json:"subgroup"`
	SubjectID        *int         `gorm:"column:subject_id" json:"subject_id"`
	SubjectName      string       `gorm:"column:subject_name" json:"subject_name"`
	TeacherID        *int         `gorm:"column:teacher_id" json:"teacher_id"`
	TeacherName      string       `gorm:"column:teacher_name" json:"teacher_name"`
	LocationID       *int         `gorm:"column:location_id" json:"location_id"`
	LocationName     string       `gorm:"column:location_name" json:"location_name"`
	LessonFormat     string       `gorm:"column:lesson_format" json:"lesson_format"`
	Status           EntityStatus `gorm:"column:status" json:"status"`
	Source           string       `gorm:"column:source" json:"source"`
	FlowKey          *string      `gorm:"column:flow_key" json:"flow_key,omitempty"`
	Comment          *string      `gorm:"column:comment" json:"comment"`
	Version          int          `gorm:"column:version" json:"version"`
	RoomAssignmentID *int64       `gorm:"column:room_assignment_id" json:"room_assignment_id,omitempty"`
}
