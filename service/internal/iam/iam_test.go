package iam

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	clientapiconnect "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1/clientapi_pbconnect"
	"github.com/jamesread/StencilBox/internal/database"
	armlayer "github.com/jamesread/armature-iam/layer"
	"github.com/jamesread/armature-iam/rbac"
	"github.com/jamesread/armature-iam/store"
	iamsqlite "github.com/jamesread/armature-iam/store/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newStore(t *testing.T) *iamsqlite.SQLite {
	t.Helper()
	db, err := database.OpenMemory(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return iamsqlite.New(db)
}

func TestRequiredPermission(t *testing.T) {
	assert.Equal(t, rbac.PermissionUsersView, RequiredPermission(clientapiconnect.StencilBoxApiServiceGetUserProcedure))
	assert.Equal(t, rbac.PermissionRbacManage, RequiredPermission(clientapiconnect.StencilBoxApiServiceSetUserGroupRbacRolesProcedure))
	assert.Equal(t, rbac.PermissionAppAccess, RequiredPermission(clientapiconnect.StencilBoxApiServiceGetUsersProcedure), "pickers")
	assert.Equal(t, PermissionBuildsRun, RequiredPermission(clientapiconnect.StencilBoxApiServiceStartBuildProcedure))
	assert.Equal(t, rbac.PermissionAppAccess, RequiredPermission(clientapiconnect.StencilBoxApiServiceInitProcedure), "unlisted")
}

func TestBootstrapCreatesDefaultAdminSuperuser(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)

	require.NoError(t, Bootstrap(ctx, st))
	require.NoError(t, Bootstrap(ctx, st), "idempotent")

	n, err := st.CountUserAccounts(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	admin, err := st.GetUserByUsername(ctx, DefaultAdminUsername)
	require.NoError(t, err)
	require.NotNil(t, admin)
	rb, err := st.LoadEffectiveRBAC(ctx, admin.ID)
	require.NoError(t, err)
	assert.True(t, rb.IsSuperuser)
	assert.True(t, UsesDefaultAdminPassword(ctx, st))

	groupIDs, err := st.ListUserGroupIDsForUser(ctx, admin.ID)
	require.NoError(t, err)
	everyone, err := st.GetUserGroupByName(ctx, rbac.GroupEveryone)
	require.NoError(t, err)
	admins, err := st.GetUserGroupByName(ctx, rbac.GroupAdministrators)
	require.NoError(t, err)
	assert.ElementsMatch(t, []int{everyone.ID, admins.ID}, groupIDs, "every user is in Everyone")
}

func TestSetUserGroupsKeepsSuperuserAndEveryone(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	require.NoError(t, Bootstrap(ctx, st))
	admin, _ := st.GetUserByUsername(ctx, DefaultAdminUsername)

	err := SetUserGroups(ctx, st.DB(), admin.ID, nil)
	require.ErrorIs(t, err, store.ErrNoSuperuser, "the only superuser cannot leave Administrators")

	bobID, err := st.CreateUserAccount(ctx, "bob", "", store.UserCreatedByAdmin)
	require.NoError(t, err)
	admins, err := st.GetUserGroupByName(ctx, rbac.GroupAdministrators)
	require.NoError(t, err)

	require.NoError(t, SetUserGroups(ctx, st.DB(), bobID, []int{admins.ID}))
	require.NoError(t, SetUserGroups(ctx, st.DB(), admin.ID, nil), "bob is now a superuser")

	rb, err := st.LoadEffectiveRBAC(ctx, admin.ID)
	require.NoError(t, err)
	assert.False(t, rb.IsSuperuser)
	assert.True(t, rb.Has(rbac.PermissionAppAccess), "still in Everyone")
}

func TestGuestOnlyReachesAllowListedProcedures(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	require.NoError(t, Bootstrap(ctx, st))
	layer, err := NewAuthLayer(st, nil)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(ctx, "POST", clientapiconnect.StencilBoxApiServiceGetStatusProcedure, nil)
	info, err := layer.Handle(ctx, req)
	require.NoError(t, err)
	assert.Nil(t, info)

	req = httptest.NewRequestWithContext(ctx, "POST", clientapiconnect.StencilBoxApiServiceStartBuildProcedure, nil)
	_, err = layer.Handle(ctx, req)
	assert.Error(t, err)
}

func TestSessionCookieAndRBAC(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	require.NoError(t, Bootstrap(ctx, st))
	layer, err := NewAuthLayer(st, nil)
	require.NoError(t, err)

	bobID, err := st.CreateUserAccount(ctx, "bob", "", store.UserCreatedByAdmin)
	require.NoError(t, err)
	require.NoError(t, st.EnsureUserInEveryoneGroup(ctx, bobID))
	require.NoError(t, st.CreateSession(ctx, "sid-bob", bobID, nil))

	call := func(procedure string) (any, error) {
		req := httptest.NewRequestWithContext(ctx, "POST", procedure, nil)
		req.AddCookie(&http.Cookie{Name: CookieName, Value: "sid-bob"})
		return layer.Handle(ctx, req)
	}

	info, err := call(clientapiconnect.StencilBoxApiServiceStartBuildProcedure)
	require.NoError(t, err, "member has app.access")
	assert.Equal(t, "bob", info.(*armlayer.AuthenticatedUser).User.Username)

	_, err = call(clientapiconnect.StencilBoxApiServiceCreateUserProcedure)
	assert.Error(t, err, "member lacks users.create")
}

func TestSessionCookieSecure(t *testing.T) {
	st := newStore(t)
	layer, err := NewAuthLayer(st, nil)
	require.NoError(t, err)

	plain := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil)
	assert.False(t, layer.SessionCookie("x", plain, plain.Header).Secure)

	proxied := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil)
	proxied.Header.Set("X-Forwarded-Proto", "https")
	assert.True(t, layer.SessionCookie("x", proxied, proxied.Header).Secure)

	direct := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil)
	direct.TLS = &tls.ConnectionState{}
	assert.True(t, layer.SessionCookie("x", direct, direct.Header).Secure)

	layer.secureCookies = "true"
	assert.True(t, layer.SessionCookie("x", plain, plain.Header).Secure)
	assert.Equal(t, -1, layer.ClearCookie(plain, plain.Header).MaxAge)
}

func TestAPIKeyScopeRoundTrip(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	require.NoError(t, Bootstrap(ctx, st))
	admin, _ := st.GetUserByUsername(ctx, DefaultAdminUsername)
	keyID, err := st.CreateAPIKey(ctx, admin.ID, "ci", APIKeyPrefix+"secret", false)
	require.NoError(t, err)

	scope, err := LoadAPIKeyScopeByValue(ctx, st.DB(), APIKeyPrefix+"secret")
	require.NoError(t, err)
	assert.False(t, scope.RestrictPermissions, "new keys act as their owner")
	assert.True(t, scope.AllowsBuildConfig("anything"))

	want := APIKeyScope{
		RestrictPermissions:  true,
		Permissions:          []string{PermissionBuildsRun, PermissionBuildsRun},
		RestrictBuildConfigs: true,
		BuildConfigs:         []string{"homepage"},
	}
	require.NoError(t, SetAPIKeyScope(ctx, st.DB(), keyID, admin.ID, want))
	scope, err = LoadAPIKeyScope(ctx, st.DB(), keyID)
	require.NoError(t, err)
	assert.Equal(t, []string{PermissionBuildsRun}, scope.Permissions)
	assert.True(t, scope.AllowsBuildConfig("homepage"))
	assert.False(t, scope.AllowsBuildConfig("sidebar"))
	assert.False(t, scope.AllowsPermission(rbac.PermissionUsersCreate))

	require.ErrorIs(t, SetAPIKeyScope(ctx, st.DB(), keyID, admin.ID+1, want), ErrAPIKeyNotFound, "only the owner")
	bad := want
	bad.Permissions = []string{"no.such.permission"}
	require.Error(t, SetAPIKeyScope(ctx, st.DB(), keyID, admin.ID, bad))

	narrowed := NarrowRBAC(&rbac.EffectiveRBAC{IsSuperuser: true}, []string{PermissionBuildsRun, rbac.PermissionUsersCreate}, scope)
	assert.False(t, narrowed.IsSuperuser)
	assert.True(t, narrowed.Has(PermissionBuildsRun))
	assert.False(t, narrowed.Has(rbac.PermissionUsersCreate))

	member := NarrowRBAC(&rbac.EffectiveRBAC{Permissions: map[string]bool{rbac.PermissionAppAccess: true}}, nil, scope)
	assert.False(t, member.Has(PermissionBuildsRun), "the owner's permissions are the ceiling")
}
