package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ispo-schedule/internal/auth"
)

type Permission string

const (
	PermAdminRead     Permission = "admin:read"
	PermDictWrite     Permission = "dict:write"
	PermScheduleWrite Permission = "schedule:write"
	PermImport        Permission = "import:templates"
)

func rolePermissions(role auth.Role) map[Permission]struct{} {
	perms := map[Permission]struct{}{}
	grant := func(p Permission) { perms[p] = struct{}{} }

	switch role {
	case auth.RoleAdmin:
		grant(PermAdminRead)
		grant(PermDictWrite)
		grant(PermScheduleWrite)
		grant(PermImport)
	case auth.RoleDispatcher:
		grant(PermAdminRead)
		grant(PermScheduleWrite)
		// Import stays admin-only by default.
	case auth.RoleViewer:
		grant(PermAdminRead)
	case auth.RoleStudent:
		// no admin permissions
	}
	return perms
}

func requireAnyPermission(perms ...Permission) gin.HandlerFunc {
	allowed := map[Permission]struct{}{}
	for _, p := range perms {
		allowed[p] = struct{}{}
	}
	return func(c *gin.Context) {
		v, ok := c.Get(ctxUserKey)
		if !ok {
			abortWithError(c, http.StatusUnauthorized, "unauthorized", "", "unauthorized")
			return
		}
		u, ok := v.(*auth.User)
		if !ok || u == nil {
			abortWithError(c, http.StatusUnauthorized, "unauthorized", "", "unauthorized")
			return
		}
		userPerms := rolePermissions(u.Role)
		for p := range allowed {
			if _, ok := userPerms[p]; ok {
				c.Next()
				return
			}
		}
		abortWithError(c, http.StatusForbidden, "forbidden", "", "forbidden")
	}
}
