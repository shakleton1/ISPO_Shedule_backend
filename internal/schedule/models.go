package schedule

import "time"

type EntityStatus string

const (
	StatusDraft     EntityStatus = "draft"
	StatusPublished EntityStatus = "published"
	StatusCancelled EntityStatus = "cancelled"
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
	Name      string    `gorm:"type:text;not null" json:"name"`
	ShortName string    `gorm:"size:30" json:"short_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Location struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	CampusID  *int      `json:"campus_id"`
	Name      string    `gorm:"size:50;not null" json:"name"`
	Kind      string    `gorm:"type:text;not null;default:'physical'" json:"kind"`
	Capacity  *int16    `json:"capacity"`
	IsActive  bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Campus struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;uniqueIndex;not null" json:"name"`
	Address   *string   `json:"address"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Campus) TableName() string { return "campuses" }

type LocationType struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	Code      string    `gorm:"size:50;uniqueIndex;not null" json:"code"`
	Name      string    `gorm:"size:100;not null" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LocationTypeLink struct {
	LocationID int       `gorm:"primaryKey" json:"location_id"`
	TypeID     int       `gorm:"primaryKey" json:"type_id"`
	CreatedAt  time.Time `json:"created_at"`
}

func (LocationTypeLink) TableName() string { return "location_type_links" }

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
	CampusID         *int         `json:"campus_id"`
	CurriculumItemID *int64       `json:"curriculum_item_id"`
	Subgroup         *int16       `json:"subgroup"`
	Notes            *string      `json:"notes"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

func (CourseAssignment) TableName() string { return "course_assignments" }

type ScheduleLesson struct {
	ID           int64        `gorm:"primaryKey" json:"id"`
	GroupID      int          `gorm:"not null" json:"group_id"`
	LessonDate   time.Time    `gorm:"type:date;not null" json:"lesson_date"`
	PairNumber   int16        `gorm:"not null" json:"pair_number"`
	Subgroup     *int16       `json:"subgroup"`
	SubjectID    *int         `json:"subject_id"`
	TeacherID    *int         `json:"teacher_id"`
	LessonFormat string       `gorm:"type:text;not null;default:'offline'" json:"lesson_format"`
	Status       EntityStatus `gorm:"type:text;not null;default:'published'" json:"status"`
	Source       string       `gorm:"type:text;not null;default:'manual'" json:"source"`
	FlowKey      *string      `gorm:"size:80" json:"flow_key"`
	Comment      *string      `json:"comment"`
	Version      int          `gorm:"not null;default:1" json:"version"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

type scheduleTemplateLegacy struct {
	ID             int64        `gorm:"primaryKey" json:"id"`
	GroupID        int          `gorm:"not null;index:idx_tpl_query,priority:1" json:"group_id"`
	DayOfWeek      int16        `gorm:"not null;index:idx_tpl_query,priority:2" json:"day_of_week"`
	WeekParity     WeekParity   `gorm:"type:text;not null;index:idx_tpl_query,priority:3" json:"week_parity"`
	PairNumber     int16        `gorm:"not null;index:idx_tpl_query,priority:4" json:"pair_number"`
	SubjectID      int          `gorm:"not null" json:"subject_id"`
	LocationID     *int         `json:"location_id"`
	LessonFormat   string       `gorm:"type:text;not null;default:'offline'" json:"lesson_format"`
	Status         EntityStatus `gorm:"type:text;not null;default:'published'" json:"status"`
	TeacherManual  bool         `gorm:"not null;default:false" json:"teacher_manual"`
	LocationManual bool         `gorm:"not null;default:false" json:"location_manual"`
	TeacherID      *int         `gorm:"" json:"-"`
	TeacherName    string       `gorm:"column:teacher_name;->" json:"teacher_name"`
	Subgroup       *int16       `gorm:"" json:"subgroup"` // nil = вся группа
	FlowKey        *string      `gorm:"size:80" json:"flow_key"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

type ScheduleOverride struct {
	ID                      int64          `gorm:"primaryKey" json:"id"`
	ScheduleLessonID        *int64         `json:"schedule_lesson_id"`
	LessonDate              time.Time      `gorm:"type:date;not null" json:"lesson_date"`
	GroupID                 int            `gorm:"not null" json:"group_id"`
	PairNumber              int16          `gorm:"not null" json:"pair_number"`
	ActionType              OverrideAction `gorm:"type:text;not null" json:"action_type"`
	SourceSubjectID         *int           `json:"source_subject_id"`
	SourceTeacherID         *int           `json:"source_teacher_id"`
	SourceLocationID        *int           `json:"source_location_id"`
	SourceLessonFormat      *string        `gorm:"type:text" json:"source_lesson_format"`
	Subgroup                *int16         `json:"subgroup"` // nil = для всех
	ReplacementSubjectID    *int           `json:"replacement_subject_id"`
	ReplacementTeacherID    *int           `json:"replacement_teacher_id"`
	ReplacementLocationID   *int           `json:"replacement_location_id"`
	ReplacementLessonFormat *string        `gorm:"type:text" json:"replacement_lesson_format"`
	Reason                  *string        `json:"reason"`
	Status                  string         `gorm:"type:text;not null;default:'applied'" json:"status"`
	ExpectedLessonVersion   *int           `json:"expected_lesson_version"`
	AppliedLessonVersion    *int           `json:"applied_lesson_version"`
	CreatedBy               *int           `json:"created_by"`
	CreatedAt               time.Time      `json:"created_at"`
	AppliedAt               *time.Time     `json:"applied_at"`
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

type CalendarDayConstraint struct {
	ID                   int64     `gorm:"primaryKey" json:"id"`
	TargetDate           time.Time `gorm:"type:date;not null;uniqueIndex" json:"target_date"`
	Title                string    `gorm:"size:200;not null" json:"title"`
	Reason               *string   `json:"reason"`
	ConstraintType       string    `gorm:"type:text;not null;default:'blocked'" json:"constraint_type"`
	AffectsLessons       bool      `gorm:"not null;default:true" json:"affects_lessons"`
	RequiresConfirmation bool      `gorm:"not null;default:false" json:"requires_confirmation"`
	StylePreset          string    `gorm:"size:30;not null;default:'warning'" json:"style_preset"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type calendarExceptionLegacy struct {
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
	CourseNumber  int16     `gorm:"not null;default:1" json:"course_number"`
	WeekNumber    int16     `gorm:"not null" json:"week_number"`
	WeekStartDate time.Time `gorm:"type:date;not null" json:"week_start_date"`
	ActivityCode  string    `gorm:"size:50;not null" json:"activity_code"`
	ActivityName  *string   `gorm:"size:200" json:"activity_name"`
	IsTeaching    bool      `gorm:"not null;default:true" json:"is_teaching"`
	Comment       *string   `json:"comment"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type AcademicCalendarDayOverride struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	CalendarID   int64     `gorm:"not null" json:"calendar_id"`
	CourseNumber int16     `gorm:"not null" json:"course_number"`
	WeekNumber   int16     `gorm:"not null" json:"week_number"`
	DayOfWeek    int16     `gorm:"not null" json:"day_of_week"`
	ActivityCode string    `gorm:"size:50;not null" json:"activity_code"`
	ActivityName *string   `gorm:"size:200" json:"activity_name"`
	IsTeaching   bool      `gorm:"not null;default:true" json:"is_teaching"`
	Comment      *string   `json:"comment"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CurriculumItem struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	CurriculumID int64     `gorm:"not null" json:"curriculum_id"`
	ParentID     *int64    `json:"parent_id"`
	IndexCode    *string   `gorm:"size:30" json:"index_code"`
	ItemType     string    `gorm:"type:text;not null" json:"item_type"`
	Name         string    `gorm:"type:text;not null" json:"name"`
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
	ID                   int64     `gorm:"primaryKey" json:"id"`
	TeacherID            int       `gorm:"not null" json:"teacher_id"`
	TargetDate           time.Time `gorm:"type:date;not null" json:"target_date"`
	Reason               string    `gorm:"size:255;not null;default:''" json:"reason"`
	ConstraintLevel      string    `gorm:"type:text;not null;default:'warning'" json:"constraint_level"`
	RequiresConfirmation bool      `gorm:"not null;default:true" json:"requires_confirmation"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
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

type LocationWeekAvailability struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	WeekStartDate time.Time `gorm:"type:date;not null" json:"week_start_date"`
	LocationID    int       `gorm:"not null" json:"location_id"`
	IsAvailable   bool      `gorm:"not null;default:true" json:"is_available"`
	Comment       *string   `json:"comment"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (LocationWeekAvailability) TableName() string { return "location_week_availability" }

type TeacherLocationPreference struct {
	ID         int64      `gorm:"primaryKey" json:"id"`
	TeacherID  int        `gorm:"not null" json:"teacher_id"`
	LocationID int        `gorm:"not null" json:"location_id"`
	Priority   int        `gorm:"not null;default:100" json:"priority"`
	ValidFrom  *time.Time `gorm:"type:date" json:"valid_from"`
	ValidTo    *time.Time `gorm:"type:date" json:"valid_to"`
	Comment    *string    `json:"comment"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type RoomRequest struct {
	ID                  int64     `gorm:"primaryKey" json:"id"`
	TeacherID           *int      `json:"teacher_id"`
	SubjectID           *int      `json:"subject_id"`
	GroupID             *int      `json:"group_id"`
	Semester            *int16    `json:"semester"`
	RequiredTypeID      *int      `json:"required_type_id"`
	PreferredLocationID *int      `json:"preferred_location_id"`
	Priority            int       `gorm:"not null;default:100" json:"priority"`
	Comment             *string   `json:"comment"`
	Status              string    `gorm:"type:text;not null;default:'pending'" json:"status"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type RoomAssignment struct {
	ID               int64        `gorm:"primaryKey" json:"id"`
	ScheduleLessonID int64        `gorm:"not null" json:"schedule_lesson_id"`
	LocationID       int          `gorm:"not null" json:"location_id"`
	Source           string       `gorm:"type:text;not null;default:'manual'" json:"source"`
	Status           EntityStatus `gorm:"type:text;not null;default:'published'" json:"status"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}
