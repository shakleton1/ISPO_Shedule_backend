package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ispo-schedule/internal/push"
	"ispo-schedule/internal/schedule"
)

func handleAdminListStudyActivities(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		var (
			rows []schedule.StudyActivity
			err  error
		)
		if p.Limit != nil {
			rows, err = repo.ListStudyActivitiesPaged(p.Limit, p.Offset)
		} else {
			rows, err = repo.ListStudyActivities()
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]studyActivityDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toStudyActivityDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handleAdminCreateStudyActivity(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.StudyActivity
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if err := repo.CreateStudyActivity(&req); err != nil {
			writeValidationError(c, "", err.Error())
			return
		}
		writeAudit(c, repo, "create", "study_activities", strconv.Itoa(req.ID), req)
		c.JSON(http.StatusCreated, toStudyActivityDTO(req))
	}
}

func handleAdminUpdateStudyActivity(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var req schedule.StudyActivity
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		row, err := repo.UpdateStudyActivity(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "update", "study_activities", strconv.Itoa(id), row)
		c.JSON(http.StatusOK, toStudyActivityDTO(*row))
	}
}

func handleAdminDeleteStudyActivity(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		if err := repo.DeleteStudyActivity(id); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "delete", "study_activities", strconv.Itoa(id), nil)
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}

func handleAdminListStudyCalendarWeeks(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		var filters schedule.StudyCalendarWeekFilters
		if v := c.Query("group_id"); v != "" {
			id, err := strconv.Atoi(v)
			if err != nil {
				writeValidationError(c, "group_id", "invalid group_id")
				return
			}
			filters.GroupID = &id
		}
		if v := c.Query("week_number"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				writeValidationError(c, "week_number", "invalid week_number")
				return
			}
			vv := int16(n)
			filters.WeekNumber = &vv
		}

		var (
			rows []schedule.StudyCalendarWeek
			err  error
		)
		if p.Limit != nil {
			rows, err = repo.ListStudyCalendarWeeksPaged(filters, p.Limit, p.Offset)
		} else {
			rows, err = repo.ListStudyCalendarWeeks(filters)
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]studyCalendarWeekDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toStudyCalendarWeekDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

type studyCalendarWeekReq struct {
	WeekNumber    int16   `json:"week_number"`
	WeekStartDate *string `json:"week_start_date"`
	ActivityID    *int    `json:"activity_id"`
	AllowsLessons *bool   `json:"allows_lessons"`
	Comment       *string `json:"comment"`
}

func handleAdminUpsertStudyCalendarWeeks(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID, err := strconv.Atoi(c.Param("id"))
		if err != nil || groupID <= 0 {
			writeValidationError(c, "id", "invalid group id")
			return
		}
		var req []studyCalendarWeekReq
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		weeks := make([]schedule.StudyCalendarWeek, 0, len(req))
		for i, r := range req {
			allows := true
			if r.AllowsLessons != nil {
				allows = *r.AllowsLessons
			}
			var weekStart *time.Time
			if r.WeekStartDate != nil && *r.WeekStartDate != "" {
				d, err := time.Parse("2006-01-02", *r.WeekStartDate)
				if err != nil {
					writeValidationError(c, "week_start_date", "row "+strconv.Itoa(i+1)+": week_start_date must be YYYY-MM-DD")
					return
				}
				weekStart = &d
			}
			weeks = append(weeks, schedule.StudyCalendarWeek{
				WeekNumber:    r.WeekNumber,
				WeekStartDate: weekStart,
				ActivityID:    r.ActivityID,
				AllowsLessons: allows,
				Comment:       r.Comment,
			})
		}
		rows, err := repo.UpsertStudyCalendarWeeks(groupID, weeks)
		if err != nil {
			writeDBError(c, err)
			return
		}
		ver, _ := bumpScheduleVersionAndGet(repo)
		if pushSvc != nil {
			pushSvc.NotifyScheduleUpdatedAsync(groupID, ver)
		}
		writeAudit(c, repo, "upsert", "study_calendar_weeks", strconv.Itoa(groupID), gin.H{"count": len(req)})
		out := make([]studyCalendarWeekDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toStudyCalendarWeekDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handleAdminDeleteStudyCalendarWeek(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		if err := repo.DeleteStudyCalendarWeek(id); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "delete", "study_calendar_weeks", strconv.FormatInt(id, 10), nil)
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}

func handleAdminListTeacherDayConstraints(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		var filters schedule.TeacherDayConstraintFilters
		if v := c.Query("teacher_id"); v != "" {
			id, err := strconv.Atoi(v)
			if err != nil {
				writeValidationError(c, "teacher_id", "invalid teacher_id")
				return
			}
			filters.TeacherID = &id
		}
		if v := c.Query("date"); v != "" {
			d, err := time.Parse("2006-01-02", v)
			if err != nil {
				writeValidationError(c, "date", "date must be YYYY-MM-DD")
				return
			}
			filters.TargetDate = &d
		}
		var (
			rows []schedule.TeacherDayConstraint
			err  error
		)
		if p.Limit != nil {
			rows, err = repo.ListTeacherDayConstraintsPaged(filters, p.Limit, p.Offset)
		} else {
			rows, err = repo.ListTeacherDayConstraints(filters)
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]teacherDayConstraintDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toTeacherDayConstraintDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

type teacherDayConstraintReq struct {
	TeacherID     int    `json:"teacher_id"`
	Date          string `json:"date"`
	Reason        string `json:"reason"`
	AllowsLessons *bool  `json:"allows_lessons"`
}

func toTeacherDayConstraint(req teacherDayConstraintReq) (*schedule.TeacherDayConstraint, error) {
	d, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, err
	}
	allows := false
	if req.AllowsLessons != nil {
		allows = *req.AllowsLessons
	}
	return &schedule.TeacherDayConstraint{
		TeacherID:     req.TeacherID,
		TargetDate:    d,
		Reason:        req.Reason,
		AllowsLessons: allows,
	}, nil
}

func handleAdminCreateTeacherDayConstraint(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req teacherDayConstraintReq
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		row, err := toTeacherDayConstraint(req)
		if err != nil {
			writeValidationError(c, "date", "date must be YYYY-MM-DD")
			return
		}
		if err := repo.CreateTeacherDayConstraint(row); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "create", "teacher_day_constraints", strconv.FormatInt(row.ID, 10), row)
		c.JSON(http.StatusCreated, toTeacherDayConstraintDTO(*row))
	}
}

func handleAdminUpdateTeacherDayConstraint(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var req teacherDayConstraintReq
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		patch, err := toTeacherDayConstraint(req)
		if err != nil {
			writeValidationError(c, "date", "date must be YYYY-MM-DD")
			return
		}
		row, err := repo.UpdateTeacherDayConstraint(id, patch)
		if err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "update", "teacher_day_constraints", strconv.FormatInt(id, 10), row)
		c.JSON(http.StatusOK, toTeacherDayConstraintDTO(*row))
	}
}

func handleAdminDeleteTeacherDayConstraint(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		if err := repo.DeleteTeacherDayConstraint(id); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "delete", "teacher_day_constraints", strconv.FormatInt(id, 10), nil)
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}

func handleAdminListScheduleReplacements(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		var filters schedule.ScheduleReplacementFilters
		if v := c.Query("group_id"); v != "" {
			id, err := strconv.Atoi(v)
			if err != nil {
				writeValidationError(c, "group_id", "invalid group_id")
				return
			}
			filters.GroupID = &id
		}
		if v := c.Query("date"); v != "" {
			d, err := time.Parse("2006-01-02", v)
			if err != nil {
				writeValidationError(c, "date", "date must be YYYY-MM-DD")
				return
			}
			filters.TargetDate = &d
		}
		var (
			rows []schedule.ScheduleReplacement
			err  error
		)
		if p.Limit != nil {
			rows, err = repo.ListScheduleReplacementsPaged(filters, p.Limit, p.Offset)
		} else {
			rows, err = repo.ListScheduleReplacements(filters)
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]scheduleReplacementDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toScheduleReplacementDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

type scheduleReplacementReq struct {
	Date                  string  `json:"date"`
	GroupID               int     `json:"group_id"`
	PairNumber            int16   `json:"pair_number"`
	Subgroup              *int16  `json:"subgroup"`
	SourceSubjectID       *int    `json:"source_subject_id"`
	SourceLocationID      *int    `json:"source_location_id"`
	SourceTeacherID       *int    `json:"source_teacher_id"`
	ReplacementSubjectID  *int    `json:"replacement_subject_id"`
	ReplacementLocationID *int    `json:"replacement_location_id"`
	ReplacementTeacherID  *int    `json:"replacement_teacher_id"`
	Reason                *string `json:"reason"`
	ScheduleOverrideID    *int64  `json:"schedule_override_id"`
}

func toScheduleReplacement(req scheduleReplacementReq) (*schedule.ScheduleReplacement, error) {
	d, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, err
	}
	return &schedule.ScheduleReplacement{
		TargetDate:            d,
		GroupID:               req.GroupID,
		PairNumber:            req.PairNumber,
		Subgroup:              req.Subgroup,
		SourceSubjectID:       req.SourceSubjectID,
		SourceLocationID:      req.SourceLocationID,
		SourceTeacherID:       req.SourceTeacherID,
		ReplacementSubjectID:  req.ReplacementSubjectID,
		ReplacementLocationID: req.ReplacementLocationID,
		ReplacementTeacherID:  req.ReplacementTeacherID,
		Reason:                req.Reason,
		ScheduleOverrideID:    req.ScheduleOverrideID,
	}, nil
}

func handleAdminCreateScheduleReplacement(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req scheduleReplacementReq
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		row, err := toScheduleReplacement(req)
		if err != nil {
			writeValidationError(c, "date", "date must be YYYY-MM-DD")
			return
		}
		if err := repo.CreateScheduleReplacement(row); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "create", "schedule_replacements", strconv.FormatInt(row.ID, 10), row)
		c.JSON(http.StatusCreated, toScheduleReplacementDTO(*row))
	}
}

func handleAdminUpdateScheduleReplacement(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var req scheduleReplacementReq
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		patch, err := toScheduleReplacement(req)
		if err != nil {
			writeValidationError(c, "date", "date must be YYYY-MM-DD")
			return
		}
		row, err := repo.UpdateScheduleReplacement(id, patch)
		if err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "update", "schedule_replacements", strconv.FormatInt(id, 10), row)
		c.JSON(http.StatusOK, toScheduleReplacementDTO(*row))
	}
}

func handleAdminDeleteScheduleReplacement(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		if err := repo.DeleteScheduleReplacement(id); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "delete", "schedule_replacements", strconv.FormatInt(id, 10), nil)
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}

func handleAdminListLocationWeekAvailability(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		var filters schedule.LocationWeekAvailabilityFilters
		if v := c.Query("week_start_date"); v != "" {
			d, err := time.Parse("2006-01-02", v)
			if err != nil {
				writeValidationError(c, "week_start_date", "week_start_date must be YYYY-MM-DD")
				return
			}
			filters.WeekStartDate = &d
		}
		if v := c.Query("location_id"); v != "" {
			id, err := strconv.Atoi(v)
			if err != nil {
				writeValidationError(c, "location_id", "invalid location_id")
				return
			}
			filters.LocationID = &id
		}

		var (
			rows []schedule.LocationWeekAvailability
			err  error
		)
		if p.Limit != nil {
			rows, err = repo.ListLocationWeekAvailabilityPaged(filters, p.Limit, p.Offset)
		} else {
			rows, err = repo.ListLocationWeekAvailability(filters)
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]locationWeekAvailabilityDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toLocationWeekAvailabilityDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

type locationWeekAvailabilityReq struct {
	LocationID  int     `json:"location_id"`
	IsAvailable *bool   `json:"is_available"`
	Comment     *string `json:"comment"`
}

func handleAdminUpsertLocationWeekAvailability(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		weekStartStr := c.Query("week_start_date")
		if weekStartStr == "" {
			writeValidationError(c, "week_start_date", "week_start_date required")
			return
		}
		weekStart, err := time.Parse("2006-01-02", weekStartStr)
		if err != nil {
			writeValidationError(c, "week_start_date", "week_start_date must be YYYY-MM-DD")
			return
		}

		var req []locationWeekAvailabilityReq
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		rows := make([]schedule.LocationWeekAvailability, 0, len(req))
		for _, item := range req {
			available := true
			if item.IsAvailable != nil {
				available = *item.IsAvailable
			}
			rows = append(rows, schedule.LocationWeekAvailability{
				LocationID:  item.LocationID,
				IsAvailable: available,
				Comment:     item.Comment,
			})
		}

		saved, err := repo.UpsertLocationWeekAvailability(weekStart, rows)
		if err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "upsert", "location_week_availability", weekStart.Format("2006-01-02"), gin.H{"count": len(req)})
		out := make([]locationWeekAvailabilityDTO, 0, len(saved))
		for _, r := range saved {
			out = append(out, toLocationWeekAvailabilityDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handleAdminDeleteLocationWeekAvailability(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		if err := repo.DeleteLocationWeekAvailability(id); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "delete", "location_week_availability", strconv.FormatInt(id, 10), nil)
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}

type locationAutofillReq struct {
	GroupID        int     `json:"group_id"`
	StartDate      string  `json:"start_date"`
	EndDate        string  `json:"end_date"`
	Campus         *string `json:"campus"`
	LocationType   *string `json:"location_type_code"`
	ReplaceVirtual *bool   `json:"replace_virtual"`
	DryRun         bool    `json:"dry_run"`
	Comment        *string `json:"comment"`
}

func handleAdminAutofillLocations(svc *schedule.Service, repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if svc == nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "", "schedule service not configured")
			return
		}
		var req locationAutofillReq
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		start, err := time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			writeValidationError(c, "start_date", "start_date must be YYYY-MM-DD")
			return
		}
		end, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			writeValidationError(c, "end_date", "end_date must be YYYY-MM-DD")
			return
		}
		replaceVirtual := true
		if req.ReplaceVirtual != nil {
			replaceVirtual = *req.ReplaceVirtual
		}

		res, err := svc.AutofillLocations(schedule.LocationAutofillRequest{
			GroupID:        req.GroupID,
			StartDate:      start,
			EndDate:        end,
			CampusName:     req.Campus,
			LocationType:   req.LocationType,
			ReplaceVirtual: replaceVirtual,
			DryRun:         req.DryRun,
			Comment:        req.Comment,
		})
		if err != nil {
			writeDBError(c, err)
			return
		}
		if !req.DryRun && (res.Created > 0 || res.Updated > 0) && pushSvc != nil {
			if st, err := repo.GetSystemState(); err == nil {
				pushSvc.NotifyScheduleUpdatedAsync(req.GroupID, st.ScheduleVersion)
			}
		}
		writeAudit(c, repo, "autofill", "location_week_availability", strconv.Itoa(req.GroupID), gin.H{
			"assigned": res.Assigned,
			"created":  res.Created,
			"updated":  res.Updated,
			"dry_run":  res.DryRun,
		})
		c.JSON(http.StatusOK, res)
	}
}
