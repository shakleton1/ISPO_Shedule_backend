package schedule

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Specialties

func (r *Repository) ListSpecialties() ([]Specialty, error) {
	var rows []Specialty
	err := r.db.Order("id asc").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListSpecialtiesPaged(limit, offset *int) ([]Specialty, error) {
	var rows []Specialty
	q := r.db.Order("id asc")
	q = applyLimitOffset(q, limit, offset)
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateSpecialty(s *Specialty) error {
	return r.db.Create(s).Error
}

func (r *Repository) UpdateSpecialty(id int, patch *Specialty) (*Specialty, error) {
	var row Specialty
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.Code = patch.Code
	row.Name = patch.Name
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteSpecialty(id int) error {
	return r.db.Delete(&Specialty{}, id).Error
}

// Curricula

type CurriculumFilters struct {
	SpecialtyID   *int
	AdmissionYear *int16
	IsActive      *bool
}

func (r *Repository) ListCurricula(filters CurriculumFilters) ([]Curriculum, error) {
	q := r.db.Where("deleted_at IS NULL").Order("specialty_id asc, admission_year asc, variant asc, id asc")
	if filters.SpecialtyID != nil {
		q = q.Where("specialty_id = ?", *filters.SpecialtyID)
	}
	if filters.AdmissionYear != nil {
		q = q.Where("admission_year = ?", *filters.AdmissionYear)
	}
	if filters.IsActive != nil {
		q = q.Where("is_active = ?", *filters.IsActive)
	}
	var rows []Curriculum
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) ListCurriculaPaged(filters CurriculumFilters, limit, offset *int) ([]Curriculum, error) {
	q := r.db.Where("deleted_at IS NULL").Order("specialty_id asc, admission_year asc, variant asc, id asc")
	if filters.SpecialtyID != nil {
		q = q.Where("specialty_id = ?", *filters.SpecialtyID)
	}
	if filters.AdmissionYear != nil {
		q = q.Where("admission_year = ?", *filters.AdmissionYear)
	}
	if filters.IsActive != nil {
		q = q.Where("is_active = ?", *filters.IsActive)
	}
	q = applyLimitOffset(q, limit, offset)
	var rows []Curriculum
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateCurriculum(c *Curriculum) error {
	return r.db.Create(c).Error
}

func (r *Repository) UpdateCurriculum(id int64, patch *Curriculum) (*Curriculum, error) {
	var row Curriculum
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.SpecialtyID = patch.SpecialtyID
	row.AdmissionYear = patch.AdmissionYear
	row.Variant = patch.Variant
	row.Title = patch.Title
	row.Notes = patch.Notes
	row.IsActive = patch.IsActive
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	// If it was soft-deleted earlier, restore.
	if err := r.db.Exec("UPDATE curricula SET deleted_at = NULL WHERE id = ?", id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteCurriculum(id int64) error {
	res := r.db.Exec("UPDATE curricula SET deleted_at = now() WHERE id = ? AND deleted_at IS NULL", id)
	return res.Error
}

// Academic calendars

type AcademicCalendarWeekFilters struct {
	CourseNumber *int16
	WeekNumber   *int16
}

type AcademicCalendarDayOverrideFilters struct {
	CourseNumber *int16
	WeekNumber   *int16
	DayOfWeek    *int16
}

func (r *Repository) ListAcademicCalendars(curriculumID int64) ([]AcademicCalendar, error) {
	var rows []AcademicCalendar
	err := r.db.Where("curriculum_id = ?", curriculumID).Order("academic_year_start asc").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListAcademicCalendarsPaged(curriculumID int64, limit, offset *int) ([]AcademicCalendar, error) {
	var rows []AcademicCalendar
	q := r.db.Where("curriculum_id = ?", curriculumID).Order("academic_year_start asc")
	q = applyLimitOffset(q, limit, offset)
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateAcademicCalendar(ac *AcademicCalendar) error {
	ac.AcademicYearStart = dateOnly(ac.AcademicYearStart)
	return r.db.Create(ac).Error
}

func (r *Repository) DeleteAcademicCalendar(id int64) error {
	return r.db.Delete(&AcademicCalendar{}, id).Error
}

func (r *Repository) ListAcademicCalendarWeeks(calendarID int64) ([]AcademicCalendarWeek, error) {
	return r.ListAcademicCalendarWeeksFiltered(calendarID, AcademicCalendarWeekFilters{})
}

func (r *Repository) ListAcademicCalendarWeeksFiltered(calendarID int64, filters AcademicCalendarWeekFilters) ([]AcademicCalendarWeek, error) {
	var rows []AcademicCalendarWeek
	q := r.db.Where("calendar_id = ?", calendarID).Order("course_number asc, week_number asc")
	q = applyAcademicCalendarWeekFilters(q, filters)
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) ListAcademicCalendarWeeksPaged(calendarID int64, limit, offset *int) ([]AcademicCalendarWeek, error) {
	return r.ListAcademicCalendarWeeksPagedFiltered(calendarID, AcademicCalendarWeekFilters{}, limit, offset)
}

func (r *Repository) ListAcademicCalendarWeeksPagedFiltered(calendarID int64, filters AcademicCalendarWeekFilters, limit, offset *int) ([]AcademicCalendarWeek, error) {
	var rows []AcademicCalendarWeek
	q := r.db.Where("calendar_id = ?", calendarID).Order("course_number asc, week_number asc")
	q = applyAcademicCalendarWeekFilters(q, filters)
	q = applyLimitOffset(q, limit, offset)
	err := q.Find(&rows).Error
	return rows, err
}

func applyAcademicCalendarWeekFilters(q *gorm.DB, filters AcademicCalendarWeekFilters) *gorm.DB {
	if filters.CourseNumber != nil {
		q = q.Where("course_number = ?", *filters.CourseNumber)
	}
	if filters.WeekNumber != nil {
		q = q.Where("week_number = ?", *filters.WeekNumber)
	}
	return q
}

func (r *Repository) UpsertAcademicCalendarWeeks(calendarID int64, weeks []AcademicCalendarWeek) ([]AcademicCalendarWeek, error) {
	if len(weeks) == 0 {
		return []AcademicCalendarWeek{}, nil
	}
	for i := range weeks {
		weeks[i].CalendarID = calendarID
		if weeks[i].CourseNumber == 0 {
			weeks[i].CourseNumber = 1
		}
		if weeks[i].CourseNumber < 1 || weeks[i].CourseNumber > 6 {
			return nil, fmt.Errorf("course_number must be 1..6")
		}
		weeks[i].WeekStartDate = mondayOfWeek(weeks[i].WeekStartDate)
		weeks[i].ActivityCode = strings.ToUpper(strings.TrimSpace(weeks[i].ActivityCode))
		if weeks[i].ActivityCode == "" {
			return nil, fmt.Errorf("activity_code required")
		}
		if weeks[i].WeekNumber <= 0 {
			return nil, fmt.Errorf("week_number must be > 0")
		}
	}

	err := r.db.Transaction(func(tx *gorm.DB) error {
		for _, w := range weeks {
			// Upsert by (calendar_id, course_number, week_number).
			if err := tx.Exec(`
INSERT INTO academic_calendar_weeks
  (calendar_id, course_number, week_number, week_start_date, activity_code, activity_name, is_teaching, comment)
VALUES
  (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (calendar_id, course_number, week_number)
DO UPDATE SET
  week_start_date = EXCLUDED.week_start_date,
  activity_code = EXCLUDED.activity_code,
  activity_name = EXCLUDED.activity_name,
  is_teaching = EXCLUDED.is_teaching,
  comment = EXCLUDED.comment`,
				w.CalendarID, w.CourseNumber, w.WeekNumber, dateOnly(w.WeekStartDate), w.ActivityCode, w.ActivityName, w.IsTeaching, w.Comment,
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.ListAcademicCalendarWeeks(calendarID)
}

func (r *Repository) ListAcademicCalendarDayOverrides(calendarID int64, filters AcademicCalendarDayOverrideFilters) ([]AcademicCalendarDayOverride, error) {
	var rows []AcademicCalendarDayOverride
	q := r.db.Where("calendar_id = ?", calendarID).Order("course_number asc, week_number asc, day_of_week asc")
	q = applyAcademicCalendarDayOverrideFilters(q, filters)
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) ListAcademicCalendarDayOverridesPaged(calendarID int64, filters AcademicCalendarDayOverrideFilters, limit, offset *int) ([]AcademicCalendarDayOverride, error) {
	var rows []AcademicCalendarDayOverride
	q := r.db.Where("calendar_id = ?", calendarID).Order("course_number asc, week_number asc, day_of_week asc")
	q = applyAcademicCalendarDayOverrideFilters(q, filters)
	q = applyLimitOffset(q, limit, offset)
	err := q.Find(&rows).Error
	return rows, err
}

func applyAcademicCalendarDayOverrideFilters(q *gorm.DB, filters AcademicCalendarDayOverrideFilters) *gorm.DB {
	if filters.CourseNumber != nil {
		q = q.Where("course_number = ?", *filters.CourseNumber)
	}
	if filters.WeekNumber != nil {
		q = q.Where("week_number = ?", *filters.WeekNumber)
	}
	if filters.DayOfWeek != nil {
		q = q.Where("day_of_week = ?", *filters.DayOfWeek)
	}
	return q
}

func (r *Repository) CreateAcademicCalendarDayOverride(row *AcademicCalendarDayOverride) error {
	if err := normalizeAcademicCalendarDayOverride(row); err != nil {
		return err
	}
	return r.db.Create(row).Error
}

func (r *Repository) UpdateAcademicCalendarDayOverride(id int64, patch *AcademicCalendarDayOverride) (*AcademicCalendarDayOverride, error) {
	if err := normalizeAcademicCalendarDayOverride(patch); err != nil {
		return nil, err
	}
	var row AcademicCalendarDayOverride
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.CalendarID = patch.CalendarID
	row.CourseNumber = patch.CourseNumber
	row.WeekNumber = patch.WeekNumber
	row.DayOfWeek = patch.DayOfWeek
	row.ActivityCode = patch.ActivityCode
	row.ActivityName = patch.ActivityName
	row.IsTeaching = patch.IsTeaching
	row.Comment = patch.Comment
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteAcademicCalendarDayOverride(id int64) error {
	return r.db.Delete(&AcademicCalendarDayOverride{}, id).Error
}

func normalizeAcademicCalendarDayOverride(row *AcademicCalendarDayOverride) error {
	if row == nil {
		return fmt.Errorf("academic calendar day override is nil")
	}
	if row.CalendarID <= 0 {
		return fmt.Errorf("calendar_id required")
	}
	if row.CourseNumber < 1 || row.CourseNumber > 6 {
		return fmt.Errorf("course_number must be 1..6")
	}
	if row.WeekNumber < 1 || row.WeekNumber > 60 {
		return fmt.Errorf("week_number must be 1..60")
	}
	if row.DayOfWeek < 1 || row.DayOfWeek > 7 {
		return fmt.Errorf("day_of_week must be 1..7")
	}
	row.ActivityCode = strings.ToUpper(strings.TrimSpace(row.ActivityCode))
	if row.ActivityCode == "" {
		return fmt.Errorf("activity_code required")
	}
	if row.ActivityName != nil {
		name := strings.TrimSpace(*row.ActivityName)
		if name == "" {
			row.ActivityName = nil
		} else {
			row.ActivityName = &name
		}
	}
	return nil
}

// Curriculum items

func (r *Repository) ListCurriculumItems(curriculumID int64) ([]CurriculumItem, error) {
	var rows []CurriculumItem
	err := r.db.Where("curriculum_id = ?", curriculumID).Order("id asc").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListCurriculumItemsPaged(curriculumID int64, limit, offset *int) ([]CurriculumItem, error) {
	var rows []CurriculumItem
	q := r.db.Where("curriculum_id = ?", curriculumID).Order("id asc")
	q = applyLimitOffset(q, limit, offset)
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateCurriculumItem(it *CurriculumItem) error {
	return r.db.Create(it).Error
}

func (r *Repository) UpdateCurriculumItem(id int64, patch *CurriculumItem) (*CurriculumItem, error) {
	var row CurriculumItem
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.ParentID = patch.ParentID
	row.IndexCode = patch.IndexCode
	row.ItemType = patch.ItemType
	row.Name = patch.Name
	row.SubjectID = patch.SubjectID
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteCurriculumItem(id int64) error {
	return r.db.Delete(&CurriculumItem{}, id).Error
}

func (r *Repository) ListCurriculumItemAllocations(itemID int64) ([]CurriculumItemAllocation, error) {
	var rows []CurriculumItemAllocation
	err := r.db.Where("item_id = ?", itemID).Order("semester asc").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListCurriculumItemAllocationsPaged(itemID int64, limit, offset *int) ([]CurriculumItemAllocation, error) {
	var rows []CurriculumItemAllocation
	q := r.db.Where("item_id = ?", itemID).Order("semester asc")
	q = applyLimitOffset(q, limit, offset)
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) UpsertCurriculumItemAllocations(itemID int64, allocs []CurriculumItemAllocation) ([]CurriculumItemAllocation, error) {
	if len(allocs) == 0 {
		return []CurriculumItemAllocation{}, nil
	}
	for i := range allocs {
		allocs[i].ItemID = itemID
		if allocs[i].Semester <= 0 {
			return nil, fmt.Errorf("semester must be > 0")
		}
	}

	err := r.db.Transaction(func(tx *gorm.DB) error {
		for _, a := range allocs {
			if err := tx.Exec(`
INSERT INTO curriculum_item_allocations
	(item_id, semester, weeks, hours_total, hours_lectures, hours_practice, hours_lab, hours_independent, hours_exam, assessment_type, comment)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (item_id, semester)
DO UPDATE SET
  weeks = EXCLUDED.weeks,
	hours_total = EXCLUDED.hours_total,
	hours_lectures = EXCLUDED.hours_lectures,
	hours_practice = EXCLUDED.hours_practice,
	hours_lab = EXCLUDED.hours_lab,
	hours_independent = EXCLUDED.hours_independent,
	hours_exam = EXCLUDED.hours_exam,
	assessment_type = EXCLUDED.assessment_type,
  comment = EXCLUDED.comment`,
				a.ItemID, a.Semester, a.Weeks, a.HoursTotal, a.HoursLectures, a.HoursPractice, a.HoursLab, a.HoursIndependent, a.HoursExam, a.AssessmentType, a.Comment,
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.ListCurriculumItemAllocations(itemID)
}

type allocatedSubjectRow struct {
	Semester  int16 `gorm:"column:semester"`
	SubjectID int   `gorm:"column:subject_id"`
}

func (r *Repository) ListAllocatedSubjectsBySemester(curriculumID int64, semesters []int16) (map[int16]map[int]bool, error) {
	if curriculumID <= 0 {
		return nil, fmt.Errorf("curriculum_id required")
	}
	if len(semesters) == 0 {
		return map[int16]map[int]bool{}, nil
	}

	var rows []allocatedSubjectRow
	err := r.db.Raw(`
SELECT DISTINCT a.semester, ci.subject_id
FROM curriculum_items ci
JOIN curriculum_item_allocations a ON a.item_id = ci.id
WHERE ci.curriculum_id = ?
  AND ci.subject_id IS NOT NULL
  AND a.semester IN ?
ORDER BY a.semester asc, ci.subject_id asc
`, curriculumID, semesters).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := map[int16]map[int]bool{}
	for _, r := range rows {
		m, ok := out[r.Semester]
		if !ok {
			m = map[int]bool{}
			out[r.Semester] = m
		}
		m[r.SubjectID] = true
	}
	return out, nil
}

// Calendar helpers for schedule building

type teachingWeekRow struct {
	WeekStartDate time.Time `gorm:"column:week_start_date"`
	IsTeaching    bool      `gorm:"column:is_teaching"`
}

type academicDayOverrideStateRow struct {
	TargetDate   time.Time `gorm:"column:target_date"`
	ActivityCode string    `gorm:"column:activity_code"`
	ActivityName *string   `gorm:"column:activity_name"`
	IsTeaching   bool      `gorm:"column:is_teaching"`
}

type studyWeekStateRow struct {
	WeekNumber    int16      `gorm:"column:week_number"`
	WeekStartDate *time.Time `gorm:"column:week_start_date"`
	ActivityCode  *string    `gorm:"column:activity_code"`
	ActivityName  *string    `gorm:"column:activity_name"`
	AllowsLessons bool       `gorm:"column:allows_lessons"`
}

func defaultStudyDayState() StudyDayState {
	name := "Учебные занятия"
	return StudyDayState{
		ActivityCode: "TEACHING",
		ActivityName: &name,
		IsTeaching:   true,
		Source:       "default",
	}
}

func (r *Repository) ListStudyDayStatesForGroupBetween(groupID int, startDate, endDate time.Time) (map[string]StudyDayState, error) {
	startDate = dateOnly(startDate)
	endDate = dateOnly(endDate)
	startWeek := mondayOfWeek(startDate)
	endWeek := mondayOfWeek(endDate)

	var group Group
	if err := r.db.First(&group, groupID).Error; err != nil {
		return nil, err
	}

	out := map[string]StudyDayState{}
	academicWeekStartByNumber := map[int16]time.Time{}

	if group.CurriculumID != nil {
		var calendar AcademicCalendar
		q := r.db.Where("curriculum_id = ?", *group.CurriculumID)
		if group.AdmissionYear != nil {
			yearStart := time.Date(int(*group.AdmissionYear), time.January, 1, 0, 0, 0, 0, time.UTC)
			yearEnd := yearStart.AddDate(1, 0, 0)
			q = q.Where("academic_year_start >= ? AND academic_year_start < ?", yearStart, yearEnd)
		} else {
			q = q.Where("academic_year_start <= ?", endDate)
		}
		err := q.Order("academic_year_start desc, id desc").First(&calendar).Error
		if err == gorm.ErrRecordNotFound && group.AdmissionYear != nil {
			err = r.db.
				Where("curriculum_id = ? AND academic_year_start <= ?", *group.CurriculumID, endDate).
				Order("academic_year_start desc, id desc").
				First(&calendar).Error
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, err
		}
		if err == nil {
			var weeks []AcademicCalendarWeek
			if err := r.db.
				Where("calendar_id = ? AND week_start_date BETWEEN ? AND ?", calendar.ID, startWeek, endWeek).
				Order("course_number asc, week_number asc").
				Find(&weeks).Error; err != nil {
				return nil, err
			}
			for _, w := range weeks {
				weekKey := dateOnly(w.WeekStartDate).Format("2006-01-02")
				if _, exists := academicWeekStartByNumber[w.WeekNumber]; !exists {
					academicWeekStartByNumber[w.WeekNumber] = dateOnly(w.WeekStartDate)
				}
				name := w.ActivityName
				out[weekKey] = StudyDayState{
					ActivityCode: w.ActivityCode,
					ActivityName: name,
					IsTeaching:   w.IsTeaching,
					Source:       "academic_week",
				}
			}

			var dayRows []academicDayOverrideStateRow
			if err := r.db.Raw(`
SELECT (w.week_start_date + ((o.day_of_week - 1)::int || ' days')::interval)::date AS target_date,
       o.activity_code,
       o.activity_name,
       o.is_teaching
FROM academic_calendar_day_overrides o
JOIN academic_calendar_weeks w
  ON w.calendar_id = o.calendar_id
 AND w.course_number = o.course_number
 AND w.week_number = o.week_number
WHERE o.calendar_id = ?
  AND (w.week_start_date + ((o.day_of_week - 1)::int || ' days')::interval)::date BETWEEN ? AND ?
`, calendar.ID, startDate, endDate).Scan(&dayRows).Error; err != nil {
				return nil, err
			}
			for _, row := range dayRows {
				dayKey := dateOnly(row.TargetDate).Format("2006-01-02")
				out[dayKey] = StudyDayState{
					ActivityCode: row.ActivityCode,
					ActivityName: row.ActivityName,
					IsTeaching:   row.IsTeaching,
					Source:       "academic_day_override",
				}
			}
		}
	}

	var studyRows []studyWeekStateRow
	err := r.db.Raw(`
SELECT scw.week_number,
       scw.week_start_date,
       sa.code AS activity_code,
       sa.name AS activity_name,
       scw.allows_lessons
FROM study_calendar_weeks scw
LEFT JOIN study_activities sa ON sa.id = scw.activity_id
WHERE scw.group_id = ?
  AND (scw.week_start_date IS NULL OR scw.week_start_date BETWEEN ? AND ?)
`, groupID, startWeek, endWeek).Scan(&studyRows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range studyRows {
		var weekStart time.Time
		if row.WeekStartDate != nil {
			weekStart = mondayOfWeek(*row.WeekStartDate)
		} else if ws, ok := academicWeekStartByNumber[row.WeekNumber]; ok {
			weekStart = ws
		} else {
			continue
		}
		if weekStart.Before(startWeek) || weekStart.After(endWeek) {
			continue
		}
		code := "TEACHING"
		if row.ActivityCode != nil && strings.TrimSpace(*row.ActivityCode) != "" {
			code = strings.ToUpper(strings.TrimSpace(*row.ActivityCode))
		}
		name := row.ActivityName
		out[weekStart.Format("2006-01-02")] = StudyDayState{
			ActivityCode: code,
			ActivityName: name,
			IsTeaching:   row.AllowsLessons,
			Source:       "study_week",
		}
	}

	return out, nil
}

func (r *Repository) ListTeachingWeeksForGroupBetween(groupID int, startDate, endDate time.Time) (map[string]bool, error) {
	states, err := r.ListStudyDayStatesForGroupBetween(groupID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for day, state := range states {
		if state.Source == "academic_day_override" {
			continue
		}
		out[day] = state.IsTeaching
	}
	return out, nil
}

func (r *Repository) listTeachingWeeksForGroupBetweenLegacy(groupID int, startDate, endDate time.Time) (map[string]bool, error) {
	start := mondayOfWeek(startDate)
	end := mondayOfWeek(endDate)

	var group Group
	if err := r.db.First(&group, groupID).Error; err != nil {
		return nil, err
	}
	out := map[string]bool{}

	var groupRows []teachingWeekRow
	err := r.db.Raw(`
SELECT COALESCE(scw.week_start_date, acw.week_start_date) AS week_start_date, scw.allows_lessons AS is_teaching
FROM study_calendar_weeks scw
JOIN groups g ON g.id = scw.group_id
LEFT JOIN academic_calendars ac ON ac.curriculum_id = g.curriculum_id
LEFT JOIN academic_calendar_weeks acw ON acw.calendar_id = ac.id AND acw.week_number = scw.week_number
WHERE scw.group_id = ?
  AND COALESCE(scw.week_start_date, acw.week_start_date) BETWEEN ? AND ?
`, groupID, dateOnly(start), dateOnly(end)).Scan(&groupRows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range groupRows {
		out[dateOnly(r.WeekStartDate).Format("2006-01-02")] = r.IsTeaching
	}

	if group.CurriculumID == nil {
		return out, nil
	}

	var rows []teachingWeekRow
	err = r.db.Raw(`
SELECT w.week_start_date, w.is_teaching
FROM groups g
JOIN curricula c ON c.id = g.curriculum_id
JOIN academic_calendars ac ON ac.curriculum_id = c.id
JOIN academic_calendar_weeks w ON w.calendar_id = ac.id
WHERE g.id = ?
  AND w.week_start_date BETWEEN ? AND ?
`, groupID, dateOnly(start), dateOnly(end)).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, r := range rows {
		k := dateOnly(r.WeekStartDate).Format("2006-01-02")
		if _, exists := out[k]; exists {
			continue
		}
		out[k] = r.IsTeaching
	}
	return out, nil
}
