package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"ispo-schedule/internal/config"
)

type CORSConfig = config.CORSConfig
type RateLimitRuleConfig = config.RateLimitRuleConfig

func TestSecurityHeadersMiddleware_Present(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)

	handler := securityHeadersMiddleware()
	handler(c)

	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	// X-Frame-Options может быть DENY или SAMEORIGIN
	assert.Contains(t, []string{"DENY", "SAMEORIGIN"}, w.Header().Get("X-Frame-Options"))
}

func TestCorsMiddleware_NoOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := CORSConfig{
		AllowedOrigins:   []string{},
		AllowCredentials: false,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)

	handler := corsMiddleware(cfg)
	handler(c)

	// Без настроенных origins CORS заголовки не добавляются
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCorsMiddleware_WithOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := CORSConfig{
		AllowedOrigins:   []string{"https://example.com"},
		AllowCredentials: false,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	c.Request.Header.Set("Origin", "https://example.com")

	handler := corsMiddleware(cfg)
	handler(c)

	assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCorsMiddleware_Preflight(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: "GET,POST,PUT,DELETE",
		AllowedHeaders: "Content-Type,Authorization",
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodOptions, "/api/v1/health", nil)
	c.Request.Header.Set("Origin", "https://example.com")
	c.Request.Header.Set("Access-Control-Request-Method", "POST")

	handler := corsMiddleware(cfg)
	handler(c)

	// CORS middleware может не обрабатывать preflight автоматически
	// Проверяем что заголовки установлены
	assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestRequestIDMiddleware_GeneratesID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)

	handler := requestIDMiddleware()
	handler(c)

	requestID := w.Header().Get("X-Request-ID")
	assert.NotEmpty(t, requestID)
}

func TestRequestIDMiddleware_UsesExisting(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	c.Request.Header.Set("X-Request-ID", "existing-id-123")

	handler := requestIDMiddleware()
	handler(c)

	requestID := w.Header().Get("X-Request-ID")
	assert.Equal(t, "existing-id-123", requestID)
}

func TestMaxBodyBytesMiddleware_WithinLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/test", nil)
	c.Request.ContentLength = 100

	handler := maxBodyBytesMiddleware(1024)
	handler(c)

	// Если тело в пределах лимита, запрос продолжается
	assert.Equal(t, http.StatusOK, w.Code)
}

// Note: TestMaxBodyBytesMiddleware_ExceedsLimit требует фактического чтения тела запроса
// и тестируется в integration тестах

func TestRateLimitMiddleware_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := newRateLimitStore(10)
	rule := RateLimitRuleConfig{Enabled: false, RPS: 1, Burst: 1}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.RemoteAddr = "127.0.0.1"

	handler := rateLimitMiddleware(store, rule, "test")
	handler(c)

	// Если rate limit отключён, запрос проходит
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimitMiddleware_WithinLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := newRateLimitStore(10)
	rule := RateLimitRuleConfig{Enabled: true, RPS: 10, Burst: 10}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.RemoteAddr = "127.0.0.1"

	handler := rateLimitMiddleware(store, rule, "test")
	handler(c)

	// Первый запрос должен пройти
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimitMiddleware_ExceedsLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := newRateLimitStore(10)
	rule := RateLimitRuleConfig{Enabled: true, RPS: 0.1, Burst: 1}

	// Первый запрос
	w1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(w1)
	c1.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c1.Request.RemoteAddr = "127.0.0.1"

	handler := rateLimitMiddleware(store, rule, "test")
	handler(c1)

	// Второй запрос сразу же должен быть заблокирован
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c2.Request.RemoteAddr = "127.0.0.1"

	handler(c2)

	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
	assert.Contains(t, w2.Body.String(), "rate_limited")
}
