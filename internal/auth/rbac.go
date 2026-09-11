package auth

import (
	"crypto/rand"
	"fmt"
)

// RBAC roles (spec section 13.3).
type Role string

const (
	RoleSuperAdmin Role = "super_admin"
	RoleAdmin      Role = "admin"
	RoleOperator   Role = "operator"
	RoleAuditor    Role = "auditor"
	RoleViewer     Role = "viewer"
)

// Permission strings. Enforcement happens in backend services and API
// middleware, never only in the UI (spec section 13.3, rule 10 of 52).
const (
	PermUsersManage     = "users.manage"
	PermFirewallRead    = "firewall.read"
	PermFirewallWrite   = "firewall.write"
	PermFirewallApply   = "firewall.apply"
	PermFirewallRestore = "firewall.restore"
	PermTemplatesManage = "templates.manage"
	PermDiagnosticsRun  = "diagnostics.run"
	PermServicesManage  = "services.manage"
	PermSettingsRead    = "settings.read"
	PermSettingsWrite   = "settings.write"
	PermAuditRead       = "audit.read"
	PermLogsRead        = "logs.read"
)

// rolePermissions maps each role to its capability set. Super admin gets
// everything implicitly.
var rolePermissions = map[Role]map[string]bool{
	RoleAdmin: {
		PermUsersManage: true, PermFirewallRead: true, PermFirewallWrite: true,
		PermFirewallApply: true, PermFirewallRestore: true, PermTemplatesManage: true,
		PermDiagnosticsRun: true, PermServicesManage: true,
		PermSettingsRead: true, PermSettingsWrite: true, PermAuditRead: true, PermLogsRead: true,
	},
	RoleOperator: {
		PermFirewallRead: true, PermFirewallWrite: true, PermFirewallApply: true,
		PermFirewallRestore: true, PermTemplatesManage: true, PermDiagnosticsRun: true,
		PermServicesManage: true, PermSettingsRead: true, PermLogsRead: true,
	},
	RoleAuditor: {
		PermFirewallRead: true, PermSettingsRead: true, PermAuditRead: true, PermLogsRead: true,
	},
	RoleViewer: {
		PermFirewallRead: true, PermSettingsRead: true,
	},
}

// HasPermission checks a role against a permission.
func HasPermission(role Role, perm string) bool {
	if role == RoleSuperAdmin {
		return true
	}
	return rolePermissions[role][perm]
}

// ValidRole reports whether r is a known role.
func ValidRole(r Role) bool {
	switch r {
	case RoleSuperAdmin, RoleAdmin, RoleOperator, RoleAuditor, RoleViewer:
		return true
	}
	return false
}

func readRandom(b []byte) {
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failing is unrecoverable for auth material.
		panic(fmt.Sprintf("crypto/rand unavailable: %v", err))
	}
}

// RandomTokenHex returns n random bytes hex-encoded, for session secrets
// and enrollment material.
func RandomTokenHex(n int) string {
	b := make([]byte, n)
	readRandom(b)
	return fmt.Sprintf("%x", b)
}
