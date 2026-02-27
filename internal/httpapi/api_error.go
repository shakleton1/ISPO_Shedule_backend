package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type apiErrorResponse struct {
	Code    string `json:"code"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

func writeError(c *gin.Context, status int, code, field, message string) {
	code = strings.TrimSpace(code)
	field = strings.TrimSpace(field)
	message = strings.TrimSpace(message)
	if code == "" {
		code = "error"
	}
	if message == "" {
		message = http.StatusText(status)
	}
	c.JSON(status, apiErrorResponse{Code: code, Field: field, Message: message})
}

func abortWithError(c *gin.Context, status int, code, field, message string) {
	writeError(c, status, code, field, message)
	c.Abort()
}

func writeInvalidJSON(c *gin.Context) {
	writeError(c, http.StatusBadRequest, "invalid_json", "", "invalid json")
}

func writeValidationError(c *gin.Context, field, message string) {
	writeError(c, http.StatusBadRequest, "validation_error", field, message)
}

func writeUnauthorized(c *gin.Context, message string) {
	if strings.TrimSpace(message) == "" {
		message = "unauthorized"
	}
	writeError(c, http.StatusUnauthorized, "unauthorized", "", message)
}

func writeForbidden(c *gin.Context, message string) {
	if strings.TrimSpace(message) == "" {
		message = "forbidden"
	}
	writeError(c, http.StatusForbidden, "forbidden", "", message)
}

func writeNotFound(c *gin.Context, message string) {
	if strings.TrimSpace(message) == "" {
		message = "not found"
	}
	writeError(c, http.StatusNotFound, "not_found", "", message)
}

func writeDBError(c *gin.Context, err error) {
	if err == nil {
		writeError(c, http.StatusInternalServerError, "internal", "", "internal error")
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		writeNotFound(c, "not found")
		return
	}
	// Keep current behavior (400) but with structured error.
	writeError(c, http.StatusBadRequest, "bad_request", "", err.Error())
}
