package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ispo-schedule/internal/schedule"
)

func handleGetCurrentSchedule(svc *schedule.Service, repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID, err := strconv.Atoi(c.Query("group_id"))
		if err != nil || groupID <= 0 {
			writeValidationError(c, "group_id", "group_id required")
			return
		}

		ref := time.Now().UTC()
		if ds := c.Query("date"); ds != "" {
			parsed, err := time.Parse("2006-01-02", ds)
			if err != nil {
				writeValidationError(c, "date", "invalid date (YYYY-MM-DD)")
				return
			}
			ref = parsed
		}

		resp, err := svc.GetCurrentWeek(groupID, ref)
		if err != nil {
			writeError(c, http.StatusBadRequest, "bad_request", "", err.Error())
			return
		}
		_ = repo // reserved for future (e.g. group name)

		c.JSON(http.StatusOK, resp)
	}
}

func handleGetSchedulePDF(svc *schedule.Service, repo *schedule.Repository, engine schedulePDFEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID, err := strconv.Atoi(c.Query("group_id"))
		if err != nil || groupID <= 0 {
			writeValidationError(c, "group_id", "group_id required")
			return
		}
		ds := c.Query("date_start")
		if ds == "" {
			writeValidationError(c, "date_start", "date_start required")
			return
		}
		start, err := time.Parse("2006-01-02", ds)
		if err != nil {
			writeValidationError(c, "date_start", "invalid date_start (YYYY-MM-DD)")
			return
		}

		week1Start := scheduleMonday(start)
		week2Start := week1Start.AddDate(0, 0, 7)
		week1End := week1Start.AddDate(0, 0, 5)
		week2End := week2Start.AddDate(0, 0, 5)

		week1, err := svc.GetRange(groupID, week1Start, week1End)
		if err != nil {
			writeError(c, http.StatusBadRequest, "bad_request", "", err.Error())
			return
		}
		week2, err := svc.GetRange(groupID, week2Start, week2End)
		if err != nil {
			writeError(c, http.StatusBadRequest, "bad_request", "", err.Error())
			return
		}

		group, err := repo.GetGroup(groupID)
		if err != nil {
			writeDBError(c, err)
			return
		}

		html, err := buildSchedulePDFHTML(group.Name, week1.Days, week2.Days)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal", "", err.Error())
			return
		}

		pdfBytes, err := engine.RenderHTMLToPDF(c.Request.Context(), html)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal", "", err.Error())
			return
		}

		filename := "schedule_" + group.Name + "_" + week1Start.Format("2006-01-02") + ".pdf"
		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
		c.Data(http.StatusOK, "application/pdf", pdfBytes)
	}
}

func handleGetScheduleRange(svc *schedule.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID, err := strconv.Atoi(c.Query("group_id"))
		if err != nil || groupID <= 0 {
			writeValidationError(c, "group_id", "group_id required")
			return
		}
		startStr := c.Query("date_start")
		endStr := c.Query("date_end")
		if startStr == "" || endStr == "" {
			writeValidationError(c, "", "date_start and date_end required")
			return
		}
		start, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			writeValidationError(c, "date_start", "invalid date_start (YYYY-MM-DD)")
			return
		}
		end, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			writeValidationError(c, "date_end", "invalid date_end (YYYY-MM-DD)")
			return
		}
		resp, err := svc.GetRange(groupID, start, end)
		if err != nil {
			writeError(c, http.StatusBadRequest, "bad_request", "", err.Error())
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func handleGetScheduleVersion(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		st, err := repo.GetSystemState()
		if err != nil {
			writeError(c, http.StatusInternalServerError, "db_error", "", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"data_version": st.ScheduleVersion.UTC().Format(time.RFC3339)})
	}
}
