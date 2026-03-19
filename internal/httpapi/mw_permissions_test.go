package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"ispo-schedule/internal/auth"
)

func TestRolePermissions_AllKnownRoles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		role     auth.Role
		expected map[Permission]struct{}
	}{
		{
			name: "admin",
			role: auth.RoleAdmin,
			expected: map[Permission]struct{}{
				PermAdminRead:     {},
				PermDictWrite:     {},
				PermScheduleWrite: {},
				PermImport:        {},
			},
		},
		{
			name: "dispatcher",
			role: auth.RoleDispatcher,
			expected: map[Permission]struct{}{
				PermAdminRead:     {},
				PermScheduleWrite: {},
			},
		},
		{
			name: "viewer",
			role: auth.RoleViewer,
			expected: map[Permission]struct{}{
				PermAdminRead: {},
			},
		},
		{
			name:     "student",
			role:     auth.RoleStudent,
			expected: map[Permission]struct{}{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, rolePermissions(tc.role))
		})
	}
}

func TestRolePermissions_UnknownRole(t *testing.T) {
	t.Parallel()

	perms := rolePermissions(auth.Role("unknown"))
	assert.Empty(t, perms)
}
