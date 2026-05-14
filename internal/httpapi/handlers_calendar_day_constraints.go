package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ispo-schedule/internal/push"
	"ispo-schedule/internal/schedule"
)

type calendarDayConstraintReq struct {
	TargetDate           string  `json:"target_date"`
	Title                string  `json:"title"`
	Reason               *string `json:"reason"`
	ConstraintType       string  `json:"constraint_type"`
	AffectsLessons       *bool   `json:"affects_lessons"`
	RequiresConfirmation *bool   `json:"requires_confirmation"`
	StylePreset          string  `json:"style_preset"`
}

func handleAdminListCalendarDayConstraints(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		dateFrom, ok := parseQueryDate(c, "date_from")
		if !ok {
			return
		}
		dateTo, ok := parseQueryDate(c, "date_to")
		if !ok {
			return
		}
		var constraintType *string
		if v := strings.TrimSpace(c.Query("constraint_type")); v != "" {
			constraintType = &v
		}
		affectsLessons, ok := parseOptionalQueryBool(c, "affects_lessons")
		if !ok {
			return
		}

		filters := schedule.CalendarDayConstraintFilters{
			DateFrom:       dateFrom,
			DateTo:         dateTo,
			ConstraintType: constraintType,
			AffectsLessons: affectsLessons,
		}
		var (
			rows []schedule.CalendarDayConstraint
			err  error
		)
		if page.Limit != nil {
			rows, err = repo.ListCalendarDayConstraintsPaged(filters, page.Limit, page.Offset)
		} else {
			rows, err = repo.ListCalendarDayConstraints(filters)
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]calendarDayConstraintDTO, 0, len(rows))
		for _, row := range rows {
			out = append(out, toCalendarDayConstraintDTO(row))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handleAdminCreateCalendarDayConstraint(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req calendarDayConstraintReq
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		row, err := toCalendarDayConstraint(req)
		if err != nil {
			writeValidationError(c, "", err.Error())
			return
		}
		if err := repo.CreateCalendarDayConstraint(row); err != nil {
			writeDBError(c, err)
			return
		}
		ver, _ := bumpScheduleVersionAndGet(repo)
		if pushSvc != nil {
			pushSvc.NotifyScheduleUpdatedAllAsync(ver)
		}
		writeAudit(c, repo, "create", "calendar_day_constraints", strconv.FormatInt(row.ID, 10), row)
		c.JSON(http.StatusCreated, toCalendarDayConstraintDTO(*row))
	}
}

func handleAdminUpdateCalendarDayConstraint(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var req calendarDayConstraintReq
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		patch, err := toCalendarDayConstraint(req)
		if err != nil {
			writeValidationError(c, "", err.Error())
			return
		}
		row, err := repo.UpdateCalendarDayConstraint(id, patch)
		if err != nil {
			writeDBError(c, err)
			return
		}
		ver, _ := bumpScheduleVersionAndGet(repo)
		if pushSvc != nil {
			pushSvc.NotifyScheduleUpdatedAllAsync(ver)
		}
		writeAudit(c, repo, "update", "calendar_day_constraints", strconv.FormatInt(id, 10), row)
		c.JSON(http.StatusOK, toCalendarDayConstraintDTO(*row))
	}
}

func handleAdminDeleteCalendarDayConstraint(repo *schedule.Repository, pushSvc *push.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		if err := repo.DeleteCalendarDayConstraint(id); err != nil {
			writeDBError(c, err)
			return
		}
		ver, _ := bumpScheduleVersionAndGet(repo)
		if pushSvc != nil {
			pushSvc.NotifyScheduleUpdatedAllAsync(ver)
		}
		writeAudit(c, repo, "delete", "calendar_day_constraints", strconv.FormatInt(id, 10), nil)
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}

func toCalendarDayConstraint(req calendarDayConstraintReq) (*schedule.CalendarDayConstraint, error) {
	d, err := time.Parse("2006-01-02", strings.TrimSpace(req.TargetDate))
	if err != nil {
		return nil, err
	}

	affects := true
	if req.AffectsLessons != nil {
		affects = *req.AffectsLessons
	}
	requires := false
	if req.RequiresConfirmation != nil {
		requires = *req.RequiresConfirmation
	}

	return &schedule.CalendarDayConstraint{
		TargetDate:           d,
		Title:                req.Title,
		Reason:               req.Reason,
		ConstraintType:       req.ConstraintType,
		AffectsLessons:       affects,
		RequiresConfirmation: requires,
		StylePreset:          req.StylePreset,
	}, nil
}

func parseOptionalQueryBool(c *gin.Context, name string) (*bool, bool) {
	v := strings.TrimSpace(c.Query(name))
	if v == "" {
		return nil, true
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		writeValidationError(c, name, "invalid "+name)
		return nil, false
	}
	return &parsed, true
}
