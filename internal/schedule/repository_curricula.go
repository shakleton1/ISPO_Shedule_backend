package schedule

import (
	"fmt"
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
	var rows []AcademicCalendarWeek
	err := r.db.Where("calendar_id = ?", calendarID).Order("week_number asc").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListAcademicCalendarWeeksPaged(calendarID int64, limit, offset *int) ([]AcademicCalendarWeek, error) {
	var rows []AcademicCalendarWeek
	q := r.db.Where("calendar_id = ?", calendarID).Order("week_number asc")
	q = applyLimitOffset(q, limit, offset)
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) UpsertAcademicCalendarWeeks(calendarID int64, weeks []AcademicCalendarWeek) ([]AcademicCalendarWeek, error) {
	if len(weeks) == 0 {
		return []AcademicCalendarWeek{}, nil
	}
	for i := range weeks {
		weeks[i].CalendarID = calendarID
		weeks[i].WeekStartDate = mondayOfWeek(weeks[i].WeekStartDate)
		if weeks[i].ActivityCode == "" {
			return nil, fmt.Errorf("activity_code required")
		}
		if weeks[i].WeekNumber <= 0 {
			return nil, fmt.Errorf("week_number must be > 0")
		}
	}

	err := r.db.Transaction(func(tx *gorm.DB) error {
		for _, w := range weeks {
			// Upsert by (calendar_id, week_number)
			// Note: also unique by (calendar_id, week_start_date), so week_start_date changes can fail if colliding.
			if err := tx.Exec(`
INSERT INTO academic_calendar_weeks
  (calendar_id, week_number, week_start_date, activity_code, activity_name, is_teaching, comment)
VALUES
  (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (calendar_id, week_number)
DO UPDATE SET
  week_start_date = EXCLUDED.week_start_date,
  activity_code = EXCLUDED.activity_code,
  activity_name = EXCLUDED.activity_name,
  is_teaching = EXCLUDED.is_teaching,
  comment = EXCLUDED.comment`,
				w.CalendarID, w.WeekNumber, dateOnly(w.WeekStartDate), w.ActivityCode, w.ActivityName, w.IsTeaching, w.Comment,
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

func (r *Repository) ListTeachingWeeksForGroupBetween(groupID int, startDate, endDate time.Time) (map[string]bool, error) {
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
