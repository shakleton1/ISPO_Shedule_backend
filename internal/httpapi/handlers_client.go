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
		groupID, start, ok := bindGroupTwoWeekExportQuery(c)
		if !ok {
			return
		}
		data, err := buildGroupTwoWeekScheduleExportData(svc, repo, groupID, start)
		if err != nil {
			writeError(c, http.StatusBadRequest, "bad_request", "", err.Error())
			return
		}

		html, err := buildTwoWeekSchedulePDFHTML(data)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal", "", err.Error())
			return
		}

		pdfBytes, err := engine.RenderHTMLToPDF(c.Request.Context(), html)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal", "", err.Error())
			return
		}

		writeBinaryExport(c, http.StatusOK, "application/pdf", exportFileName(data.Title, start, "pdf"), pdfBytes)
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
