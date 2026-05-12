package schedule

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	studentWeeklyHoursLimit         = 36
	teacherWeeklyHoursLimit         = 40
	physicalEducationRoomGroupLimit = 3
	academicHoursPerPair            = 2
)

type ScheduleValidationWarning struct {
	Code        string `json:"code"`
	Date        string `json:"date"`
	PairNumber  int16  `json:"pair_number"`
	Subgroup    *int16 `json:"subgroup"`
	SubjectID   int    `json:"subject_id"`
	SubjectName string `json:"subject_name"`
	Semester    int16  `json:"semester"`
	Message     string `json:"message"`
}

type ScheduleValidationResponse struct {
	GroupID      int                         `json:"group_id"`
	StartDate    string                      `json:"start_date"`
	EndDate      string                      `json:"end_date"`
	Warnings     []ScheduleValidationWarning `json:"warnings"`
	WarnCount    int                         `json:"warn_count"`
	Validated    bool                        `json:"validated"`
	NoCurriculum bool                        `json:"no_curriculum"`
}

func (s *Service) ValidateScheduleRange(groupID int, startDate, endDate time.Time) (*ScheduleValidationResponse, error) {
	startDate = dateOnly(startDate)
	endDate = dateOnly(endDate)

	group, err := s.repo.GetGroup(groupID)
	if err != nil {
		return nil, err
	}

	resp := &ScheduleValidationResponse{
		GroupID:      groupID,
		StartDate:    startDate.Format("2006-01-02"),
		EndDate:      endDate.Format("2006-01-02"),
		Warnings:     []ScheduleValidationWarning{},
		Validated:    false,
		NoCurriculum: false,
	}

	days, err := s.buildDays(groupID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	locationMeta, err := s.locationMetaForDays(days)
	if err != nil {
		return nil, err
	}
	resp.Warnings = append(resp.Warnings, validateScheduleBusinessRules(days, *group, locationMeta)...)

	peFacilityWarnings, err := s.validatePhysicalEducationFacilitySchedule(groupID, *group, startDate, endDate, days)
	if err != nil {
		return nil, err
	}
	resp.Warnings = append(resp.Warnings, peFacilityWarnings...)

	if group.CurriculumID == nil {
		resp.NoCurriculum = true
	} else {
		// Gather semesters we can infer for the requested date range.
		semSet := map[int16]bool{}
		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			sem := inferSemesterForDate(d, group.Course)
			if sem != nil {
				semSet[*sem] = true
			}
		}
		semList := make([]int16, 0, len(semSet))
		for sem := range semSet {
			semList = append(semList, sem)
		}

		allowedBySem, err := s.repo.ListAllocatedSubjectsBySemester(*group.CurriculumID, semList)
		if err != nil {
			return nil, err
		}

		for _, day := range days {
			d, err := time.Parse("2006-01-02", day.Date)
			if err != nil {
				continue
			}
			sem := inferSemesterForDate(d, group.Course)
			if sem == nil {
				continue
			}
			allowed := allowedBySem[*sem]
			for _, l := range day.Lessons {
				if l.SubjectID == nil {
					continue
				}
				if allowed != nil && allowed[*l.SubjectID] {
					continue
				}
				// If allowed is nil/empty: treat as not allocated.
				resp.Warnings = append(resp.Warnings, ScheduleValidationWarning{
					Code:        "subject_not_in_curriculum",
					Date:        day.Date,
					PairNumber:  l.PairNumber,
					Subgroup:    l.Subgroup,
					SubjectID:   *l.SubjectID,
					SubjectName: l.SubjectName,
					Semester:    *sem,
					Message:     "subject is not allocated in curriculum for semester",
				})
			}
		}
	}

	blockedTeachers, err := s.repo.ListBlockingTeacherConstraintsBetween(startDate, endDate)
	if err != nil {
		return nil, err
	}
	blockedByDateName := map[string]map[string]TeacherDayConstraintView{}
	for _, b := range blockedTeachers {
		dateKey := dateOnly(b.TargetDate).Format("2006-01-02")
		m, ok := blockedByDateName[dateKey]
		if !ok {
			m = map[string]TeacherDayConstraintView{}
			blockedByDateName[dateKey] = m
		}
		m[b.TeacherName] = b
	}

	for _, day := range days {
		blocked := blockedByDateName[day.Date]
		if len(blocked) == 0 {
			continue
		}
		for _, l := range day.Lessons {
			if l.TeacherName == "" {
				continue
			}
			b, ok := blocked[l.TeacherName]
			if !ok {
				continue
			}
			subjectID := 0
			if l.SubjectID != nil {
				subjectID = *l.SubjectID
			}
			sem := int16(0)
			if d, err := time.Parse("2006-01-02", day.Date); err == nil {
				if inferred := inferSemesterForDate(d, group.Course); inferred != nil {
					sem = *inferred
				}
			}
			resp.Warnings = append(resp.Warnings, ScheduleValidationWarning{
				Code:        "teacher_day_constraint",
				Date:        day.Date,
				PairNumber:  l.PairNumber,
				Subgroup:    l.Subgroup,
				SubjectID:   subjectID,
				SubjectName: l.SubjectName,
				Semester:    sem,
				Message:     "teacher is unavailable: " + b.Reason,
			})
		}
	}

	resp.WarnCount = len(resp.Warnings)
	resp.Validated = true
	return resp, nil
}

func (s *Service) locationMetaForDays(days []DaySchedule) (map[int]LocationMeta, error) {
	seen := map[int]bool{}
	ids := make([]int, 0)
	for _, day := range days {
		for _, lesson := range day.Lessons {
			if lesson.LocationID == nil || *lesson.LocationID <= 0 {
				continue
			}
			if seen[*lesson.LocationID] {
				continue
			}
			seen[*lesson.LocationID] = true
			ids = append(ids, *lesson.LocationID)
		}
	}
	return s.repo.ListLocationMetaByIDs(ids)
}

func (s *Service) validatePhysicalEducationFacilitySchedule(groupID int, group Group, startDate, endDate time.Time, targetDays []DaySchedule) ([]ScheduleValidationWarning, error) {
	if !hasPhysicalEducationLessons(targetDays) {
		return nil, nil
	}

	groups, err := s.repo.ListGroups()
	if err != nil {
		return nil, err
	}

	groupsByID := map[int]Group{groupID: group}
	allDaysByGroup := map[int][]DaySchedule{groupID: targetDays}
	locationIDs := map[int]bool{}
	collectLocationIDsFromDays(targetDays, locationIDs)

	for _, g := range groups {
		groupsByID[g.ID] = g
		if g.ID == groupID {
			continue
		}
		days, err := s.buildDays(g.ID, startDate, endDate)
		if err != nil {
			return nil, err
		}
		allDaysByGroup[g.ID] = days
		collectLocationIDsFromDays(days, locationIDs)
	}

	ids := make([]int, 0, len(locationIDs))
	for id := range locationIDs {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	locationMeta, err := s.repo.ListLocationMetaByIDs(ids)
	if err != nil {
		return nil, err
	}
	return validatePhysicalEducationFacilityRules(groupID, groupsByID, allDaysByGroup, locationMeta), nil
}

func validateScheduleBusinessRules(days []DaySchedule, group Group, locationMeta map[int]LocationMeta) []ScheduleValidationWarning {
	warnings := make([]ScheduleValidationWarning, 0)
	warnings = append(warnings, validateWeeklyStudentHours(days)...)
	warnings = append(warnings, validateMissingTeachers(days, group)...)
	warnings = append(warnings, validateWeeklyTeacherHours(days, group)...)
	warnings = append(warnings, validateFloatingDayOff(days, group)...)
	warnings = append(warnings, validatePhysicalEducationFifthPair(days, group)...)
	warnings = append(warnings, validateThreePairsPracticeOnly(days, group)...)
	warnings = append(warnings, validateSingleCampusPerDay(days, group, locationMeta)...)
	return warnings
}

func validateMissingTeachers(days []DaySchedule, group Group) []ScheduleValidationWarning {
	var warnings []ScheduleValidationWarning
	for _, day := range days {
		for _, lesson := range day.Lessons {
			if strings.TrimSpace(lesson.TeacherName) != "" {
				continue
			}
			if lesson.SubjectID == nil && strings.TrimSpace(lesson.SubjectName) == "" {
				continue
			}
			warnings = append(warnings, warningFromLesson(
				"missing_teacher_placeholder",
				day.Date,
				lesson,
				group,
				fmt.Sprintf("lesson has no assigned teacher; pending load is %d hours", academicHoursPerPair),
			))
		}
	}
	return warnings
}

type physicalEducationRoomSlot struct {
	Date       string
	PairNumber int16
	LocationID int
}

func validatePhysicalEducationFacilityRules(targetGroupID int, groupsByID map[int]Group, allDaysByGroup map[int][]DaySchedule, locationMeta map[int]LocationMeta) []ScheduleValidationWarning {
	groupsByRoomSlot := map[physicalEducationRoomSlot]map[int]bool{}
	for groupID, days := range allDaysByGroup {
		for _, day := range days {
			for _, lesson := range day.Lessons {
				if lesson.LocationID == nil || !isPhysicalEducationSubject(lesson.SubjectName) {
					continue
				}
				meta := locationMeta[*lesson.LocationID]
				if !isPhysicalEducationFacilityKind(meta.LocationKind) {
					continue
				}
				slot := physicalEducationRoomSlot{
					Date:       day.Date,
					PairNumber: lesson.PairNumber,
					LocationID: *lesson.LocationID,
				}
				if _, ok := groupsByRoomSlot[slot]; !ok {
					groupsByRoomSlot[slot] = map[int]bool{}
				}
				groupsByRoomSlot[slot][groupID] = true
			}
		}
	}

	targetGroup := groupsByID[targetGroupID]
	var warnings []ScheduleValidationWarning
	roomLimitWarned := map[physicalEducationRoomSlot]bool{}
	for _, day := range allDaysByGroup[targetGroupID] {
		for _, lesson := range day.Lessons {
			if !isPhysicalEducationSubject(lesson.SubjectName) {
				continue
			}
			if lesson.LocationID == nil {
				warnings = append(warnings, warningFromLesson("physical_education_location_kind", day.Date, lesson, targetGroup, "physical education must be scheduled in gym or pool"))
				continue
			}

			meta := locationMeta[*lesson.LocationID]
			if !isPhysicalEducationFacilityKind(meta.LocationKind) {
				warnings = append(warnings, warningFromLesson("physical_education_location_kind", day.Date, lesson, targetGroup, "physical education must be scheduled in gym or pool"))
				continue
			}

			slot := physicalEducationRoomSlot{
				Date:       day.Date,
				PairNumber: lesson.PairNumber,
				LocationID: *lesson.LocationID,
			}
			if roomLimitWarned[slot] {
				continue
			}
			groupCount := len(groupsByRoomSlot[slot])
			if groupCount <= physicalEducationRoomGroupLimit {
				continue
			}
			roomLimitWarned[slot] = true
			warnings = append(warnings, warningFromLesson(
				"physical_education_room_group_limit",
				day.Date,
				lesson,
				targetGroup,
				fmt.Sprintf("physical education room has %d groups in one slot, limit is %d", groupCount, physicalEducationRoomGroupLimit),
			))
		}
	}
	return warnings
}

func validateWeeklyStudentHours(days []DaySchedule) []ScheduleValidationWarning {
	pairsByWeek := map[string]map[string]bool{}
	for _, day := range days {
		weekKey := weekKeyFromDateString(day.Date)
		if weekKey == "" {
			continue
		}
		m := ensureStringBoolMap(pairsByWeek, weekKey)
		for _, lesson := range day.Lessons {
			if lesson.PairNumber <= 0 {
				continue
			}
			m[fmt.Sprintf("%s:%d", day.Date, lesson.PairNumber)] = true
		}
	}

	var warnings []ScheduleValidationWarning
	for _, weekKey := range sortedMapKeys(pairsByWeek) {
		hours := len(pairsByWeek[weekKey]) * academicHoursPerPair
		if hours <= studentWeeklyHoursLimit {
			continue
		}
		warnings = append(warnings, ScheduleValidationWarning{
			Code:    "student_week_hours_limit",
			Date:    weekKey,
			Message: fmt.Sprintf("student weekly load is %d hours, limit is %d", hours, studentWeeklyHoursLimit),
		})
	}
	return warnings
}

func validateWeeklyTeacherHours(days []DaySchedule, group Group) []ScheduleValidationWarning {
	type weekTeacherKey struct {
		Week    string
		Teacher string
	}
	slotsByTeacherWeek := map[weekTeacherKey]map[string]bool{}
	for _, day := range days {
		weekKey := weekKeyFromDateString(day.Date)
		if weekKey == "" {
			continue
		}
		for _, lesson := range day.Lessons {
			teacher := strings.TrimSpace(lesson.TeacherName)
			if teacher == "" || lesson.PairNumber <= 0 {
				continue
			}
			k := weekTeacherKey{Week: weekKey, Teacher: teacher}
			m := ensureSlotMap(slotsByTeacherWeek, k)
			m[fmt.Sprintf("%s:%d:%d", day.Date, lesson.PairNumber, subgroupKey(lesson.Subgroup))] = true
		}
	}

	keys := make([]weekTeacherKey, 0, len(slotsByTeacherWeek))
	for k := range slotsByTeacherWeek {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Week != keys[j].Week {
			return keys[i].Week < keys[j].Week
		}
		return keys[i].Teacher < keys[j].Teacher
	})

	var warnings []ScheduleValidationWarning
	for _, k := range keys {
		hours := len(slotsByTeacherWeek[k]) * academicHoursPerPair
		if hours <= teacherWeeklyHoursLimit {
			continue
		}
		warnings = append(warnings, ScheduleValidationWarning{
			Code:     "teacher_week_hours_limit",
			Date:     k.Week,
			Semester: semesterForDateString(k.Week, group.Course),
			Message:  fmt.Sprintf("teacher %s weekly load is %d hours in validated schedule, limit is %d", k.Teacher, hours, teacherWeeklyHoursLimit),
		})
	}
	return warnings
}

func validateFloatingDayOff(days []DaySchedule, group Group) []ScheduleValidationWarning {
	type weekTeacherKey struct {
		Week    string
		Teacher string
	}
	coveredDays := map[string]map[string]bool{}
	groupLessonDays := map[string]map[string]bool{}
	teacherLessonDays := map[weekTeacherKey]map[string]bool{}

	for _, day := range days {
		weekKey := weekKeyFromDateString(day.Date)
		if weekKey == "" {
			continue
		}
		ensureStringBoolMap(coveredDays, weekKey)[day.Date] = true
		if len(day.Lessons) > 0 {
			ensureStringBoolMap(groupLessonDays, weekKey)[day.Date] = true
		}
		for _, lesson := range day.Lessons {
			teacher := strings.TrimSpace(lesson.TeacherName)
			if teacher == "" {
				continue
			}
			k := weekTeacherKey{Week: weekKey, Teacher: teacher}
			ensureSlotMap(teacherLessonDays, k)[day.Date] = true
		}
	}

	var warnings []ScheduleValidationWarning
	for _, weekKey := range sortedMapKeys(coveredDays) {
		if len(coveredDays[weekKey]) < 6 {
			continue
		}
		if len(groupLessonDays[weekKey]) >= 6 {
			warnings = append(warnings, ScheduleValidationWarning{
				Code:     "group_floating_day_off",
				Date:     weekKey,
				Semester: semesterForDateString(weekKey, group.Course),
				Message:  "group has lessons on all six study days in the week",
			})
		}
	}

	keys := make([]weekTeacherKey, 0, len(teacherLessonDays))
	for k := range teacherLessonDays {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Week != keys[j].Week {
			return keys[i].Week < keys[j].Week
		}
		return keys[i].Teacher < keys[j].Teacher
	})
	for _, k := range keys {
		if len(coveredDays[k.Week]) < 6 || len(teacherLessonDays[k]) < 6 {
			continue
		}
		warnings = append(warnings, ScheduleValidationWarning{
			Code:     "teacher_floating_day_off",
			Date:     k.Week,
			Semester: semesterForDateString(k.Week, group.Course),
			Message:  "teacher " + k.Teacher + " has lessons on all six study days in the week",
		})
	}
	return warnings
}

func validatePhysicalEducationFifthPair(days []DaySchedule, group Group) []ScheduleValidationWarning {
	var warnings []ScheduleValidationWarning
	for _, day := range days {
		for _, lesson := range day.Lessons {
			if lesson.PairNumber != 5 || !isPhysicalEducationSubject(lesson.SubjectName) {
				continue
			}
			warnings = append(warnings, warningFromLesson("physical_education_fifth_pair", day.Date, lesson, group, "physical education must not be scheduled as fifth pair"))
		}
	}
	return warnings
}

func validateThreePairsPracticeOnly(days []DaySchedule, group Group) []ScheduleValidationWarning {
	var warnings []ScheduleValidationWarning
	for _, day := range days {
		pairs := map[int16]bool{}
		for _, lesson := range day.Lessons {
			if lesson.PairNumber > 0 {
				pairs[lesson.PairNumber] = true
			}
		}
		if len(pairs) < 3 {
			continue
		}
		for _, lesson := range day.Lessons {
			if isPracticeLikeSubject(lesson.SubjectName) {
				continue
			}
			warnings = append(warnings, warningFromLesson("three_pairs_requires_practice", day.Date, lesson, group, "three or more pairs in a day are allowed only for practice-like activities"))
			break
		}
	}
	return warnings
}

func validateSingleCampusPerDay(days []DaySchedule, group Group, locationMeta map[int]LocationMeta) []ScheduleValidationWarning {
	var warnings []ScheduleValidationWarning
	for _, day := range days {
		campuses := map[string]bool{}
		var first Lesson
		hasFirst := false
		for _, lesson := range day.Lessons {
			if lesson.LocationID == nil {
				continue
			}
			meta := locationMeta[*lesson.LocationID]
			campus := strings.TrimSpace(meta.Campus)
			if campus == "" {
				continue
			}
			campuses[campus] = true
			if !hasFirst {
				first = lesson
				hasFirst = true
			}
		}
		if len(campuses) <= 1 {
			continue
		}
		campusNames := sortedStringKeys(campuses)
		w := warningFromLesson("multiple_campuses_day", day.Date, first, group, "group has lessons in multiple campuses in one day: "+strings.Join(campusNames, ", "))
		warnings = append(warnings, w)
	}
	return warnings
}

func warningFromLesson(code, date string, lesson Lesson, group Group, message string) ScheduleValidationWarning {
	subjectID := 0
	if lesson.SubjectID != nil {
		subjectID = *lesson.SubjectID
	}
	return ScheduleValidationWarning{
		Code:        code,
		Date:        date,
		PairNumber:  lesson.PairNumber,
		Subgroup:    lesson.Subgroup,
		SubjectID:   subjectID,
		SubjectName: lesson.SubjectName,
		Semester:    semesterForDateString(date, group.Course),
		Message:     message,
	}
}

func isPhysicalEducationSubject(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	return strings.Contains(normalized, "физическая культура") ||
		strings.Contains(normalized, "физкультура") ||
		strings.Contains(normalized, "физ-ра") ||
		strings.Contains(normalized, "физра") ||
		strings.Contains(normalized, "physical education") ||
		normalized == "pe"
}

func isPhysicalEducationFacilityKind(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "gym", "pool":
		return true
	default:
		return false
	}
}

func hasPhysicalEducationLessons(days []DaySchedule) bool {
	for _, day := range days {
		for _, lesson := range day.Lessons {
			if isPhysicalEducationSubject(lesson.SubjectName) {
				return true
			}
		}
	}
	return false
}

func collectLocationIDsFromDays(days []DaySchedule, out map[int]bool) {
	for _, day := range days {
		for _, lesson := range day.Lessons {
			if lesson.LocationID == nil || *lesson.LocationID <= 0 {
				continue
			}
			out[*lesson.LocationID] = true
		}
	}
}

func isPracticeLikeSubject(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	return strings.Contains(normalized, "практик") ||
		strings.Contains(normalized, "practice") ||
		strings.Contains(normalized, "стажиров")
}

func weekKeyFromDateString(date string) string {
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return ""
	}
	return mondayOfWeek(d).Format("2006-01-02")
}

func semesterForDateString(date string, course int) int16 {
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return 0
	}
	sem := inferSemesterForDate(d, course)
	if sem == nil {
		return 0
	}
	return *sem
}

func ensureStringBoolMap(m map[string]map[string]bool, key string) map[string]bool {
	if _, ok := m[key]; !ok {
		m[key] = map[string]bool{}
	}
	return m[key]
}

func ensureSlotMap[K comparable](m map[K]map[string]bool, key K) map[string]bool {
	if _, ok := m[key]; !ok {
		m[key] = map[string]bool{}
	}
	return m[key]
}

func sortedStringKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
