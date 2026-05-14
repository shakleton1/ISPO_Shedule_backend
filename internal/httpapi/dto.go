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
	ID        int       `json:"id"`
	CampusID  *int      `json:"campus_id"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	Capacity  *int16    `json:"capacity"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toLocationDTO(l schedule.Location) locationDTO {
	return locationDTO{
		ID:        l.ID,
		CampusID:  l.CampusID,
		Name:      l.Name,
		Kind:      l.Kind,
		Capacity:  l.Capacity,
		IsActive:  l.IsActive,
		CreatedAt: l.CreatedAt,
		UpdatedAt: l.UpdatedAt,
	}
}

type campusDTO struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Address   *string   `json:"address"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toCampusDTO(c schedule.Campus) campusDTO {
	return campusDTO{ID: c.ID, Name: c.Name, Address: c.Address, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt}
}

type locationTypeDTO struct {
	ID        int       `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toLocationTypeDTO(t schedule.LocationType) locationTypeDTO {
	return locationTypeDTO{ID: t.ID, Code: t.Code, Name: t.Name, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt}
}

type locationTypeLinkDTO struct {
	LocationID int       `json:"location_id"`
	TypeID     int       `json:"type_id"`
	CreatedAt  time.Time `json:"created_at"`
}

func toLocationTypeLinkDTO(l schedule.LocationTypeLink) locationTypeLinkDTO {
	return locationTypeLinkDTO{LocationID: l.LocationID, TypeID: l.TypeID, CreatedAt: l.CreatedAt}
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

type scheduleLessonDTO struct {
	ID           int64                 `json:"id"`
	GroupID      int                   `json:"group_id"`
	LessonDate   time.Time             `json:"lesson_date"`
	PairNumber   int16                 `json:"pair_number"`
	Subgroup     *int16                `json:"subgroup"`
	SubjectID    *int                  `json:"subject_id"`
	TeacherID    *int                  `json:"teacher_id"`
	LessonFormat string                `json:"lesson_format"`
	Status       schedule.EntityStatus `json:"status"`
	Source       string                `json:"source"`
	FlowKey      *string               `json:"flow_key,omitempty"`
	Comment      *string               `json:"comment"`
	Version      int                   `json:"version"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
}

func toScheduleLessonDTO(l schedule.ScheduleLesson) scheduleLessonDTO {
	return scheduleLessonDTO{
		ID:           l.ID,
		GroupID:      l.GroupID,
		LessonDate:   l.LessonDate,
		PairNumber:   l.PairNumber,
		Subgroup:     l.Subgroup,
		SubjectID:    l.SubjectID,
		TeacherID:    l.TeacherID,
		LessonFormat: l.LessonFormat,
		Status:       l.Status,
		Source:       l.Source,
		FlowKey:      l.FlowKey,
		Comment:      l.Comment,
		Version:      l.Version,
		CreatedAt:    l.CreatedAt,
		UpdatedAt:    l.UpdatedAt,
	}
}

type scheduleOverrideDTO struct {
	ID                      int64                   `json:"id"`
	ScheduleLessonID        *int64                  `json:"schedule_lesson_id"`
	GroupID                 int                     `json:"group_id"`
	LessonDate              time.Time               `json:"lesson_date"`
	PairNumber              int16                   `json:"pair_number"`
	Subgroup                *int16                  `json:"subgroup"`
	ActionType              schedule.OverrideAction `json:"action_type"`
	SourceSubjectID         *int                    `json:"source_subject_id"`
	SourceTeacherID         *int                    `json:"source_teacher_id"`
	SourceLocationID        *int                    `json:"source_location_id"`
	SourceLessonFormat      *string                 `json:"source_lesson_format"`
	ReplacementSubjectID    *int                    `json:"replacement_subject_id"`
	ReplacementTeacherID    *int                    `json:"replacement_teacher_id"`
	ReplacementLocationID   *int                    `json:"replacement_location_id"`
	ReplacementLessonFormat *string                 `json:"replacement_lesson_format"`
	Reason                  *string                 `json:"reason"`
	Status                  string                  `json:"status"`
	ExpectedLessonVersion   *int                    `json:"expected_lesson_version"`
	AppliedLessonVersion    *int                    `json:"applied_lesson_version"`
	CreatedBy               *int                    `json:"created_by"`
	CreatedAt               time.Time               `json:"created_at"`
	AppliedAt               *time.Time              `json:"applied_at"`
}

func toScheduleOverrideDTO(o schedule.ScheduleOverride) scheduleOverrideDTO {
	return scheduleOverrideDTO{
		ID:                      o.ID,
		ScheduleLessonID:        o.ScheduleLessonID,
		GroupID:                 o.GroupID,
		LessonDate:              o.LessonDate,
		PairNumber:              o.PairNumber,
		Subgroup:                o.Subgroup,
		ActionType:              o.ActionType,
		SourceSubjectID:         o.SourceSubjectID,
		SourceTeacherID:         o.SourceTeacherID,
		SourceLocationID:        o.SourceLocationID,
		SourceLessonFormat:      o.SourceLessonFormat,
		ReplacementSubjectID:    o.ReplacementSubjectID,
		ReplacementTeacherID:    o.ReplacementTeacherID,
		ReplacementLocationID:   o.ReplacementLocationID,
		ReplacementLessonFormat: o.ReplacementLessonFormat,
		Reason:                  o.Reason,
		Status:                  o.Status,
		ExpectedLessonVersion:   o.ExpectedLessonVersion,
		AppliedLessonVersion:    o.AppliedLessonVersion,
		CreatedBy:               o.CreatedBy,
		CreatedAt:               o.CreatedAt,
		AppliedAt:               o.AppliedAt,
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
	ID                   int64     `json:"id"`
	TeacherID            int       `json:"teacher_id"`
	TargetDate           time.Time `json:"target_date"`
	Reason               string    `json:"reason"`
	ConstraintLevel      string    `json:"constraint_level"`
	RequiresConfirmation bool      `json:"requires_confirmation"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func toTeacherDayConstraintDTO(d schedule.TeacherDayConstraint) teacherDayConstraintDTO {
	return teacherDayConstraintDTO{
		ID:                   d.ID,
		TeacherID:            d.TeacherID,
		TargetDate:           d.TargetDate,
		Reason:               d.Reason,
		ConstraintLevel:      d.ConstraintLevel,
		RequiresConfirmation: d.RequiresConfirmation,
		CreatedAt:            d.CreatedAt,
		UpdatedAt:            d.UpdatedAt,
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

type locationWeekAvailabilityDTO struct {
	ID            int64     `json:"id"`
	WeekStartDate time.Time `json:"week_start_date"`
	LocationID    int       `json:"location_id"`
	IsAvailable   bool      `json:"is_available"`
	Comment       *string   `json:"comment"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func toLocationWeekAvailabilityDTO(a schedule.LocationWeekAvailability) locationWeekAvailabilityDTO {
	return locationWeekAvailabilityDTO{
		ID:            a.ID,
		WeekStartDate: a.WeekStartDate,
		LocationID:    a.LocationID,
		IsAvailable:   a.IsAvailable,
		Comment:       a.Comment,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
}

type teacherLocationPreferenceDTO struct {
	ID         int64      `json:"id"`
	TeacherID  int        `json:"teacher_id"`
	LocationID int        `json:"location_id"`
	Priority   int        `json:"priority"`
	ValidFrom  *time.Time `json:"valid_from"`
	ValidTo    *time.Time `json:"valid_to"`
	Comment    *string    `json:"comment"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func toTeacherLocationPreferenceDTO(p schedule.TeacherLocationPreference) teacherLocationPreferenceDTO {
	return teacherLocationPreferenceDTO{
		ID:         p.ID,
		TeacherID:  p.TeacherID,
		LocationID: p.LocationID,
		Priority:   p.Priority,
		ValidFrom:  p.ValidFrom,
		ValidTo:    p.ValidTo,
		Comment:    p.Comment,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}

type roomRequestDTO struct {
	ID                  int64     `json:"id"`
	TeacherID           *int      `json:"teacher_id"`
	SubjectID           *int      `json:"subject_id"`
	GroupID             *int      `json:"group_id"`
	Semester            *int16    `json:"semester"`
	RequiredTypeID      *int      `json:"required_type_id"`
	PreferredLocationID *int      `json:"preferred_location_id"`
	Priority            int       `json:"priority"`
	Comment             *string   `json:"comment"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func toRoomRequestDTO(r schedule.RoomRequest) roomRequestDTO {
	return roomRequestDTO{
		ID:                  r.ID,
		TeacherID:           r.TeacherID,
		SubjectID:           r.SubjectID,
		GroupID:             r.GroupID,
		Semester:            r.Semester,
		RequiredTypeID:      r.RequiredTypeID,
		PreferredLocationID: r.PreferredLocationID,
		Priority:            r.Priority,
		Comment:             r.Comment,
		Status:              r.Status,
		CreatedAt:           r.CreatedAt,
		UpdatedAt:           r.UpdatedAt,
	}
}

type roomAssignmentDTO struct {
	ID               int64                 `json:"id"`
	ScheduleLessonID int64                 `json:"schedule_lesson_id"`
	LocationID       int                   `json:"location_id"`
	Source           string                `json:"source"`
	Status           schedule.EntityStatus `json:"status"`
	CreatedAt        time.Time             `json:"created_at"`
	UpdatedAt        time.Time             `json:"updated_at"`
}

func toRoomAssignmentDTO(a schedule.RoomAssignment) roomAssignmentDTO {
	return roomAssignmentDTO{
		ID:               a.ID,
		ScheduleLessonID: a.ScheduleLessonID,
		LocationID:       a.LocationID,
		Source:           a.Source,
		Status:           a.Status,
		CreatedAt:        a.CreatedAt,
		UpdatedAt:        a.UpdatedAt,
	}
}
