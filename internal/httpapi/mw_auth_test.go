package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ispo-schedule/internal/auth"
	"ispo-schedule/internal/schedule"
)

func TestAuthMiddleware_MissingAuthorizationHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokens, err := auth.NewTokenManager("test-secret", time.Hour)
	require.NoError(t, err)

	repo := &schedule.Repository{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups", nil)

	authMiddleware(tokens, repo)(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unauthorized")
	assert.Contains(t, w.Body.String(), "missing Authorization header")
}

func TestAuthMiddleware_InvalidBearerFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokens, err := auth.NewTokenManager("test-secret", time.Hour)
	require.NoError(t, err)

	repo := &schedule.Repository{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups", nil)
	c.Request.Header.Set("Authorization", "InvalidFormat")

	authMiddleware(tokens, repo)(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unauthorized")
	assert.Contains(t, w.Body.String(), "invalid Authorization header")
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokens, err := auth.NewTokenManager("test-secret", time.Hour)
	require.NoError(t, err)

	repo := &schedule.Repository{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups", nil)
	c.Request.Header.Set("Authorization", "Bearer invalid.token.string")

	authMiddleware(tokens, repo)(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unauthorized")
	assert.Contains(t, w.Body.String(), "invalid token")
}

// Note: Integration tests для auth middleware с реальной БД находятся в
// internal/integration/auth_middleware_test.go с build tag integration

func TestRequireAnyRole_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups", nil)

	handler := requireAnyRole(auth.RoleAdmin)
	handler(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unauthorized")
}

func TestRequireAnyRole_InvalidUserType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups", nil)
	c.Set(ctxUserKey, "not a user")

	handler := requireAnyRole(auth.RoleAdmin)
	handler(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unauthorized")
}

func TestRequireAnyRole_AllowedRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups", nil)
	c.Set(ctxUserKey, &auth.User{ID: 1, Role: auth.RoleAdmin})

	handler := requireAnyRole(auth.RoleAdmin)
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, c.IsAborted() == false)
}

func TestRequireAnyRole_ForbiddenRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups", nil)
	c.Set(ctxUserKey, &auth.User{ID: 1, Role: auth.RoleStudent})

	handler := requireAnyRole(auth.RoleAdmin)
	handler(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "forbidden")
}

func TestRequireAnyRole_MultipleRoles_Allowed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups", nil)
	c.Set(ctxUserKey, &auth.User{ID: 1, Role: auth.RoleDispatcher})

	handler := requireAnyRole(auth.RoleAdmin, auth.RoleDispatcher)
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdminGateMiddleware_XAdminKey_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokens, err := auth.NewTokenManager("test-secret", time.Hour)
	require.NoError(t, err)

	repo := &schedule.Repository{}
	apiKey := "test-api-key"

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups", nil)
	c.Request.Header.Set("X-Admin-Key", apiKey)

	handler := adminGateMiddleware(apiKey, tokens, repo)
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)

	v, ok := c.Get(ctxUserKey)
	require.True(t, ok)
	u, ok := v.(*auth.User)
	require.True(t, ok)
	assert.Equal(t, int64(0), u.ID)
	assert.Equal(t, "api_key", u.Login)
	assert.Equal(t, auth.RoleAdmin, u.Role)
}

func TestAdminGateMiddleware_NoApiKey_RequiresJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokens, err := auth.NewTokenManager("test-secret", time.Hour)
	require.NoError(t, err)

	repo := &schedule.Repository{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups", nil)

	handler := adminGateMiddleware("", tokens, repo)
	handler(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unauthorized")
}

func TestParseSubjectID_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"42", 42},
		{"0", 0},
		{"999999", 999999},
		{"", 0},
		{"abc", 0},
		{"123abc", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseSubjectID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
