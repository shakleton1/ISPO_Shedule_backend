package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestWriteError_Basic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	writeError(c, http.StatusBadRequest, "validation_error", "field_name", "Field is required")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "validation_error")
	assert.Contains(t, w.Body.String(), "field_name")
	assert.Contains(t, w.Body.String(), "Field is required")
}

func TestWriteError_EmptyCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	writeError(c, http.StatusBadRequest, "", "", "Message")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "error") // Default code
}

func TestWriteError_EmptyMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	writeError(c, http.StatusBadRequest, "validation_error", "", "")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Bad Request") // Default message from status
}

func TestWriteInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	writeInvalidJSON(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_json")
}

func TestWriteValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	writeValidationError(c, "email", "Invalid email format")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "validation_error")
	assert.Contains(t, w.Body.String(), "email")
	assert.Contains(t, w.Body.String(), "Invalid email format")
}

func TestWriteUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	writeUnauthorized(c, "Token expired")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unauthorized")
	assert.Contains(t, w.Body.String(), "Token expired")
}

func TestWriteUnauthorized_EmptyMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	writeUnauthorized(c, "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unauthorized")
}

func TestAbortWithError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	abortWithError(c, http.StatusForbidden, "forbidden", "", "Access denied")

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "forbidden")
	assert.Contains(t, w.Body.String(), "Access denied")
	assert.True(t, c.IsAborted())
}

// Note: writeDBError requires gorm errors for proper handling
// Integration tests cover DB error scenarios
