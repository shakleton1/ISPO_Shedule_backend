package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

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

// Bulk overrides

type adminBulkOverridesRequest struct {
	GroupID          int     `json:"group_id"`
	StartDate        string  `json:"start_date"`
	EndDate          string  `json:"end_date"`
	DaysOfWeek       []int16 `json:"days_of_week"`
	PairNumber       *int16  `json:"pair_number"`
	PairNumbers      []int16 `json:"pair_numbers"`
	ActionType       string  `json:"action_type"`
	NewSubjectID     *int    `json:"new_subject_id"`
	NewLocationID    *int    `json:"new_location_id"`
	NewTeacherName   *string `json:"new_teacher_name"`
	Comment          *string `json:"comment"`
	Subgroup         *int16  `json:"subgroup"`
	OnlyTeachingDays *bool   `json:"only_teaching_days"`
	OnConflict       string  `json:"on_conflict"` // error|skip
}

func handleAdminBulkOverrides(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req adminBulkOverridesRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if req.GroupID <= 0 || req.StartDate == "" || req.EndDate == "" || req.ActionType == "" {
			writeValidationError(c, "", "group_id, start_date, end_date, action_type required")
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
		if end.Before(start) {
			writeValidationError(c, "end_date", "end_date before start_date")
			return
		}

		pairNums := make([]int16, 0)
		pairNums = append(pairNums, req.PairNumbers...)
		if req.PairNumber != nil {
			pairNums = append(pairNums, *req.PairNumber)
		}
		if len(pairNums) == 0 {
			writeValidationError(c, "", "pair_number or pair_numbers required")
			return
		}

		onConflict := strings.ToLower(strings.TrimSpace(req.OnConflict))
		if onConflict == "" {
			onConflict = "error"
		}
		if onConflict != "error" && onConflict != "skip" {
			writeValidationError(c, "on_conflict", "on_conflict must be error or skip")
			return
		}

		onlyTeaching := true
		if req.OnlyTeachingDays != nil {
			onlyTeaching = *req.OnlyTeachingDays
		}

		created := 0
		skipped := 0

		// preload calendar exceptions for DoW mapping
		exceptions, err := repo.ListCalendarExceptionsBetween(start, end)
		if err != nil {
			writeDBError(c, err)
			return
		}
		worksAs := map[string]int16{}
		for _, e := range exceptions {
			worksAs[e.TargetDate.Format("2006-01-02")] = e.WorksAsDay
		}

		teachingWeeks := map[string]bool{}
		if onlyTeaching {
			m, err := repo.ListTeachingWeeksForGroupBetween(req.GroupID, start, end)
			if err != nil {
				writeDBError(c, err)
				return
			}
			teachingWeeks = m
		}

		dowAllowed := map[int16]bool{}
		if len(req.DaysOfWeek) > 0 {
			for _, d := range req.DaysOfWeek {
				dowAllowed[d] = true
			}
		}

		err = repo.DB().Transaction(func(tx *gorm.DB) error {
			txRepo := schedule.NewRepository(tx)
			for d := scheduleDateOnly(start); !d.After(end); d = d.AddDate(0, 0, 1) {
				dayKey := d.Format("2006-01-02")
				dow := scheduleDayOfWeekForDate(d, worksAs)
				if len(dowAllowed) > 0 && !dowAllowed[dow] {
					continue
				}
				if onlyTeaching {
					wk := scheduleMondayOfWeek(d).Format("2006-01-02")
					if v, ok := teachingWeeks[wk]; ok && !v {
						continue
					}
				}

				for _, pn := range pairNums {
					exists, err := txRepo.HasAnyOverrideForSlot(req.GroupID, d, pn, req.Subgroup)
					if err != nil {
						return err
					}
					if exists {
						if onConflict == "skip" {
							skipped++
							continue
						}
						return fmt.Errorf("override already exists for %s pair %d", dayKey, pn)
					}
					o := &schedule.ScheduleOverride{
						TargetDate:     d,
						GroupID:        req.GroupID,
						PairNumber:     pn,
						ActionType:     schedule.OverrideAction(strings.ToUpper(strings.TrimSpace(req.ActionType))),
						NewSubjectID:   req.NewSubjectID,
						NewLocationID:  req.NewLocationID,
						NewTeacherName: req.NewTeacherName,
						Comment:        req.Comment,
						Subgroup:       req.Subgroup,
					}
					if err := txRepo.CreateOverride(o); err != nil {
						return err
					}
					created++
				}
			}
			return nil
		})
		if err != nil {
			writeDBError(c, err)
			return
		}

		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "bulk_create", "schedule_overrides", strconv.Itoa(req.GroupID), gin.H{"created": created, "skipped": skipped})
		c.JSON(http.StatusOK, gin.H{"created": created, "skipped": skipped})
	}
}

// Move pair (atomically: CANCEL + ADD)

type adminMovePairRequest struct {
	GroupID    int     `json:"group_id"`
	TargetDate string  `json:"target_date"`
	FromPair   int16   `json:"from_pair_number"`
	ToPair     int16   `json:"to_pair_number"`
	Subgroup   *int16  `json:"subgroup"`
	Comment    *string `json:"comment"`
}

func handleAdminMovePair(svc *schedule.Service, repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req adminMovePairRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if req.GroupID <= 0 || req.TargetDate == "" || req.FromPair <= 0 || req.ToPair <= 0 {
			writeValidationError(c, "", "group_id, target_date, from_pair_number, to_pair_number required")
			return
		}
		date, err := time.Parse("2006-01-02", req.TargetDate)
		if err != nil {
			writeValidationError(c, "target_date", "target_date must be YYYY-MM-DD")
			return
		}
		resp, err := svc.GetRange(req.GroupID, date, date)
		if err != nil {
			writeDBError(c, err)
			return
		}
		if len(resp.Days) != 1 {
			writeError(c, http.StatusBadRequest, "bad_request", "", "no schedule for date")
			return
		}
		day := resp.Days[0]

		var found *schedule.Lesson
		for i := range day.Lessons {
			l := day.Lessons[i]
			if l.PairNumber != req.FromPair {
				continue
			}
			if req.Subgroup != nil {
				if l.Subgroup == nil || *l.Subgroup != *req.Subgroup {
					continue
				}
			} else {
				if l.Subgroup != nil {
					continue
				}
			}
			found = &l
			break
		}
		if found == nil {
			writeError(c, http.StatusBadRequest, "bad_request", "", "source lesson not found")
			return
		}
		if found.SubjectID == nil || found.LocationID == nil {
			writeError(c, http.StatusBadRequest, "bad_request", "", "source lesson has no subject/location ids")
			return
		}

		// Be conservative: if any overrides already exist for these slots, refuse.
		dateOnly := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
		existsFrom, err := repo.HasAnyOverrideForSlot(req.GroupID, dateOnly, req.FromPair, req.Subgroup)
		if err != nil {
			writeDBError(c, err)
			return
		}
		existsTo, err := repo.HasAnyOverrideForSlot(req.GroupID, dateOnly, req.ToPair, req.Subgroup)
		if err != nil {
			writeDBError(c, err)
			return
		}
		if existsFrom || existsTo {
			writeError(c, http.StatusConflict, "conflict", "", "overrides already exist for from/to slot")
			return
		}

		err = repo.DB().Transaction(func(tx *gorm.DB) error {
			txRepo := schedule.NewRepository(tx)
			// CANCEL at source
			cancel := &schedule.ScheduleOverride{
				TargetDate: dateOnly,
				GroupID:    req.GroupID,
				PairNumber: req.FromPair,
				ActionType: schedule.OverrideCancel,
				Subgroup:   req.Subgroup,
				Comment:    req.Comment,
			}
			if err := txRepo.CreateOverride(cancel); err != nil {
				return err
			}
			// ADD at destination
			add := &schedule.ScheduleOverride{
				TargetDate:     dateOnly,
				GroupID:        req.GroupID,
				PairNumber:     req.ToPair,
				ActionType:     schedule.OverrideAdd,
				NewSubjectID:   found.SubjectID,
				NewLocationID:  found.LocationID,
				NewTeacherName: &found.TeacherName,
				Subgroup:       req.Subgroup,
				Comment:        req.Comment,
			}
			if strings.TrimSpace(found.TeacherName) == "" {
				add.NewTeacherName = nil
			}
			if err := txRepo.CreateOverride(add); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			writeDBError(c, err)
			return
		}
		_ = repo.BumpScheduleVersion()
		writeAudit(c, repo, "move_pair", "schedule_overrides", strconv.Itoa(req.GroupID), req)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

// helpers for bulk endpoint (keep local to httpapi)

func scheduleDateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func scheduleMondayOfWeek(t time.Time) time.Time {
	d := scheduleDateOnly(t)
	wd := int(d.Weekday())
	offset := (wd + 6) % 7
	return d.AddDate(0, 0, -offset)
}

func scheduleDayOfWeekForDate(d time.Time, worksAs map[string]int16) int16 {
	dayKey := scheduleDateOnly(d).Format("2006-01-02")
	dayOfWeek := int16((int(d.Weekday()) + 6) % 7)
	if v, ok := worksAs[dayKey]; ok {
		return v
	}
	return dayOfWeek
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

// Templates

func handleAdminListTemplates(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}

		var filters schedule.TemplateFilters
		if v := c.Query("group_id"); v != "" {
			i, err := strconv.Atoi(v)
			if err != nil {
				writeValidationError(c, "group_id", "invalid group_id")
				return
			}
			filters.GroupID = &i
		}
		if v := c.Query("day_of_week"); v != "" {
			i, err := strconv.Atoi(v)
			if err != nil {
				writeValidationError(c, "day_of_week", "invalid day_of_week")
				return
			}
			d := int16(i)
			filters.DayOfWeek = &d
		}
		if v := c.Query("week_parity"); v != "" {
			p := schedule.WeekParity(v)
			filters.WeekParity = &p
		}
		if v := c.Query("status"); v != "" {
			s := schedule.EntityStatus(v)
			filters.Status = &s
		}
		var rows []schedule.ScheduleTemplate
		var err error
		if page.Limit != nil {
			rows, err = repo.ListTemplatesPaged(filters, page.Limit, page.Offset)
		} else {
			rows, err = repo.ListTemplates(filters)
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]scheduleTemplateDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toScheduleTemplateDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handleAdminCreateTemplate(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.ScheduleTemplate
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if req.GroupID <= 0 || req.SubjectID <= 0 || req.LocationID <= 0 || req.PairNumber <= 0 {
			writeValidationError(c, "", "group_id, subject_id, location_id, pair_number required")
			return
		}
		if err := repo.CreateTemplate(&req); err != nil {
			writeDBError(c, err)
			return
		}
		if req.Status == schedule.StatusPublished {
			ver, _ := bumpScheduleVersionAndGet(repo)
			if pushSvc != nil {
				pushSvc.NotifyScheduleUpdatedAsync(req.GroupID, ver)
			}
		}
		writeAudit(c, repo, "create", "schedule_templates", strconv.FormatInt(req.ID, 10), req)
		c.JSON(http.StatusCreated, toScheduleTemplateDTO(req))
	}
}

func handleAdminUpdateTemplate(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var req schedule.ScheduleTemplate
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		row, err := repo.UpdateTemplate(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		if row.Status == schedule.StatusPublished {
			ver, _ := bumpScheduleVersionAndGet(repo)
			if pushSvc != nil {
				pushSvc.NotifyScheduleUpdatedAsync(row.GroupID, ver)
			}
		}
		writeAudit(c, repo, "update", "schedule_templates", strconv.FormatInt(id, 10), row)
		c.JSON(http.StatusOK, toScheduleTemplateDTO(*row))
	}
}

func handleAdminDeleteTemplate(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		tpl, err := repo.GetTemplateByID(id)
		if err != nil {
			writeDBError(c, err)
			return
		}
		if err := repo.DeleteTemplate(id); err != nil {
			writeDBError(c, err)
			return
		}
		if tpl.Status == schedule.StatusPublished {
			ver, _ := bumpScheduleVersionAndGet(repo)
			if pushSvc != nil {
				pushSvc.NotifyScheduleUpdatedAsync(tpl.GroupID, ver)
			}
		}
		writeAudit(c, repo, "delete", "schedule_templates", strconv.FormatInt(id, 10), nil)
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}

func handleAdminPublishDraftTemplates(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID, err := strconv.Atoi(c.Query("group_id"))
		if err != nil || groupID <= 0 {
			writeValidationError(c, "group_id", "group_id required")
			return
		}
		moved, err := repo.PublishDraftTemplates(groupID)
		if err != nil {
			writeDBError(c, err)
			return
		}
		var ver time.Time
		if moved > 0 {
			ver, _ = bumpScheduleVersionAndGet(repo)
			if pushSvc != nil {
				pushSvc.NotifyScheduleUpdatedAsync(groupID, ver)
			}
		}
		writeAudit(c, repo, "publish", "schedule_templates", strconv.Itoa(groupID), gin.H{"group_id": groupID, "moved": moved})
		c.JSON(http.StatusOK, gin.H{"group_id": groupID, "moved": moved, "data_version": ver.UTC().Format(time.RFC3339)})
	}
}

func handleAdminDiscardDraftTemplates(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID, err := strconv.Atoi(c.Query("group_id"))
		if err != nil || groupID <= 0 {
			writeValidationError(c, "group_id", "group_id required")
			return
		}
		deleted, err := repo.DiscardDraftTemplates(groupID)
		if err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "discard_drafts", "schedule_templates", strconv.Itoa(groupID), gin.H{"group_id": groupID, "deleted": deleted})
		c.JSON(http.StatusOK, gin.H{"group_id": groupID, "deleted": deleted})
	}
}

// Overrides

func handleAdminListOverrides(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}

		var filters schedule.OverrideFilters
		if v := c.Query("group_id"); v != "" {
			i, err := strconv.Atoi(v)
			if err != nil {
				writeValidationError(c, "group_id", "invalid group_id")
				return
			}
			filters.GroupID = &i
		}
		if v := c.Query("date"); v != "" {
			d, err := time.Parse("2006-01-02", v)
			if err != nil {
				writeValidationError(c, "date", "invalid date")
				return
			}
			filters.TargetDate = &d
		}
		var rows []schedule.ScheduleOverride
		var err error
		if page.Limit != nil {
			rows, err = repo.ListOverridesPaged(filters, page.Limit, page.Offset)
		} else {
			rows, err = repo.ListOverrides(filters)
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]scheduleOverrideDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toScheduleOverrideDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

type adminOverrideRequest struct {
	GroupID          int     `json:"group_id"`
	Date             string  `json:"date"`
	Pair             int16   `json:"pair"`
	Action           string  `json:"action"`
	NewSubjectID     *int    `json:"new_subject_id"`
	NewLocationID    *int    `json:"new_location_id"`
	NewTeacherManual bool    `json:"new_teacher_manual"`
	NewTeacherName   *string `json:"new_teacher_name"`
	Comment          *string `json:"comment"`
	Subgroup         *int16  `json:"subgroup"`
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
		if o.NewSubjectID != nil || o.NewLocationID != nil || o.NewTeacherName != nil || o.NewTeacherManual {
			return errors.New("CANCEL must not include new_* fields")
		}
	case schedule.OverrideAdd:
		if o.NewSubjectID == nil {
			return errors.New("ADD requires new_subject_id")
		}
	case schedule.OverrideReplace:
		if o.NewSubjectID == nil && o.NewLocationID == nil && o.NewTeacherName == nil && o.Comment == nil && !o.NewTeacherManual {
			return errors.New("REPLACE requires at least one change field")
		}
	default:
		return errors.New("invalid action")
	}
	return nil
}

func handleAdminCreateOverride(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req adminOverrideRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		d, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			writeValidationError(c, "date", "invalid date")
			return
		}
		o := schedule.ScheduleOverride{
			GroupID:          req.GroupID,
			TargetDate:       d,
			PairNumber:       req.Pair,
			ActionType:       schedule.OverrideAction(req.Action),
			NewSubjectID:     req.NewSubjectID,
			NewLocationID:    req.NewLocationID,
			NewTeacherManual: req.NewTeacherManual,
			NewTeacherName:   req.NewTeacherName,
			Comment:          req.Comment,
			Subgroup:         req.Subgroup,
		}
		if err := validateOverrideRequest(o); err != nil {
			writeValidationError(c, "", err.Error())
			return
		}
		if err := repo.CreateOverride(&o); err != nil {
			writeDBError(c, err)
			return
		}
		ver, _ := bumpScheduleVersionAndGet(repo)
		if pushSvc != nil {
			pushSvc.NotifyScheduleUpdatedAsync(o.GroupID, ver)
		}
		writeAudit(c, repo, "create", "schedule_overrides", strconv.FormatInt(o.ID, 10), o)
		c.JSON(http.StatusCreated, toScheduleOverrideDTO(o))
	}
}

func handleAdminUpdateOverride(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var req schedule.ScheduleOverride
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if err := validateOverrideRequest(req); err != nil {
			writeValidationError(c, "", err.Error())
			return
		}
		row, err := repo.UpdateOverride(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		ver, _ := bumpScheduleVersionAndGet(repo)
		if pushSvc != nil {
			pushSvc.NotifyScheduleUpdatedAsync(row.GroupID, ver)
		}
		writeAudit(c, repo, "update", "schedule_overrides", strconv.FormatInt(id, 10), row)
		c.JSON(http.StatusOK, toScheduleOverrideDTO(*row))
	}
}

func handleAdminDeleteOverride(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		o, err := repo.GetOverrideByID(id)
		if err != nil {
			writeDBError(c, err)
			return
		}
		if err := repo.DeleteOverride(id); err != nil {
			writeDBError(c, err)
			return
		}
		ver, _ := bumpScheduleVersionAndGet(repo)
		if pushSvc != nil {
			pushSvc.NotifyScheduleUpdatedAsync(o.GroupID, ver)
		}
		writeAudit(c, repo, "delete", "schedule_overrides", strconv.FormatInt(id, 10), nil)
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

// Calendar exceptions

func handleAdminListCalendarExceptions(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		var rows []schedule.CalendarException
		var err error
		if page.Limit != nil {
			rows, err = repo.ListCalendarExceptionsPaged(page.Limit, page.Offset)
		} else {
			rows, err = repo.ListCalendarExceptions()
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]calendarExceptionDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toCalendarExceptionDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

type adminCalendarExceptionRequest struct {
	Date       string  `json:"date"`
	WorksAsDay int16   `json:"works_as_day"`
	Comment    *string `json:"comment"`
}

func handleAdminUpsertCalendarException(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req adminCalendarExceptionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		d, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			writeValidationError(c, "date", "invalid date")
			return
		}
		row, err := repo.UpsertCalendarException(d, req.WorksAsDay, req.Comment)
		if err != nil {
			writeDBError(c, err)
			return
		}
		ver, _ := bumpScheduleVersionAndGet(repo)
		if pushSvc != nil {
			pushSvc.NotifyScheduleUpdatedAllAsync(ver)
		}
		writeAudit(c, repo, "upsert", "calendar_exceptions", strconv.FormatInt(row.ID, 10), row)
		c.JSON(http.StatusOK, toCalendarExceptionDTO(*row))
	}
}

func handleAdminDeleteCalendarException(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		dateStr := c.Param("date")
		if dateStr == "" {
			writeValidationError(c, "date", "date required")
			return
		}
		if err := repo.DeleteCalendarExceptionByDate(dateStr); err != nil {
			writeDBError(c, err)
			return
		}
		ver, _ := bumpScheduleVersionAndGet(repo)
		if pushSvc != nil {
			pushSvc.NotifyScheduleUpdatedAllAsync(ver)
		}
		writeAudit(c, repo, "delete", "calendar_exceptions", dateStr, nil)
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}
