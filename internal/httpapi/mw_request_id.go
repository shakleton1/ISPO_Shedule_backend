package httpapi

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

const ctxRequestIDKey = "request.id"

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			b := make([]byte, 16)
			if _, err := rand.Read(b); err == nil {
				rid = hex.EncodeToString(b)
			}
		}
		if rid != "" {
			c.Set(ctxRequestIDKey, rid)
			c.Header("X-Request-ID", rid)
		}
		c.Next()
	}
}

func requestIDFromContext(c *gin.Context) string {
	if v, ok := c.Get(ctxRequestIDKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
