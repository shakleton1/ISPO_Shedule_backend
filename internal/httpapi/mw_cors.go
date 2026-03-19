package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"ispo-schedule/internal/config"
)

func corsMiddleware(cfg config.CORSConfig) gin.HandlerFunc {
	allowedOrigins := make(map[string]struct{}, len(cfg.AllowedOrigins))
	allowAll := false
	for _, o := range cfg.AllowedOrigins {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if o == "*" {
			allowAll = true
			continue
		}
		allowedOrigins[o] = struct{}{}
	}

	allowedMethods := strings.TrimSpace(cfg.AllowedMethods)
	if allowedMethods == "" {
		allowedMethods = "GET,POST,PUT,PATCH,DELETE,OPTIONS"
	}

	allowedHeaders := strings.TrimSpace(cfg.AllowedHeaders)
	if allowedHeaders == "" {
		allowedHeaders = "Authorization,Content-Type,X-Request-ID,X-Admin-Key"
	}

	exposedHeaders := strings.TrimSpace(cfg.ExposedHeaders)
	if exposedHeaders == "" {
		exposedHeaders = "X-Request-ID"
	}

	// Safe default: if no allowed origins configured, do nothing (CORS disabled).
	if !allowAll && len(allowedOrigins) == 0 {
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin == "" {
			c.Next()
			return
		}

		originAllowed := allowAll
		if !originAllowed {
			_, originAllowed = allowedOrigins[origin]
		}
		if !originAllowed {
			// No CORS headers for unknown origin.
			c.Next()
			return
		}

		// If credentials are allowed, we must echo origin (cannot use '*').
		if cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		} else {
			if allowAll {
				c.Header("Access-Control-Allow-Origin", "*")
			} else {
				c.Header("Access-Control-Allow-Origin", origin)
			}
		}

		c.Writer.Header().Add("Vary", "Origin")
		c.Header("Access-Control-Allow-Methods", allowedMethods)
		c.Header("Access-Control-Allow-Headers", allowedHeaders)
		c.Header("Access-Control-Expose-Headers", exposedHeaders)

		if c.Request.Method == http.MethodOptions {
			c.Writer.WriteHeader(http.StatusNoContent)
			c.Abort()
			return
		}

		c.Next()
	}
}
