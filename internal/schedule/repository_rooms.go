package schedule

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type LocationTypeLinkFilters struct {
	LocationID *int
	TypeID     *int
}

type TeacherLocationPreferenceFilters struct {
	TeacherID  *int
	LocationID *int
}

type RoomRequestFilters struct {
	TeacherID *int
	SubjectID *int
	GroupID   *int
	Semester  *int16
	Status    *string
}

type RoomAssignmentFilters struct {
	ScheduleTemplateID *int64
	ScheduleOverrideID *int64
	LocationID         *int
	Status             *EntityStatus
}

func normalizeLocation(row *Location) error {
	if row == nil {
		return fmt.Errorf("location is nil")
	}
	row.Name = strings.TrimSpace(row.Name)
	if row.Name == "" {
		return fmt.Errorf("name required")
	}
	row.Kind = strings.ToLower(strings.TrimSpace(row.Kind))
	if row.Kind == "" {
		row.Kind = "physical"
	}
	if row.Kind != "physical" && row.Kind != "virtual" {
		return fmt.Errorf("kind must be physical or virtual")
	}
	return nil
}

func normalizeCampus(row *Campus) error {
	if row == nil {
		return fmt.Errorf("campus is nil")
	}
	row.Name = strings.TrimSpace(row.Name)
	if row.Name == "" {
		return fmt.Errorf("name required")
	}
	return nil
}

func normalizeLocationType(row *LocationType) error {
	if row == nil {
		return fmt.Errorf("location type is nil")
	}
	row.Code = strings.ToLower(strings.TrimSpace(row.Code))
	row.Name = strings.TrimSpace(row.Name)
	if row.Code == "" || row.Name == "" {
		return fmt.Errorf("code and name required")
	}
	return nil
}

func normalizePreference(row *TeacherLocationPreference) error {
	if row == nil {
		return fmt.Errorf("teacher location preference is nil")
	}
	if row.TeacherID <= 0 || row.LocationID <= 0 {
		return fmt.Errorf("teacher_id and location_id required")
	}
	if row.Priority <= 0 {
		row.Priority = 100
	}
	return nil
}

func normalizeRoomRequest(row *RoomRequest) error {
	if row == nil {
		return fmt.Errorf("room request is nil")
	}
	if row.Priority <= 0 {
		row.Priority = 100
	}
	row.Status = strings.ToLower(strings.TrimSpace(row.Status))
	if row.Status == "" {
		row.Status = "pending"
	}
	switch row.Status {
	case "pending", "approved", "rejected", "cancelled":
		return nil
	default:
		return fmt.Errorf("invalid room request status: %s", row.Status)
	}
}

func normalizeRoomAssignment(row *RoomAssignment) error {
	if row == nil {
		return fmt.Errorf("room assignment is nil")
	}
	if row.LocationID <= 0 {
		return fmt.Errorf("location_id required")
	}
	if (row.ScheduleTemplateID == nil) == (row.ScheduleOverrideID == nil) {
		return fmt.Errorf("exactly one of schedule_template_id or schedule_override_id required")
	}
	row.Source = strings.ToLower(strings.TrimSpace(row.Source))
	if row.Source == "" {
		row.Source = "manual"
	}
	switch row.Source {
	case "manual", "auto", "imported", "teacher_preference", "request":
	default:
		return fmt.Errorf("invalid room assignment source: %s", row.Source)
	}
	if row.Status == "" {
		row.Status = StatusPublished
	}
	if row.Status != StatusDraft && row.Status != StatusPublished {
		return fmt.Errorf("invalid room assignment status: %s", row.Status)
	}
	return nil
}

func (r *Repository) ListCampuses() ([]Campus, error) {
	var rows []Campus
	err := r.db.Order("name asc, id asc").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListCampusesPaged(limit, offset *int) ([]Campus, error) {
	var rows []Campus
	q := applyLimitOffset(r.db.Order("name asc, id asc"), limit, offset)
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateCampus(row *Campus) error {
	if err := normalizeCampus(row); err != nil {
		return err
	}
	return r.db.Create(row).Error
}

func (r *Repository) UpdateCampus(id int, patch *Campus) (*Campus, error) {
	if err := normalizeCampus(patch); err != nil {
		return nil, err
	}
	var row Campus
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.Name = patch.Name
	row.Address = patch.Address
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteCampus(id int) error {
	return r.db.Delete(&Campus{}, id).Error
}

func (r *Repository) ListLocationTypes() ([]LocationType, error) {
	var rows []LocationType
	err := r.db.Order("code asc, id asc").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListLocationTypesPaged(limit, offset *int) ([]LocationType, error) {
	var rows []LocationType
	q := applyLimitOffset(r.db.Order("code asc, id asc"), limit, offset)
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateLocationType(row *LocationType) error {
	if err := normalizeLocationType(row); err != nil {
		return err
	}
	return r.db.Create(row).Error
}

func (r *Repository) UpdateLocationType(id int, patch *LocationType) (*LocationType, error) {
	if err := normalizeLocationType(patch); err != nil {
		return nil, err
	}
	var row LocationType
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

func (r *Repository) DeleteLocationType(id int) error {
	return r.db.Delete(&LocationType{}, id).Error
}

func (r *Repository) ListLocationTypeLinks(filters LocationTypeLinkFilters) ([]LocationTypeLink, error) {
	q := r.db.Model(&LocationTypeLink{}).Order("location_id asc, type_id asc")
	if filters.LocationID != nil {
		q = q.Where("location_id = ?", *filters.LocationID)
	}
	if filters.TypeID != nil {
		q = q.Where("type_id = ?", *filters.TypeID)
	}
	var rows []LocationTypeLink
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateLocationTypeLink(row *LocationTypeLink) error {
	if row == nil || row.LocationID <= 0 || row.TypeID <= 0 {
		return fmt.Errorf("location_id and type_id required")
	}
	return r.db.Create(row).Error
}

func (r *Repository) DeleteLocationTypeLink(locationID, typeID int) error {
	return r.db.Where("location_id = ? AND type_id = ?", locationID, typeID).Delete(&LocationTypeLink{}).Error
}

func (r *Repository) ListTeacherLocationPreferences(filters TeacherLocationPreferenceFilters) ([]TeacherLocationPreference, error) {
	q := r.db.Model(&TeacherLocationPreference{}).Order("teacher_id asc, priority asc, id asc")
	if filters.TeacherID != nil {
		q = q.Where("teacher_id = ?", *filters.TeacherID)
	}
	if filters.LocationID != nil {
		q = q.Where("location_id = ?", *filters.LocationID)
	}
	var rows []TeacherLocationPreference
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) ListTeacherLocationPreferencesPaged(filters TeacherLocationPreferenceFilters, limit, offset *int) ([]TeacherLocationPreference, error) {
	q := r.db.Model(&TeacherLocationPreference{}).Order("teacher_id asc, priority asc, id asc")
	if filters.TeacherID != nil {
		q = q.Where("teacher_id = ?", *filters.TeacherID)
	}
	if filters.LocationID != nil {
		q = q.Where("location_id = ?", *filters.LocationID)
	}
	q = applyLimitOffset(q, limit, offset)
	var rows []TeacherLocationPreference
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateTeacherLocationPreference(row *TeacherLocationPreference) error {
	if err := normalizePreference(row); err != nil {
		return err
	}
	return r.db.Create(row).Error
}

func (r *Repository) UpdateTeacherLocationPreference(id int64, patch *TeacherLocationPreference) (*TeacherLocationPreference, error) {
	if err := normalizePreference(patch); err != nil {
		return nil, err
	}
	var row TeacherLocationPreference
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.TeacherID = patch.TeacherID
	row.LocationID = patch.LocationID
	row.Priority = patch.Priority
	row.ValidFrom = patch.ValidFrom
	row.ValidTo = patch.ValidTo
	row.Comment = patch.Comment
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteTeacherLocationPreference(id int64) error {
	return r.db.Delete(&TeacherLocationPreference{}, id).Error
}

func (r *Repository) ListRoomRequests(filters RoomRequestFilters) ([]RoomRequest, error) {
	q := r.db.Model(&RoomRequest{}).Order("priority asc, id asc")
	q = applyRoomRequestFilters(q, filters)
	var rows []RoomRequest
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) ListRoomRequestsPaged(filters RoomRequestFilters, limit, offset *int) ([]RoomRequest, error) {
	q := r.db.Model(&RoomRequest{}).Order("priority asc, id asc")
	q = applyRoomRequestFilters(q, filters)
	q = applyLimitOffset(q, limit, offset)
	var rows []RoomRequest
	err := q.Find(&rows).Error
	return rows, err
}

func applyRoomRequestFilters(q *gorm.DB, filters RoomRequestFilters) *gorm.DB {
	if filters.TeacherID != nil {
		q = q.Where("teacher_id = ?", *filters.TeacherID)
	}
	if filters.SubjectID != nil {
		q = q.Where("subject_id = ?", *filters.SubjectID)
	}
	if filters.GroupID != nil {
		q = q.Where("group_id = ?", *filters.GroupID)
	}
	if filters.Semester != nil {
		q = q.Where("semester = ?", *filters.Semester)
	}
	if filters.Status != nil && strings.TrimSpace(*filters.Status) != "" {
		q = q.Where("status = ?", strings.ToLower(strings.TrimSpace(*filters.Status)))
	}
	return q
}

func (r *Repository) CreateRoomRequest(row *RoomRequest) error {
	if err := normalizeRoomRequest(row); err != nil {
		return err
	}
	return r.db.Create(row).Error
}

func (r *Repository) UpdateRoomRequest(id int64, patch *RoomRequest) (*RoomRequest, error) {
	if err := normalizeRoomRequest(patch); err != nil {
		return nil, err
	}
	var row RoomRequest
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.TeacherID = patch.TeacherID
	row.SubjectID = patch.SubjectID
	row.GroupID = patch.GroupID
	row.Semester = patch.Semester
	row.RequiredTypeID = patch.RequiredTypeID
	row.PreferredLocationID = patch.PreferredLocationID
	row.Priority = patch.Priority
	row.Comment = patch.Comment
	row.Status = patch.Status
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteRoomRequest(id int64) error {
	return r.db.Delete(&RoomRequest{}, id).Error
}

func (r *Repository) ListRoomAssignments(filters RoomAssignmentFilters) ([]RoomAssignment, error) {
	q := r.db.Model(&RoomAssignment{}).Order("id asc")
	q = applyRoomAssignmentFilters(q, filters)
	var rows []RoomAssignment
	err := q.Find(&rows).Error
	return rows, err
}

func (r *Repository) ListRoomAssignmentsPaged(filters RoomAssignmentFilters, limit, offset *int) ([]RoomAssignment, error) {
	q := r.db.Model(&RoomAssignment{}).Order("id asc")
	q = applyRoomAssignmentFilters(q, filters)
	q = applyLimitOffset(q, limit, offset)
	var rows []RoomAssignment
	err := q.Find(&rows).Error
	return rows, err
}

func applyRoomAssignmentFilters(q *gorm.DB, filters RoomAssignmentFilters) *gorm.DB {
	if filters.ScheduleTemplateID != nil {
		q = q.Where("schedule_template_id = ?", *filters.ScheduleTemplateID)
	}
	if filters.ScheduleOverrideID != nil {
		q = q.Where("schedule_override_id = ?", *filters.ScheduleOverrideID)
	}
	if filters.LocationID != nil {
		q = q.Where("location_id = ?", *filters.LocationID)
	}
	if filters.Status != nil {
		q = q.Where("status = ?", *filters.Status)
	}
	return q
}

func (r *Repository) CreateRoomAssignment(row *RoomAssignment) error {
	if err := normalizeRoomAssignment(row); err != nil {
		return err
	}
	return r.db.Create(row).Error
}

func (r *Repository) UpdateRoomAssignment(id int64, patch *RoomAssignment) (*RoomAssignment, error) {
	if err := normalizeRoomAssignment(patch); err != nil {
		return nil, err
	}
	var row RoomAssignment
	if err := r.db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.ScheduleTemplateID = patch.ScheduleTemplateID
	row.ScheduleOverrideID = patch.ScheduleOverrideID
	row.LocationID = patch.LocationID
	row.Source = patch.Source
	row.Status = patch.Status
	if err := r.db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) DeleteRoomAssignment(id int64) error {
	return r.db.Delete(&RoomAssignment{}, id).Error
}

func (r *Repository) ResolveTeacherPreferredLocation(teacherName string, targetDate time.Time) (*Location, error) {
	teacherName = strings.TrimSpace(teacherName)
	if teacherName == "" {
		return nil, nil
	}
	var row Location
	q := r.db.Table("teacher_location_preferences tlp").
		Select("l.id, l.campus_id, l.name, l.kind, l.capacity, l.is_active, l.created_at, l.updated_at").
		Joins("JOIN teachers t ON t.id = tlp.teacher_id").
		Joins("JOIN locations l ON l.id = tlp.location_id").
		Where("t.name = ? AND t.deleted_at IS NULL AND l.is_active = TRUE", teacherName)
	if !targetDate.IsZero() {
		d := dateOnly(targetDate)
		q = q.Where("(tlp.valid_from IS NULL OR tlp.valid_from <= ?) AND (tlp.valid_to IS NULL OR tlp.valid_to >= ?)", d, d)
	}
	err := q.Order("tlp.priority asc, tlp.id asc").Limit(1).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, nil
	}
	return &row, nil
}

func (r *Repository) ResolveRequestedLocation(groupID int, subjectID int, semester *int16) (*Location, error) {
	if groupID <= 0 || subjectID <= 0 {
		return nil, nil
	}

	type requestCandidate struct {
		PreferredLocationID *int `gorm:"column:preferred_location_id"`
		RequiredTypeID      *int `gorm:"column:required_type_id"`
	}
	var req requestCandidate
	q := r.db.Table("room_requests").
		Select("preferred_location_id, required_type_id").
		Where("status IN ? AND (group_id IS NULL OR group_id = ?) AND (subject_id IS NULL OR subject_id = ?)", []string{"pending", "approved"}, groupID, subjectID).
		Where("(preferred_location_id IS NOT NULL OR required_type_id IS NOT NULL)")
	if semester != nil {
		q = q.Where("(semester IS NULL OR semester = ?)", *semester)
	} else {
		q = q.Where("semester IS NULL")
	}
	if err := q.Order("priority asc, id asc").Limit(1).Scan(&req).Error; err != nil {
		return nil, err
	}
	if req.PreferredLocationID == nil && req.RequiredTypeID == nil {
		return nil, nil
	}

	var row Location
	if req.PreferredLocationID != nil {
		if err := r.db.Where("id = ? AND is_active = TRUE", *req.PreferredLocationID).First(&row).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, nil
			}
			return nil, err
		}
		return &row, nil
	}

	if err := r.db.Table("locations l").
		Select("l.id, l.campus_id, l.name, l.kind, l.capacity, l.is_active, l.created_at, l.updated_at").
		Joins("JOIN location_type_links ltl ON ltl.location_id = l.id").
		Where("ltl.type_id = ? AND l.is_active = TRUE", *req.RequiredTypeID).
		Order("l.name asc, l.id asc").
		Limit(1).
		Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, nil
	}
	return &row, nil
}
