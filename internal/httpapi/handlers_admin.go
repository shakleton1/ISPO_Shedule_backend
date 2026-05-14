package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ispo-schedule/internal/auth"
	"ispo-schedule/internal/push"
	"ispo-schedule/internal/schedule"
)

// Exported handlers for integration tests

// HandleAdminCreateGroupForTest exports handleAdminCreateGroup for integration tests
func HandleAdminCreateGroupForTest(repo *schedule.Repository) gin.HandlerFunc {
	return handleAdminCreateGroup(repo, nil)
}

// HandleAdminListGroupsForTest exports handleAdminListGroups for integration tests
func HandleAdminListGroupsForTest(repo *schedule.Repository) gin.HandlerFunc {
	return handleAdminListGroups(repo)
}

// HandleAdminUpdateGroupForTest exports handleAdminUpdateGroup for integration tests
func HandleAdminUpdateGroupForTest(repo *schedule.Repository) gin.HandlerFunc {
	return handleAdminUpdateGroup(repo, nil)
}

// HandleAdminDeleteGroupForTest exports handleAdminDeleteGroup for integration tests
func HandleAdminDeleteGroupForTest(repo *schedule.Repository) gin.HandlerFunc {
	return handleAdminDeleteGroup(repo, nil)
}

// Day events

func handleAdminListDayEvents(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}

		var filters schedule.DayEventFilters
		if v := c.Query("group_id"); v != "" {
			id, err := strconv.Atoi(v)
			if err != nil {
				writeValidationError(c, "group_id", "invalid group_id")
				return
			}
			filters.GroupID = &id
		}
		if v := c.Query("target_date"); v != "" {
			d, err := time.Parse("2006-01-02", v)
			if err != nil {
				writeValidationError(c, "target_date", "target_date must be YYYY-MM-DD")
				return
			}
			filters.TargetDate = &d
		}
		var rows []schedule.ScheduleDayEvent
		var err error
		if page.Limit != nil {
			rows, err = repo.ListDayEventsPaged(filters, page.Limit, page.Offset)
		} else {
			rows, err = repo.ListDayEvents(filters)
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]scheduleDayEventDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toScheduleDayEventDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handleAdminCreateDayEvent(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.ScheduleDayEvent
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if req.GroupID <= 0 || req.TargetDate.IsZero() || req.EventType == "" || req.Title == "" {
			writeValidationError(c, "", "group_id, target_date, event_type, title required")
			return
		}
		req.EventType = strings.ToUpper(strings.TrimSpace(req.EventType))
		if err := repo.CreateDayEvent(&req); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "create", "schedule_day_events", strconv.FormatInt(req.ID, 10), req)
		c.JSON(http.StatusCreated, toScheduleDayEventDTO(req))
	}
}

func handleAdminUpdateDayEvent(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var req schedule.ScheduleDayEvent
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if req.GroupID <= 0 || req.TargetDate.IsZero() || req.EventType == "" || req.Title == "" {
			writeValidationError(c, "", "group_id, target_date, event_type, title required")
			return
		}
		req.EventType = strings.ToUpper(strings.TrimSpace(req.EventType))
		row, err := repo.UpdateDayEvent(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "update", "schedule_day_events", strconv.FormatInt(id, 10), row)
		c.JSON(http.StatusOK, toScheduleDayEventDTO(*row))
	}
}

func handleAdminDeleteDayEvent(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		if err := repo.DeleteDayEvent(id); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "delete", "schedule_day_events", strconv.FormatInt(id, 10), nil)
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}

func auditActorFromContext(c *gin.Context) (actorType string, actorID *int64, actorLogin *string, actorRole *string) {
	if v, ok := c.Get(ctxUserKey); ok {
		if u, ok := v.(*auth.User); ok && u != nil {
			actorType = "jwt"
			id := u.ID
			actorID = &id
			login := u.Login
			actorLogin = &login
			role := string(u.Role)
			actorRole = &role
			return
		}
	}
	if c.GetHeader("X-Admin-Key") != "" {
		return "admin_key", nil, nil, nil
	}
	return "unknown", nil, nil, nil
}

func writeAudit(c *gin.Context, repo *schedule.Repository, action, entityType, entityID string, payload any) {
	actorType, actorID, actorLogin, actorRole := auditActorFromContext(c)
	b := sanitizeAuditPayload(payload)
	rid := requestIDFromContext(c)
	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")
	var ridPtr, ipPtr, uaPtr *string
	if rid != "" {
		ridPtr = &rid
	}
	if ip != "" {
		ipPtr = &ip
	}
	if ua != "" {
		uaPtr = &ua
	}
	_ = repo.CreateAuditLog(&schedule.AuditLog{
		ActorType:  actorType,
		ActorID:    actorID,
		ActorLogin: actorLogin,
		ActorRole:  actorRole,
		Method:     c.Request.Method,
		Path:       c.FullPath(),
		RequestID:  ridPtr,
		IP:         ipPtr,
		UserAgent:  uaPtr,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Payload:    b,
	})
}

func bumpScheduleVersionAndGet(repo *schedule.Repository) (time.Time, error) {
	if err := repo.BumpScheduleVersion(); err != nil {
		return time.Time{}, err
	}
	st, err := repo.GetSystemState()
	if err != nil {
		return time.Time{}, err
	}
	return st.ScheduleVersion, nil
}

// Groups

func handleAdminListGroups(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		var rows []schedule.Group
		var err error
		if page.Limit != nil {
			rows, err = repo.ListGroupsPaged(page.Limit, page.Offset)
		} else {
			rows, err = repo.ListGroups()
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]groupDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toGroupDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handleAdminCreateGroup(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.Group
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if req.Name == "" || req.Course <= 0 {
			writeValidationError(c, "", "name and course required")
			return
		}
		if err := repo.CreateGroup(&req); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "create", "groups", strconv.Itoa(req.ID), req)
		c.JSON(http.StatusCreated, toGroupDTO(req))
	}
}

func handleAdminUpdateGroup(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var req schedule.Group
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		row, err := repo.UpdateGroup(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "update", "groups", strconv.Itoa(id), row)
		c.JSON(http.StatusOK, toGroupDTO(*row))
	}
}

func handleAdminDeleteGroup(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		if err := repo.DeleteGroup(id); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "delete", "groups", strconv.Itoa(id), nil)
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}

// Subjects

func handleAdminListSubjects(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		var rows []schedule.Subject
		var err error
		if page.Limit != nil {
			rows, err = repo.ListSubjectsPaged(page.Limit, page.Offset)
		} else {
			rows, err = repo.ListSubjects()
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]subjectDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toSubjectDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handleAdminCreateSubject(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.Subject
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if req.Name == "" {
			writeValidationError(c, "name", "name required")
			return
		}
		if err := repo.CreateSubject(&req); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "create", "subjects", strconv.Itoa(req.ID), req)
		c.JSON(http.StatusCreated, toSubjectDTO(req))
	}
}

func handleAdminUpdateSubject(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var req schedule.Subject
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		row, err := repo.UpdateSubject(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "update", "subjects", strconv.Itoa(id), row)
		c.JSON(http.StatusOK, toSubjectDTO(*row))
	}
}

func handleAdminDeleteSubject(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		if err := repo.DeleteSubject(id); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "delete", "subjects", strconv.Itoa(id), nil)
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}

// Locations

func handleAdminListLocations(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		var rows []schedule.Location
		var err error
		if page.Limit != nil {
			rows, err = repo.ListLocationsPaged(page.Limit, page.Offset)
		} else {
			rows, err = repo.ListLocations()
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]locationDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toLocationDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handleAdminCreateLocation(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.Location
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if req.Name == "" {
			writeValidationError(c, "name", "name required")
			return
		}
		if err := repo.CreateLocation(&req); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "create", "locations", strconv.Itoa(req.ID), req)
		c.JSON(http.StatusCreated, toLocationDTO(req))
	}
}

func handleAdminUpdateLocation(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var req schedule.Location
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		row, err := repo.UpdateLocation(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "update", "locations", strconv.Itoa(id), row)
		c.JSON(http.StatusOK, toLocationDTO(*row))
	}
}

func handleAdminDeleteLocation(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		if err := repo.DeleteLocation(id); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "delete", "locations", strconv.Itoa(id), nil)
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}

func handleAdminValidateSchedule(svc *schedule.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if svc == nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "", "schedule service not configured")
			return
		}
		groupID, err := strconv.Atoi(strings.TrimSpace(c.Query("group_id")))
		if err != nil || groupID <= 0 {
			writeValidationError(c, "group_id", "group_id required")
			return
		}
		startStr := strings.TrimSpace(c.Query("start_date"))
		endStr := strings.TrimSpace(c.Query("end_date"))
		if startStr == "" || endStr == "" {
			writeValidationError(c, "", "start_date and end_date required")
			return
		}
		start, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			writeValidationError(c, "start_date", "invalid start_date")
			return
		}
		end, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			writeValidationError(c, "end_date", "invalid end_date")
			return
		}
		if end.Before(start) {
			writeValidationError(c, "end_date", "end_date must be >= start_date")
			return
		}

		res, err := svc.ValidateScheduleRange(groupID, start, end)
		if err != nil {
			writeDBError(c, err)
			return
		}
		c.JSON(http.StatusOK, res)
	}
}

// Overlay

type adminOverlayRequest struct {
	GroupID     int    `json:"group_id"`
	Date        string `json:"date"`
	Text        string `json:"text"`
	StylePreset string `json:"style_preset"`
}

func handleAdminUpsertOverlay(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req adminOverlayRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		d, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			writeValidationError(c, "date", "invalid date")
			return
		}
		if req.GroupID <= 0 || req.Text == "" {
			writeValidationError(c, "", "group_id and text required")
			return
		}
		row, err := repo.UpsertOverlay(req.GroupID, d, req.Text, req.StylePreset)
		if err != nil {
			writeDBError(c, err)
			return
		}
		ver, _ := bumpScheduleVersionAndGet(repo)
		if pushSvc != nil {
			pushSvc.NotifyScheduleUpdatedAsync(req.GroupID, ver)
		}
		writeAudit(c, repo, "upsert", "schedule_day_overlays", strconv.FormatInt(row.ID, 10), row)
		c.JSON(http.StatusOK, toScheduleDayOverlayDTO(*row))
	}
}
