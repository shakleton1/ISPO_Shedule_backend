//go:build integration

package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ispo-schedule/internal/auth"
	"ispo-schedule/internal/httpapi"
	"ispo-schedule/internal/schedule"
)

func TestAuthMiddleware_Integration_UserNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	gin.SetMode(gin.TestMode)

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	tokens, err := auth.NewTokenManager("test-secret", time.Hour)
	require.NoError(t, err)

	// Создаём токен для несуществующего пользователя
	user := &auth.User{ID: 999999, Role: auth.RoleAdmin}
	token, _, err := tokens.IssueAccessToken(user, time.Now())
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	httpapi.AuthMiddlewareForTest(tokens, repo)(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unauthorized")
}

func TestAuthMiddleware_Integration_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	gin.SetMode(gin.TestMode)

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	tokens, err := auth.NewTokenManager("test-secret", time.Hour)
	require.NoError(t, err)

	// Создаём реального пользователя в БД
	testUser := &auth.User{
		Login:        "test_middleware_user",
		PasswordHash: "hash",
		Role:         auth.RoleAdmin,
	}

	// Очищаем если существует
	db.Where("login = ?", testUser.Login).Delete(&auth.User{})

	err = repo.CreateUser(testUser)
	require.NoError(t, err)
	defer db.Where("login = ?", testUser.Login).Delete(&auth.User{})

	// Создаём токен
	token, _, err := tokens.IssueAccessToken(testUser, time.Now())
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	httpapi.AuthMiddlewareForTest(tokens, repo)(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// Проверяем что пользователь установлен в контекст
	v, ok := c.Get("auth.user")
	require.True(t, ok)
	u, ok := v.(*auth.User)
	require.True(t, ok)
	assert.Equal(t, testUser.ID, u.ID)
	assert.Equal(t, testUser.Role, u.Role)
}

func TestService_GetCurrentWeek_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	svc := schedule.NewService(schedule.ServiceDeps{
		Repo:              repo,
		SemesterStartDate: "2026-02-09",
		Now:               time.Now,
	})

	// Создаём тестовую группу
	group := &schedule.Group{
		Name:   "Integration Test Group",
		Course: 1,
	}
	db.Where("name = ?", group.Name).Delete(&schedule.Group{})
	err := db.Create(group).Error
	require.NoError(t, err)
	defer db.Where("name = ?", group.Name).Delete(&schedule.Group{})

	refDate := time.Date(2026, 2, 26, 12, 0, 0, 0, time.UTC) // Thursday
	resp, err := svc.GetCurrentWeek(group.ID, refDate)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, group.ID, resp.GroupID)
	assert.Equal(t, "2026-02-23", resp.DateStart) // Monday
	assert.Equal(t, "2026-02-28", resp.DateEnd)   // Saturday
	assert.NotEmpty(t, resp.DataVersion)
}

func TestService_GetRange_InvalidGroupID_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	svc := schedule.NewService(schedule.ServiceDeps{
		Repo:              repo,
		SemesterStartDate: "2026-02-09",
		Now:               time.Now,
	})

	resp, err := svc.GetRange(0, time.Now(), time.Now())

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "group_id required")
}

func TestService_GetRange_EndDateBeforeStartDate_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	svc := schedule.NewService(schedule.ServiceDeps{
		Repo:              repo,
		SemesterStartDate: "2026-02-09",
		Now:               time.Now,
	})

	startDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 2, 24, 0, 0, 0, 0, time.UTC)

	resp, err := svc.GetRange(1, startDate, endDate)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "date_end before date_start")
}

func TestService_GetRange_GroupNotFound_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := getTestDB(t)
	repo := schedule.NewRepository(db)

	svc := schedule.NewService(schedule.ServiceDeps{
		Repo:              repo,
		SemesterStartDate: "2026-02-09",
		Now:               time.Now,
	})

	// Group 999999 does not exist
	resp, err := svc.GetRange(999999, time.Now(), time.Now())

	assert.Error(t, err)
	assert.Nil(t, resp)
}
