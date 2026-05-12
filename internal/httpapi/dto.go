package httpapi

import (
	"time"

	"ispo-schedule/internal/schedule"
)

type groupDTO struct {
	ID                    int       `json:"id"`
	Name                  string    `json:"name"`
	Course                int       `json:"course"`
	ScheduleSourceGroupID *int      `json:"schedule_source_group_id"`
	CurriculumID          *int64    `json:"curriculum_id"`
	AdmissionYear         *int16    `json:"admission_year"`
	SpecialtyID           *int      `json:"specialty_id"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func toGroupDTO(g schedule.Group) groupDTO {
	return groupDTO{
		ID:                    g.ID,
		Name:                  g.Name,
		Course:                g.Course,
		ScheduleSourceGroupID: g.ScheduleSourceGroupID,
		CurriculumID:          g.CurriculumID,
		AdmissionYear:         g.AdmissionYear,
		SpecialtyID:           g.SpecialtyID,
		CreatedAt:             g.CreatedAt,
		UpdatedAt:             g.UpdatedAt,
	}
}

type subjectDTO struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	ShortName string    `json:"short_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toSubjectDTO(s schedule.Subject) subjectDTO {
	return subjectDTO{
		ID:        s.ID,
		Name:      s.Name,
		ShortName: s.ShortName,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

type locationDTO struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	IsVirtual    bool      `json:"is_virtual"`
	Campus       string    `json:"campus"`
	LocationKind string    `json:"location_kind"`
	Capacity     *int16    `json:"capacity"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func toLocationDTO(l schedule.Location) locationDTO {
	return locationDTO{
		ID:           l.ID,
		Name:         l.Name,
		IsVirtual:    l.IsVirtual,
		Campus:       l.Campus,
		LocationKind: l.LocationKind,
		Capacity:     l.Capacity,
		CreatedAt:    l.CreatedAt,
		UpdatedAt:    l.UpdatedAt,
	}
}

type teacherDTO struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toTeacherDTO(t schedule.Teacher) teacherDTO {
	return teacherDTO{
		ID:        t.ID,
		Name:      t.Name,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

type teacherSubjectDTO struct {
	TeacherID int `json:"teacher_id"`
	SubjectID int `json:"subject_id"`
}

func toTeacherSubjectDTO(ts schedule.TeacherSubject) teacherSubjectDTO {
	return teacherSubjectDTO{TeacherID: ts.TeacherID, SubjectID: ts.SubjectID}
}

type courseAssignmentDTO struct {
	ID               int64                 `json:"id"`
	GroupID          int                   `json:"group_id"`
	Semester         int16                 `json:"semester"`
	SubjectID        int                   `json:"subject_id"`
	Status           schedule.EntityStatus `json:"status"`
	TeacherID        *int                  `json:"teacher_id"`
	LocationID       *int                  `json:"location_id"`
	CurriculumItemID *int64                `json:"curriculum_item_id"`
	Subgroup         *int16                `json:"subgroup"`
	Notes            *string               `json:"notes"`
	CreatedAt        time.Time             `json:"created_at"`
	UpdatedAt        time.Time             `json:"updated_at"`
}

func toCourseAssignmentDTO(a schedule.CourseAssignment) courseAssignmentDTO {
	return courseAssignmentDTO{
		ID:               a.ID,
		GroupID:          a.GroupID,
		Semester:         a.Semester,
		SubjectID:        a.SubjectID,
		Status:           a.Status,
		TeacherID:        a.TeacherID,
		LocationID:       a.LocationID,
		CurriculumItemID: a.CurriculumItemID,
		Subgroup:         a.Subgroup,
		Notes:            a.Notes,
		CreatedAt:        a.CreatedAt,
		UpdatedAt:        a.UpdatedAt,
	}
}

type specialtyDTO struct {
	ID        int       `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toSpecialtyDTO(s schedule.Specialty) specialtyDTO {
	return specialtyDTO{ID: s.ID, Code: s.Code, Name: s.Name, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt}
}

type curriculumDTO struct {
	ID            int64     `json:"id"`
	SpecialtyID   int       `json:"specialty_id"`
	AdmissionYear int16     `json:"admission_year"`
	Variant       string    `json:"variant"`
	Title         string    `json:"title"`
	Notes         *string   `json:"notes"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func toCurriculumDTO(c schedule.Curriculum) curriculumDTO {
	return curriculumDTO{
		ID:            c.ID,
		SpecialtyID:   c.SpecialtyID,
		AdmissionYear: c.AdmissionYear,
		Variant:       c.Variant,
		Title:         c.Title,
		Notes:         c.Notes,
		IsActive:      c.IsActive,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}

type academicCalendarDTO struct {
	ID                int64     `json:"id"`
	CurriculumID      int64     `json:"curriculum_id"`
	AcademicYearStart time.Time `json:"academic_year_start"`
	WeeksTotal        int16     `json:"weeks_total"`
	Notes             *string   `json:"notes"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func toAcademicCalendarDTO(a schedule.AcademicCalendar) academicCalendarDTO {
	return academicCalendarDTO{
		ID:                a.ID,
		CurriculumID:      a.CurriculumID,
		AcademicYearStart: a.AcademicYearStart,
		WeeksTotal:        a.WeeksTotal,
		Notes:             a.Notes,
		CreatedAt:         a.CreatedAt,
		UpdatedAt:         a.UpdatedAt,
	}
}

type academicCalendarWeekDTO struct {
	ID            int64     `json:"id"`
	CalendarID    int64     `json:"calendar_id"`
	WeekNumber    int16     `json:"week_number"`
	WeekStartDate time.Time `json:"week_start_date"`
	ActivityCode  string    `json:"activity_code"`
	ActivityName  *string   `json:"activity_name"`
	IsTeaching    bool      `json:"is_teaching"`
	Comment       *string   `json:"comment"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func toAcademicCalendarWeekDTO(w schedule.AcademicCalendarWeek) academicCalendarWeekDTO {
	return academicCalendarWeekDTO{
		ID:            w.ID,
		CalendarID:    w.CalendarID,
		WeekNumber:    w.WeekNumber,
		WeekStartDate: w.WeekStartDate,
		ActivityCode:  w.ActivityCode,
		ActivityName:  w.ActivityName,
		IsTeaching:    w.IsTeaching,
		Comment:       w.Comment,
		CreatedAt:     w.CreatedAt,
		UpdatedAt:     w.UpdatedAt,
	}
}

type curriculumItemDTO struct {
	ID           int64     `json:"id"`
	CurriculumID int64     `json:"curriculum_id"`
	ParentID     *int64    `json:"parent_id"`
	IndexCode    *string   `json:"index_code"`
	ItemType     string    `json:"item_type"`
	Name         string    `json:"name"`
	SubjectID    *int      `json:"subject_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func toCurriculumItemDTO(it schedule.CurriculumItem) curriculumItemDTO {
	return curriculumItemDTO{
		ID:           it.ID,
		CurriculumID: it.CurriculumID,
		ParentID:     it.ParentID,
		IndexCode:    it.IndexCode,
		ItemType:     it.ItemType,
		Name:         it.Name,
		SubjectID:    it.SubjectID,
		CreatedAt:    it.CreatedAt,
		UpdatedAt:    it.UpdatedAt,
	}
}

type curriculumItemAllocationDTO struct {
	ID               int64     `json:"id"`
	ItemID           int64     `json:"item_id"`
	Semester         int16     `json:"semester"`
	Weeks            *int16    `json:"weeks"`
	HoursTotal       *int      `json:"hours_total"`
	HoursLectures    *int      `json:"hours_lectures"`
	HoursPractice    *int      `json:"hours_practice"`
	HoursLab         *int      `json:"hours_lab"`
	HoursIndependent *int      `json:"hours_independent"`
	HoursExam        *int      `json:"hours_exam"`
	AssessmentType   *string   `json:"assessment_type"`
	Comment          *string   `json:"comment"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func toCurriculumItemAllocationDTO(a schedule.CurriculumItemAllocation) curriculumItemAllocationDTO {
	return curriculumItemAllocationDTO{
		ID:               a.ID,
		ItemID:           a.ItemID,
		Semester:         a.Semester,
		Weeks:            a.Weeks,
		HoursTotal:       a.HoursTotal,
		HoursLectures:    a.HoursLectures,
		HoursPractice:    a.HoursPractice,
		HoursLab:         a.HoursLab,
		HoursIndependent: a.HoursIndependent,
		HoursExam:        a.HoursExam,
		AssessmentType:   a.AssessmentType,
		Comment:          a.Comment,
		CreatedAt:        a.CreatedAt,
		UpdatedAt:        a.UpdatedAt,
	}
}

type scheduleTemplateDTO struct {
	ID             int64                 `json:"id"`
	GroupID        int                   `json:"group_id"`
	DayOfWeek      int16                 `json:"day_of_week"`
	WeekParity     schedule.WeekParity   `json:"week_parity"`
	PairNumber     int16                 `json:"pair_number"`
	SubjectID      int                   `json:"subject_id"`
	LocationID     int                   `json:"location_id"`
	Status         schedule.EntityStatus `json:"status"`
	TeacherManual  bool                  `json:"teacher_manual"`
	LocationManual bool                  `json:"location_manual"`
	TeacherName    string                `json:"teacher_name"`
	Subgroup       *int16                `json:"subgroup"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

func toScheduleTemplateDTO(t schedule.ScheduleTemplate) scheduleTemplateDTO {
	return scheduleTemplateDTO{
		ID:             t.ID,
		GroupID:        t.GroupID,
		DayOfWeek:      t.DayOfWeek,
		WeekParity:     t.WeekParity,
		PairNumber:     t.PairNumber,
		SubjectID:      t.SubjectID,
		LocationID:     t.LocationID,
		Status:         t.Status,
		TeacherManual:  t.TeacherManual,
		LocationManual: t.LocationManual,
		TeacherName:    t.TeacherName,
		Subgroup:       t.Subgroup,
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
	}
}

type scheduleOverrideDTO struct {
	ID               int64                   `json:"id"`
	TargetDate       time.Time               `json:"target_date"`
	GroupID          int                     `json:"group_id"`
	PairNumber       int16                   `json:"pair_number"`
	ActionType       schedule.OverrideAction `json:"action_type"`
	NewSubjectID     *int                    `json:"new_subject_id"`
	NewLocationID    *int                    `json:"new_location_id"`
	NewTeacherManual bool                    `json:"new_teacher_manual"`
	NewTeacherName   *string                 `json:"new_teacher_name"`
	Comment          *string                 `json:"comment"`
	Subgroup         *int16                  `json:"subgroup"`
	CreatedAt        time.Time               `json:"created_at"`
	UpdatedAt        time.Time               `json:"updated_at"`
}

func toScheduleOverrideDTO(o schedule.ScheduleOverride) scheduleOverrideDTO {
	return scheduleOverrideDTO{
		ID:               o.ID,
		TargetDate:       o.TargetDate,
		GroupID:          o.GroupID,
		PairNumber:       o.PairNumber,
		ActionType:       o.ActionType,
		NewSubjectID:     o.NewSubjectID,
		NewLocationID:    o.NewLocationID,
		NewTeacherManual: o.NewTeacherManual,
		NewTeacherName:   o.NewTeacherName,
		Comment:          o.Comment,
		Subgroup:         o.Subgroup,
		CreatedAt:        o.CreatedAt,
		UpdatedAt:        o.UpdatedAt,
	}
}

type scheduleDayOverlayDTO struct {
	ID          int64     `json:"id"`
	TargetDate  time.Time `json:"target_date"`
	GroupID     int       `json:"group_id"`
	Text        string    `json:"text"`
	StylePreset string    `json:"style_preset"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toScheduleDayOverlayDTO(o schedule.ScheduleDayOverlay) scheduleDayOverlayDTO {
	return scheduleDayOverlayDTO{
		ID:          o.ID,
		TargetDate:  o.TargetDate,
		GroupID:     o.GroupID,
		Text:        o.Text,
		StylePreset: o.StylePreset,
		CreatedAt:   o.CreatedAt,
		UpdatedAt:   o.UpdatedAt,
	}
}

type calendarExceptionDTO struct {
	ID         int64     `json:"id"`
	TargetDate time.Time `json:"target_date"`
	WorksAsDay int16     `json:"works_as_day"`
	Comment    *string   `json:"comment"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func toCalendarExceptionDTO(e schedule.CalendarException) calendarExceptionDTO {
	return calendarExceptionDTO{
		ID:         e.ID,
		TargetDate: e.TargetDate,
		WorksAsDay: e.WorksAsDay,
		Comment:    e.Comment,
		CreatedAt:  e.CreatedAt,
		UpdatedAt:  e.UpdatedAt,
	}
}

type scheduleDayEventDTO struct {
	ID         int64     `json:"id"`
	TargetDate time.Time `json:"target_date"`
	GroupID    int       `json:"group_id"`
	EventType  string    `json:"event_type"`
	Title      string    `json:"title"`
	Details    *string   `json:"details"`
	LocationID *int      `json:"location_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func toScheduleDayEventDTO(e schedule.ScheduleDayEvent) scheduleDayEventDTO {
	return scheduleDayEventDTO{
		ID:         e.ID,
		TargetDate: e.TargetDate,
		GroupID:    e.GroupID,
		EventType:  e.EventType,
		Title:      e.Title,
		Details:    e.Details,
		LocationID: e.LocationID,
		CreatedAt:  e.CreatedAt,
		UpdatedAt:  e.UpdatedAt,
	}
}

type studyActivityDTO struct {
	ID            int       `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	ActivityKind  string    `json:"activity_kind"`
	AllowsLessons bool      `json:"allows_lessons"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func toStudyActivityDTO(a schedule.StudyActivity) studyActivityDTO {
	return studyActivityDTO{
		ID:            a.ID,
		Code:          a.Code,
		Name:          a.Name,
		ActivityKind:  a.ActivityKind,
		AllowsLessons: a.AllowsLessons,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
}

type studyCalendarWeekDTO struct {
	ID            int64      `json:"id"`
	GroupID       int        `json:"group_id"`
	WeekNumber    int16      `json:"week_number"`
	WeekStartDate *time.Time `json:"week_start_date"`
	ActivityID    *int       `json:"activity_id"`
	AllowsLessons bool       `json:"allows_lessons"`
	Comment       *string    `json:"comment"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func toStudyCalendarWeekDTO(w schedule.StudyCalendarWeek) studyCalendarWeekDTO {
	return studyCalendarWeekDTO{
		ID:            w.ID,
		GroupID:       w.GroupID,
		WeekNumber:    w.WeekNumber,
		WeekStartDate: w.WeekStartDate,
		ActivityID:    w.ActivityID,
		AllowsLessons: w.AllowsLessons,
		Comment:       w.Comment,
		CreatedAt:     w.CreatedAt,
		UpdatedAt:     w.UpdatedAt,
	}
}

type teacherDayConstraintDTO struct {
	ID            int64     `json:"id"`
	TeacherID     int       `json:"teacher_id"`
	TargetDate    time.Time `json:"target_date"`
	Reason        string    `json:"reason"`
	AllowsLessons bool      `json:"allows_lessons"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func toTeacherDayConstraintDTO(d schedule.TeacherDayConstraint) teacherDayConstraintDTO {
	return teacherDayConstraintDTO{
		ID:            d.ID,
		TeacherID:     d.TeacherID,
		TargetDate:    d.TargetDate,
		Reason:        d.Reason,
		AllowsLessons: d.AllowsLessons,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}

type scheduleReplacementDTO struct {
	ID                    int64     `json:"id"`
	TargetDate            time.Time `json:"target_date"`
	GroupID               int       `json:"group_id"`
	PairNumber            int16     `json:"pair_number"`
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

func toScheduleReplacementDTO(r schedule.ScheduleReplacement) scheduleReplacementDTO {
	return scheduleReplacementDTO{
		ID:                    r.ID,
		TargetDate:            r.TargetDate,
		GroupID:               r.GroupID,
		PairNumber:            r.PairNumber,
		Subgroup:              r.Subgroup,
		SourceSubjectID:       r.SourceSubjectID,
		SourceLocationID:      r.SourceLocationID,
		SourceTeacherID:       r.SourceTeacherID,
		ReplacementSubjectID:  r.ReplacementSubjectID,
		ReplacementLocationID: r.ReplacementLocationID,
		ReplacementTeacherID:  r.ReplacementTeacherID,
		Reason:                r.Reason,
		ScheduleOverrideID:    r.ScheduleOverrideID,
		CreatedAt:             r.CreatedAt,
		UpdatedAt:             r.UpdatedAt,
	}
}
