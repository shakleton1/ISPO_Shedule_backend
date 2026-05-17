package httpapi

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ispo-schedule/internal/schedule"
)

func handleGetScheduleXLSX(svc *schedule.Service, repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID, refDate, ok := bindGroupTwoWeekExportQuery(c)
		if !ok {
			return
		}
		data, err := buildGroupTwoWeekScheduleExportData(svc, repo, groupID, refDate)
		if err != nil {
			writeError(c, http.StatusBadRequest, "bad_request", "", err.Error())
			return
		}
		body, err := buildTwoWeekScheduleXLSX(data)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal", "", err.Error())
			return
		}
		writeBinaryExport(c, http.StatusOK, xlsxContentType, exportFileName(data.Title, refDate, "xlsx"), body)
	}
}

func handleAdminExportGroupSchedulePDF(svc *schedule.Service, repo *schedule.Repository, engine schedulePDFEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID, refDate, ok := bindGroupTwoWeekExportQuery(c)
		if !ok {
			return
		}
		data, err := buildGroupTwoWeekScheduleExportData(svc, repo, groupID, refDate)
		if err != nil {
			writeError(c, http.StatusBadRequest, "bad_request", "", err.Error())
			return
		}
		html, err := buildTwoWeekSchedulePDFHTML(data)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal", "", err.Error())
			return
		}
		body, err := engine.RenderHTMLToPDF(c.Request.Context(), html)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal", "", err.Error())
			return
		}
		writeBinaryExport(c, http.StatusOK, "application/pdf", exportFileName(data.Title, refDate, "pdf"), body)
	}
}

func handleAdminExportGroupScheduleXLSX(svc *schedule.Service, repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID, refDate, ok := bindGroupTwoWeekExportQuery(c)
		if !ok {
			return
		}
		data, err := buildGroupTwoWeekScheduleExportData(svc, repo, groupID, refDate)
		if err != nil {
			writeError(c, http.StatusBadRequest, "bad_request", "", err.Error())
			return
		}
		body, err := buildTwoWeekScheduleXLSX(data)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal", "", err.Error())
			return
		}
		writeBinaryExport(c, http.StatusOK, xlsxContentType, exportFileName(data.Title, refDate, "xlsx"), body)
	}
}

func handleAdminExportTeacherSchedulePDF(svc *schedule.Service, repo *schedule.Repository, engine schedulePDFEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		teacherID, refDate, ok := bindTeacherTwoWeekExportQuery(c)
		if !ok {
			return
		}
		data, err := buildTeacherTwoWeekScheduleExportData(svc, repo, teacherID, refDate)
		if err != nil {
			writeError(c, http.StatusBadRequest, "bad_request", "", err.Error())
			return
		}
		html, err := buildTwoWeekSchedulePDFHTML(data)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal", "", err.Error())
			return
		}
		body, err := engine.RenderHTMLToPDF(c.Request.Context(), html)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal", "", err.Error())
			return
		}
		writeBinaryExport(c, http.StatusOK, "application/pdf", exportFileName(data.Title, refDate, "pdf"), body)
	}
}

func handleAdminExportTeacherScheduleXLSX(svc *schedule.Service, repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		teacherID, refDate, ok := bindTeacherTwoWeekExportQuery(c)
		if !ok {
			return
		}
		data, err := buildTeacherTwoWeekScheduleExportData(svc, repo, teacherID, refDate)
		if err != nil {
			writeError(c, http.StatusBadRequest, "bad_request", "", err.Error())
			return
		}
		body, err := buildTwoWeekScheduleXLSX(data)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal", "", err.Error())
			return
		}
		writeBinaryExport(c, http.StatusOK, xlsxContentType, exportFileName(data.Title, refDate, "xlsx"), body)
	}
}

func handleAdminExportScheduleOverridesPDF(repo *schedule.Repository, engine schedulePDFEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		filters, startDate, endDate, ok := bindScheduleOverrideExportQuery(c)
		if !ok {
			return
		}
		data, err := buildScheduleOverridesExportData(repo, filters, startDate, endDate)
		if err != nil {
			writeDBError(c, err)
			return
		}
		html, err := buildScheduleOverridesPDFHTML(data)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal", "", err.Error())
			return
		}
		body, err := engine.RenderHTMLToPDF(c.Request.Context(), html)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal", "", err.Error())
			return
		}
		writeBinaryExport(c, http.StatusOK, "application/pdf", exportFileName(data.Title, time.Now().UTC(), "pdf"), body)
	}
}

func handleAdminExportScheduleOverridesXLSX(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		filters, startDate, endDate, ok := bindScheduleOverrideExportQuery(c)
		if !ok {
			return
		}
		data, err := buildScheduleOverridesExportData(repo, filters, startDate, endDate)
		if err != nil {
			writeDBError(c, err)
			return
		}
		body, err := buildScheduleOverridesXLSX(data)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal", "", err.Error())
			return
		}
		writeBinaryExport(c, http.StatusOK, xlsxContentType, exportFileName(data.Title, time.Now().UTC(), "xlsx"), body)
	}
}

func bindGroupTwoWeekExportQuery(c *gin.Context) (int, time.Time, bool) {
	groupID, err := strconv.Atoi(c.Query("group_id"))
	if err != nil || groupID <= 0 {
		writeValidationError(c, "group_id", "group_id required")
		return 0, time.Time{}, false
	}
	refDate, ok := parseRequiredExportDate(c)
	return groupID, refDate, ok
}

func bindTeacherTwoWeekExportQuery(c *gin.Context) (int, time.Time, bool) {
	teacherID, err := strconv.Atoi(c.Query("teacher_id"))
	if err != nil || teacherID <= 0 {
		writeValidationError(c, "teacher_id", "teacher_id required")
		return 0, time.Time{}, false
	}
	refDate, ok := parseRequiredExportDate(c)
	return teacherID, refDate, ok
}

func bindScheduleOverrideExportQuery(c *gin.Context) (schedule.ScheduleOverrideFilters, *time.Time, *time.Time, bool) {
	groupID, ok := parseQueryInt(c, "group_id")
	if !ok {
		return schedule.ScheduleOverrideFilters{}, nil, nil, false
	}
	teacherID, ok := parseQueryInt(c, "teacher_id")
	if !ok {
		return schedule.ScheduleOverrideFilters{}, nil, nil, false
	}
	startDate, ok := parseQueryDateWithAliases(c, "start_date", "date_start")
	if !ok {
		return schedule.ScheduleOverrideFilters{}, nil, nil, false
	}
	endDate, ok := parseQueryDateWithAliases(c, "end_date", "date_end")
	if !ok {
		return schedule.ScheduleOverrideFilters{}, nil, nil, false
	}
	var action *schedule.OverrideAction
	if v := strings.TrimSpace(c.Query("action_type")); v != "" {
		a := schedule.OverrideAction(v)
		action = &a
	}
	return schedule.ScheduleOverrideFilters{
		GroupID:    groupID,
		TeacherID:  teacherID,
		StartDate:  startDate,
		EndDate:    endDate,
		ActionType: action,
	}, startDate, endDate, true
}

func parseRequiredExportDate(c *gin.Context) (time.Time, bool) {
	v := strings.TrimSpace(c.Query("date_start"))
	if v == "" {
		v = strings.TrimSpace(c.Query("start_date"))
	}
	if v == "" {
		writeValidationError(c, "date_start", "date_start required")
		return time.Time{}, false
	}
	d, err := time.Parse("2006-01-02", v)
	if err != nil {
		writeValidationError(c, "date_start", "invalid date_start (YYYY-MM-DD)")
		return time.Time{}, false
	}
	return d, true
}

func parseQueryDateWithAliases(c *gin.Context, primary string, alias string) (*time.Time, bool) {
	v := strings.TrimSpace(c.Query(primary))
	if v == "" {
		v = strings.TrimSpace(c.Query(alias))
	}
	if v == "" {
		return nil, true
	}
	d, err := time.Parse("2006-01-02", v)
	if err != nil {
		writeValidationError(c, primary, "invalid "+primary)
		return nil, false
	}
	return &d, true
}

func writeBinaryExport(c *gin.Context, status int, contentType string, filename string, body []byte) {
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", contentDisposition(filename))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(status, contentType, body)
}

func exportFileName(title string, refDate time.Time, ext string) string {
	title = strings.ToLower(strings.TrimSpace(title))
	title = strings.NewReplacer(
		" ", "_",
		"\"", "",
		"'", "",
		"«", "",
		"»", "",
	).Replace(title)
	return fmt.Sprintf("%s_%s.%s", title, scheduleMonday(refDate).Format("2006-01-02"), ext)
}
