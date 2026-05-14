package schedule

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ApplyScheduleOverrideRequest struct {
	ScheduleLessonID *int64    `json:"schedule_lesson_id"`
	GroupID          int       `json:"group_id"`
	LessonDate       time.Time `json:"lesson_date"`
	PairNumber       int16     `json:"pair_number"`
	Subgroup         *int16    `json:"subgroup"`
	ActionType       string    `json:"action_type"`

	ReplacementSubjectID    *int    `json:"replacement_subject_id"`
	ReplacementTeacherID    *int    `json:"replacement_teacher_id"`
	ReplacementLocationID   *int    `json:"replacement_location_id"`
	ReplacementLessonFormat *string `json:"replacement_lesson_format"`
	Reason                  *string `json:"reason"`
	ExpectedLessonVersion   *int    `json:"expected_lesson_version"`
	ConfirmConstraints      bool    `json:"confirm_constraints"`
	CreatedBy               *int    `json:"created_by"`
}

type TeacherConstraintConfirmationRequiredError struct {
	Constraint TeacherDayConstraint
}

func (e *TeacherConstraintConfirmationRequiredError) Error() string {
	return "teacher day constraint confirmation required"
}

type TeacherConstraintHardBlockError struct {
	Constraint TeacherDayConstraint
}

func (e *TeacherConstraintHardBlockError) Error() string {
	return "teacher day constraint hard block"
}

type RoomConflictError struct {
	LocationID int
	Date       time.Time
	PairNumber int16
}

func (e *RoomConflictError) Error() string {
	return "room conflict"
}

func (s *Service) ApplyScheduleOverride(req ApplyScheduleOverrideRequest) (*ScheduleOverride, error) {
	action := normalizeOverrideAction(OverrideAction(req.ActionType))
	if action == "" {
		return nil, fmt.Errorf("invalid action_type")
	}
	req.LessonDate = dateOnly(req.LessonDate)
	if req.GroupID <= 0 {
		return nil, fmt.Errorf("group_id required")
	}
	if req.PairNumber < 1 || req.PairNumber > 8 {
		return nil, fmt.Errorf("pair_number must be 1..8")
	}
	if req.Subgroup != nil && (*req.Subgroup < 1 || *req.Subgroup > 2) {
		return nil, fmt.Errorf("subgroup must be 1 or 2")
	}

	var out ScheduleOverride
	err := s.repo.DB().Transaction(func(tx *gorm.DB) error {
		switch action {
		case OverrideAdd:
			row, err := s.applyAddOverride(tx, req)
			if err != nil {
				return err
			}
			out = *row
		case OverrideReplace, OverrideCancel, OverrideRestore:
			row, err := s.applyExistingLessonOverride(tx, req, action)
			if err != nil {
				return err
			}
			out = *row
		default:
			return fmt.Errorf("unsupported action_type")
		}
		return bumpScheduleVersionTx(tx)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Service) CreateScheduleLesson(row *ScheduleLesson, confirmed bool) error {
	if row == nil {
		return fmt.Errorf("schedule lesson is nil")
	}
	return s.repo.DB().Transaction(func(tx *gorm.DB) error {
		if err := normalizeScheduleLesson(row); err != nil {
			return err
		}
		if err := s.checkTeacherConstraintsTx(tx, row.TeacherID, row.LessonDate, confirmed); err != nil {
			return err
		}
		return tx.Create(row).Error
	})
}

func (s *Service) UpdateScheduleLesson(id int64, patch *ScheduleLesson, expectedVersion *int, confirmed bool) (*ScheduleLesson, error) {
	if patch == nil {
		return nil, fmt.Errorf("schedule lesson patch is nil")
	}
	var out ScheduleLesson
	err := s.repo.DB().Transaction(func(tx *gorm.DB) error {
		var row ScheduleLesson
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, id).Error; err != nil {
			return err
		}
		if expectedVersion != nil && row.Version != *expectedVersion {
			return ErrLessonVersionConflict
		}

		row.GroupID = patch.GroupID
		row.LessonDate = dateOnly(patch.LessonDate)
		row.PairNumber = patch.PairNumber
		row.Subgroup = patch.Subgroup
		row.SubjectID = patch.SubjectID
		row.TeacherID = patch.TeacherID
		row.LessonFormat = normalizeLessonFormat(patch.LessonFormat)
		row.Status = patch.Status
		row.Source = normalizeLessonSource(patch.Source)
		row.FlowKey = patch.FlowKey
		row.Comment = patch.Comment
		row.Version++
		if err := normalizeScheduleLesson(&row); err != nil {
			return err
		}
		if err := s.checkTeacherConstraintsTx(tx, row.TeacherID, row.LessonDate, confirmed); err != nil {
			return err
		}
		if err := tx.Save(&row).Error; err != nil {
			return err
		}
		out = row
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Service) applyExistingLessonOverride(tx *gorm.DB, req ApplyScheduleOverrideRequest, action OverrideAction) (*ScheduleOverride, error) {
	lesson, err := lockScheduleLessonForOverride(tx, req)
	if err != nil {
		return nil, err
	}
	if req.ExpectedLessonVersion != nil && lesson.Version != *req.ExpectedLessonVersion {
		return nil, ErrLessonVersionConflict
	}

	currentRoom, err := getRoomAssignmentForLesson(tx, lesson.ID)
	if err != nil {
		return nil, err
	}

	sourceFormat := lesson.LessonFormat
	override := &ScheduleOverride{
		ScheduleLessonID:      &lesson.ID,
		GroupID:               lesson.GroupID,
		LessonDate:            lesson.LessonDate,
		PairNumber:            lesson.PairNumber,
		Subgroup:              lesson.Subgroup,
		ActionType:            action,
		SourceSubjectID:       lesson.SubjectID,
		SourceTeacherID:       lesson.TeacherID,
		SourceLessonFormat:    &sourceFormat,
		Reason:                req.Reason,
		Status:                "applied",
		ExpectedLessonVersion: req.ExpectedLessonVersion,
		CreatedBy:             req.CreatedBy,
	}
	if currentRoom != nil {
		override.SourceLocationID = &currentRoom.LocationID
	}

	if action == OverrideCancel {
		lesson.Status = StatusCancelled
		lesson.Source = "replacement"
		lesson.Version++
		applied := lesson.Version
		override.AppliedLessonVersion = &applied
		now := time.Now().UTC()
		override.AppliedAt = &now
		if err := tx.Save(lesson).Error; err != nil {
			return nil, err
		}
		if err := tx.Create(override).Error; err != nil {
			return nil, err
		}
		return override, nil
	}

	nextSubjectID := lesson.SubjectID
	if req.ReplacementSubjectID != nil {
		nextSubjectID = req.ReplacementSubjectID
	}
	nextTeacherID := lesson.TeacherID
	if req.ReplacementTeacherID != nil {
		nextTeacherID = req.ReplacementTeacherID
	}
	nextFormat := lesson.LessonFormat
	if req.ReplacementLessonFormat != nil {
		nextFormat = normalizeLessonFormat(*req.ReplacementLessonFormat)
	}
	nextLocationID := (*int)(nil)
	if currentRoom != nil {
		v := currentRoom.LocationID
		nextLocationID = &v
	}
	if req.ReplacementLocationID != nil {
		nextLocationID = req.ReplacementLocationID
	}

	if err := s.checkTeacherConstraintsTx(tx, nextTeacherID, lesson.LessonDate, req.ConfirmConstraints); err != nil {
		return nil, err
	}
	if nextLocationID != nil {
		if err := ensureRoomAvailableTx(tx, *nextLocationID, lesson.LessonDate, lesson.PairNumber, lesson.FlowKey, &lesson.ID); err != nil {
			return nil, err
		}
	}

	override.ReplacementSubjectID = nextSubjectID
	override.ReplacementTeacherID = nextTeacherID
	override.ReplacementLocationID = nextLocationID
	override.ReplacementLessonFormat = &nextFormat

	lesson.SubjectID = nextSubjectID
	lesson.TeacherID = nextTeacherID
	lesson.LessonFormat = nextFormat
	lesson.Status = StatusPublished
	lesson.Source = "replacement"
	lesson.Version++
	applied := lesson.Version
	override.AppliedLessonVersion = &applied
	now := time.Now().UTC()
	override.AppliedAt = &now

	if err := tx.Save(lesson).Error; err != nil {
		return nil, err
	}
	if nextLocationID != nil {
		if _, err := upsertRoomAssignmentForLesson(tx, lesson.ID, *nextLocationID, "replacement"); err != nil {
			return nil, err
		}
	}
	if err := tx.Create(override).Error; err != nil {
		return nil, err
	}
	return override, nil
}

func (s *Service) applyAddOverride(tx *gorm.DB, req ApplyScheduleOverrideRequest) (*ScheduleOverride, error) {
	var cnt int64
	q := tx.Model(&ScheduleLesson{}).
		Where("group_id = ? AND lesson_date = ? AND pair_number = ? AND status <> ?", req.GroupID, req.LessonDate, req.PairNumber, StatusCancelled)
	if req.Subgroup == nil {
		q = q.Where("subgroup IS NULL")
	} else {
		q = q.Where("subgroup = ?", *req.Subgroup)
	}
	if err := q.Count(&cnt).Error; err != nil {
		return nil, err
	}
	if cnt > 0 {
		return nil, fmt.Errorf("schedule lesson slot is occupied")
	}
	if req.ReplacementSubjectID == nil {
		return nil, fmt.Errorf("replacement_subject_id required for add")
	}
	if err := s.checkTeacherConstraintsTx(tx, req.ReplacementTeacherID, req.LessonDate, req.ConfirmConstraints); err != nil {
		return nil, err
	}
	lessonFormat := "offline"
	if req.ReplacementLessonFormat != nil {
		lessonFormat = normalizeLessonFormat(*req.ReplacementLessonFormat)
	}
	lesson := &ScheduleLesson{
		GroupID:      req.GroupID,
		LessonDate:   req.LessonDate,
		PairNumber:   req.PairNumber,
		Subgroup:     req.Subgroup,
		SubjectID:    req.ReplacementSubjectID,
		TeacherID:    req.ReplacementTeacherID,
		LessonFormat: lessonFormat,
		Status:       StatusPublished,
		Source:       "replacement",
		Version:      1,
	}
	if err := normalizeScheduleLesson(lesson); err != nil {
		return nil, err
	}
	if req.ReplacementLocationID != nil {
		if err := ensureRoomAvailableTx(tx, *req.ReplacementLocationID, req.LessonDate, req.PairNumber, lesson.FlowKey, nil); err != nil {
			return nil, err
		}
	}
	if err := tx.Create(lesson).Error; err != nil {
		return nil, err
	}
	if req.ReplacementLocationID != nil {
		if _, err := upsertRoomAssignmentForLesson(tx, lesson.ID, *req.ReplacementLocationID, "replacement"); err != nil {
			return nil, err
		}
	}
	applied := lesson.Version
	now := time.Now().UTC()
	override := &ScheduleOverride{
		ScheduleLessonID:        &lesson.ID,
		GroupID:                 req.GroupID,
		LessonDate:              req.LessonDate,
		PairNumber:              req.PairNumber,
		Subgroup:                req.Subgroup,
		ActionType:              OverrideAdd,
		ReplacementSubjectID:    req.ReplacementSubjectID,
		ReplacementTeacherID:    req.ReplacementTeacherID,
		ReplacementLocationID:   req.ReplacementLocationID,
		ReplacementLessonFormat: &lessonFormat,
		Reason:                  req.Reason,
		Status:                  "applied",
		AppliedLessonVersion:    &applied,
		CreatedBy:               req.CreatedBy,
		AppliedAt:               &now,
	}
	if err := tx.Create(override).Error; err != nil {
		return nil, err
	}
	return override, nil
}

func lockScheduleLessonForOverride(tx *gorm.DB, req ApplyScheduleOverrideRequest) (*ScheduleLesson, error) {
	var lesson ScheduleLesson
	q := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Model(&ScheduleLesson{})
	if req.ScheduleLessonID != nil {
		q = q.Where("id = ?", *req.ScheduleLessonID)
	} else {
		q = q.Where("group_id = ? AND lesson_date = ? AND pair_number = ? AND status <> ?", req.GroupID, req.LessonDate, req.PairNumber, StatusCancelled)
		if req.Subgroup == nil {
			q = q.Where("subgroup IS NULL")
		} else {
			q = q.Where("subgroup = ?", *req.Subgroup)
		}
	}
	if err := q.First(&lesson).Error; err != nil {
		return nil, err
	}
	return &lesson, nil
}

func (s *Service) checkTeacherConstraintsTx(tx *gorm.DB, teacherID *int, lessonDate time.Time, confirmed bool) error {
	if teacherID == nil || *teacherID <= 0 {
		return nil
	}
	var rows []TeacherDayConstraint
	if err := tx.Model(&TeacherDayConstraint{}).
		Where("teacher_id = ? AND target_date = ?", *teacherID, dateOnly(lessonDate)).
		Order("CASE constraint_level WHEN 'hard_block' THEN 0 ELSE 1 END, id asc").
		Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if row.ConstraintLevel == "hard_block" {
			return &TeacherConstraintHardBlockError{Constraint: row}
		}
		if row.ConstraintLevel == "warning" && row.RequiresConfirmation && !confirmed {
			return &TeacherConstraintConfirmationRequiredError{Constraint: row}
		}
	}
	return nil
}

func ensureRoomAvailableTx(tx *gorm.DB, locationID int, lessonDate time.Time, pairNumber int16, flowKey *string, excludeLessonID *int64) error {
	if locationID <= 0 {
		return fmt.Errorf("location_id required")
	}
	virtual, err := locationIsVirtualTx(tx, locationID)
	if err != nil {
		return err
	}
	if virtual {
		return nil
	}
	flow := strings.ToLower(strings.TrimSpace(stringValue(flowKey)))
	q := tx.Table("schedule_lessons sl").
		Joins("JOIN room_assignments ra ON ra.schedule_lesson_id = sl.id AND ra.status = ?", StatusPublished).
		Where("sl.lesson_date = ? AND sl.pair_number = ? AND ra.location_id = ? AND sl.status <> ?", dateOnly(lessonDate), pairNumber, locationID, StatusCancelled).
		Where("(COALESCE(sl.flow_key, '') = '' OR ? = '' OR lower(sl.flow_key) <> ?)", flow, flow)
	if excludeLessonID != nil {
		q = q.Where("sl.id <> ?", *excludeLessonID)
	}
	var cnt int64
	if err := q.Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return &RoomConflictError{LocationID: locationID, Date: dateOnly(lessonDate), PairNumber: pairNumber}
	}
	return nil
}

func locationIsVirtualTx(tx *gorm.DB, locationID int) (bool, error) {
	var out struct {
		Kind      string
		HasOnline bool
	}
	err := tx.Table("locations l").
		Select(`l.kind, EXISTS (
			SELECT 1
			FROM location_type_links ltl
			JOIN location_types lt ON lt.id = ltl.type_id
			WHERE ltl.location_id = l.id AND lt.code = 'online'
		) AS has_online`).
		Where("l.id = ?", locationID).
		Scan(&out).Error
	if err != nil {
		return false, err
	}
	return strings.EqualFold(out.Kind, "virtual") || out.HasOnline, nil
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func bumpScheduleVersionTx(tx *gorm.DB) error {
	return tx.Exec(`
INSERT INTO system_state (id, schedule_version)
VALUES (1, now())
ON CONFLICT (id)
DO UPDATE SET schedule_version = EXCLUDED.schedule_version`).Error
}

func IsLessonVersionConflict(err error) bool {
	return errors.Is(err, ErrLessonVersionConflict)
}
