package clientapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	pb "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1"
	clientapiconnect "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1/clientapi_pbconnect"
	"github.com/jamesread/StencilBox/internal/buildconfigs"
	"github.com/jamesread/StencilBox/internal/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Without a database the server still works for localhost, and refuses everyone else.
func TestNoDatabaseLocalhostOnly(t *testing.T) {
	ctx := context.Background()
	api := &ClientApi{buildConfigs: map[string]*buildconfigs.BuildConfig{"homepage": {Name: "homepage"}}}
	api.refreshDefaultAdminPassword(ctx) // nil-safe without a database
	_, handler := clientapiconnect.NewStencilBoxApiServiceHandler(api, connect.WithInterceptors(api.ScopeInterceptor()))
	srv := httptest.NewServer(iam.LocalhostAuth{}.WrapHandler(handler)) // listens on 127.0.0.1
	t.Cleanup(srv.Close)
	c := clientapiconnect.NewStencilBoxApiServiceClient(http.DefaultClient, srv.URL)

	status, err := c.GetStatus(ctx, connect.NewRequest(&pb.GetStatusRequest{}))
	require.NoError(t, err)
	assert.True(t, status.Msg.GetIsLoggedIn())
	assert.Equal(t, iam.LocalhostUsername, status.Msg.GetUsername())
	assert.False(t, status.Msg.GetDatabaseEnabled())
	assert.False(t, status.Msg.GetRbacIsSuperuser())
	assert.Contains(t, status.Msg.GetRbacPermissions(), "builds.run")
	assert.NotContains(t, status.Msg.GetRbacPermissions(), "users.view")

	configs, err := c.GetBuildConfigs(ctx, connect.NewRequest(&pb.GetBuildConfigsRequest{}))
	require.NoError(t, err)
	assert.Len(t, configs.Msg.GetBuildConfigs(), 1)

	_, err = c.LoginWithUsernameAndPassword(ctx, connect.NewRequest(&pb.LoginWithUsernameAndPasswordRequest{Username: "admin", Password: "admin"}))
	assert.Equal(t, connect.CodeUnavailable, connect.CodeOf(err), "no accounts without a database")
	_, err = c.ListApiKeys(ctx, connect.NewRequest(&pb.ListApiKeysRequest{}))
	assert.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
	_, err = c.GetUser(ctx, connect.NewRequest(&pb.GetUserRequest{UserId: 1}))
	require.Error(t, err)

	// Through a proxy on the same host the request is not trusted as local.
	proxied := connect.NewRequest(&pb.GetBuildConfigsRequest{})
	proxied.Header().Set("X-Forwarded-For", "203.0.113.9")
	_, err = c.GetBuildConfigs(ctx, proxied)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	proxiedStatus := connect.NewRequest(&pb.GetStatusRequest{})
	proxiedStatus.Header().Set("X-Forwarded-For", "203.0.113.9")
	status, err = c.GetStatus(ctx, proxiedStatus)
	require.NoError(t, err)
	assert.False(t, status.Msg.GetIsLoggedIn())
	assert.False(t, status.Msg.GetDatabaseEnabled(), "the web UI explains instead of showing a login form")
}
