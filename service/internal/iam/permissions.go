package iam

import (
	clientapiconnect "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1/clientapi_pbconnect"
	"github.com/jamesread/armature-iam/rbac"
)

// procedurePermissions maps Connect procedures to the permission they require. Names must match
// rows in rbac_permissions. Unlisted procedures require app.access.
var procedurePermissions = map[string]string{
	// Builds. These are separate from app.access so API keys can be limited to them.
	clientapiconnect.StencilBoxApiServiceStartBuildProcedure:        PermissionBuildsRun,
	clientapiconnect.StencilBoxApiServiceClearBuildCacheProcedure:   PermissionBuildsRun,
	clientapiconnect.StencilBoxApiServiceGetBuildConfigsProcedure:   PermissionBuildConfigsView,
	clientapiconnect.StencilBoxApiServiceGetBuildConfigProcedure:    PermissionBuildConfigsView,
	clientapiconnect.StencilBoxApiServiceGetBuildHistoryProcedure:   PermissionBuildConfigsView,
	clientapiconnect.StencilBoxApiServiceListDataFilesProcedure:     PermissionBuildConfigsView,
	clientapiconnect.StencilBoxApiServiceGetDataFileProcedure:       PermissionBuildConfigsView,
	clientapiconnect.StencilBoxApiServiceGetTemplatesProcedure:      PermissionBuildConfigsView,
	clientapiconnect.StencilBoxApiServiceGetTemplateProcedure:       PermissionBuildConfigsView,
	clientapiconnect.StencilBoxApiServiceUpdateDataFileProcedure:    PermissionDataFilesEdit,
	clientapiconnect.StencilBoxApiServiceCreateBuildConfigProcedure: PermissionBuildConfigsManage,
	clientapiconnect.StencilBoxApiServiceCreateDataFileProcedure:    PermissionBuildConfigsManage,
	clientapiconnect.StencilBoxApiServiceGitPullProcedure:           PermissionBuildConfigsGitPull,

	// Users. GetUsers stays on app.access so pickers work; details need users.view.
	clientapiconnect.StencilBoxApiServiceGetUserProcedure:           rbac.PermissionUsersView,
	clientapiconnect.StencilBoxApiServiceGetUserAccessProcedure:     rbac.PermissionUsersView,
	clientapiconnect.StencilBoxApiServiceCreateUserProcedure:        rbac.PermissionUsersCreate,
	clientapiconnect.StencilBoxApiServiceDeleteUserProcedure:        rbac.PermissionUsersDelete,
	clientapiconnect.StencilBoxApiServiceResetUserPasswordProcedure: rbac.PermissionUsersResetPassword,

	// Groups and membership.
	clientapiconnect.StencilBoxApiServiceListUserGroupsProcedure:      rbac.PermissionUserGroupsView,
	clientapiconnect.StencilBoxApiServiceGetUserGroupMembersProcedure: rbac.PermissionUserGroupsView,
	clientapiconnect.StencilBoxApiServiceCreateUserGroupProcedure:     rbac.PermissionUserGroupsManage,
	clientapiconnect.StencilBoxApiServiceDeleteUserGroupProcedure:     rbac.PermissionUserGroupsManage,
	clientapiconnect.StencilBoxApiServiceSetUserGroupMembersProcedure: rbac.PermissionUserGroupsManage,
	clientapiconnect.StencilBoxApiServiceSetUserGroupsProcedure:       rbac.PermissionUserGroupsManage,

	// Roles and permissions.
	clientapiconnect.StencilBoxApiServiceListRbacPermissionsProcedure:   rbac.PermissionRbacView,
	clientapiconnect.StencilBoxApiServiceListRbacRolesProcedure:         rbac.PermissionRbacView,
	clientapiconnect.StencilBoxApiServiceGetUserGroupRbacRolesProcedure: rbac.PermissionRbacView,
	clientapiconnect.StencilBoxApiServiceGetRbacRoleGroupsProcedure:     rbac.PermissionRbacView,
	clientapiconnect.StencilBoxApiServiceCreateRbacRoleProcedure:        rbac.PermissionRbacManage,
	clientapiconnect.StencilBoxApiServiceUpdateRbacRoleProcedure:        rbac.PermissionRbacManage,
	clientapiconnect.StencilBoxApiServiceDeleteRbacRoleProcedure:        rbac.PermissionRbacManage,
	clientapiconnect.StencilBoxApiServiceSetUserGroupRbacRolesProcedure: rbac.PermissionRbacManage,
}

// RequiredPermission returns the permission a procedure needs ("" means any authenticated user).
// Procedures in AllowUnauthenticated never reach this check.
func RequiredPermission(procedure string) string {
	if p, ok := procedurePermissions[procedure]; ok {
		return p
	}
	return rbac.PermissionAppAccess
}

// SessionOnlyProcedures cannot be called with an API key, so a narrowly scoped key cannot mint a
// broader one or take over its owner's account.
var SessionOnlyProcedures = map[string]bool{
	clientapiconnect.StencilBoxApiServiceCreateApiKeyProcedure:   true,
	clientapiconnect.StencilBoxApiServiceUpdateApiKeyProcedure:   true,
	clientapiconnect.StencilBoxApiServiceDeleteApiKeyProcedure:   true,
	clientapiconnect.StencilBoxApiServiceChangePasswordProcedure: true,
}
