package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ispo-schedule/internal/auth"
	"ispo-schedule/internal/schedule"
)

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
	var b []byte
	if payload != nil {
		b, _ = json.Marshal(payload)
	}
	_ = repo.CreateAuditLog(&schedule.AuditLog{
		ActorType:  actorType,
		ActorID:    actorID,
		ActorLogin: actorLogin,
		ActorRole:  actorRole,
		Method:     c.Request.Method,
		Path:       c.FullPath(),
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Payload:    b,
	})
}

func writeDBError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

// Groups

func handleAdminListGroups(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := repo.ListGroups()
		if err != nil {
			writeDBError(c, err)
			return
		}
		c.JSON(http.StatusOK, rows)
	}
}

func handleAdminCreateGroup(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.Group
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if req.Name == "" || req.Course <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name and course required"})
			return
		}
		if err := repo.CreateGroup(&req); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "create", "groups", strconv.Itoa(req.ID), req)
		c.JSON(http.StatusCreated, req)
	}
}

func handleAdminUpdateGroup(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req schedule.Group
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		row, err := repo.UpdateGroup(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "update", "groups", strconv.Itoa(id), row)
		c.JSON(http.StatusOK, row)
	}
}

func handleAdminDeleteGroup(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := repo.DeleteGroup(id); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "delete", "groups", strconv.Itoa(id), nil)
		c.Status(http.StatusNoContent)
	}
}

// Subjects

func handleAdminListSubjects(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := repo.ListSubjects()
		if err != nil {
			writeDBError(c, err)
			return
		}
		c.JSON(http.StatusOK, rows)
	}
}

func handleAdminCreateSubject(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.Subject
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if req.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
			return
		}
		if err := repo.CreateSubject(&req); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "create", "subjects", strconv.Itoa(req.ID), req)
		c.JSON(http.StatusCreated, req)
	}
}

func handleAdminUpdateSubject(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req schedule.Subject
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		row, err := repo.UpdateSubject(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "update", "subjects", strconv.Itoa(id), row)
		c.JSON(http.StatusOK, row)
	}
}

func handleAdminDeleteSubject(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := repo.DeleteSubject(id); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "delete", "subjects", strconv.Itoa(id), nil)
		c.Status(http.StatusNoContent)
	}
}

// Locations

func handleAdminListLocations(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := repo.ListLocations()
		if err != nil {
			writeDBError(c, err)
			return
		}
		c.JSON(http.StatusOK, rows)
	}
}

func handleAdminCreateLocation(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.Location
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if req.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
			return
		}
		if err := repo.CreateLocation(&req); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "create", "locations", strconv.Itoa(req.ID), req)
		c.JSON(http.StatusCreated, req)
	}
}

func handleAdminUpdateLocation(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req schedule.Location
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		row, err := repo.UpdateLocation(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "update", "locations", strconv.Itoa(id), row)
		c.JSON(http.StatusOK, row)
	}
}

func handleAdminDeleteLocation(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := repo.DeleteLocation(id); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "delete", "locations", strconv.Itoa(id), nil)
		c.Status(http.StatusNoContent)
	}
}

// Templates

func handleAdminListTemplates(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var filters schedule.TemplateFilters
		if v := c.Query("group_id"); v != "" {
			i, err := strconv.Atoi(v)
			if err == nil {
				filters.GroupID = &i
			}
		}
		if v := c.Query("day_of_week"); v != "" {
			i, err := strconv.Atoi(v)
			if err == nil {
				d := int16(i)
				filters.DayOfWeek = &d
			}
		}
		if v := c.Query("week_parity"); v != "" {
			p := schedule.WeekParity(v)
			filters.WeekParity = &p
		}
		rows, err := repo.ListTemplates(filters)
		if err != nil {
			writeDBError(c, err)
			return
		}
		c.JSON(http.StatusOK, rows)
	}
}

func handleAdminCreateTemplate(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.ScheduleTemplate
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if req.GroupID <= 0 || req.SubjectID <= 0 || req.LocationID <= 0 || req.PairNumber <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "group_id, subject_id, location_id, pair_number required"})
			return
		}
		if err := repo.CreateTemplate(&req); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "create", "schedule_templates", strconv.FormatInt(req.ID, 10), req)
		c.JSON(http.StatusCreated, req)
	}
}

func handleAdminUpdateTemplate(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req schedule.ScheduleTemplate
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		row, err := repo.UpdateTemplate(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "update", "schedule_templates", strconv.FormatInt(id, 10), row)
		c.JSON(http.StatusOK, row)
	}
}

func handleAdminDeleteTemplate(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := repo.DeleteTemplate(id); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "delete", "schedule_templates", strconv.FormatInt(id, 10), nil)
		c.Status(http.StatusNoContent)
	}
}

// Overrides

func handleAdminListOverrides(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var filters schedule.OverrideFilters
		if v := c.Query("group_id"); v != "" {
			i, err := strconv.Atoi(v)
			if err == nil {
				filters.GroupID = &i
			}
		}
		if v := c.Query("date"); v != "" {
			d, err := time.Parse("2006-01-02", v)
			if err == nil {
				filters.TargetDate = &d
			}
		}
		rows, err := repo.ListOverrides(filters)
		if err != nil {
			writeDBError(c, err)
			return
		}
		c.JSON(http.StatusOK, rows)
	}
}

type adminOverrideRequest struct {
	GroupID        int     `json:"group_id"`
	Date           string  `json:"date"`
	Pair           int16   `json:"pair"`
	Action         string  `json:"action"`
	NewSubjectID   *int    `json:"new_subject_id"`
	NewLocationID  *int    `json:"new_location_id"`
	NewTeacherName *string `json:"new_teacher_name"`
	Comment        *string `json:"comment"`
	Subgroup       *int16  `json:"subgroup"`
}

func validateOverrideRequest(o schedule.ScheduleOverride) error {
	if o.GroupID <= 0 {
		return errors.New("group_id required")
	}
	if o.PairNumber < 1 || o.PairNumber > 8 {
		return errors.New("pair must be 1..8")
	}
	if o.Subgroup != nil && (*o.Subgroup < 1 || *o.Subgroup > 2) {
		return errors.New("subgroup must be 1 or 2")
	}
	switch o.ActionType {
	case schedule.OverrideCancel:
		if o.NewSubjectID != nil || o.NewLocationID != nil || o.NewTeacherName != nil {
			return errors.New("CANCEL must not include new_* fields")
		}
	case schedule.OverrideAdd:
		if o.NewSubjectID == nil || o.NewLocationID == nil {
			return errors.New("ADD requires new_subject_id and new_location_id")
		}
	case schedule.OverrideReplace:
		if o.NewSubjectID == nil && o.NewLocationID == nil && o.NewTeacherName == nil && o.Comment == nil {
			return errors.New("REPLACE requires at least one change field")
		}
	default:
		return errors.New("invalid action")
	}
	return nil
}

func handleAdminCreateOverride(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req adminOverrideRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		d, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date"})
			return
		}
		o := schedule.ScheduleOverride{
			GroupID:        req.GroupID,
			TargetDate:     d,
			PairNumber:     req.Pair,
			ActionType:     schedule.OverrideAction(req.Action),
			NewSubjectID:   req.NewSubjectID,
			NewLocationID:  req.NewLocationID,
			NewTeacherName: req.NewTeacherName,
			Comment:        req.Comment,
			Subgroup:       req.Subgroup,
		}
		if err := validateOverrideRequest(o); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := repo.CreateOverride(&o); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "create", "schedule_overrides", strconv.FormatInt(o.ID, 10), o)
		c.JSON(http.StatusCreated, o)
	}
}

func handleAdminUpdateOverride(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req schedule.ScheduleOverride
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if err := validateOverrideRequest(req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		row, err := repo.UpdateOverride(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "update", "schedule_overrides", strconv.FormatInt(id, 10), row)
		c.JSON(http.StatusOK, row)
	}
}

func handleAdminDeleteOverride(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := repo.DeleteOverride(id); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "delete", "schedule_overrides", strconv.FormatInt(id, 10), nil)
		c.Status(http.StatusNoContent)
	}
}

// Overlay

type adminOverlayRequest struct {
	GroupID     int    `json:"group_id"`
	Date        string `json:"date"`
	Text        string `json:"text"`
	StylePreset string `json:"style_preset"`
}

func handleAdminUpsertOverlay(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req adminOverlayRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		d, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date"})
			return
		}
		if req.GroupID <= 0 || req.Text == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "group_id and text required"})
			return
		}
		row, err := repo.UpsertOverlay(req.GroupID, d, req.Text, req.StylePreset)
		if err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "upsert", "schedule_day_overlays", strconv.FormatInt(row.ID, 10), row)
		c.JSON(http.StatusOK, row)
	}
}

// Calendar exceptions

func handleAdminListCalendarExceptions(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := repo.ListCalendarExceptions()
		if err != nil {
			writeDBError(c, err)
			return
		}
		c.JSON(http.StatusOK, rows)
	}
}

type adminCalendarExceptionRequest struct {
	Date       string  `json:"date"`
	WorksAsDay int16   `json:"works_as_day"`
	Comment    *string `json:"comment"`
}

func handleAdminUpsertCalendarException(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req adminCalendarExceptionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		d, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date"})
			return
		}
		row, err := repo.UpsertCalendarException(d, req.WorksAsDay, req.Comment)
		if err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "upsert", "calendar_exceptions", strconv.FormatInt(row.ID, 10), row)
		c.JSON(http.StatusOK, row)
	}
}

func handleAdminDeleteCalendarException(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		dateStr := c.Param("date")
		if dateStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "date required"})
			return
		}
		if err := repo.DeleteCalendarExceptionByDate(dateStr); err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "delete", "calendar_exceptions", dateStr, nil)
		c.Status(http.StatusNoContent)
	}
}
