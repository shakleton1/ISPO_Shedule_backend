package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWriteError_UnifiedShape(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	writeError(c, http.StatusBadRequest, "validation_error", "field_x", "bad")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := got["code"]; !ok {
		t.Fatalf("expected key code")
	}
	if _, ok := got["field"]; !ok {
		t.Fatalf("expected key field")
	}
	if _, ok := got["message"]; !ok {
		t.Fatalf("expected key message")
	}

	if got["code"] != "validation_error" {
		t.Fatalf("expected code validation_error, got %#v", got["code"])
	}
	if got["field"] != "field_x" {
		t.Fatalf("expected field field_x, got %#v", got["field"])
	}
	if got["message"] != "bad" {
		t.Fatalf("expected message bad, got %#v", got["message"])
	}

	// Legacy shape should not appear.
	if _, ok := got["error"]; ok {
		t.Fatalf("did not expect legacy key error")
	}
}
