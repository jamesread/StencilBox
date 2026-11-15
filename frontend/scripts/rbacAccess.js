// RBAC helpers for the web UI. These only hide UI: every RPC is also checked by the server
// (service/internal/iam/permissions.go). Keep the permission sets here in sync with Go.

export const IAM_PERMISSIONS = [
	'users.view',
	'usergroups.view',
	'rbac.view',
]

// The System Control Panel is shown to anyone who can use one of its children.
export const CONTROL_PANEL_PERMISSIONS = [...IAM_PERMISSIONS]

export const SYSTEM_ROLES = new Set(['superuser', 'member'])
export const SYSTEM_GROUPS = new Set(['Everyone', 'Administrators'])

export function hasPermission(st, name) {
	if (!st?.isLoggedIn) {
		return false
	}
	return Boolean(st.rbacIsSuperuser) || (st.rbacPermissions || []).includes(name)
}

export function hasAnyPermission(st, names) {
	return names.some((name) => hasPermission(st, name))
}

export function canAccessIamFromStatus(st) {
	return hasAnyPermission(st, IAM_PERMISSIONS)
}

export function canAccessControlPanelFromStatus(st) {
	return hasAnyPermission(st, CONTROL_PANEL_PERMISSIONS)
}

export function isSystemRole(name) {
	return SYSTEM_ROLES.has(name)
}

export function isSystemGroup(name) {
	return SYSTEM_GROUPS.has(name)
}
