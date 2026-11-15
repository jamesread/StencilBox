package clientapi

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	pb "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1"
	"github.com/jamesread/StencilBox/internal/iam"
	"github.com/jamesread/StencilBox/internal/reqctx"
	armlayer "github.com/jamesread/armature-iam/layer"
	armpassword "github.com/jamesread/armature-iam/password"
	"github.com/jamesread/armature-iam/rbac"
	"github.com/jamesread/armature-iam/store"
	log "github.com/sirupsen/logrus"
)

// loginFailedMessage is deliberately the same for unknown users and bad passwords.
const loginFailedMessage = "Invalid username or password"

var (
	errAuthRequired = connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	errIamDisabled  = connect.NewError(connect.CodeUnavailable, errors.New("this needs a database, and StencilBox is running without one"))
)

// dummyHash is verified against when a login names an unknown user, so the response takes as
// long as a wrong password and does not reveal which usernames exist.
var dummyHash = sync.OnceValue(func() string {
	h, _ := armpassword.Hash(uuid.NewString())
	return h
})

func authUser(ctx context.Context) *armlayer.AuthenticatedUser {
	au := armlayer.UserFromContext(ctx)
	if au == nil || au.User == nil {
		return nil
	}
	return au
}

func (c *ClientApi) iamStore() (store.Store, error) {
	if c.auth == nil {
		return nil, errIamDisabled
	}
	return c.auth.Store, nil
}

func toInt32(v int) int32 {
	if v > math.MaxInt32 || v < math.MinInt32 {
		return 0
	}
	return int32(v)
}

func intsToInt32(in []int) []int32 {
	out := make([]int32, 0, len(in))
	for _, v := range in {
		out = append(out, toInt32(v))
	}
	return out
}

func int32sToInt(in []int32) []int {
	out := make([]int, 0, len(in))
	for _, v := range in {
		out = append(out, int(v))
	}
	return out
}

func internalErr(op string, err error) error {
	log.WithError(err).Errorf("IAM: %s", op)
	return connect.NewError(connect.CodeInternal, fmt.Errorf("%s failed", op))
}

// storeErr maps armature-iam guard errors (system role/group, last superuser) to InvalidArgument
// so their message reaches the UI.
func storeErr(op string, err error) error {
	for _, guard := range []error{store.ErrNoSuperuser, store.ErrSystemRole, store.ErrSystemGroup, store.ErrRenameSystemRole} {
		if errors.Is(err, guard) {
			return connect.NewError(connect.CodeInvalidArgument, err)
		}
	}
	return internalErr(op, err)
}

func requireID(name string, id int32) (int, error) {
	if id <= 0 {
		return 0, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%s is required", name))
	}
	return int(id), nil
}

func passwordProblem(password string) string {
	if len(password) < iam.MinPasswordLength {
		return fmt.Sprintf("password must be at least %d characters", iam.MinPasswordLength)
	}
	return ""
}

// rbacPermissionNames returns the user's effective permissions, sorted. A superuser gets the
// whole catalog.
func (c *ClientApi) rbacPermissionNames(ctx context.Context, au *armlayer.AuthenticatedUser) []string {
	names := []string{}
	if au == nil || au.RBAC == nil {
		return names
	}
	if au.RBAC.IsSuperuser && c.auth != nil {
		perms, err := c.auth.Store.ListRBACPermissions(ctx)
		if err == nil {
			for _, p := range perms {
				names = append(names, p.Name)
			}
		}
	} else {
		for name, ok := range au.RBAC.Permissions {
			if ok {
				names = append(names, name)
			}
		}
	}
	sort.Strings(names)
	return names
}

func (c *ClientApi) refreshDefaultAdminPassword(ctx context.Context) {
	if c.auth == nil {
		return
	}
	c.defaultAdminPassword.Store(iam.UsesDefaultAdminPassword(ctx, c.auth.Store))
}

func sessionIDFromHeader(header http.Header) string {
	cookie, err := (&http.Request{Header: header}).Cookie(iam.CookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// ---- Authentication ----

func (c *ClientApi) LoginWithUsernameAndPassword(ctx context.Context, req *connect.Request[pb.LoginWithUsernameAndPasswordRequest]) (*connect.Response[pb.LoginWithUsernameAndPasswordResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	username := strings.TrimSpace(req.Msg.GetUsername())
	password := req.Msg.GetPassword()
	fail := connect.NewResponse(&pb.LoginWithUsernameAndPasswordResponse{Success: false, Message: loginFailedMessage})
	if username == "" || password == "" {
		return fail, nil
	}

	user, err := st.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, internalErr("login", err)
	}
	hash := dummyHash()
	if user != nil && user.PasswordHash != "" {
		hash = user.PasswordHash
	}
	ok, err := armpassword.Verify(hash, password)
	if err != nil || !ok || user == nil || user.PasswordHash == "" {
		log.WithField("username", username).Info("Login failed")
		return fail, nil
	}

	sid := uuid.NewString()
	if err := st.CreateSession(ctx, sid, user.ID, nil); err != nil {
		return nil, internalErr("create session", err)
	}
	log.WithField("username", user.Username).Info("Login succeeded")

	res := connect.NewResponse(&pb.LoginWithUsernameAndPasswordResponse{Success: true, Message: "Logged in"})
	cookie := c.auth.SessionCookie(sid, reqctx.HttpRequest(ctx), req.Header())
	res.Header().Add("Set-Cookie", cookie.String())
	return res, nil
}

func (c *ClientApi) Logout(ctx context.Context, req *connect.Request[pb.LogoutRequest]) (*connect.Response[pb.LogoutResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	if sid := sessionIDFromHeader(req.Header()); sid != "" {
		if err := st.DeleteSession(ctx, sid); err != nil {
			return nil, internalErr("delete session", err)
		}
	}
	res := connect.NewResponse(&pb.LogoutResponse{Success: true})
	cookie := c.auth.ClearCookie(reqctx.HttpRequest(ctx), req.Header())
	res.Header().Add("Set-Cookie", cookie.String())
	return res, nil
}

func (c *ClientApi) ChangePassword(ctx context.Context, req *connect.Request[pb.ChangePasswordRequest]) (*connect.Response[pb.ChangePasswordResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	au := authUser(ctx)
	if au == nil || au.User.ID <= 0 {
		return nil, errAuthRequired
	}
	reply := func(success bool, msg string) (*connect.Response[pb.ChangePasswordResponse], error) {
		return connect.NewResponse(&pb.ChangePasswordResponse{Success: success, Message: msg}), nil
	}
	if problem := passwordProblem(req.Msg.GetNewPassword()); problem != "" {
		return reply(false, problem)
	}
	user, err := st.GetUserByID(ctx, au.User.ID)
	if err != nil || user == nil {
		return nil, internalErr("load user", err)
	}
	// Accounts without a password (SSO, API key only) may set a first one.
	if user.PasswordHash != "" {
		ok, err := armpassword.Verify(user.PasswordHash, req.Msg.GetCurrentPassword())
		if err != nil || !ok {
			return reply(false, "current password is incorrect")
		}
	}
	hash, err := armpassword.Hash(req.Msg.GetNewPassword())
	if err != nil {
		return nil, internalErr("hash password", err)
	}
	if err := st.UpdateUserPassword(ctx, user.ID, hash); err != nil {
		return nil, internalErr("update password", err)
	}
	c.refreshDefaultAdminPassword(ctx)
	return reply(true, "Password changed")
}

// ---- Users ----

func iamUserToProto(u *store.UserAccountRow) *pb.IamUser {
	return &pb.IamUser{
		Id:          toInt32(u.ID),
		Username:    u.Username,
		CreatedBy:   u.CreatedBy,
		HasPassword: u.PasswordHash != "",
	}
}

func (c *ClientApi) GetUsers(ctx context.Context, _ *connect.Request[pb.GetUsersRequest]) (*connect.Response[pb.GetUsersResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	rows, err := st.ListUserAccounts(ctx)
	if err != nil {
		return nil, internalErr("list users", err)
	}
	users := make([]*pb.IamUser, 0, len(rows))
	for i := range rows {
		// Pickers only need id and username; details stay behind users.view on GetUser.
		users = append(users, &pb.IamUser{Id: toInt32(rows[i].ID), Username: rows[i].Username})
	}
	return connect.NewResponse(&pb.GetUsersResponse{Users: users}), nil
}

func (c *ClientApi) GetUser(ctx context.Context, req *connect.Request[pb.GetUserRequest]) (*connect.Response[pb.GetUserResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	id, err := requireID("user_id", req.Msg.GetUserId())
	if err != nil {
		return nil, err
	}
	user, err := st.GetUserByID(ctx, id)
	if err != nil {
		return nil, internalErr("get user", err)
	}
	if user == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
	}
	return connect.NewResponse(&pb.GetUserResponse{User: iamUserToProto(user)}), nil
}

func (c *ClientApi) CreateUser(ctx context.Context, req *connect.Request[pb.CreateUserRequest]) (*connect.Response[pb.CreateUserResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	username := strings.TrimSpace(req.Msg.GetUsername())
	password := req.Msg.GetPassword()
	if username == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("username is required"))
	}
	if password != "" {
		if problem := passwordProblem(password); problem != "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New(problem))
		}
	}
	if existing, _ := st.GetUserByUsername(ctx, username); existing != nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("username already taken"))
	}
	hash := ""
	if password != "" {
		if hash, err = armpassword.Hash(password); err != nil {
			return nil, internalErr("hash password", err)
		}
	}
	id, err := st.CreateUserAccount(ctx, username, hash, store.UserCreatedByAdmin)
	if err != nil {
		return nil, internalErr("create user", err)
	}
	if err := st.EnsureUserInEveryoneGroup(ctx, id); err != nil {
		return nil, internalErr("add user to Everyone", err)
	}
	user, err := st.GetUserByID(ctx, id)
	if err != nil || user == nil {
		return nil, internalErr("load created user", err)
	}
	return connect.NewResponse(&pb.CreateUserResponse{User: iamUserToProto(user)}), nil
}

func (c *ClientApi) DeleteUser(ctx context.Context, req *connect.Request[pb.DeleteUserRequest]) (*connect.Response[pb.DeleteUserResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	id, err := requireID("user_id", req.Msg.GetUserId())
	if err != nil {
		return nil, err
	}
	if au := authUser(ctx); au != nil && au.User.ID == id {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("you cannot delete your own account"))
	}
	if err := c.refuseRemovingLastSuperuser(ctx, st, id); err != nil {
		return nil, err
	}
	if err := st.DeleteUserAccount(ctx, id); err != nil {
		return nil, internalErr("delete user", err)
	}
	c.refreshDefaultAdminPassword(ctx)
	return connect.NewResponse(&pb.DeleteUserResponse{Success: true}), nil
}

func (c *ClientApi) refuseRemovingLastSuperuser(ctx context.Context, st store.Store, userID int) error {
	rb, err := st.LoadEffectiveRBAC(ctx, userID)
	if err != nil {
		return internalErr("load user rbac", err)
	}
	if !rb.IsSuperuser {
		return nil
	}
	n, err := st.CountUsersWithSuperuserViaGroups(ctx)
	if err != nil {
		return internalErr("count superusers", err)
	}
	if n <= 1 {
		return connect.NewError(connect.CodeInvalidArgument, store.ErrNoSuperuser)
	}
	return nil
}

func (c *ClientApi) ResetUserPassword(ctx context.Context, req *connect.Request[pb.ResetUserPasswordRequest]) (*connect.Response[pb.ResetUserPasswordResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	id, err := requireID("user_id", req.Msg.GetUserId())
	if err != nil {
		return nil, err
	}
	if problem := passwordProblem(req.Msg.GetNewPassword()); problem != "" {
		return connect.NewResponse(&pb.ResetUserPasswordResponse{Success: false, Message: problem}), nil
	}
	user, err := st.GetUserByID(ctx, id)
	if err != nil {
		return nil, internalErr("get user", err)
	}
	if user == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
	}
	hash, err := armpassword.Hash(req.Msg.GetNewPassword())
	if err != nil {
		return nil, internalErr("hash password", err)
	}
	if err := st.UpdateUserPassword(ctx, id, hash); err != nil {
		return nil, internalErr("update password", err)
	}
	// Sign the user out everywhere, unless the admin is resetting their own password.
	if au := authUser(ctx); au == nil || au.User.ID != id {
		if err := st.DeleteSessionsForUser(ctx, id); err != nil {
			return nil, internalErr("delete sessions", err)
		}
	}
	c.refreshDefaultAdminPassword(ctx)
	return connect.NewResponse(&pb.ResetUserPasswordResponse{Success: true, Message: "Password reset"}), nil
}

func auditRowsToProto(rows []store.MyPermissionAuditRow) []*pb.PermissionAuditRow {
	out := make([]*pb.PermissionAuditRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, &pb.PermissionAuditRow{
			Permission:     r.Permission,
			Granted:        r.Granted,
			GrantingGroups: r.GrantingGroups,
		})
	}
	return out
}

func (c *ClientApi) GetUserAccess(ctx context.Context, req *connect.Request[pb.GetUserAccessRequest]) (*connect.Response[pb.GetUserAccessResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	id, err := requireID("user_id", req.Msg.GetUserId())
	if err != nil {
		return nil, err
	}
	if user, _ := st.GetUserByID(ctx, id); user == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
	}
	groupIDs, err := st.ListUserGroupIDsForUser(ctx, id)
	if err != nil {
		return nil, internalErr("list user groups", err)
	}
	groupNames, roleNames, isSuper, rows, err := st.GetMyPermissionsAudit(ctx, id)
	if err != nil {
		return nil, internalErr("permissions audit", err)
	}
	return connect.NewResponse(&pb.GetUserAccessResponse{
		GroupIds:    intsToInt32(groupIDs),
		GroupNames:  groupNames,
		RoleNames:   roleNames,
		IsSuperuser: isSuper,
		Permissions: auditRowsToProto(rows),
	}), nil
}

func (c *ClientApi) SetUserGroups(ctx context.Context, req *connect.Request[pb.SetUserGroupsRequest]) (*connect.Response[pb.SetUserGroupsResponse], error) {
	if c.auth == nil || c.iamDB == nil {
		return nil, errIamDisabled
	}
	id, err := requireID("user_id", req.Msg.GetUserId())
	if err != nil {
		return nil, err
	}
	if user, _ := c.auth.Store.GetUserByID(ctx, id); user == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
	}
	if err := iam.SetUserGroups(ctx, c.iamDB, id, int32sToInt(req.Msg.GetGroupIds())); err != nil {
		return nil, storeErr("set user groups", err)
	}
	return connect.NewResponse(&pb.SetUserGroupsResponse{Success: true}), nil
}

func (c *ClientApi) GetMyPermissionsAudit(ctx context.Context, _ *connect.Request[pb.GetMyPermissionsAuditRequest]) (*connect.Response[pb.GetMyPermissionsAuditResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	au := authUser(ctx)
	if au == nil || au.User.ID <= 0 {
		return nil, errAuthRequired
	}
	groupNames, roleNames, isSuper, rows, err := st.GetMyPermissionsAudit(ctx, au.User.ID)
	if err != nil {
		return nil, internalErr("permissions audit", err)
	}
	return connect.NewResponse(&pb.GetMyPermissionsAuditResponse{
		GroupNames:  groupNames,
		RoleNames:   roleNames,
		IsSuperuser: isSuper,
		Permissions: auditRowsToProto(rows),
	}), nil
}

// ---- User groups ----

func userGroupToProto(g *store.UserGroupRow) *pb.UserGroup {
	return &pb.UserGroup{Id: toInt32(g.ID), Name: g.Name, MemberCount: toInt32(g.MemberCount)}
}

func (c *ClientApi) ListUserGroups(ctx context.Context, _ *connect.Request[pb.ListUserGroupsRequest]) (*connect.Response[pb.ListUserGroupsResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	rows, err := st.ListUserGroups(ctx)
	if err != nil {
		return nil, internalErr("list groups", err)
	}
	groups := make([]*pb.UserGroup, 0, len(rows))
	for i := range rows {
		groups = append(groups, userGroupToProto(&rows[i]))
	}
	return connect.NewResponse(&pb.ListUserGroupsResponse{Groups: groups}), nil
}

func (c *ClientApi) CreateUserGroup(ctx context.Context, req *connect.Request[pb.CreateUserGroupRequest]) (*connect.Response[pb.CreateUserGroupResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	if existing, _ := st.GetUserGroupByName(ctx, name); existing != nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("a group with that name already exists"))
	}
	id, err := st.CreateUserGroup(ctx, name)
	if err != nil {
		return nil, internalErr("create group", err)
	}
	g, err := st.GetUserGroupByID(ctx, id)
	if err != nil || g == nil {
		return nil, internalErr("load created group", err)
	}
	return connect.NewResponse(&pb.CreateUserGroupResponse{Group: userGroupToProto(g)}), nil
}

func (c *ClientApi) DeleteUserGroup(ctx context.Context, req *connect.Request[pb.DeleteUserGroupRequest]) (*connect.Response[pb.DeleteUserGroupResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	id, err := requireID("group_id", req.Msg.GetGroupId())
	if err != nil {
		return nil, err
	}
	if err := st.DeleteUserGroup(ctx, id); err != nil {
		return nil, storeErr("delete group", err)
	}
	return connect.NewResponse(&pb.DeleteUserGroupResponse{Success: true}), nil
}

func (c *ClientApi) requireGroup(ctx context.Context, st store.Store, id int32) (int, error) {
	gid, err := requireID("group_id", id)
	if err != nil {
		return 0, err
	}
	if g, _ := st.GetUserGroupByID(ctx, gid); g == nil {
		return 0, connect.NewError(connect.CodeNotFound, errors.New("group not found"))
	}
	return gid, nil
}

func (c *ClientApi) GetUserGroupMembers(ctx context.Context, req *connect.Request[pb.GetUserGroupMembersRequest]) (*connect.Response[pb.GetUserGroupMembersResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	gid, err := c.requireGroup(ctx, st, req.Msg.GetGroupId())
	if err != nil {
		return nil, err
	}
	ids, err := st.ListUserGroupMemberIDs(ctx, gid)
	if err != nil {
		return nil, internalErr("list members", err)
	}
	return connect.NewResponse(&pb.GetUserGroupMembersResponse{UserIds: intsToInt32(ids)}), nil
}

func (c *ClientApi) SetUserGroupMembers(ctx context.Context, req *connect.Request[pb.SetUserGroupMembersRequest]) (*connect.Response[pb.SetUserGroupMembersResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	gid, err := c.requireGroup(ctx, st, req.Msg.GetGroupId())
	if err != nil {
		return nil, err
	}
	if err := st.SetUserGroupMembers(ctx, gid, int32sToInt(req.Msg.GetUserIds())); err != nil {
		return nil, storeErr("set members", err)
	}
	return connect.NewResponse(&pb.SetUserGroupMembersResponse{Success: true}), nil
}

// ---- RBAC ----

func rbacRoleToProto(r *store.RBACRoleRow) *pb.RbacRole {
	return &pb.RbacRole{
		Id:            toInt32(r.ID),
		Name:          r.Name,
		Description:   r.Description,
		PermissionIds: intsToInt32(r.PermissionIDs),
		GroupCount:    toInt32(r.GroupCount),
		UserCount:     toInt32(r.UserCount),
	}
}

func (c *ClientApi) ListRbacPermissions(ctx context.Context, _ *connect.Request[pb.ListRbacPermissionsRequest]) (*connect.Response[pb.ListRbacPermissionsResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	rows, err := st.ListRBACPermissions(ctx)
	if err != nil {
		return nil, internalErr("list permissions", err)
	}
	perms := make([]*pb.RbacPermission, 0, len(rows))
	for _, p := range rows {
		perms = append(perms, &pb.RbacPermission{Id: toInt32(p.ID), Name: p.Name, Description: p.Description})
	}
	return connect.NewResponse(&pb.ListRbacPermissionsResponse{Permissions: perms}), nil
}

func (c *ClientApi) ListRbacRoles(ctx context.Context, _ *connect.Request[pb.ListRbacRolesRequest]) (*connect.Response[pb.ListRbacRolesResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	rows, err := st.ListRBACRoles(ctx)
	if err != nil {
		return nil, internalErr("list roles", err)
	}
	roles := make([]*pb.RbacRole, 0, len(rows))
	for i := range rows {
		roles = append(roles, rbacRoleToProto(&rows[i]))
	}
	return connect.NewResponse(&pb.ListRbacRolesResponse{Roles: roles}), nil
}

func (c *ClientApi) CreateRbacRole(ctx context.Context, req *connect.Request[pb.CreateRbacRoleRequest]) (*connect.Response[pb.CreateRbacRoleResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	if rbac.IsSystemRole(name) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("that role name is reserved"))
	}
	id, err := st.CreateRBACRole(ctx, name, strings.TrimSpace(req.Msg.GetDescription()), int32sToInt(req.Msg.GetPermissionIds()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("create role: %w", err))
	}
	role, err := st.GetRBACRole(ctx, id)
	if err != nil || role == nil {
		return nil, internalErr("load created role", err)
	}
	return connect.NewResponse(&pb.CreateRbacRoleResponse{Role: rbacRoleToProto(role)}), nil
}

func (c *ClientApi) UpdateRbacRole(ctx context.Context, req *connect.Request[pb.UpdateRbacRoleRequest]) (*connect.Response[pb.UpdateRbacRoleResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	id, err := requireID("role_id", req.Msg.GetRoleId())
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	existing, err := st.GetRBACRole(ctx, id)
	if err != nil {
		return nil, internalErr("get role", err)
	}
	if existing == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("role not found"))
	}
	if existing.Name == rbac.RoleSuperuser {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("the superuser role grants every permission and cannot be edited"))
	}
	if rbac.IsSystemRole(name) && name != existing.Name {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("that role name is reserved"))
	}
	if err := st.UpdateRBACRole(ctx, id, name, strings.TrimSpace(req.Msg.GetDescription()), int32sToInt(req.Msg.GetPermissionIds())); err != nil {
		return nil, storeErr("update role", err)
	}
	role, err := st.GetRBACRole(ctx, id)
	if err != nil || role == nil {
		return nil, internalErr("load updated role", err)
	}
	return connect.NewResponse(&pb.UpdateRbacRoleResponse{Role: rbacRoleToProto(role)}), nil
}

func (c *ClientApi) DeleteRbacRole(ctx context.Context, req *connect.Request[pb.DeleteRbacRoleRequest]) (*connect.Response[pb.DeleteRbacRoleResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	id, err := requireID("role_id", req.Msg.GetRoleId())
	if err != nil {
		return nil, err
	}
	if err := st.DeleteRBACRole(ctx, id); err != nil {
		return nil, storeErr("delete role", err)
	}
	return connect.NewResponse(&pb.DeleteRbacRoleResponse{Success: true}), nil
}

func (c *ClientApi) GetUserGroupRbacRoles(ctx context.Context, req *connect.Request[pb.GetUserGroupRbacRolesRequest]) (*connect.Response[pb.GetUserGroupRbacRolesResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	gid, err := c.requireGroup(ctx, st, req.Msg.GetGroupId())
	if err != nil {
		return nil, err
	}
	ids, err := st.GetUserGroupRbacRoleIDs(ctx, gid)
	if err != nil {
		return nil, internalErr("get group roles", err)
	}
	return connect.NewResponse(&pb.GetUserGroupRbacRolesResponse{RoleIds: intsToInt32(ids)}), nil
}

func (c *ClientApi) SetUserGroupRbacRoles(ctx context.Context, req *connect.Request[pb.SetUserGroupRbacRolesRequest]) (*connect.Response[pb.SetUserGroupRbacRolesResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	gid, err := c.requireGroup(ctx, st, req.Msg.GetGroupId())
	if err != nil {
		return nil, err
	}
	if err := st.SetUserGroupRbacRoles(ctx, gid, int32sToInt(req.Msg.GetRoleIds())); err != nil {
		return nil, storeErr("set group roles", err)
	}
	return connect.NewResponse(&pb.SetUserGroupRbacRolesResponse{Success: true}), nil
}

func (c *ClientApi) GetRbacRoleGroups(ctx context.Context, req *connect.Request[pb.GetRbacRoleGroupsRequest]) (*connect.Response[pb.GetRbacRoleGroupsResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	id, err := requireID("role_id", req.Msg.GetRoleId())
	if err != nil {
		return nil, err
	}
	names, err := st.ListRbacRoleGroupNames(ctx, id)
	if err != nil {
		return nil, internalErr("list role groups", err)
	}
	return connect.NewResponse(&pb.GetRbacRoleGroupsResponse{GroupNames: names}), nil
}

// ---- API keys ----

func keyHint(key string) string {
	if len(key) <= 4 {
		return ""
	}
	return "…" + key[len(key)-4:]
}

func (c *ClientApi) ListApiKeys(ctx context.Context, _ *connect.Request[pb.ListApiKeysRequest]) (*connect.Response[pb.ListApiKeysResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	au := authUser(ctx)
	if au == nil || au.User.ID <= 0 {
		return nil, errAuthRequired
	}
	rows, err := st.ListAPIKeysForUser(ctx, au.User.ID)
	if err != nil {
		return nil, internalErr("list api keys", err)
	}
	keys := make([]*pb.ApiKey, 0, len(rows))
	for _, k := range rows {
		scope, err := iam.LoadAPIKeyScope(ctx, c.iamDB, k.ID)
		if err != nil {
			return nil, internalErr("load api key scope", err)
		}
		keys = append(keys, &pb.ApiKey{
			Id:         toInt32(k.ID),
			Name:       k.Name,
			CreatedAt:  k.CreatedAt,
			LastUsedAt: k.LastUsedAt,
			KeyHint:    keyHint(k.KeyValue),
			Scope:      scopeToProto(scope),
		})
	}
	return connect.NewResponse(&pb.ListApiKeysResponse{ApiKeys: keys}), nil
}

func (c *ClientApi) CreateApiKey(ctx context.Context, req *connect.Request[pb.CreateApiKeyRequest]) (*connect.Response[pb.CreateApiKeyResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	au := authUser(ctx)
	if au == nil || au.User.ID <= 0 {
		return nil, errAuthRequired
	}
	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	key, err := armpassword.GenerateAPIKey(iam.APIKeyPrefix)
	if err != nil {
		return nil, internalErr("generate api key", err)
	}
	scope := scopeFromProto(req.Msg.GetScope())
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	id, err := st.CreateAPIKey(ctx, au.User.ID, name, key, false)
	if err != nil {
		return nil, internalErr("create api key", err)
	}
	if err := iam.SetAPIKeyScope(ctx, c.iamDB, id, au.User.ID, scope); err != nil {
		// Never leave behind a key with more access than was asked for.
		_ = st.DeleteAPIKey(ctx, id, au.User.ID)
		return nil, scopeErr(err)
	}
	return connect.NewResponse(&pb.CreateApiKeyResponse{
		ApiKey: &pb.ApiKey{Id: toInt32(id), Name: name, KeyHint: keyHint(key), Scope: scopeToProto(&scope)},
		Key:    key,
	}), nil
}

func (c *ClientApi) UpdateApiKey(ctx context.Context, req *connect.Request[pb.UpdateApiKeyRequest]) (*connect.Response[pb.UpdateApiKeyResponse], error) {
	if _, err := c.iamStore(); err != nil {
		return nil, err
	}
	au := authUser(ctx)
	if au == nil || au.User.ID <= 0 {
		return nil, errAuthRequired
	}
	id, err := requireID("api_key_id", req.Msg.GetApiKeyId())
	if err != nil {
		return nil, err
	}
	scope := scopeFromProto(req.Msg.GetScope())
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	// Scoped to the caller, so users can only change their own keys.
	if err := iam.SetAPIKeyScope(ctx, c.iamDB, id, au.User.ID, scope); err != nil {
		return nil, scopeErr(err)
	}
	return connect.NewResponse(&pb.UpdateApiKeyResponse{
		ApiKey: &pb.ApiKey{Id: toInt32(id), Scope: scopeToProto(&scope)},
	}), nil
}

func scopeFromProto(p *pb.ApiKeyScope) iam.APIKeyScope {
	return iam.APIKeyScope{
		RestrictPermissions:  p.GetRestrictPermissions(),
		Permissions:          p.GetPermissions(),
		RestrictBuildConfigs: p.GetRestrictBuildConfigs(),
		BuildConfigs:         p.GetBuildConfigs(),
	}
}

func scopeToProto(s *iam.APIKeyScope) *pb.ApiKeyScope {
	if s == nil {
		return &pb.ApiKeyScope{}
	}
	return &pb.ApiKeyScope{
		RestrictPermissions:  s.RestrictPermissions,
		Permissions:          s.Permissions,
		RestrictBuildConfigs: s.RestrictBuildConfigs,
		BuildConfigs:         s.BuildConfigs,
	}
}

// validateScope refuses scopes that would make a useless key. Restricting to no build configs
// is allowed: that key can still use permissions that are not about build configs.
func validateScope(s iam.APIKeyScope) error {
	if s.RestrictPermissions && len(s.Permissions) == 0 {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("choose at least one permission, or give the key all of your permissions"))
	}
	return nil
}

func scopeErr(err error) error {
	if errors.Is(err, iam.ErrAPIKeyNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	if strings.HasPrefix(err.Error(), "unknown permission") {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return internalErr("set api key scope", err)
}

func (c *ClientApi) DeleteApiKey(ctx context.Context, req *connect.Request[pb.DeleteApiKeyRequest]) (*connect.Response[pb.DeleteApiKeyResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	au := authUser(ctx)
	if au == nil || au.User.ID <= 0 {
		return nil, errAuthRequired
	}
	id, err := requireID("api_key_id", req.Msg.GetApiKeyId())
	if err != nil {
		return nil, err
	}
	// Scoped to the caller, so users can only delete their own keys.
	if err := st.DeleteAPIKey(ctx, id, au.User.ID); err != nil {
		return nil, internalErr("delete api key", err)
	}
	return connect.NewResponse(&pb.DeleteApiKeyResponse{Success: true}), nil
}
