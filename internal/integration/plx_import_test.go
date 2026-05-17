//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"ispo-schedule/internal/httpapi"
	"ispo-schedule/internal/schedule"
)

func TestHandleAdminImportPLXCurriculumXLSX_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	h := httpapi.HandleAdminImportPLXCurriculumXLSXForTest(repo)

	f := newPLXWorkbookForIntegration(t)
	defer func() { _ = f.Close() }()
	body, ctype := multipartXLSXForIntegration(t, "file", "99.02.01_26_11.plx.xlsx", f)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/import/plx-curriculum/xlsx?academic_year_start=2026-09-01", body)
	c.Request.Header.Set("Content-Type", ctype)

	h(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp struct {
		SpecialtyID   int   `json:"specialty_id"`
		CurriculumID  int64 `json:"curriculum_id"`
		Items         int   `json:"items"`
		Allocations   int   `json:"allocations"`
		CalendarWeeks int   `json:"calendar_weeks"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	t.Cleanup(func() {
		_ = db.Exec("DELETE FROM curricula WHERE id = ?", resp.CurriculumID).Error
		_ = db.Exec("DELETE FROM specialties WHERE id = ?", resp.SpecialtyID).Error
	})
	assert.Equal(t, 3, resp.Items)
	assert.Equal(t, 2, resp.Allocations)
	assert.Equal(t, 104, resp.CalendarWeeks)
	assert.NotZero(t, resp.CurriculumID)

	var items int64
	require.NoError(t, db.Model(&schedule.CurriculumItem{}).Where("curriculum_id = ?", resp.CurriculumID).Count(&items).Error)
	assert.Equal(t, int64(3), items)

	var calendars []schedule.AcademicCalendar
	require.NoError(t, db.Where("curriculum_id = ?", resp.CurriculumID).Find(&calendars).Error)
	require.NotEmpty(t, calendars)
	var weeks int64
	require.NoError(t, db.Model(&schedule.AcademicCalendarWeek{}).Where("calendar_id = ?", calendars[0].ID).Count(&weeks).Error)
	assert.Equal(t, int64(104), weeks)

	var practiceWeek schedule.AcademicCalendarWeek
	require.NoError(t, db.Where("calendar_id = ? AND course_number = ? AND week_number = ?", calendars[0].ID, 1, 2).First(&practiceWeek).Error)
	assert.Equal(t, "PRACTICE_EDU", practiceWeek.ActivityCode)
	assert.False(t, practiceWeek.IsTeaching)

	var vacationWeek schedule.AcademicCalendarWeek
	require.NoError(t, db.Where("calendar_id = ? AND course_number = ? AND week_number = ?", calendars[0].ID, 1, 4).First(&vacationWeek).Error)
	assert.Equal(t, "VACATION", vacationWeek.ActivityCode)
	assert.False(t, vacationWeek.IsTeaching)

	var secondCourseWeek schedule.AcademicCalendarWeek
	require.NoError(t, db.Where("calendar_id = ? AND course_number = ? AND week_number = ?", calendars[0].ID, 2, 2).First(&secondCourseWeek).Error)
	assert.Equal(t, "PRACTICE_PREGRAD", secondCourseWeek.ActivityCode)
	assert.False(t, secondCourseWeek.IsTeaching)
}

func multipartXLSXForIntegration(t *testing.T, field, filename string, f *excelize.File) (*bytes.Buffer, string) {
	t.Helper()
	xlsx, err := f.WriteToBuffer()
	require.NoError(t, err)
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	fw, err := mw.CreateFormFile(field, filename)
	require.NoError(t, err)
	_, err = fw.Write(xlsx.Bytes())
	require.NoError(t, err)
	require.NoError(t, mw.Close())
	return buf, mw.FormDataContentType()
}

func newPLXWorkbookForIntegration(t *testing.T) *excelize.File {
	t.Helper()
	f := excelize.NewFile()
	plan := f.GetSheetName(0)
	require.NoError(t, f.SetSheetName(plan, "План"))
	plan = "План"
	_, err := f.NewSheet("График")
	require.NoError(t, err)

	_ = f.SetCellValue(plan, "Q1", "Курс 1")
	_ = f.SetCellValue(plan, "Q2", "Семестр 1 [16 нед]")
	_ = f.SetCellValue(plan, "Y2", "Семестр 2 [21 нед]")
	_ = f.SetCellValue(plan, "A3", "Считать в плане")
	_ = f.SetCellValue(plan, "B3", "Индекс")
	_ = f.SetCellValue(plan, "C3", "Наименование")
	_ = f.SetCellValue(plan, "D3", "Экза мен")
	_ = f.SetCellValue(plan, "E3", "Зачет")
	_ = f.SetCellValue(plan, "F3", "Зачет с оц.")
	_ = f.SetCellValue(plan, "Q3", "Итого")
	_ = f.SetCellValue(plan, "S3", "Лек")
	_ = f.SetCellValue(plan, "Y3", "Итого")
	_ = f.SetCellValue(plan, "AA3", "Лек")
	_ = f.SetCellValue(plan, "A4", "ПП.ПРОФЕССИОНАЛЬНАЯ ПОДГОТОВКА")
	_ = f.SetCellValue(plan, "A5", "СГЦ.Социально-гуманитарный цикл")
	_ = f.SetCellValue(plan, "A6", "+")
	_ = f.SetCellValue(plan, "B6", "ОГСЭ.01")
	_ = f.SetCellValue(plan, "C6", "История России")
	_ = f.SetCellValue(plan, "F6", 2)
	_ = f.SetCellValue(plan, "Q6", 34)
	_ = f.SetCellValue(plan, "S6", 34)
	_ = f.SetCellValue(plan, "Y6", 32)
	_ = f.SetCellValue(plan, "AA6", 32)

	graph := "График"
	for week := 1; week <= 52; week++ {
		cellName, err := excelize.CoordinatesToCellName(week+1, 4)
		require.NoError(t, err)
		_ = f.SetCellValue(graph, cellName, week)
	}
	_ = f.SetCellValue(graph, "A13", "I")
	_ = f.SetCellValue(graph, "B13", "=")
	_ = f.SetCellValue(graph, "C13", "У")
	_ = f.SetCellValue(graph, "D13", "Э")
	_ = f.SetCellValue(graph, "E13", "К")
	_ = f.SetCellValue(graph, "A14", "II")
	_ = f.SetCellValue(graph, "B14", "=")
	_ = f.SetCellValue(graph, "C14", "ПП")
	return f
}
