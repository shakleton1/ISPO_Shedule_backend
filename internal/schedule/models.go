package schedule

import "time"

type EntityStatus string

const (
	StatusDraft     EntityStatus = "draft"
	StatusPublished EntityStatus = "published"
)

type Group struct {
	ID                    int       `gorm:"primaryKey" json:"id"`
	Name                  string    `gorm:"size:50;uniqueIndex;not null" json:"name"`
	Course                int       `gorm:"not null" json:"course"`
	ScheduleSourceGroupID *int      `gorm:"" json:"schedule_source_group_id"`
	CurriculumID          *int64    `gorm:"" json:"curriculum_id"`
	AdmissionYear         *int16    `gorm:"" json:"admission_year"`
	SpecialtyID           *int      `gorm:"" json:"specialty_id"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type Subject struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	ShortName string    `gorm:"size:30" json:"short_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Location struct {
	ID           int       `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"size:50;not null" json:"name"`
	IsVirtual    bool      `gorm:"not null;default:false" json:"is_virtual"`
	Campus       string    `gorm:"size:50;not null;default:''" json:"campus"`
	LocationKind string    `gorm:"type:text;not null;default:'classroom'" json:"location_kind"`
	Capacity     *int16    `json:"capacity"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Teacher struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TeacherSubject struct {
	TeacherID int `gorm:"primaryKey" json:"teacher_id"`
	SubjectID int `gorm:"primaryKey" json:"subject_id"`
}

func (TeacherSubject) TableName() string { return "teacher_subjects" }

type CourseAssignment struct {
	ID               int64        `gorm:"primaryKey" json:"id"`
	GroupID          int          `gorm:"not null" json:"group_id"`
	Semester         int16        `gorm:"not null" json:"semester"`
	SubjectID        int          `gorm:"not null" json:"subject_id"`
	Status           EntityStatus `gorm:"type:text;not null;default:'published'" json:"status"`
	TeacherID        *int         `json:"teacher_id"`
	LocationID       *int         `json:"location_id"`
	CurriculumItemID *int64       `json:"curriculum_item_id"`
	Subgroup         *int16       `json:"subgroup"`
	Notes            *string      `json:"notes"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

func (CourseAssignment) TableName() string { return "course_assignments" }

type ScheduleTemplate struct {
	ID             int64        `gorm:"primaryKey" json:"id"`
	GroupID        int          `gorm:"not null;index:idx_tpl_query,priority:1" json:"group_id"`
	DayOfWeek      int16        `gorm:"not null;index:idx_tpl_query,priority:2" json:"day_of_week"`
	WeekParity     WeekParity   `gorm:"type:text;not null;index:idx_tpl_query,priority:3" json:"week_parity"`
	PairNumber     int16        `gorm:"not null;index:idx_tpl_query,priority:4" json:"pair_number"`
	SubjectID      int          `gorm:"not null" json:"subject_id"`
	LocationID     int          `gorm:"not null" json:"location_id"`
	Status         EntityStatus `gorm:"type:text;not null;default:'published'" json:"status"`
	TeacherManual  bool         `gorm:"not null;default:false" json:"teacher_manual"`
	LocationManual bool         `gorm:"not null;default:false" json:"location_manual"`
	TeacherID      *int         `gorm:"" json:"-"`
	TeacherName    string       `gorm:"column:teacher_name;->" json:"teacher_name"`
	Subgroup       *int16       `gorm:"" json:"subgroup"` // nil = вся группа
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

type ScheduleOverride struct {
	ID               int64          `gorm:"primaryKey" json:"id"`
	TargetDate       time.Time      `gorm:"type:date;not null;index:idx_ovr_query,priority:2" json:"target_date"`
	GroupID          int            `gorm:"not null;index:idx_ovr_query,priority:1" json:"group_id"`
	PairNumber       int16          `gorm:"not null" json:"pair_number"`
	ActionType       OverrideAction `gorm:"type:text;not null" json:"action_type"`
	NewSubjectID     *int           `json:"new_subject_id"`
	NewLocationID    *int           `json:"new_location_id"`
	NewTeacherID     *int           `json:"-"`
	NewTeacherManual bool           `gorm:"not null;default:false" json:"new_teacher_manual"`
	NewTeacherName   *string        `gorm:"column:new_teacher_name;->" json:"new_teacher_name"`
	Comment          *string        `json:"comment"`
	Subgroup         *int16         `json:"subgroup"` // nil = для всех
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type ScheduleDayOverlay struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	TargetDate  time.Time `gorm:"type:date;not null;uniqueIndex:uidx_overlay" json:"target_date"`
	GroupID     int       `gorm:"not null;uniqueIndex:uidx_overlay" json:"group_id"`
	Text        string    `gorm:"size:255;not null" json:"text"`
	StylePreset string    `gorm:"size:30;not null;default:standard" json:"style_preset"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CalendarException struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	TargetDate time.Time `gorm:"type:date;not null;uniqueIndex" json:"target_date"`
	WorksAsDay int16     `gorm:"not null" json:"works_as_day"`
	Comment    *string   `json:"comment"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ScheduleDayEvent struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	TargetDate time.Time `gorm:"type:date;not null;index:idx_day_events_lookup,priority:2" json:"target_date"`
	GroupID    int       `gorm:"not null;index:idx_day_events_lookup,priority:1" json:"group_id"`
	EventType  string    `gorm:"type:text;not null" json:"event_type"`
	Title      string    `gorm:"size:200;not null" json:"title"`
	Details    *string   `json:"details"`
	LocationID *int      `json:"location_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type SystemState struct {
	ID              int16     `gorm:"primaryKey" json:"id"`
	ScheduleVersion time.Time `gorm:"not null" json:"schedule_version"`
}

func (SystemState) TableName() string { return "system_state" }

type Specialty struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	Code      string    `gorm:"size:20;uniqueIndex;not null" json:"code"`
	Name      string    `gorm:"size:200;not null" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Curriculum struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	SpecialtyID   int       `gorm:"not null" json:"specialty_id"`
	AdmissionYear int16     `gorm:"not null" json:"admission_year"`
	Variant       string    `gorm:"size:50;not null;default:''" json:"variant"`
	Title         string    `gorm:"size:200;not null;default:''" json:"title"`
	Notes         *string   `json:"notes"`
	IsActive      bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Curriculum) TableName() string { return "curricula" }

type AcademicCalendar struct {
	ID                int64     `gorm:"primaryKey" json:"id"`
	CurriculumID      int64     `gorm:"not null" json:"curriculum_id"`
	AcademicYearStart time.Time `gorm:"type:date;not null" json:"academic_year_start"`
	WeeksTotal        int16     `gorm:"not null;default:52" json:"weeks_total"`
	Notes             *string   `json:"notes"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type AcademicCalendarWeek struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	CalendarID    int64     `gorm:"not null" json:"calendar_id"`
	WeekNumber    int16     `gorm:"not null" json:"week_number"`
	WeekStartDate time.Time `gorm:"type:date;not null" json:"week_start_date"`
	ActivityCode  string    `gorm:"size:10;not null" json:"activity_code"`
	ActivityName  *string   `gorm:"size:100" json:"activity_name"`
	IsTeaching    bool      `gorm:"not null;default:true" json:"is_teaching"`
	Comment       *string   `json:"comment"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CurriculumItem struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	CurriculumID int64     `gorm:"not null" json:"curriculum_id"`
	ParentID     *int64    `json:"parent_id"`
	IndexCode    *string   `gorm:"size:30" json:"index_code"`
	ItemType     string    `gorm:"type:text;not null" json:"item_type"`
	Name         string    `gorm:"size:200;not null" json:"name"`
	SubjectID    *int      `json:"subject_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CurriculumItemAllocation struct {
	ID               int64     `gorm:"primaryKey" json:"id"`
	ItemID           int64     `gorm:"not null" json:"item_id"`
	Semester         int16     `gorm:"not null" json:"semester"`
	Weeks            *int16    `json:"weeks"`
	HoursTotal       *int      `json:"hours_total"`
	HoursLectures    *int      `json:"hours_lectures"`
	HoursPractice    *int      `json:"hours_practice"`
	HoursLab         *int      `json:"hours_lab"`
	HoursIndependent *int      `json:"hours_independent"`
	HoursExam        *int      `json:"hours_exam"`
	AssessmentType   *string   `gorm:"type:text" json:"assessment_type"`
	Comment          *string   `json:"comment"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type StudyActivity struct {
	ID            int       `gorm:"primaryKey" json:"id"`
	Code          string    `gorm:"size:50;not null" json:"code"`
	Name          string    `gorm:"size:200;not null" json:"name"`
	ActivityKind  string    `gorm:"type:text;not null;default:'OTHER'" json:"activity_kind"`
	AllowsLessons bool      `gorm:"not null;default:true" json:"allows_lessons"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type StudyCalendarWeek struct {
	ID            int64      `gorm:"primaryKey" json:"id"`
	GroupID       int        `gorm:"not null" json:"group_id"`
	WeekNumber    int16      `gorm:"not null" json:"week_number"`
	WeekStartDate *time.Time `gorm:"type:date" json:"week_start_date"`
	ActivityID    *int       `json:"activity_id"`
	AllowsLessons bool       `gorm:"not null;default:true" json:"allows_lessons"`
	Comment       *string    `json:"comment"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type TeacherDayConstraint struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	TeacherID     int       `gorm:"not null" json:"teacher_id"`
	TargetDate    time.Time `gorm:"type:date;not null" json:"target_date"`
	Reason        string    `gorm:"size:255;not null;default:''" json:"reason"`
	AllowsLessons bool      `gorm:"not null;default:false" json:"allows_lessons"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ScheduleReplacement struct {
	ID                    int64     `gorm:"primaryKey" json:"id"`
	TargetDate            time.Time `gorm:"type:date;not null" json:"target_date"`
	GroupID               int       `gorm:"not null" json:"group_id"`
	PairNumber            int16     `gorm:"not null" json:"pair_number"`
	Subgroup              *int16    `json:"subgroup"`
	SourceSubjectID       *int      `json:"source_subject_id"`
	SourceLocationID      *int      `json:"source_location_id"`
	SourceTeacherID       *int      `json:"source_teacher_id"`
	ReplacementSubjectID  *int      `json:"replacement_subject_id"`
	ReplacementLocationID *int      `json:"replacement_location_id"`
	ReplacementTeacherID  *int      `json:"replacement_teacher_id"`
	Reason                *string   `json:"reason"`
	ScheduleOverrideID    *int64    `json:"schedule_override_id"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}
