//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
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

func TestHandleAdminCreateGroup_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	
	gin.SetMode(gin.TestMode)
	
	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	
	// Create admin user for auth
	adminUser := &auth.User{
		Login:        "admin_test",
		PasswordHash: "hash",
		Role:         auth.RoleAdmin,
	}
	db.Where("login = ?", adminUser.Login).Delete(&auth.User{})
	err := db.Create(adminUser).Error
	require.NoError(t, err)
	defer db.Where("login = ?", adminUser.Login).Delete(&auth.User{})
	
	handler := httpapi.HandleAdminCreateGroupForTest(repo)
	
	body := `{"name":"Integration Test Group","course":2}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/groups", bytes.NewReader([]byte(body)))
	c.Request.Header.Set("Content-Type", "application/json")
	
	// Mock auth context
	c.Set("auth.user", adminUser)
	
	handler(c)
	
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Integration Test Group")
	
	// Cleanup
	db.Where("name = ?", "Integration Test Group").Delete(&schedule.Group{})
}

func TestHandleAdminCreateGroup_EmptyName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	
	gin.SetMode(gin.TestMode)
	
	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	
	adminUser := &auth.User{
		Login:        "admin_test2",
		PasswordHash: "hash",
		Role:         auth.RoleAdmin,
	}
	db.Where("login = ?", adminUser.Login).Delete(&auth.User{})
	err := db.Create(adminUser).Error
	require.NoError(t, err)
	defer db.Where("login = ?", adminUser.Login).Delete(&auth.User{})
	
	handler := httpapi.HandleAdminCreateGroupForTest(repo)
	
	body := `{"name":"","course":1}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/groups", bytes.NewReader([]byte(body)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("auth.user", adminUser)
	
	handler(c)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "validation_error")
}

func TestHandleAdminCreateGroup_InvalidCourse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	
	gin.SetMode(gin.TestMode)
	
	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	
	adminUser := &auth.User{
		Login:        "admin_test3",
		PasswordHash: "hash",
		Role:         auth.RoleAdmin,
	}
	db.Where("login = ?", adminUser.Login).Delete(&auth.User{})
	err := db.Create(adminUser).Error
	require.NoError(t, err)
	defer db.Where("login = ?", adminUser.Login).Delete(&auth.User{})
	
	handler := httpapi.HandleAdminCreateGroupForTest(repo)
	
	body := `{"name":"Test","course":0}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/groups", bytes.NewReader([]byte(body)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("auth.user", adminUser)
	
	handler(c)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "validation_error")
}

func TestHandleAdminListGroups_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	
	gin.SetMode(gin.TestMode)
	
	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	
	// Create test group
	group := setupTestGroup(t, db)
	defer db.Where("id = ?", group.ID).Delete(&schedule.Group{})
	
	adminUser := &auth.User{
		Login:        "admin_test4",
		PasswordHash: "hash",
		Role:         auth.RoleAdmin,
	}
	db.Where("login = ?", adminUser.Login).Delete(&auth.User{})
	err := db.Create(adminUser).Error
	require.NoError(t, err)
	defer db.Where("login = ?", adminUser.Login).Delete(&auth.User{})
	
	handler := httpapi.HandleAdminListGroupsForTest(repo)
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups", nil)
	c.Set("auth.user", adminUser)
	
	handler(c)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Test Group")
}

func TestHandleAdminUpdateGroup_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	
	gin.SetMode(gin.TestMode)
	
	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	
	// Create test group
	group := setupTestGroup(t, db)
	defer db.Where("id = ?", group.ID).Delete(&schedule.Group{})
	
	adminUser := &auth.User{
		Login:        "admin_test5",
		PasswordHash: "hash",
		Role:         auth.RoleAdmin,
	}
	db.Where("login = ?", adminUser.Login).Delete(&auth.User{})
	err := db.Create(adminUser).Error
	require.NoError(t, err)
	defer db.Where("login = ?", adminUser.Login).Delete(&auth.User{})
	
	handler := httpapi.HandleAdminUpdateGroupForTest(repo)
	
	body := `{"name":"Updated Group","course":3}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/groups/"+string(rune(group.ID)), bytes.NewReader([]byte(body)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("auth.user", adminUser)
	
	handler(c)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Updated Group")
}

func TestHandleAdminDeleteGroup_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	
	gin.SetMode(gin.TestMode)
	
	db := getTestDB(t)
	repo := schedule.NewRepository(db)
	
	// Create test group
	group := setupTestGroup(t, db)
	
	adminUser := &auth.User{
		Login:        "admin_test6",
		PasswordHash: "hash",
		Role:         auth.RoleAdmin,
	}
	db.Where("login = ?", adminUser.Login).Delete(&auth.User{})
	err := db.Create(adminUser).Error
	require.NoError(t, err)
	defer db.Where("login = ?", adminUser.Login).Delete(&auth.User{})
	
	handler := httpapi.HandleAdminDeleteGroupForTest(repo)
	
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/v1/admin/groups/"+string(rune(group.ID)), nil)
	c.Set("auth.user", adminUser)
	
	handler(c)
	
	assert.Equal(t, http.StatusNoContent, w.Code)
	
	// Verify deleted
	var count int64
	db.Model(&schedule.Group{}).Where("id = ?", group.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

// Helper to marshal JSON
func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
