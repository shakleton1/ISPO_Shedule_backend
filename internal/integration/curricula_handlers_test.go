//go:build integration

package integration

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"ispo-schedule/internal/auth"
	"ispo-schedule/internal/httpapi"
	"ispo-schedule/internal/schedule"
)

func makeAdminForCurriculaTests(t *testing.T, db *gorm.DB) *auth.User {
	t.Helper()

	u := &auth.User{
		Login:        fmt.Sprintf("admin_curricula_handlers_%d", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         auth.RoleAdmin,
	}
	_ = db.Where("login = ?", u.Login).Delete(&auth.User{}).Error
	require.NoError(t, db.Create(u).Error)
	return u
}

func testSpecialtyCode(prefix string) string {
	return fmt.Sprintf("%s-%05d", prefix, time.Now().Unix()%100000)
}

func TestHandleAdminListCurricula_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	admin := makeAdminForCurriculaTests(t, db)
	defer db.Where("id = ?", admin.ID).Delete(&auth.User{})

	spec := &schedule.Specialty{Code: testSpecialtyCode("CL"), Name: "Curricula List Spec"}
	require.NoError(t, db.Create(spec).Error)
	defer db.Where("id = ?", spec.ID).Delete(&schedule.Specialty{})

	c1 := &schedule.Curriculum{SpecialtyID: spec.ID, AdmissionYear: 2024, Variant: "v1", Title: "Curr 1", IsActive: true}
	c2 := &schedule.Curriculum{SpecialtyID: spec.ID, AdmissionYear: 2025, Variant: "v2", Title: "Curr 2", IsActive: true}
	require.NoError(t, db.Create(c1).Error)
	require.NoError(t, db.Create(c2).Error)
	defer db.Exec("DELETE FROM curricula WHERE id IN ?", []int64{c1.ID, c2.ID})

	h := httpapi.HandleAdminListCurriculaForTest(repo)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/curricula?specialty_id="+strconv.Itoa(spec.ID)+"&limit=1&offset=0", nil)
	c.Set("auth.user", admin)

	h(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Curr 1")
	assert.NotContains(t, w.Body.String(), "Curr 2")
}

func TestHandleAdminCreateCurriculum_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	admin := makeAdminForCurriculaTests(t, db)
	defer db.Where("id = ?", admin.ID).Delete(&auth.User{})

	spec := &schedule.Specialty{Code: testSpecialtyCode("CC"), Name: "Curricula Create Spec"}
	require.NoError(t, db.Create(spec).Error)
	defer db.Where("id = ?", spec.ID).Delete(&schedule.Specialty{})

	h := httpapi.HandleAdminCreateCurriculumForTest(repo)
	body := fmt.Sprintf(`{"specialty_id":%d,"admission_year":2025,"variant":"full","title":"Created Curriculum","is_active":true}`, spec.ID)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/curricula", bytes.NewReader([]byte(body)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("auth.user", admin)

	h(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Created Curriculum")

	var created schedule.Curriculum
	require.NoError(t, db.Where("specialty_id = ? AND title = ?", spec.ID, "Created Curriculum").First(&created).Error)
	defer db.Exec("DELETE FROM curricula WHERE id = ?", created.ID)
}

func TestHandleAdminUpdateCurriculum_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	admin := makeAdminForCurriculaTests(t, db)
	defer db.Where("id = ?", admin.ID).Delete(&auth.User{})

	spec := &schedule.Specialty{Code: testSpecialtyCode("CU"), Name: "Curricula Update Spec"}
	require.NoError(t, db.Create(spec).Error)
	defer db.Where("id = ?", spec.ID).Delete(&schedule.Specialty{})

	curr := &schedule.Curriculum{SpecialtyID: spec.ID, AdmissionYear: 2024, Variant: "old", Title: "Old Title", IsActive: true}
	require.NoError(t, db.Create(curr).Error)
	defer db.Exec("DELETE FROM curricula WHERE id = ?", curr.ID)

	h := httpapi.HandleAdminUpdateCurriculumForTest(repo)
	body := fmt.Sprintf(`{"specialty_id":%d,"admission_year":2026,"variant":"new","title":"Updated Curriculum","is_active":false}`, spec.ID)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = []gin.Param{{Key: "id", Value: strconv.FormatInt(curr.ID, 10)}}
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/curricula/"+strconv.FormatInt(curr.ID, 10), bytes.NewReader([]byte(body)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("auth.user", admin)

	h(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Updated Curriculum")
}

func TestHandleAdminDeleteCurriculum_SoftDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	admin := makeAdminForCurriculaTests(t, db)
	defer db.Where("id = ?", admin.ID).Delete(&auth.User{})

	spec := &schedule.Specialty{Code: testSpecialtyCode("CD"), Name: "Curricula Delete Spec"}
	require.NoError(t, db.Create(spec).Error)
	defer db.Where("id = ?", spec.ID).Delete(&schedule.Specialty{})

	curr := &schedule.Curriculum{SpecialtyID: spec.ID, AdmissionYear: 2023, Variant: "del", Title: "Delete Me", IsActive: true}
	require.NoError(t, db.Create(curr).Error)
	defer db.Exec("DELETE FROM curricula WHERE id = ?", curr.ID)

	h := httpapi.HandleAdminDeleteCurriculumForTest(repo)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("auth.user", admin)
		c.Next()
	})
	r.DELETE("/api/v1/admin/curricula/:id", h)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/curricula/"+strconv.FormatInt(curr.ID, 10), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	var cnt int64
	require.NoError(t, db.Raw("SELECT count(*) FROM curricula WHERE id = ? AND deleted_at IS NOT NULL", curr.ID).Scan(&cnt).Error)
	assert.Equal(t, int64(1), cnt)
}

func TestHandleAdminListAcademicCalendars_List(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	admin := makeAdminForCurriculaTests(t, db)
	defer db.Where("id = ?", admin.ID).Delete(&auth.User{})

	spec := &schedule.Specialty{Code: testSpecialtyCode("AL"), Name: "AC List Spec"}
	require.NoError(t, db.Create(spec).Error)
	defer db.Where("id = ?", spec.ID).Delete(&schedule.Specialty{})

	curr := &schedule.Curriculum{SpecialtyID: spec.ID, AdmissionYear: 2024, Variant: "v", Title: "Curr", IsActive: true}
	require.NoError(t, db.Create(curr).Error)
	defer db.Exec("DELETE FROM curricula WHERE id = ?", curr.ID)

	ac := &schedule.AcademicCalendar{CurriculumID: curr.ID, AcademicYearStart: time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC), WeeksTotal: 52}
	require.NoError(t, db.Create(ac).Error)
	defer db.Where("id = ?", ac.ID).Delete(&schedule.AcademicCalendar{})

	h := httpapi.HandleAdminListAcademicCalendarsForTest(repo)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = []gin.Param{{Key: "id", Value: strconv.FormatInt(curr.ID, 10)}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/curricula/"+strconv.FormatInt(curr.ID, 10)+"/calendars?limit=10", nil)
	c.Set("auth.user", admin)

	h(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "academic_year_start")
}

func TestHandleAdminUpsertAcademicCalendarWeeks_Weeks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	admin := makeAdminForCurriculaTests(t, db)
	defer db.Where("id = ?", admin.ID).Delete(&auth.User{})

	spec := &schedule.Specialty{Code: testSpecialtyCode("AW"), Name: "AC Weeks Spec"}
	require.NoError(t, db.Create(spec).Error)
	defer db.Where("id = ?", spec.ID).Delete(&schedule.Specialty{})

	curr := &schedule.Curriculum{SpecialtyID: spec.ID, AdmissionYear: 2024, Variant: "v", Title: "Curr", IsActive: true}
	require.NoError(t, db.Create(curr).Error)
	defer db.Exec("DELETE FROM curricula WHERE id = ?", curr.ID)

	ac := &schedule.AcademicCalendar{CurriculumID: curr.ID, AcademicYearStart: time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC), WeeksTotal: 52}
	require.NoError(t, db.Create(ac).Error)
	defer db.Where("id = ?", ac.ID).Delete(&schedule.AcademicCalendar{})

	h := httpapi.HandleAdminUpsertAcademicCalendarWeeksForTest(repo)
	body := `[{"week_number":1,"week_start_date":"2025-09-01T00:00:00Z","activity_code":"ST","activity_name":"Study","is_teaching":true}]`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = []gin.Param{{Key: "id", Value: strconv.FormatInt(ac.ID, 10)}}
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/calendars/"+strconv.FormatInt(ac.ID, 10)+"/weeks", bytes.NewReader([]byte(body)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("auth.user", admin)

	h(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"week_number\":1")
}

func TestHandleAdminListCurriculumItems_Items(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	admin := makeAdminForCurriculaTests(t, db)
	defer db.Where("id = ?", admin.ID).Delete(&auth.User{})

	spec := &schedule.Specialty{Code: testSpecialtyCode("IL"), Name: "Items Spec"}
	require.NoError(t, db.Create(spec).Error)
	defer db.Where("id = ?", spec.ID).Delete(&schedule.Specialty{})

	curr := &schedule.Curriculum{SpecialtyID: spec.ID, AdmissionYear: 2024, Variant: "v", Title: "Curr", IsActive: true}
	require.NoError(t, db.Create(curr).Error)
	defer db.Exec("DELETE FROM curricula WHERE id = ?", curr.ID)

	item := &schedule.CurriculumItem{CurriculumID: curr.ID, ItemType: "DISCIPLINE", Name: "Algorithms"}
	require.NoError(t, db.Create(item).Error)
	defer db.Where("id = ?", item.ID).Delete(&schedule.CurriculumItem{})

	h := httpapi.HandleAdminListCurriculumItemsForTest(repo)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = []gin.Param{{Key: "id", Value: strconv.FormatInt(curr.ID, 10)}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/curricula/"+strconv.FormatInt(curr.ID, 10)+"/items?limit=10", nil)
	c.Set("auth.user", admin)

	h(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Algorithms")
}

func TestHandleAdminUpsertCurriculumItemAllocations_Allocations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	admin := makeAdminForCurriculaTests(t, db)
	defer db.Where("id = ?", admin.ID).Delete(&auth.User{})

	spec := &schedule.Specialty{Code: testSpecialtyCode("AU"), Name: "Alloc Spec"}
	require.NoError(t, db.Create(spec).Error)
	defer db.Where("id = ?", spec.ID).Delete(&schedule.Specialty{})

	curr := &schedule.Curriculum{SpecialtyID: spec.ID, AdmissionYear: 2024, Variant: "v", Title: "Curr", IsActive: true}
	require.NoError(t, db.Create(curr).Error)
	defer db.Exec("DELETE FROM curricula WHERE id = ?", curr.ID)

	item := &schedule.CurriculumItem{CurriculumID: curr.ID, ItemType: "DISCIPLINE", Name: "Math"}
	require.NoError(t, db.Create(item).Error)
	defer db.Where("id = ?", item.ID).Delete(&schedule.CurriculumItem{})

	h := httpapi.HandleAdminUpsertCurriculumItemAllocationsForTest(repo)
	body := `[{"semester":1,"weeks":16,"hours_total":64,"hours_lectures":32,"hours_practice":32,"assessment_type":"exam"}]`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = []gin.Param{{Key: "id", Value: strconv.FormatInt(item.ID, 10)}}
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/curriculum-items/"+strconv.FormatInt(item.ID, 10)+"/allocations", bytes.NewReader([]byte(body)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("auth.user", admin)

	h(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"semester\":1")
}
