package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"ispo-schedule/internal/auth"
	"ispo-schedule/internal/schedule"
)

const ctxUserKey = "auth.user"

func authMiddleware(tokens *auth.TokenManager, repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		ah := c.GetHeader("Authorization")
		if ah == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
			return
		}
		parts := strings.SplitN(ah, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid Authorization header"})
			return
		}
		claims, err := tokens.Parse(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		// Load user for existence check (and future revocation).
		u, err := repo.GetUserByID(parseSubjectID(claims.Subject))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unknown user"})
			return
		}
		c.Set(ctxUserKey, u)
		c.Next()
	}
}

func requireAnyRole(roles ...auth.Role) gin.HandlerFunc {
	allowed := map[auth.Role]struct{}{}
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		v, ok := c.Get(ctxUserKey)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		u, ok := v.(*auth.User)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if _, ok := allowed[u.Role]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}

func parseSubjectID(sub string) int64 {
	// subject is always issued as int64 string; on error -> 0
	var id int64
	for i := 0; i < len(sub); i++ {
		ch := sub[i]
		if ch < '0' || ch > '9' {
			return 0
		}
		id = id*10 + int64(ch-'0')
	}
	return id
}

// adminGateMiddleware allows either:
// - X-Admin-Key (if configured), OR
// - JWT auth
//
// RBAC role checks are enforced per-route in the admin router.
func adminGateMiddleware(apiKey string, tokens *auth.TokenManager, repo *schedule.Repository) gin.HandlerFunc {
	if apiKey != "" {
		return func(c *gin.Context) {
			if c.GetHeader("X-Admin-Key") == apiKey {
				// Synthetic user for audit/RBAC (treated as admin).
				c.Set(ctxUserKey, &auth.User{ID: 0, Login: "api_key", Role: auth.RoleAdmin})
				c.Next()
				return
			}
			// fallback to JWT
			authMiddleware(tokens, repo)(c)
			if c.IsAborted() {
				return
			}
			c.Next()
		}
	}
	return func(c *gin.Context) {
		authMiddleware(tokens, repo)(c)
		if c.IsAborted() {
			return
		}
		c.Next()
	}
}
