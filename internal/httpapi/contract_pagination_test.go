package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseLimitOffset_NoLimit_DisablesPaginationAndIgnoresOffset(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x?offset=10", nil)

	got, ok := parseLimitOffset(c, nil, 500)
	if !ok {
		t.Fatalf("expected ok")
	}
	if got.Limit != nil {
		t.Fatalf("expected nil limit, got %v", *got.Limit)
	}
	if got.Offset != nil {
		t.Fatalf("expected nil offset when limit is nil, got %v", *got.Offset)
	}
}

func TestParseLimitOffset_LimitSetsDefaultOffset(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x?limit=10", nil)

	got, ok := parseLimitOffset(c, nil, 500)
	if !ok {
		t.Fatalf("expected ok")
	}
	if got.Limit == nil || *got.Limit != 10 {
		t.Fatalf("expected limit=10, got %+v", got.Limit)
	}
	if got.Offset == nil || *got.Offset != 0 {
		t.Fatalf("expected offset=0, got %+v", got.Offset)
	}
}

func TestParseLimitOffset_InvalidLimit_ReturnsUnifiedValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x?limit=0", nil)

	_, ok := parseLimitOffset(c, nil, 500)
	if ok {
		t.Fatalf("expected not ok")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp apiErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != "validation_error" {
		t.Fatalf("expected code validation_error, got %q", resp.Code)
	}
	if resp.Field != "limit" {
		t.Fatalf("expected field limit, got %q", resp.Field)
	}
	if resp.Message == "" {
		t.Fatalf("expected message")
	}
}
