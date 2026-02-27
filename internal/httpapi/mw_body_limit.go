package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var errRequestBodyTooLarge = errors.New("request body too large")

func maxBodyBytesMiddleware(limitBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limitBytes > 0 {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limitBytes)
		}
		c.Next()
	}
}

func isRequestBodyTooLarge(err error) bool {
	if err == nil {
		return false
	}
	var mbe *http.MaxBytesError
	if errors.As(err, &mbe) {
		return true
	}
	// Some parsers wrap the error, keep a safe fallback.
	return strings.Contains(err.Error(), "request body too large")
}
