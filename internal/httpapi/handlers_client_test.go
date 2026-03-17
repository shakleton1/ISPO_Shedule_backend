package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"ispo-schedule/internal/schedule"
)

// Note: Integration tests для client handlers с реальной БД находятся в
// internal/integration/client_handlers_test.go с build tag integration

func TestHandleGetCurrentSchedule_MissingGroupID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &schedule.Repository{}
	svc := schedule.NewService(schedule.ServiceDeps{
		Repo:              repo,
		SemesterStartDate: "2026-02-09",
		Now:               time.Now,
	})

	handler := handleGetCurrentSchedule(svc, repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/schedule/current", nil)

	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "group_id required")
}

func TestHandleGetCurrentSchedule_InvalidGroupID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &schedule.Repository{}
	svc := schedule.NewService(schedule.ServiceDeps{
		Repo:              repo,
		SemesterStartDate: "2026-02-09",
		Now:               time.Now,
	})

	handler := handleGetCurrentSchedule(svc, repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/schedule/current?group_id=invalid", nil)

	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "group_id required")
}

func TestHandleGetCurrentSchedule_InvalidDate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &schedule.Repository{}
	svc := schedule.NewService(schedule.ServiceDeps{
		Repo:              repo,
		SemesterStartDate: "2026-02-09",
		Now:               time.Now,
	})

	handler := handleGetCurrentSchedule(svc, repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/schedule/current?group_id=1&date=invalid", nil)

	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid date")
}

func TestHandleGetScheduleRange_MissingGroupID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := schedule.NewService(schedule.ServiceDeps{
		Repo:              &schedule.Repository{},
		SemesterStartDate: "2026-02-09",
		Now:               time.Now,
	})

	handler := handleGetScheduleRange(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/schedule/range", nil)

	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "group_id required")
}

func TestHandleGetScheduleRange_MissingDates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := schedule.NewService(schedule.ServiceDeps{
		Repo:              &schedule.Repository{},
		SemesterStartDate: "2026-02-09",
		Now:               time.Now,
	})

	handler := handleGetScheduleRange(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/schedule/range?group_id=1", nil)

	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "date_start and date_end required")
}

func TestHandleGetScheduleRange_InvalidDates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := schedule.NewService(schedule.ServiceDeps{
		Repo:              &schedule.Repository{},
		SemesterStartDate: "2026-02-09",
		Now:               time.Now,
	})

	handler := handleGetScheduleRange(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/schedule/range?group_id=1&date_start=invalid&date_end=2026-03-01", nil)

	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid date_start")
}

// Note: TestHandleGetScheduleVersion_Success requires integration test with real DB

func TestHandleGetSchedulePDF_MissingGroupID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &schedule.Repository{}
	svc := schedule.NewService(schedule.ServiceDeps{
		Repo:              repo,
		SemesterStartDate: "2026-02-09",
		Now:               time.Now,
	})

	handler := handleGetSchedulePDF(svc, repo, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/schedule/pdf", nil)

	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "group_id required")
}

func TestHandleGetSchedulePDF_MissingDateStart(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &schedule.Repository{}
	svc := schedule.NewService(schedule.ServiceDeps{
		Repo:              repo,
		SemesterStartDate: "2026-02-09",
		Now:               time.Now,
	})

	handler := handleGetSchedulePDF(svc, repo, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/schedule/pdf?group_id=1", nil)

	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "date_start required")
}

func TestHandleGetSchedulePDF_InvalidDate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &schedule.Repository{}
	svc := schedule.NewService(schedule.ServiceDeps{
		Repo:              repo,
		SemesterStartDate: "2026-02-09",
		Now:               time.Now,
	})

	handler := handleGetSchedulePDF(svc, repo, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/schedule/pdf?group_id=1&date_start=invalid", nil)

	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid date_start")
}
