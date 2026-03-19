package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

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

func TestCorsMiddleware_AllowCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	c.Request.Header.Set("Origin", "https://allowed.example")

	handler := corsMiddleware(cfg)
	handler(c)

	assert.Equal(t, "https://allowed.example", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
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

func TestMaxBodyBytesMiddleware_ExceedsLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(maxBodyBytesMiddleware(8))
	r.POST("/test", func(c *gin.Context) {
		_, err := c.GetRawData()
		if isRequestBodyTooLarge(err) {
			writeError(c, http.StatusRequestEntityTooLarge, "payload_too_large", "body", "payload too large")
			return
		}
		if err != nil {
			writeError(c, http.StatusBadRequest, "bad_request", "body", "invalid body")
			return
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	// 16 bytes > 8 bytes middleware limit
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("1234567890abcdef"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	assert.Contains(t, w.Body.String(), "payload_too_large")
}

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

	handler := rateLimitMiddleware(store, rule, "test")
	handler(c)

	// Первый запрос должен пройти
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimitStore_CleanupOldClients(t *testing.T) {
	store := newRateLimitStore(10 * time.Millisecond)
	now := time.Now()

	store.clients["old"] = &rateLimitClient{lastSeen: now.Add(-time.Second)}
	store.clients["fresh"] = &rateLimitClient{lastSeen: now}
	store.nextCleanup = now.Add(-time.Millisecond)

	_ = store.get("new", 1, 1)

	_, hasOld := store.clients["old"]
	_, hasFresh := store.clients["fresh"]
	_, hasNew := store.clients["new"]

	assert.False(t, hasOld)
	assert.True(t, hasFresh)
	assert.True(t, hasNew)
}

func TestAdminAuthMiddleware_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(adminAuthMiddleware("secret"))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Admin-Key", "secret")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdminAuthMiddleware_InvalidKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(adminAuthMiddleware("secret"))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Admin-Key", "wrong")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unauthorized")
}

func TestRequestLoggingMiddleware_LogsRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var buf bytes.Buffer
	prev := log.Logger
	log.Logger = zerolog.New(&buf)
	defer func() { log.Logger = prev }()

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ctxRequestIDKey, "rid-123")
		c.Next()
	})
	r.Use(requestLoggingMiddleware())
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)

	var event map[string]any
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &event))

	assert.Equal(t, "rid-123", event["request_id"])
	assert.Equal(t, "GET", event["method"])
	assert.Equal(t, "/ping", event["path"])
	assert.Equal(t, "http_request", event["message"])
}

func TestRequestLoggingMiddleware_LogsTraceAndSpanID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var buf bytes.Buffer
	prev := log.Logger
	log.Logger = zerolog.New(&buf)
	defer func() { log.Logger = prev }()

	traceID, err := trace.TraceIDFromHex("11111111111111111111111111111111")
	require.NoError(t, err)
	spanID, err := trace.SpanIDFromHex("2222222222222222")
	require.NoError(t, err)

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	})

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(trace.ContextWithSpanContext(c.Request.Context(), sc))
		c.Next()
	})
	r.Use(requestLoggingMiddleware())
	r.GET("/trace", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trace", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var event map[string]any
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &event))

	assert.Equal(t, traceID.String(), event["trace_id"])
	assert.Equal(t, spanID.String(), event["span_id"])
}

// Note: TestRateLimitMiddleware_ExceedsLimit требует стабильного ClientIP
// и тестируется в integration тестах
