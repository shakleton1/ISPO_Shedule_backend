package schedule

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type StudyCalendarWeekFilters struct {
	GroupID    *int
	WeekNumber *int16
}

type TeacherDayConstraintFilters struct {
	TeacherID  *int
	TargetDate *time.Time
}

type ScheduleReplacementFilters struct {
	GroupID    *int
	TargetDate *time.Time
}

type LocationWeekAvailabilityFilters struct {
	WeekStartDate *time.Time
	LocationID    *int
}

type TeacherDayConstraintView struct {
	ID            int64     `gorm:"column:id"`
	TeacherID     int       `gorm:"column:teacher_id"`
	TeacherName   string    `gorm:"column:teacher_name"`
	TargetDate    time.Time `gorm:"column:target_date"`
	Reason        string    `gorm:"column:reason"`
	AllowsLessons bool      `gorm:"column:allows_lessons"`
}

type LocationMeta struct {
	ID         int    `gorm:"column:id"`
	CampusID   *int   `gorm:"column:campus_id"`
	CampusName string `gorm:"column:campus_name"`
	Kind       string `gorm:"column:kind"`
	TypeCodes  string `gorm:"column:type_codes"`
}

func (m LocationMeta) HasType(code string) bool {
	code = strings.ToLower(strings.TrimSpace(code))
	if code == "" {
		return false
	}
	for _, part := range strings.Split(m.TypeCodes, ",") {
		if strings.ToLower(strings.TrimSpace(part)) == code {
			return true
		}
	}
	return false
}

func (m LocationMeta) IsVirtual() bool {
	return strings.EqualFold(strings.TrimSpace(m.Kind), "virtual") || m.HasType("online")
}

func (m LocationMeta) IsPhysicalEducationFacility() bool {
	return m.HasType("gym") || m.HasType("pool")
}

func (r *Repository) ListLocationMetaByIDs(ids []int) (map[int]LocationMeta, error) {
	if len(ids) == 0 {
		return map[int]LocationMeta{}, nil
	}
	var rows []LocationMeta
	err := r.db.Table("locations l").
		Select(`
l.id,
l.campus_id,
COALESCE(c.name, '') AS campus_name,
l.kind,
COALESCE(string_agg(DISTINCT lt.code, ',' ORDER BY lt.code), '') AS type_codes`).
		Joins("LEFT JOIN campuses c ON c.id = l.campus_id").
		Joins("LEFT JOIN location_type_links ltl ON ltl.location_id = l.id").
		Joins("LEFT JOIN location_types lt ON lt.id = ltl.type_id").
		Where("l.id IN ?", ids).
		Group("l.id, c.name").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[int]LocationMeta, len(rows))
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

func (r *Repository) ListLocationWeekAvailability(filters LocationWeekAvailabilityFilters) ([]LocationWeekAvailability, error) {
	q := r.db.Model(&LocationWeekAvailability{}).Order("week_start_date desc, location_id asc")
	if filters.WeekStartDate != nil {
		q = q.Where("week_start_date = ?", mondayOfWeek(*filters.WeekStartDate))
	}
	if filters.LocationID != nil {
		q = q.Where("location_id = ?", *filters.LocationID)
	}
	var rows []LocationWeekAvailability
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) ListLocationWeekAvailabilityPaged(filters LocationWeekAvailabilityFilters, limit, offset *int) ([]LocationWeekAvailability, error) {
	q := r.db.Model(&LocationWeekAvailability{}).Order("week_start_date desc, location_id asc")
	if filters.WeekStartDate != nil {
		q = q.Where("week_start_date = ?", mondayOfWeek(*filters.WeekStartDate))
	}
	if filters.LocationID != nil {
		q = q.Where("location_id = ?", *filters.LocationID)
	}
	q = applyLimitOffset(q, limit, offset)
	var rows []LocationWeekAvailability
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) UpsertLocationWeekAvailability(weekStart time.Time, rows []LocationWeekAvailability) ([]LocationWeekAvailability, error) {
	weekStart = mondayOfWeek(weekStart)
	if weekStart.IsZero() {
		return nil, fmt.Errorf("week_start_date required")
	}
	if len(rows) == 0 {
		return r.ListLocationWeekAvailability(LocationWeekAvailabilityFilters{WeekStartDate: &weekStart})
	}

	err := r.db.Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			if row.LocationID <= 0 {
				return fmt.Errorf("location_id required")
			}
			if err := tx.Exec(`
INSERT INTO location_week_availability
  (week_start_date, location_id, is_available, comment)
VALUES
  (?, ?, ?, ?)
ON CONFLICT (week_start_date, location_id)
DO UPDATE SET
  is_available = EXCLUDED.is_available,
  comment = EXCLUDED.comment`,
				weekStart, row.LocationID, row.IsAvailable, row.Comment,
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.ListLocationWeekAvailability(LocationWeekAvailabilityFilters{WeekStartDate: &weekStart})
}

func (r *Repository) DeleteLocationWeekAvailability(id int64) error {
	return r.db.Delete(&LocationWeekAvailability{}, id).Error
}

func (r *Repository) UpsertLocationOverrideForSlot(groupID int, date time.Time, pairNumber int16, subgroup *int16, locationID int, comment *string) (string, error) {
	if groupID <= 0 {
		return "", fmt.Errorf("group_id required")
	}
	if pairNumber < 1 || pairNumber > 8 {
		return "", fmt.Errorf("pair_number must be 1..8")
	}
	if locationID <= 0 {
		return "", fmt.Errorf("location_id required")
	}

	date = dateOnly(date)
	var row ScheduleOverride
	q := r.db.Where("group_id = ? AND target_date = ? AND pair_number = ?", groupID, date, pairNumber)
	if subgroup == nil {
		q = q.Where("subgroup IS NULL")
	} else {
		q = q.Where("subgroup = ?", *subgroup)
	}

	err := q.First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			o := &ScheduleOverride{
				TargetDate:    date,
				GroupID:       groupID,
				PairNumber:    pairNumber,
				ActionType:    OverrideReplace,
				NewLocationID: &locationID,
				Comment:       comment,
				Subgroup:      subgroup,
			}
			if err := r.CreateOverride(o); err != nil {
				return "", err
			}
			return "created", nil
		}
		return "", err
	}

	if row.ActionType == OverrideCancel {
		return "blocked_cancel", nil
	}

	updates := map[string]any{"new_location_id": locationID}
	if comment != nil {
		updates["comment"] = comment
	}
	if err := r.db.Model(&ScheduleOverride{}).Where("id = ?", row.ID).Updates(updates).Error; err != nil {
		return "", err
	}
	return "updated", nil
}

func (r *Repository) ListAvailableLocationsForWeek(weekStart time.Time, campusName, locationTypeCode *string) ([]Location, error) {
	weekStart = mondayOfWeek(weekStart)
	q := r.db.Table("location_week_availability lwa").
		Select("l.id, l.campus_id, l.name, l.kind, l.capacity, l.is_active, l.created_at, l.updated_at").
		Joins("JOIN locations l ON l.id = lwa.location_id").
		Joins("LEFT JOIN campuses c ON c.id = l.campus_id").
		Where("lwa.week_start_date = ? AND lwa.is_available = TRUE AND l.is_active = TRUE", weekStart)

	if campusName != nil && strings.TrimSpace(*campusName) != "" {
		q = q.Where("c.name = ?", strings.TrimSpace(*campusName))
	}
	if locationTypeCode != nil && strings.TrimSpace(*locationTypeCode) != "" {
		q = q.Where(`EXISTS (
			SELECT 1
			FROM location_type_links ltl
			JOIN location_types lt ON lt.id = ltl.type_id
			WHERE ltl.location_id = l.id AND lt.code = ?
		)`, strings.TrimSpace(*locationTypeCode))
	} else {
		q = q.Where("l.kind = ?", "physical")
	}

	var rows []Location
	err := q.Order("l.name asc, l.id asc").Scan(&rows).Error
	return rows, err
}

func (r *Repository) ListStudyActivities() ([]StudyActivity, error) {
	var rows []StudyActivity
	err := r.db.Order("code asc, id asc").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListStudyActivitiesPaged(limit, offset *int) ([]StudyActivity, error) {
	var rows []StudyActivity
	q := applyLimitOffset(r.db.Order("code asc, id asc"), limit, offset)
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateStudyActivity(a *StudyActivity) error {
	if err := normalizeStudyActivity(a); err != nil {
		return err
	}
	return r.db.Create(a).Error
}

func (r *Repository) UpdateStudyActivity(id int, patch *StudyActivity) (*StudyActivity, error) {
	if err := normalizeStudyActivity(patch); err != nil {
		return nil, err
	}
	var row StudyActivity
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.Code = patch.Code
	row.Name = patch.Name
	row.ActivityKind = patch.ActivityKind
	row.AllowsLessons = patch.AllowsLessons
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteStudyActivity(id int) error {
	return r.db.Delete(&StudyActivity{}, id).Error
}

func normalizeStudyActivity(a *StudyActivity) error {
	if a == nil {
		return fmt.Errorf("study activity is nil")
	}
	a.Code = strings.TrimSpace(a.Code)
	a.Name = strings.TrimSpace(a.Name)
	a.ActivityKind = strings.ToUpper(strings.TrimSpace(a.ActivityKind))
	if a.Code == "" || a.Name == "" {
		return fmt.Errorf("code and name required")
	}
	if a.ActivityKind == "" {
		a.ActivityKind = "OTHER"
	}
	return nil
}

func (r *Repository) GetOrCreateStudyActivity(code, name, kind string, allowsLessons bool) (int, error) {
	code = strings.TrimSpace(code)
	name = strings.TrimSpace(name)
	kind = strings.ToUpper(strings.TrimSpace(kind))
	if code == "" {
		code = name
	}
	if name == "" {
		name = code
	}
	if code == "" {
		return 0, fmt.Errorf("activity code or name required")
	}
	if kind == "" {
		kind = "OTHER"
	}
	var out struct {
		ID int `gorm:"column:id"`
	}
	err := r.db.Raw(`
INSERT INTO study_activities (code, name, activity_kind, allows_lessons)
VALUES (?, ?, ?, ?)
ON CONFLICT (code)
DO UPDATE SET
  name = EXCLUDED.name,
  activity_kind = EXCLUDED.activity_kind,
  allows_lessons = EXCLUDED.allows_lessons
RETURNING id`, code, name, kind, allowsLessons).Scan(&out).Error
	if err != nil {
		return 0, err
	}
	return out.ID, nil
}

func (r *Repository) ListStudyCalendarWeeks(filters StudyCalendarWeekFilters) ([]StudyCalendarWeek, error) {
	q := r.db.Model(&StudyCalendarWeek{}).Order("group_id asc, week_number asc")
	if filters.GroupID != nil {
		q = q.Where("group_id = ?", *filters.GroupID)
	}
	if filters.WeekNumber != nil {
		q = q.Where("week_number = ?", *filters.WeekNumber)
	}
	var rows []StudyCalendarWeek
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) ListStudyCalendarWeeksPaged(filters StudyCalendarWeekFilters, limit, offset *int) ([]StudyCalendarWeek, error) {
	q := r.db.Model(&StudyCalendarWeek{}).Order("group_id asc, week_number asc")
	if filters.GroupID != nil {
		q = q.Where("group_id = ?", *filters.GroupID)
	}
	if filters.WeekNumber != nil {
		q = q.Where("week_number = ?", *filters.WeekNumber)
	}
	q = applyLimitOffset(q, limit, offset)
	var rows []StudyCalendarWeek
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) UpsertStudyCalendarWeeks(groupID int, weeks []StudyCalendarWeek) ([]StudyCalendarWeek, error) {
	if groupID <= 0 {
		return nil, fmt.Errorf("group_id required")
	}
	if len(weeks) == 0 {
		return []StudyCalendarWeek{}, nil
	}
	for i := range weeks {
		weeks[i].GroupID = groupID
		if weeks[i].WeekNumber <= 0 {
			return nil, fmt.Errorf("week_number must be > 0")
		}
		if weeks[i].WeekStartDate != nil {
			d := mondayOfWeek(*weeks[i].WeekStartDate)
			weeks[i].WeekStartDate = &d
		}
	}

	err := r.db.Transaction(func(tx *gorm.DB) error {
		for _, w := range weeks {
			if err := tx.Exec(`
INSERT INTO study_calendar_weeks
  (group_id, week_number, week_start_date, activity_id, allows_lessons, comment)
VALUES
  (?, ?, ?, ?, ?, ?)
ON CONFLICT (group_id, week_number)
DO UPDATE SET
  week_start_date = EXCLUDED.week_start_date,
  activity_id = EXCLUDED.activity_id,
  allows_lessons = EXCLUDED.allows_lessons,
  comment = EXCLUDED.comment`,
				w.GroupID, w.WeekNumber, w.WeekStartDate, w.ActivityID, w.AllowsLessons, w.Comment,
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.ListStudyCalendarWeeks(StudyCalendarWeekFilters{GroupID: &groupID})
}

func (r *Repository) DeleteStudyCalendarWeek(id int64) error {
	return r.db.Delete(&StudyCalendarWeek{}, id).Error
}

func (r *Repository) ListTeacherDayConstraints(filters TeacherDayConstraintFilters) ([]TeacherDayConstraint, error) {
	q := r.db.Model(&TeacherDayConstraint{}).Order("target_date asc, teacher_id asc")
	if filters.TeacherID != nil {
		q = q.Where("teacher_id = ?", *filters.TeacherID)
	}
	if filters.TargetDate != nil {
		q = q.Where("target_date = ?", dateOnly(*filters.TargetDate))
	}
	var rows []TeacherDayConstraint
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) ListTeacherDayConstraintsPaged(filters TeacherDayConstraintFilters, limit, offset *int) ([]TeacherDayConstraint, error) {
	q := r.db.Model(&TeacherDayConstraint{}).Order("target_date asc, teacher_id asc")
	if filters.TeacherID != nil {
		q = q.Where("teacher_id = ?", *filters.TeacherID)
	}
	if filters.TargetDate != nil {
		q = q.Where("target_date = ?", dateOnly(*filters.TargetDate))
	}
	q = applyLimitOffset(q, limit, offset)
	var rows []TeacherDayConstraint
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateTeacherDayConstraint(d *TeacherDayConstraint) error {
	if d == nil {
		return fmt.Errorf("teacher day constraint is nil")
	}
	if d.TeacherID <= 0 {
		return fmt.Errorf("teacher_id required")
	}
	d.TargetDate = dateOnly(d.TargetDate)
	return r.db.Create(d).Error
}

func (r *Repository) UpdateTeacherDayConstraint(id int64, patch *TeacherDayConstraint) (*TeacherDayConstraint, error) {
	if patch == nil {
		return nil, fmt.Errorf("teacher day constraint patch is nil")
	}
	var row TeacherDayConstraint
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.TeacherID = patch.TeacherID
	row.TargetDate = dateOnly(patch.TargetDate)
	row.Reason = patch.Reason
	row.AllowsLessons = patch.AllowsLessons
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteTeacherDayConstraint(id int64) error {
	return r.db.Delete(&TeacherDayConstraint{}, id).Error
}

func (r *Repository) ListBlockingTeacherConstraintsBetween(startDate, endDate time.Time) ([]TeacherDayConstraintView, error) {
	var rows []TeacherDayConstraintView
	err := r.db.Table("teacher_day_constraints tdc").
		Select("tdc.id, tdc.teacher_id, t.name AS teacher_name, tdc.target_date, tdc.reason, tdc.allows_lessons").
		Joins("JOIN teachers t ON t.id = tdc.teacher_id").
		Where("tdc.target_date BETWEEN ? AND ? AND tdc.allows_lessons = FALSE", dateOnly(startDate), dateOnly(endDate)).
		Order("tdc.target_date asc, t.name asc").
		Scan(&rows).Error
	return rows, err
}

func (r *Repository) ListScheduleReplacements(filters ScheduleReplacementFilters) ([]ScheduleReplacement, error) {
	q := r.db.Model(&ScheduleReplacement{}).Order("target_date asc, group_id asc, pair_number asc, COALESCE(subgroup, 0) asc")
	if filters.GroupID != nil {
		q = q.Where("group_id = ?", *filters.GroupID)
	}
	if filters.TargetDate != nil {
		q = q.Where("target_date = ?", dateOnly(*filters.TargetDate))
	}
	var rows []ScheduleReplacement
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) ListScheduleReplacementsPaged(filters ScheduleReplacementFilters, limit, offset *int) ([]ScheduleReplacement, error) {
	q := r.db.Model(&ScheduleReplacement{}).Order("target_date asc, group_id asc, pair_number asc, COALESCE(subgroup, 0) asc")
	if filters.GroupID != nil {
		q = q.Where("group_id = ?", *filters.GroupID)
	}
	if filters.TargetDate != nil {
		q = q.Where("target_date = ?", dateOnly(*filters.TargetDate))
	}
	q = applyLimitOffset(q, limit, offset)
	var rows []ScheduleReplacement
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateScheduleReplacement(rep *ScheduleReplacement) error {
	if err := validateScheduleReplacement(rep); err != nil {
		return err
	}
	rep.TargetDate = dateOnly(rep.TargetDate)
	return r.db.Create(rep).Error
}

func (r *Repository) UpdateScheduleReplacement(id int64, patch *ScheduleReplacement) (*ScheduleReplacement, error) {
	if err := validateScheduleReplacement(patch); err != nil {
		return nil, err
	}
	var row ScheduleReplacement
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.TargetDate = dateOnly(patch.TargetDate)
	row.GroupID = patch.GroupID
	row.PairNumber = patch.PairNumber
	row.Subgroup = patch.Subgroup
	row.SourceSubjectID = patch.SourceSubjectID
	row.SourceLocationID = patch.SourceLocationID
	row.SourceTeacherID = patch.SourceTeacherID
	row.ReplacementSubjectID = patch.ReplacementSubjectID
	row.ReplacementLocationID = patch.ReplacementLocationID
	row.ReplacementTeacherID = patch.ReplacementTeacherID
	row.Reason = patch.Reason
	row.ScheduleOverrideID = patch.ScheduleOverrideID
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteScheduleReplacement(id int64) error {
	return r.db.Delete(&ScheduleReplacement{}, id).Error
}

func validateScheduleReplacement(rep *ScheduleReplacement) error {
	if rep == nil {
		return fmt.Errorf("replacement is nil")
	}
	if rep.GroupID <= 0 {
		return fmt.Errorf("group_id required")
	}
	if rep.PairNumber < 1 || rep.PairNumber > 8 {
		return fmt.Errorf("pair_number must be 1..8")
	}
	if rep.Subgroup != nil && (*rep.Subgroup < 1 || *rep.Subgroup > 2) {
		return fmt.Errorf("subgroup must be 1 or 2")
	}
	return nil
}
