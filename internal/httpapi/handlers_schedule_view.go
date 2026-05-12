package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ispo-schedule/internal/schedule"
)

func handleAdminScheduleView(svc *schedule.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		start, end, ok := parseScheduleViewDateRange(c)
		if !ok {
			return
		}

		filter := schedule.ScheduleViewFilter{Scope: c.DefaultQuery("scope", "group")}
		if v := c.Query("group_id"); v != "" {
			id, err := strconv.Atoi(v)
			if err != nil || id <= 0 {
				writeValidationError(c, "group_id", "invalid group_id")
				return
			}
			filter.GroupID = &id
		}
		if v := c.Query("teacher_id"); v != "" {
			id, err := strconv.Atoi(v)
			if err != nil || id <= 0 {
				writeValidationError(c, "teacher_id", "invalid teacher_id")
				return
			}
			filter.TeacherID = &id
		}
		if v := strings.TrimSpace(c.Query("teacher_name")); v != "" {
			filter.TeacherName = &v
		}
		if v := c.Query("location_id"); v != "" {
			id, err := strconv.Atoi(v)
			if err != nil || id <= 0 {
				writeValidationError(c, "location_id", "invalid location_id")
				return
			}
			filter.LocationID = &id
		}

		resp, err := svc.GetScheduleView(filter, start, end)
		if err != nil {
			writeError(c, http.StatusBadRequest, "bad_request", "", err.Error())
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func parseScheduleViewDateRange(c *gin.Context) (time.Time, time.Time, bool) {
	startStr := c.Query("date_start")
	if startStr == "" {
		startStr = c.Query("start_date")
	}
	endStr := c.Query("date_end")
	if endStr == "" {
		endStr = c.Query("end_date")
	}
	if startStr == "" || endStr == "" {
		writeValidationError(c, "", "date_start and date_end required")
		return time.Time{}, time.Time{}, false
	}
	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		writeValidationError(c, "date_start", "invalid date_start (YYYY-MM-DD)")
		return time.Time{}, time.Time{}, false
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		writeValidationError(c, "date_end", "invalid date_end (YYYY-MM-DD)")
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}
