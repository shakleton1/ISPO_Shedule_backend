package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ispo-schedule/internal/auth"
	"ispo-schedule/internal/schedule"
)

// Note: Integration tests для auth handlers с реальной БД находятся в
// internal/integration/auth_handlers_test.go с build tag integration

func TestHandleLogin_EmptyCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokens, err := auth.NewTokenManager("test-secret", time.Hour)
	require.NoError(t, err)

	repo := &schedule.Repository{}
	handler := handleLogin(tokens, repo, time.Hour*720)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "login and password required")
}

func TestHandleLogin_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokens, err := auth.NewTokenManager("test-secret", time.Hour)
	require.NoError(t, err)

	repo := &schedule.Repository{}
	handler := handleLogin(tokens, repo, time.Hour*720)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{invalid json}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "validation_error")
}

func TestHandleRefresh_EmptyToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokens, err := auth.NewTokenManager("test-secret", time.Hour)
	require.NoError(t, err)

	repo := &schedule.Repository{}
	handler := handleRefresh(tokens, repo, time.Hour*720)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "refresh_token required")
}

func TestHandleLogout_EmptyToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &schedule.Repository{}
	handler := handleLogout(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "refresh_token required")
}

// Note: TestHandleLogout_Success requires integration test with real DB

func TestHandleMe_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &schedule.Repository{}
	handler := handleMe(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	// No user in context

	handler(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unauthorized")
}

// Note: TestHandleMe_Success requires integration test with real DB
