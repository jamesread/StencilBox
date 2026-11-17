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

func invalidArg(msg string) error {
	return connect.NewError(connect.CodeInvalidArgument, errors.New(msg))
}

func notFoundErr(msg string) error {
	return connect.NewError(connect.CodeNotFound, errors.New(msg))
}

// requireName trims a required name field.
func requireName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", invalidArg("name is required")
	}
	return name, nil
}

// orInternal turns a failed or empty lookup into an internal error for op.
func orInternal[T any](v *T, err error, op string) (*T, error) {
	if err != nil || v == nil {
		return nil, internalErr(op, err)
	}
	return v, nil
}

// iamAuthed returns the IAM store and the signed-in user.
func (c *ClientApi) iamAuthed(ctx context.Context) (store.Store, *armlayer.AuthenticatedUser, error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, nil, err
	}
	au := authUser(ctx)
	if au == nil || au.User.ID <= 0 {
		return nil, nil, errAuthRequired
	}
	return st, au, nil
}

// iamWithID returns the IAM store and a validated identifier.
func (c *ClientApi) iamWithID(field string, id int32) (store.Store, int, error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, 0, err
	}
	n, err := requireID(field, id)
	if err != nil {
		return nil, 0, err
	}
	return st, n, nil
}

// iamWithGroup returns the IAM store and the id of an existing group.
func (c *ClientApi) iamWithGroup(ctx context.Context, id int32) (store.Store, int, error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, 0, err
	}
	gid, err := c.requireGroup(ctx, st, id)
	if err != nil {
		return nil, 0, err
	}
	return st, gid, nil
}

// rbacPermissionNames returns the user's effective permissions, sorted. A superuser gets the
// whole catalog.
func (c *ClientApi) rbacPermissionNames(ctx context.Context, au *armlayer.AuthenticatedUser) []string {
	if au == nil || au.RBAC == nil {
		return []string{}
	}
	var names []string
	if au.RBAC.IsSuperuser && c.auth != nil {
		names = c.catalogPermissionNames(ctx)
	} else {
		names = grantedPermissionNames(au.RBAC.Permissions)
	}
	sort.Strings(names)
	return names
}

func (c *ClientApi) catalogPermissionNames(ctx context.Context) []string {
	names := []string{}
	perms, err := c.auth.Store.ListRBACPermissions(ctx)
	if err != nil {
		return names
	}
	for _, p := range perms {
		names = append(names, p.Name)
	}
	return names
}

func grantedPermissionNames(permissions map[string]bool) []string {
	names := []string{}
	for name, ok := range permissions {
		if ok {
			names = append(names, name)
		}
	}
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
	user, err := authenticate(ctx, st, strings.TrimSpace(req.Msg.GetUsername()), req.Msg.GetPassword())
	if err != nil {
		return nil, err
	}
	if user == nil {
		return connect.NewResponse(&pb.LoginWithUsernameAndPasswordResponse{Success: false, Message: loginFailedMessage}), nil
	}

	sid := uuid.NewString()
	if err = st.CreateSession(ctx, sid, user.ID, nil); err != nil {
		return nil, internalErr("create session", err)
	}
	log.WithField("username", user.Username).Info("Login succeeded")

	res := connect.NewResponse(&pb.LoginWithUsernameAndPasswordResponse{Success: true, Message: "Logged in"})
	cookie := c.auth.SessionCookie(sid, reqctx.HttpRequest(ctx), req.Header())
	res.Header().Add("Set-Cookie", cookie.String())
	return res, nil
}

// authenticate returns the user when the credentials are right, and nil when they are not.
func authenticate(ctx context.Context, st store.Store, username, password string) (*store.UserAccountRow, error) {
	if username == "" || password == "" {
		return nil, nil
	}
	user, err := st.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, internalErr("login", err)
	}
	if !credentialsValid(user, password) {
		log.WithField("username", username).Info("Login failed")
		return nil, nil
	}
	return user, nil
}

// credentialsValid always verifies a hash (a dummy one for unknown users), so timing does not
// reveal which usernames exist.
func credentialsValid(user *store.UserAccountRow, password string) bool {
	matches := passwordMatches(loginHash(user), password)
	return matches && user != nil && user.PasswordHash != ""
}

func loginHash(user *store.UserAccountRow) string {
	if user != nil && user.PasswordHash != "" {
		return user.PasswordHash
	}
	return dummyHash()
}

func passwordMatches(hash, password string) bool {
	ok, err := armpassword.Verify(hash, password)
	return err == nil && ok
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
	st, au, err := c.iamAuthed(ctx)
	if err != nil {
		return nil, err
	}
	problem, err := changePassword(ctx, st, au.User.ID, req.Msg.GetCurrentPassword(), req.Msg.GetNewPassword())
	if err != nil {
		return nil, err
	}
	if problem != "" {
		return changePasswordReply(false, problem), nil
	}
	c.refreshDefaultAdminPassword(ctx)
	return changePasswordReply(true, "Password changed"), nil
}

func changePasswordReply(success bool, msg string) *connect.Response[pb.ChangePasswordResponse] {
	return connect.NewResponse(&pb.ChangePasswordResponse{Success: success, Message: msg})
}

// changePassword returns a message for the user when the change is refused, and an error when it fails.
func changePassword(ctx context.Context, st store.Store, userID int, current, newPassword string) (string, error) {
	if problem := passwordProblem(newPassword); problem != "" {
		return problem, nil
	}
	user, err := st.GetUserByID(ctx, userID)
	if user, err = orInternal(user, err, "load user"); err != nil {
		return "", err
	}
	// Accounts without a password (SSO, API key only) may set a first one.
	if user.PasswordHash != "" && !passwordMatches(user.PasswordHash, current) {
		return "current password is incorrect", nil
	}
	return "", setPassword(ctx, st, user.ID, newPassword)
}

func setPassword(ctx context.Context, st store.Store, userID int, newPassword string) error {
	hash, err := armpassword.Hash(newPassword)
	if err != nil {
		return internalErr("hash password", err)
	}
	if err = st.UpdateUserPassword(ctx, userID, hash); err != nil {
		return internalErr("update password", err)
	}
	return nil
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
	st, id, err := c.iamWithID("user_id", req.Msg.GetUserId())
	if err != nil {
		return nil, err
	}
	user, err := loadUser(ctx, st, id)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.GetUserResponse{User: iamUserToProto(user)}), nil
}

func loadUser(ctx context.Context, st store.Store, id int) (*store.UserAccountRow, error) {
	user, err := st.GetUserByID(ctx, id)
	if err != nil {
		return nil, internalErr("get user", err)
	}
	if user == nil {
		return nil, notFoundErr("user not found")
	}
	return user, nil
}
func (c *ClientApi) CreateUser(ctx context.Context, req *connect.Request[pb.CreateUserRequest]) (*connect.Response[pb.CreateUserResponse], error) {
	st, err := c.iamStore()
	if err != nil {
		return nil, err
	}
	username, hash, err := prepareNewUser(ctx, st, req.Msg)
	if err != nil {
		return nil, err
	}
	user, err := createUserAccount(ctx, st, username, hash)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.CreateUserResponse{User: iamUserToProto(user)}), nil
}

// prepareNewUser validates a new user and returns the username and password hash (empty when
// the user has no password yet).
func prepareNewUser(ctx context.Context, st store.Store, msg *pb.CreateUserRequest) (string, string, error) {
	username, err := requireName(msg.GetUsername())
	if err != nil {
		return "", "", invalidArg("username is required")
	}
	if err = checkNewUserPassword(msg.GetPassword()); err != nil {
		return "", "", err
	}
	if existing, _ := st.GetUserByUsername(ctx, username); existing != nil {
		return "", "", connect.NewError(connect.CodeAlreadyExists, errors.New("username already taken"))
	}
	hash, err := hashOptionalPassword(msg.GetPassword())
	return username, hash, err
}

func checkNewUserPassword(password string) error {
	if password == "" {
		return nil
	}
	if problem := passwordProblem(password); problem != "" {
		return invalidArg(problem)
	}
	return nil
}

func hashOptionalPassword(password string) (string, error) {
	if password == "" {
		return "", nil
	}
	hash, err := armpassword.Hash(password)
	if err != nil {
		return "", internalErr("hash password", err)
	}
	return hash, nil
}

func createUserAccount(ctx context.Context, st store.Store, username, hash string) (*store.UserAccountRow, error) {
	id, err := st.CreateUserAccount(ctx, username, hash, store.UserCreatedByAdmin)
	if err != nil {
		return nil, internalErr("create user", err)
	}
	if err = st.EnsureUserInEveryoneGroup(ctx, id); err != nil {
		return nil, internalErr("add user to Everyone", err)
	}
	user, err := st.GetUserByID(ctx, id)
	return orInternal(user, err, "load created user")
}
func (c *ClientApi) DeleteUser(ctx context.Context, req *connect.Request[pb.DeleteUserRequest]) (*connect.Response[pb.DeleteUserResponse], error) {
	st, id, err := c.iamWithID("user_id", req.Msg.GetUserId())
	if err != nil {
		return nil, err
	}
	if err = c.checkUserDeletable(ctx, st, id); err != nil {
		return nil, err
	}
	if err = st.DeleteUserAccount(ctx, id); err != nil {
		return nil, internalErr("delete user", err)
	}
	c.refreshDefaultAdminPassword(ctx)
	return connect.NewResponse(&pb.DeleteUserResponse{Success: true}), nil
}

func (c *ClientApi) checkUserDeletable(ctx context.Context, st store.Store, id int) error {
	if au := authUser(ctx); au != nil && au.User.ID == id {
		return invalidArg("you cannot delete your own account")
	}
	return c.refuseRemovingLastSuperuser(ctx, st, id)
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
	st, id, err := c.iamWithID("user_id", req.Msg.GetUserId())
	if err != nil {
		return nil, err
	}
	if problem := passwordProblem(req.Msg.GetNewPassword()); problem != "" {
		return connect.NewResponse(&pb.ResetUserPasswordResponse{Success: false, Message: problem}), nil
	}
	if err = resetPassword(ctx, st, id, req.Msg.GetNewPassword()); err != nil {
		return nil, err
	}
	c.refreshDefaultAdminPassword(ctx)
	return connect.NewResponse(&pb.ResetUserPasswordResponse{Success: true, Message: "Password reset"}), nil
}

func resetPassword(ctx context.Context, st store.Store, id int, newPassword string) error {
	if _, err := loadUser(ctx, st, id); err != nil {
		return err
	}
	if err := setPassword(ctx, st, id, newPassword); err != nil {
		return err
	}
	return signOutEverywhere(ctx, st, id)
}

// signOutEverywhere ends the user's sessions, unless the admin is resetting their own password.
func signOutEverywhere(ctx context.Context, st store.Store, id int) error {
	if au := authUser(ctx); au != nil && au.User.ID == id {
		return nil
	}
	if err := st.DeleteSessionsForUser(ctx, id); err != nil {
		return internalErr("delete sessions", err)
	}
	return nil
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
	st, id, err := c.iamWithID("user_id", req.Msg.GetUserId())
	if err != nil {
		return nil, err
	}
	if user, _ := st.GetUserByID(ctx, id); user == nil {
		return nil, notFoundErr("user not found")
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
	if err := c.requireIamDB(); err != nil {
		return nil, err
	}
	id, err := requireID("user_id", req.Msg.GetUserId())
	if err != nil {
		return nil, err
	}
	if user, _ := c.auth.Store.GetUserByID(ctx, id); user == nil {
		return nil, notFoundErr("user not found")
	}
	if err = iam.SetUserGroups(ctx, c.iamDB, id, int32sToInt(req.Msg.GetGroupIds())); err != nil {
		return nil, storeErr("set user groups", err)
	}
	return connect.NewResponse(&pb.SetUserGroupsResponse{Success: true}), nil
}

func (c *ClientApi) requireIamDB() error {
	if c.auth == nil || c.iamDB == nil {
		return errIamDisabled
	}
	return nil
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
	name, err := requireName(req.Msg.GetName())
	if err != nil {
		return nil, err
	}
	if existing, _ := st.GetUserGroupByName(ctx, name); existing != nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("a group with that name already exists"))
	}
	g, err := createGroup(ctx, st, name)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.CreateUserGroupResponse{Group: userGroupToProto(g)}), nil
}

func createGroup(ctx context.Context, st store.Store, name string) (*store.UserGroupRow, error) {
	id, err := st.CreateUserGroup(ctx, name)
	if err != nil {
		return nil, internalErr("create group", err)
	}
	g, err := st.GetUserGroupByID(ctx, id)
	return orInternal(g, err, "load created group")
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
	st, gid, err := c.iamWithGroup(ctx, req.Msg.GetGroupId())
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
	st, gid, err := c.iamWithGroup(ctx, req.Msg.GetGroupId())
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
	name, err := requireName(req.Msg.GetName())
	if err != nil {
		return nil, err
	}
	if rbac.IsSystemRole(name) {
		return nil, invalidArg("that role name is reserved")
	}
	role, err := createRole(ctx, st, name, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.CreateRbacRoleResponse{Role: rbacRoleToProto(role)}), nil
}

func createRole(ctx context.Context, st store.Store, name string, msg *pb.CreateRbacRoleRequest) (*store.RBACRoleRow, error) {
	id, err := st.CreateRBACRole(ctx, name, strings.TrimSpace(msg.GetDescription()), int32sToInt(msg.GetPermissionIds()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("create role: %w", err))
	}
	role, err := st.GetRBACRole(ctx, id)
	return orInternal(role, err, "load created role")
}
func (c *ClientApi) UpdateRbacRole(ctx context.Context, req *connect.Request[pb.UpdateRbacRoleRequest]) (*connect.Response[pb.UpdateRbacRoleResponse], error) {
	st, id, err := c.iamWithID("role_id", req.Msg.GetRoleId())
	if err != nil {
		return nil, err
	}
	name, err := requireName(req.Msg.GetName())
	if err != nil {
		return nil, err
	}
	role, err := updateRole(ctx, st, id, name, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.UpdateRbacRoleResponse{Role: rbacRoleToProto(role)}), nil
}

func updateRole(ctx context.Context, st store.Store, id int, name string, msg *pb.UpdateRbacRoleRequest) (*store.RBACRoleRow, error) {
	existing, err := loadRole(ctx, st, id)
	if err != nil {
		return nil, err
	}
	if err = checkRoleEditable(existing, name); err != nil {
		return nil, err
	}
	if err = st.UpdateRBACRole(ctx, id, name, strings.TrimSpace(msg.GetDescription()), int32sToInt(msg.GetPermissionIds())); err != nil {
		return nil, storeErr("update role", err)
	}
	role, err := st.GetRBACRole(ctx, id)
	return orInternal(role, err, "load updated role")
}

func loadRole(ctx context.Context, st store.Store, id int) (*store.RBACRoleRow, error) {
	role, err := st.GetRBACRole(ctx, id)
	if err != nil {
		return nil, internalErr("get role", err)
	}
	if role == nil {
		return nil, notFoundErr("role not found")
	}
	return role, nil
}

func checkRoleEditable(existing *store.RBACRoleRow, newName string) error {
	if existing.Name == rbac.RoleSuperuser {
		return invalidArg("the superuser role grants every permission and cannot be edited")
	}
	if rbac.IsSystemRole(newName) && newName != existing.Name {
		return invalidArg("that role name is reserved")
	}
	return nil
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
	st, gid, err := c.iamWithGroup(ctx, req.Msg.GetGroupId())
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
	st, gid, err := c.iamWithGroup(ctx, req.Msg.GetGroupId())
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
	st, au, err := c.iamAuthed(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := st.ListAPIKeysForUser(ctx, au.User.ID)
	if err != nil {
		return nil, internalErr("list api keys", err)
	}
	keys := make([]*pb.ApiKey, 0, len(rows))
	for _, k := range rows {
		key, keyErr := c.apiKeyToProto(ctx, k)
		if keyErr != nil {
			return nil, keyErr
		}
		keys = append(keys, key)
	}
	return connect.NewResponse(&pb.ListApiKeysResponse{ApiKeys: keys}), nil
}

func (c *ClientApi) apiKeyToProto(ctx context.Context, k store.APIKeyRow) (*pb.ApiKey, error) {
	scope, err := iam.LoadAPIKeyScope(ctx, c.iamDB, k.ID)
	if err != nil {
		return nil, internalErr("load api key scope", err)
	}
	return &pb.ApiKey{
		Id:         toInt32(k.ID),
		Name:       k.Name,
		CreatedAt:  k.CreatedAt,
		LastUsedAt: k.LastUsedAt,
		KeyHint:    keyHint(k.KeyValue),
		Scope:      scopeToProto(scope),
	}, nil
}
func (c *ClientApi) CreateApiKey(ctx context.Context, req *connect.Request[pb.CreateApiKeyRequest]) (*connect.Response[pb.CreateApiKeyResponse], error) {
	st, au, err := c.iamAuthed(ctx)
	if err != nil {
		return nil, err
	}
	name, err := requireName(req.Msg.GetName())
	if err != nil {
		return nil, err
	}
	scope := scopeFromProto(req.Msg.GetScope())
	if err = validateScope(scope); err != nil {
		return nil, err
	}
	id, key, err := c.createKeyWithScope(ctx, st, au.User.ID, name, scope)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.CreateApiKeyResponse{
		ApiKey: &pb.ApiKey{Id: toInt32(id), Name: name, KeyHint: keyHint(key), Scope: scopeToProto(&scope)},
		Key:    key,
	}), nil
}

func (c *ClientApi) createKeyWithScope(ctx context.Context, st store.Store, userID int, name string, scope iam.APIKeyScope) (int, string, error) {
	key, err := armpassword.GenerateAPIKey(iam.APIKeyPrefix)
	if err != nil {
		return 0, "", internalErr("generate api key", err)
	}
	id, err := st.CreateAPIKey(ctx, userID, name, key, false)
	if err != nil {
		return 0, "", internalErr("create api key", err)
	}
	if err = iam.SetAPIKeyScope(ctx, c.iamDB, id, userID, scope); err != nil {
		// Never leave behind a key with more access than was asked for.
		_ = st.DeleteAPIKey(ctx, id, userID)
		return 0, "", scopeErr(err)
	}
	return id, key, nil
}
func (c *ClientApi) UpdateApiKey(ctx context.Context, req *connect.Request[pb.UpdateApiKeyRequest]) (*connect.Response[pb.UpdateApiKeyResponse], error) {
	_, au, err := c.iamAuthed(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requireID("api_key_id", req.Msg.GetApiKeyId())
	if err != nil {
		return nil, err
	}
	scope := scopeFromProto(req.Msg.GetScope())
	if err = validateScope(scope); err != nil {
		return nil, err
	}
	// Scoped to the caller, so users can only change their own keys.
	if err = iam.SetAPIKeyScope(ctx, c.iamDB, id, au.User.ID, scope); err != nil {
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
	st, au, err := c.iamAuthed(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requireID("api_key_id", req.Msg.GetApiKeyId())
	if err != nil {
		return nil, err
	}
	// Scoped to the caller, so users can only delete their own keys.
	if err = st.DeleteAPIKey(ctx, id, au.User.ID); err != nil {
		return nil, internalErr("delete api key", err)
	}
	return connect.NewResponse(&pb.DeleteApiKeyResponse{Success: true}), nil
}
