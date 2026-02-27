package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func adminAuthMiddleware(apiKey string) gin.HandlerFunc {
	// Deprecated: kept for backward compatibility in case older code references it.
	if apiKey == "" {
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		if c.GetHeader("X-Admin-Key") != apiKey {
			abortWithError(c, http.StatusUnauthorized, "unauthorized", "X-Admin-Key", "unauthorized")
			return
		}
		c.Next()
	}
}
