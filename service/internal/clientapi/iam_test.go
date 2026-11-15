package clientapi

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	pb "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1"
	clientapiconnect "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1/clientapi_pbconnect"
	"github.com/jamesread/StencilBox/internal/buildconfigs"
	"github.com/jamesread/StencilBox/internal/database"
	"github.com/jamesread/StencilBox/internal/iam"
	iamsqlite "github.com/jamesread/armature-iam/store/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newIamTestClient serves the IAM handlers behind the real auth middleware and returns a client
// with its own cookie jar.
func newIamTestClient(t *testing.T) (func() clientapiconnect.StencilBoxApiServiceClient, *ClientApi) {
	t.Helper()
	ctx := context.Background()
	db, err := database.OpenMemory(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	st := iamsqlite.New(db)
	require.NoError(t, iam.Bootstrap(ctx, st))
	layer, err := iam.NewAuthLayer(st, nil)
	require.NoError(t, err)

	api := &ClientApi{auth: layer, iamDB: db}
	api.refreshDefaultAdminPassword(ctx)
	_, handler := clientapiconnect.NewStencilBoxApiServiceHandler(api, connect.WithInterceptors(api.ScopeInterceptor()))
	srv := httptest.NewServer(layer.WrapHandler(handler))
	t.Cleanup(srv.Close)

	newClient := func() clientapiconnect.StencilBoxApiServiceClient {
		jar, err := cookiejar.New(nil)
		require.NoError(t, err)
		return clientapiconnect.NewStencilBoxApiServiceClient(&http.Client{Jar: jar}, srv.URL)
	}
	return newClient, api
}

func login(t *testing.T, c clientapiconnect.StencilBoxApiServiceClient, username, password string) bool {
	t.Helper()
	res, err := c.LoginWithUsernameAndPassword(context.Background(), connect.NewRequest(&pb.LoginWithUsernameAndPasswordRequest{
		Username: username,
		Password: password,
	}))
	require.NoError(t, err)
	return res.Msg.GetSuccess()
}

func TestLoginStatusLogout(t *testing.T) {
	newClient, _ := newIamTestClient(t)
	ctx := context.Background()
	c := newClient()

	status, err := c.GetStatus(ctx, connect.NewRequest(&pb.GetStatusRequest{}))
	require.NoError(t, err)
	assert.False(t, status.Msg.GetIsLoggedIn())
	assert.Empty(t, status.Msg.GetOutputPath(), "guests do not see server paths")

	_, err = c.GetBuildHistory(ctx, connect.NewRequest(&pb.GetBuildHistoryRequest{}))
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	assert.False(t, login(t, c, iam.DefaultAdminUsername, "wrong"))
	assert.False(t, login(t, c, "nobody", "whatever"))
	require.True(t, login(t, c, iam.DefaultAdminUsername, iam.DefaultAdminPassword))

	status, err = c.GetStatus(ctx, connect.NewRequest(&pb.GetStatusRequest{}))
	require.NoError(t, err)
	assert.True(t, status.Msg.GetIsLoggedIn())
	assert.Equal(t, iam.DefaultAdminUsername, status.Msg.GetUsername())
	assert.True(t, status.Msg.GetRbacIsSuperuser())
	assert.Contains(t, status.Msg.GetRbacPermissions(), "rbac.manage")
	assert.True(t, status.Msg.GetAdminUsesDefaultPassword())

	_, err = c.Logout(ctx, connect.NewRequest(&pb.LogoutRequest{}))
	require.NoError(t, err)
	status, err = c.GetStatus(ctx, connect.NewRequest(&pb.GetStatusRequest{}))
	require.NoError(t, err)
	assert.False(t, status.Msg.GetIsLoggedIn())
}

func TestMemberCannotAdminister(t *testing.T) {
	newClient, _ := newIamTestClient(t)
	ctx := context.Background()

	admin := newClient()
	require.True(t, login(t, admin, iam.DefaultAdminUsername, iam.DefaultAdminPassword))
	created, err := admin.CreateUser(ctx, connect.NewRequest(&pb.CreateUserRequest{Username: "bob", Password: "bobpassword"}))
	require.NoError(t, err)

	bob := newClient()
	require.True(t, login(t, bob, "bob", "bobpassword"))

	status, err := bob.GetStatus(ctx, connect.NewRequest(&pb.GetStatusRequest{}))
	require.NoError(t, err)
	assert.Equal(t, []string{"app.access", "buildconfigs.git-pull", "buildconfigs.view", "builds.run", "datafiles.edit"}, status.Msg.GetRbacPermissions())
	assert.False(t, status.Msg.GetAdminUsesDefaultPassword(), "only shown to superusers")

	_, err = bob.CreateUser(ctx, connect.NewRequest(&pb.CreateUserRequest{Username: "eve"}))
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), "Forbidden from the auth layer")

	users, err := bob.GetUsers(ctx, connect.NewRequest(&pb.GetUsersRequest{}))
	require.NoError(t, err, "pickers work with app.access")
	assert.Len(t, users.Msg.GetUsers(), 2)

	audit, err := bob.GetMyPermissionsAudit(ctx, connect.NewRequest(&pb.GetMyPermissionsAuditRequest{}))
	require.NoError(t, err)
	assert.Equal(t, []string{"Everyone"}, audit.Msg.GetGroupNames())

	// Resetting bob's password signs him out.
	_, err = admin.ResetUserPassword(ctx, connect.NewRequest(&pb.ResetUserPasswordRequest{
		UserId:      created.Msg.GetUser().GetId(),
		NewPassword: "newbobpassword",
	}))
	require.NoError(t, err)
	status, err = bob.GetStatus(ctx, connect.NewRequest(&pb.GetStatusRequest{}))
	require.NoError(t, err)
	assert.False(t, status.Msg.GetIsLoggedIn())
}

func TestCannotRemoveLastSuperuser(t *testing.T) {
	newClient, _ := newIamTestClient(t)
	ctx := context.Background()
	admin := newClient()
	require.True(t, login(t, admin, iam.DefaultAdminUsername, iam.DefaultAdminPassword))

	status, err := admin.GetUsers(ctx, connect.NewRequest(&pb.GetUsersRequest{}))
	require.NoError(t, err)
	adminID := status.Msg.GetUsers()[0].GetId()

	_, err = admin.SetUserGroups(ctx, connect.NewRequest(&pb.SetUserGroupsRequest{UserId: adminID}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = admin.DeleteUser(ctx, connect.NewRequest(&pb.DeleteUserRequest{UserId: adminID}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "cannot delete yourself")
}

func TestApiKeyBearerAuth(t *testing.T) {
	newClient, _ := newIamTestClient(t)
	ctx := context.Background()
	admin := newClient()
	require.True(t, login(t, admin, iam.DefaultAdminUsername, iam.DefaultAdminPassword))

	created, err := admin.CreateApiKey(ctx, connect.NewRequest(&pb.CreateApiKeyRequest{Name: "ci"}))
	require.NoError(t, err)
	key := created.Msg.GetKey()
	assert.Contains(t, key, iam.APIKeyPrefix)

	req := connect.NewRequest(&pb.GetStatusRequest{})
	req.Header().Set("Authorization", "Bearer "+key)
	status, err := newClient().GetStatus(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, iam.DefaultAdminUsername, status.Msg.GetUsername())

	keys, err := admin.ListApiKeys(ctx, connect.NewRequest(&pb.ListApiKeysRequest{}))
	require.NoError(t, err)
	require.Len(t, keys.Msg.GetApiKeys(), 1)
	assert.Equal(t, keyHint(key), keys.Msg.GetApiKeys()[0].GetKeyHint())
}

func TestChangePasswordClearsDefaultAdminFlag(t *testing.T) {
	newClient, api := newIamTestClient(t)
	ctx := context.Background()
	admin := newClient()
	require.True(t, login(t, admin, iam.DefaultAdminUsername, iam.DefaultAdminPassword))
	require.True(t, api.defaultAdminPassword.Load())

	res, err := admin.ChangePassword(ctx, connect.NewRequest(&pb.ChangePasswordRequest{
		CurrentPassword: "wrong",
		NewPassword:     "a-much-better-password",
	}))
	require.NoError(t, err)
	assert.False(t, res.Msg.GetSuccess())

	res, err = admin.ChangePassword(ctx, connect.NewRequest(&pb.ChangePasswordRequest{
		CurrentPassword: iam.DefaultAdminPassword,
		NewPassword:     "a-much-better-password",
	}))
	require.NoError(t, err)
	assert.True(t, res.Msg.GetSuccess())
	assert.False(t, api.defaultAdminPassword.Load())
}

func TestScopedApiKey(t *testing.T) {
	newClient, api := newIamTestClient(t)
	ctx := context.Background()
	api.buildConfigs = map[string]*buildconfigs.BuildConfig{
		"homepage": {Name: "homepage"},
		"sidebar":  {Name: "sidebar"},
	}
	admin := newClient()
	require.True(t, login(t, admin, iam.DefaultAdminUsername, iam.DefaultAdminPassword))

	_, err := admin.CreateApiKey(ctx, connect.NewRequest(&pb.CreateApiKeyRequest{
		Name:  "empty",
		Scope: &pb.ApiKeyScope{RestrictPermissions: true},
	}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "a key with no permissions is refused")

	created, err := admin.CreateApiKey(ctx, connect.NewRequest(&pb.CreateApiKeyRequest{
		Name: "homepage-viewer",
		Scope: &pb.ApiKeyScope{
			RestrictPermissions:  true,
			Permissions:          []string{"buildconfigs.view"},
			RestrictBuildConfigs: true,
			BuildConfigs:         []string{"homepage"},
		},
	}))
	require.NoError(t, err)
	key := created.Msg.GetKey()
	withKey := func(r interface{ Header() http.Header }) {
		r.Header().Set("Authorization", "Bearer "+key)
	}
	c := newClient()

	listReq := connect.NewRequest(&pb.GetBuildConfigsRequest{})
	withKey(listReq)
	list, err := c.GetBuildConfigs(ctx, listReq)
	require.NoError(t, err)
	require.Len(t, list.Msg.GetBuildConfigs(), 1, "only the key's build configs are listed")
	assert.Equal(t, "homepage", list.Msg.GetBuildConfigs()[0].GetName())

	historyReq := connect.NewRequest(&pb.GetBuildHistoryRequest{ConfigName: "sidebar"})
	withKey(historyReq)
	_, err = c.GetBuildHistory(ctx, historyReq)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), "another build config")

	historyReq = connect.NewRequest(&pb.GetBuildHistoryRequest{ConfigName: "homepage"})
	withKey(historyReq)
	_, err = c.GetBuildHistory(ctx, historyReq)
	require.NoError(t, err)

	buildReq := connect.NewRequest(&pb.BuildRequest{ConfigName: "homepage"})
	withKey(buildReq)
	stream, err := c.StartBuild(ctx, buildReq)
	require.NoError(t, err)
	assert.False(t, stream.Receive())
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(stream.Err()), "no builds.run")

	usersReq := connect.NewRequest(&pb.GetUserRequest{UserId: 1})
	withKey(usersReq)
	_, err = c.GetUser(ctx, usersReq)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), "the owner is a superuser, the key is not")

	mintReq := connect.NewRequest(&pb.CreateApiKeyRequest{Name: "escalate"})
	withKey(mintReq)
	_, err = c.CreateApiKey(ctx, mintReq)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), "keys cannot mint keys")

	statusReq := connect.NewRequest(&pb.GetStatusRequest{})
	withKey(statusReq)
	status, err := c.GetStatus(ctx, statusReq)
	require.NoError(t, err)
	assert.False(t, status.Msg.GetRbacIsSuperuser())
	assert.Equal(t, []string{"buildconfigs.view"}, status.Msg.GetRbacPermissions())

	builder, err := admin.CreateApiKey(ctx, connect.NewRequest(&pb.CreateApiKeyRequest{
		Name: "homepage-builder",
		Scope: &pb.ApiKeyScope{
			RestrictPermissions:  true,
			Permissions:          []string{"builds.run"},
			RestrictBuildConfigs: true,
			BuildConfigs:         []string{"homepage"},
		},
	}))
	require.NoError(t, err)
	buildReq = connect.NewRequest(&pb.BuildRequest{ConfigName: "sidebar"})
	buildReq.Header().Set("Authorization", "Bearer "+builder.Msg.GetKey())
	stream, err = c.StartBuild(ctx, buildReq)
	require.NoError(t, err)
	assert.False(t, stream.Receive())
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(stream.Err()), "builds.run, but not for this build config")

	// Widening the key from the web UI session takes effect on its next request.
	_, err = admin.UpdateApiKey(ctx, connect.NewRequest(&pb.UpdateApiKeyRequest{
		ApiKeyId: created.Msg.GetApiKey().GetId(),
		Scope:    &pb.ApiKeyScope{},
	}))
	require.NoError(t, err)
	listReq = connect.NewRequest(&pb.GetBuildConfigsRequest{})
	withKey(listReq)
	list, err = c.GetBuildConfigs(ctx, listReq)
	require.NoError(t, err)
	assert.Len(t, list.Msg.GetBuildConfigs(), 2)
}
