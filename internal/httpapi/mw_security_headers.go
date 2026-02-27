package httpapi

import (
	"github.com/gin-gonic/gin"
)

func securityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Minimal, safe defaults. Avoid CSP here to not break Swagger UI assets.
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-DNS-Prefetch-Control", "off")
		c.Next()
	}
}
