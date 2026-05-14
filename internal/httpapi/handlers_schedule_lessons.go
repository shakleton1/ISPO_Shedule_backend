package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ispo-schedule/internal/push"
	"ispo-schedule/internal/schedule"
)

type adminScheduleLessonRequest struct {
	GroupID            int                   `json:"group_id"`
	LessonDate         string                `json:"lesson_date"`
	PairNumber         int16                 `json:"pair_number"`
	Subgroup           *int16                `json:"subgroup"`
	SubjectID          *int                  `json:"subject_id"`
	TeacherID          *int                  `json:"teacher_id"`
	LessonFormat       string                `json:"lesson_format"`
	Status             schedule.EntityStatus `json:"status"`
	Source             string                `json:"source"`
	FlowKey            *string               `json:"flow_key"`
	Comment            *string               `json:"comment"`
	ExpectedVersion    *int                  `json:"expected_version"`
	ConfirmConstraints bool                  `json:"confirm_constraints"`
}

type applyScheduleOverrideRequest struct {
	ScheduleLessonID        *int64  `json:"schedule_lesson_id"`
	GroupID                 int     `json:"group_id"`
	LessonDate              string  `json:"lesson_date"`
	PairNumber              int16   `json:"pair_number"`
	Subgroup                *int16  `json:"subgroup"`
	ActionType              string  `json:"action_type"`
	ReplacementSubjectID    *int    `json:"replacement_subject_id"`
	ReplacementTeacherID    *int    `json:"replacement_teacher_id"`
	ReplacementLocationID   *int    `json:"replacement_location_id"`
	ReplacementLessonFormat *string `json:"replacement_lesson_format"`
	Reason                  *string `json:"reason"`
	ExpectedLessonVersion   *int    `json:"expected_lesson_version"`
	ConfirmConstraints      bool    `json:"confirm_constraints"`
}

func handleAdminListScheduleLessons(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		groupID, ok := parseQueryInt(c, "group_id")
		if !ok {
			return
		}
		teacherID, ok := parseQueryInt(c, "teacher_id")
		if !ok {
			return
		}
		lessonDate, ok := parseQueryDate(c, "lesson_date")
		if !ok {
			return
		}
		startDate, ok := parseQueryDate(c, "start_date")
		if !ok {
			return
		}
		endDate, ok := parseQueryDate(c, "end_date")
		if !ok {
			return
		}
		var status *schedule.EntityStatus
		if s := strings.TrimSpace(c.Query("status")); s != "" {
			v := schedule.EntityStatus(s)
			status = &v
		}
		filters := schedule.ScheduleLessonFilters{
			GroupID:    groupID,
			TeacherID:  teacherID,
			LessonDate: lessonDate,
			StartDate:  startDate,
			EndDate:    endDate,
			Status:     status,
		}
		var rows []schedule.ScheduleLesson
		var err error
		if page.Limit != nil {
			rows, err = repo.ListScheduleLessonsPaged(filters, page.Limit, page.Offset)
		} else {
			rows, err = repo.ListScheduleLessons(filters)
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]scheduleLessonDTO, 0, len(rows))
		for _, row := range rows {
			out = append(out, toScheduleLessonDTO(row))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handleAdminCreateScheduleLesson(svc *schedule.Service, repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		row, _, confirmed, ok := bindScheduleLessonRequest(c)
		if !ok {
			return
		}
		if err := svc.CreateScheduleLesson(row, confirmed); err != nil {
			writeScheduleWriteError(c, err)
			return
		}
		version, _ := bumpScheduleVersionAndGet(repo)
		if pushSvc != nil {
			pushSvc.NotifyScheduleUpdatedAsync(row.GroupID, version)
		}
		writeAudit(c, repo, "create", "schedule_lessons", strconv.FormatInt(row.ID, 10), row)
		c.JSON(http.StatusCreated, toScheduleLessonDTO(*row))
	}
}

func handleAdminUpdateScheduleLesson(svc *schedule.Service, repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		row, expectedVersion, confirmed, ok := bindScheduleLessonRequest(c)
		if !ok {
			return
		}
		updated, err := svc.UpdateScheduleLesson(id, row, expectedVersion, confirmed)
		if err != nil {
			writeScheduleWriteError(c, err)
			return
		}
		version, _ := bumpScheduleVersionAndGet(repo)
		if pushSvc != nil {
			pushSvc.NotifyScheduleUpdatedAsync(updated.GroupID, version)
		}
		writeAudit(c, repo, "update", "schedule_lessons", strconv.FormatInt(id, 10), updated)
		c.JSON(http.StatusOK, toScheduleLessonDTO(*updated))
	}
}

func handleAdminDeleteScheduleLesson(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		row, _ := repo.GetScheduleLesson(id)
		if err := repo.DeleteScheduleLesson(id); err != nil {
			writeDBError(c, err)
			return
		}
		version, _ := bumpScheduleVersionAndGet(repo)
		if pushSvc != nil && row != nil {
			pushSvc.NotifyScheduleUpdatedAsync(row.GroupID, version)
		}
		writeAudit(c, repo, "delete", "schedule_lessons", strconv.FormatInt(id, 10), nil)
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}

func handleAdminCancelScheduleLesson(svc *schedule.Service, repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var req struct {
			ExpectedVersion *int    `json:"expected_version"`
			Reason          *string `json:"reason"`
		}
		_ = c.ShouldBindJSON(&req)
		row, err := repo.GetScheduleLesson(id)
		if err != nil {
			writeDBError(c, err)
			return
		}
		applied, err := svc.ApplyScheduleOverride(schedule.ApplyScheduleOverrideRequest{
			ScheduleLessonID:      &id,
			GroupID:               row.GroupID,
			LessonDate:            row.LessonDate,
			PairNumber:            row.PairNumber,
			Subgroup:              row.Subgroup,
			ActionType:            string(schedule.OverrideCancel),
			Reason:                req.Reason,
			ExpectedLessonVersion: req.ExpectedVersion,
			ConfirmConstraints:    true,
		})
		if err != nil {
			writeScheduleWriteError(c, err)
			return
		}
		updated, err := repo.GetScheduleLesson(id)
		if err != nil {
			writeDBError(c, err)
			return
		}
		st, _ := repo.GetSystemState()
		if pushSvc != nil {
			var version time.Time
			if st != nil {
				version = st.ScheduleVersion
			}
			pushSvc.NotifyScheduleUpdatedAsync(updated.GroupID, version)
		}
		writeAudit(c, repo, "cancel", "schedule_lessons", strconv.FormatInt(id, 10), applied)
		c.JSON(http.StatusOK, toScheduleLessonDTO(*updated))
	}
}

func handleAdminApplyScheduleOverride(svc *schedule.Service, repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req applyScheduleOverrideRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeValidationError(c, "body", "invalid json")
			return
		}
		lessonDate, err := time.Parse("2006-01-02", strings.TrimSpace(req.LessonDate))
		if err != nil {
			writeValidationError(c, "lesson_date", "invalid lesson_date")
			return
		}
		_, actorID, _, _ := auditActorFromContext(c)
		var createdBy *int
		if actorID != nil {
			v := int(*actorID)
			createdBy = &v
		}
		applied, err := svc.ApplyScheduleOverride(schedule.ApplyScheduleOverrideRequest{
			ScheduleLessonID:        req.ScheduleLessonID,
			GroupID:                 req.GroupID,
			LessonDate:              lessonDate,
			PairNumber:              req.PairNumber,
			Subgroup:                req.Subgroup,
			ActionType:              req.ActionType,
			ReplacementSubjectID:    req.ReplacementSubjectID,
			ReplacementTeacherID:    req.ReplacementTeacherID,
			ReplacementLocationID:   req.ReplacementLocationID,
			ReplacementLessonFormat: req.ReplacementLessonFormat,
			Reason:                  req.Reason,
			ExpectedLessonVersion:   req.ExpectedLessonVersion,
			ConfirmConstraints:      req.ConfirmConstraints,
			CreatedBy:               createdBy,
		})
		if err != nil {
			writeScheduleWriteError(c, err)
			return
		}
		st, _ := repo.GetSystemState()
		if pushSvc != nil && st != nil {
			pushSvc.NotifyScheduleUpdatedAsync(applied.GroupID, st.ScheduleVersion)
		}
		writeAudit(c, repo, "apply", "schedule_overrides", strconv.FormatInt(applied.ID, 10), applied)
		c.JSON(http.StatusCreated, toScheduleOverrideDTO(*applied))
	}
}

func handleAdminListAppliedScheduleOverrides(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		groupID, ok := parseQueryInt(c, "group_id")
		if !ok {
			return
		}
		teacherID, ok := parseQueryInt(c, "teacher_id")
		if !ok {
			return
		}
		startDate, ok := parseQueryDate(c, "start_date")
		if !ok {
			return
		}
		endDate, ok := parseQueryDate(c, "end_date")
		if !ok {
			return
		}
		var action *schedule.OverrideAction
		if v := strings.TrimSpace(c.Query("action_type")); v != "" {
			a := schedule.OverrideAction(v)
			action = &a
		}
		filters := schedule.ScheduleOverrideFilters{
			GroupID:    groupID,
			TeacherID:  teacherID,
			StartDate:  startDate,
			EndDate:    endDate,
			ActionType: action,
		}
		var rows []schedule.ScheduleOverride
		var err error
		if page.Limit != nil {
			rows, err = repo.ListScheduleOverridesPaged(filters, page.Limit, page.Offset)
		} else {
			rows, err = repo.ListScheduleOverrides(filters)
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]scheduleOverrideDTO, 0, len(rows))
		for _, row := range rows {
			out = append(out, toScheduleOverrideDTO(row))
		}
		c.JSON(http.StatusOK, out)
	}
}

func bindScheduleLessonRequest(c *gin.Context) (*schedule.ScheduleLesson, *int, bool, bool) {
	var req adminScheduleLessonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, "body", "invalid json")
		return nil, nil, false, false
	}
	d, err := time.Parse("2006-01-02", strings.TrimSpace(req.LessonDate))
	if err != nil {
		writeValidationError(c, "lesson_date", "invalid lesson_date")
		return nil, nil, false, false
	}
	return &schedule.ScheduleLesson{
		GroupID:      req.GroupID,
		LessonDate:   d,
		PairNumber:   req.PairNumber,
		Subgroup:     req.Subgroup,
		SubjectID:    req.SubjectID,
		TeacherID:    req.TeacherID,
		LessonFormat: req.LessonFormat,
		Status:       req.Status,
		Source:       req.Source,
		FlowKey:      req.FlowKey,
		Comment:      req.Comment,
	}, req.ExpectedVersion, req.ConfirmConstraints, true
}

func parseQueryDate(c *gin.Context, name string) (*time.Time, bool) {
	v := strings.TrimSpace(c.Query(name))
	if v == "" {
		return nil, true
	}
	d, err := time.Parse("2006-01-02", v)
	if err != nil {
		writeValidationError(c, name, "invalid "+name)
		return nil, false
	}
	return &d, true
}

func writeScheduleWriteError(c *gin.Context, err error) {
	if schedule.IsLessonVersionConflict(err) {
		abortWithError(c, http.StatusConflict, "lesson_version_conflict", "", "lesson version conflict")
		return
	}
	var confirm *schedule.TeacherConstraintConfirmationRequiredError
	if errors.As(err, &confirm) {
		c.JSON(http.StatusConflict, gin.H{
			"error":       "teacher_day_constraint_confirmation_required",
			"teacher_id":  confirm.Constraint.TeacherID,
			"target_date": confirm.Constraint.TargetDate.Format("2006-01-02"),
			"reason":      confirm.Constraint.Reason,
			"message":     "Teacher has a soft day constraint. Confirmation required.",
		})
		return
	}
	var hardBlock *schedule.TeacherConstraintHardBlockError
	if errors.As(err, &hardBlock) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":       "teacher_day_constraint_hard_block",
			"teacher_id":  hardBlock.Constraint.TeacherID,
			"target_date": hardBlock.Constraint.TargetDate.Format("2006-01-02"),
			"reason":      hardBlock.Constraint.Reason,
		})
		return
	}
	var roomConflict *schedule.RoomConflictError
	if errors.As(err, &roomConflict) {
		c.JSON(http.StatusConflict, gin.H{
			"error":       "room_conflict",
			"location_id": roomConflict.LocationID,
			"date":        roomConflict.Date.Format("2006-01-02"),
			"pair_number": roomConflict.PairNumber,
		})
		return
	}
	writeDBError(c, err)
}
