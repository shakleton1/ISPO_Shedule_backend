package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func adminAuthMiddleware(apiKey string) gin.HandlerFunc {
	if apiKey == "" {
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		if c.GetHeader("X-Admin-Key") != apiKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}
