//go:build integration

package integration

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ispo-schedule/internal/auth"
	"ispo-schedule/internal/httpapi"
	"ispo-schedule/internal/schedule"
)

func TestHandleAdminCreateSpecialty_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	gin.SetMode(gin.TestMode)

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	adminUser := &auth.User{
		Login:        "admin_curricula_test",
		PasswordHash: "hash",
		Role:         auth.RoleAdmin,
	}
	db.Where("login = ?", adminUser.Login).Delete(&auth.User{})
	err := db.Create(adminUser).Error
	require.NoError(t, err)
	defer db.Where("login = ?", adminUser.Login).Delete(&auth.User{})

	handler := httpapi.HandleAdminCreateSpecialtyForTest(repo)

	body := `{"code":"TEST-001","name":"Test Specialty"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/specialties", bytes.NewReader([]byte(body)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("auth.user", adminUser)

	handler(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "TEST-001")
	assert.Contains(t, w.Body.String(), "Test Specialty")

	// Cleanup
	db.Where("code = ?", "TEST-001").Delete(&schedule.Specialty{})
}

func TestHandleAdminCreateSpecialty_EmptyCode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	gin.SetMode(gin.TestMode)

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	adminUser := &auth.User{
		Login:        "admin_curricula_test2",
		PasswordHash: "hash",
		Role:         auth.RoleAdmin,
	}
	db.Where("login = ?", adminUser.Login).Delete(&auth.User{})
	err := db.Create(adminUser).Error
	require.NoError(t, err)
	defer db.Where("login = ?", adminUser.Login).Delete(&auth.User{})

	handler := httpapi.HandleAdminCreateSpecialtyForTest(repo)

	body := `{"code":"","name":"Test"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/specialties", bytes.NewReader([]byte(body)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("auth.user", adminUser)

	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "validation_error")
}

func TestHandleAdminListSpecialties_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	gin.SetMode(gin.TestMode)

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	// Create test specialty
	specialty := &schedule.Specialty{
		Code: "LIST-TEST",
		Name: "List Test Specialty",
	}
	db.Where("code = ?", specialty.Code).Delete(&schedule.Specialty{})
	db.Create(specialty)
	defer db.Where("code = ?", specialty.Code).Delete(&schedule.Specialty{})

	adminUser := &auth.User{
		Login:        "admin_curricula_test3",
		PasswordHash: "hash",
		Role:         auth.RoleAdmin,
	}
	db.Where("login = ?", adminUser.Login).Delete(&auth.User{})
	err := db.Create(adminUser).Error
	require.NoError(t, err)
	defer db.Where("login = ?", adminUser.Login).Delete(&auth.User{})

	handler := httpapi.HandleAdminListSpecialtiesForTest(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/specialties", nil)
	c.Set("auth.user", adminUser)

	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "LIST-TEST")
}

func TestHandleAdminUpdateSpecialty_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	gin.SetMode(gin.TestMode)

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	// Create test specialty
	specialty := &schedule.Specialty{
		Code: "UPDATE-TEST",
		Name: "Update Test Specialty",
	}
	db.Where("code = ?", specialty.Code).Delete(&schedule.Specialty{})
	db.Create(specialty)
	defer db.Where("code = ?", specialty.Code).Delete(&schedule.Specialty{})

	adminUser := &auth.User{
		Login:        "admin_curricula_test4",
		PasswordHash: "hash",
		Role:         auth.RoleAdmin,
	}
	db.Where("login = ?", adminUser.Login).Delete(&auth.User{})
	err := db.Create(adminUser).Error
	require.NoError(t, err)
	defer db.Where("login = ?", adminUser.Login).Delete(&auth.User{})

	handler := httpapi.HandleAdminUpdateSpecialtyForTest(repo)

	body := `{"code":"UPDATE-TEST-NEW","name":"Updated Specialty"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/admin/specialties/%d", specialty.ID), bytes.NewReader([]byte(body)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("auth.user", adminUser)

	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Updated Specialty")
}

func TestHandleAdminDeleteSpecialty_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	gin.SetMode(gin.TestMode)

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	// Create test specialty
	specialty := &schedule.Specialty{
		Code: "DELETE-TEST",
		Name: "Delete Test Specialty",
	}
	db.Where("code = ?", specialty.Code).Delete(&schedule.Specialty{})
	db.Create(specialty)

	adminUser := &auth.User{
		Login:        "admin_curricula_test5",
		PasswordHash: "hash",
		Role:         auth.RoleAdmin,
	}
	db.Where("login = ?", adminUser.Login).Delete(&auth.User{})
	err := db.Create(adminUser).Error
	require.NoError(t, err)
	defer db.Where("login = ?", adminUser.Login).Delete(&auth.User{})

	// Use router for proper 204 handling
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("auth.user", adminUser)
		c.Next()
	})

	handler := httpapi.HandleAdminDeleteSpecialtyForTest(repo)
	r.DELETE("/api/v1/admin/specialties/:id", handler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/admin/specialties/%d", specialty.ID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, 0, w.Body.Len())

	// Verify deleted
	var count int64
	db.Model(&schedule.Specialty{}).Where("id = ?", specialty.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}
