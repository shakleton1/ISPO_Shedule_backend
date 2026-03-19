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

	"ispo-schedule/internal/httpapi"
	"ispo-schedule/internal/schedule"
)

func TestHandlePushRegister_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	gin.SetMode(gin.TestMode)

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	// Create test group
	group := setupTestGroup(t, db)
	defer db.Where("id = ?", group.ID).Delete(&schedule.Group{})

	handler := httpapi.HandlePushRegisterForTest(repo)

	body := `{"group_id":` + fmt.Sprintf("%d", group.ID) + `,"token":"test_device_token_123"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/push/register", bytes.NewReader([]byte(body)))
	c.Request.Header.Set("Content-Type", "application/json")

	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// Cleanup
	db.Where("token = ?", "test_device_token_123").Delete(&schedule.DeviceToken{})
}

func TestHandlePushRegister_InvalidToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	gin.SetMode(gin.TestMode)

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	group := setupTestGroup(t, db)
	defer db.Where("id = ?", group.ID).Delete(&schedule.Group{})

	handler := httpapi.HandlePushRegisterForTest(repo)

	body := `{"group_id":` + fmt.Sprintf("%d", group.ID) + `,"token":""}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/push/register", bytes.NewReader([]byte(body)))
	c.Request.Header.Set("Content-Type", "application/json")

	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "bad_request")
}

func TestHandlePushUnregister_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	gin.SetMode(gin.TestMode)

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	group := setupTestGroup(t, db)
	defer db.Where("id = ?", group.ID).Delete(&schedule.Group{})

	// Create device token first
	token := "test_token_unregister"
	deviceToken := &schedule.DeviceToken{
		GroupID: group.ID,
		Token:   token,
	}
	db.Create(deviceToken)

	// Use router for proper request handling
	r := gin.New()
	handler := httpapi.HandlePushUnregisterForTest(repo)
	r.DELETE("/api/v1/push/unregister/:token", handler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/push/unregister/%s", token), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify deleted
	var count int64
	db.Model(&schedule.DeviceToken{}).Where("token = ?", token).Count(&count)
	assert.Equal(t, int64(0), count)
}
