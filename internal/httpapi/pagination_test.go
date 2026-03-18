package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Note: Большинство pagination тестов уже есть в contract_pagination_test.go
// Здесь только дополнительные сценарии

func TestParseLimitOffset_InvalidLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/test?limit=invalid", nil)

	page, ok := parseLimitOffset(c, nil, 50)

	assert.False(t, ok)
	assert.Nil(t, page.Limit)
}

func TestParseLimitOffset_InvalidOffset(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/test?limit=10&offset=invalid", nil)

	page, ok := parseLimitOffset(c, nil, 50)

	assert.False(t, ok)
	assert.Nil(t, page.Limit)
}

func TestParseLimitOffset_NegativeLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/test?limit=-5", nil)

	page, ok := parseLimitOffset(c, nil, 50)

	assert.False(t, ok)
	assert.Nil(t, page.Limit)
}

func TestParseLimitOffset_NegativeOffset(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/test?limit=10&offset=-5", nil)

	page, ok := parseLimitOffset(c, nil, 50)

	assert.False(t, ok)
	assert.Nil(t, page.Limit)
}

func TestParseLimitOffset_ExceedsMaxLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/test?limit=1000", nil)

	page, ok := parseLimitOffset(c, nil, 50)

	assert.True(t, ok)
	assert.NotNil(t, page.Limit)
	assert.Equal(t, 50, *page.Limit) // Should be capped at max
}

func TestParseLimitOffset_ZeroLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/test?limit=0", nil)

	page, ok := parseLimitOffset(c, nil, 50)

	assert.False(t, ok)
	assert.Nil(t, page.Limit)
}
