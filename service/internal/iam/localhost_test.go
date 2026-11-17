package iam

import (
	"context"
	"net/http/httptest"
	"testing"

	clientapiconnect "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1/clientapi_pbconnect"
	armlayer "github.com/jamesread/armature-iam/layer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsLocalRequest(t *testing.T) {
	req := func(addr string, headers ...string) bool {
		r := httptest.NewRequestWithContext(t.Context(), "POST", "/", nil)
		r.RemoteAddr = addr
		for i := 0; i+1 < len(headers); i += 2 {
			r.Header.Set(headers[i], headers[i+1])
		}
		return IsLocalRequest(r)
	}
	assert.True(t, req("127.0.0.1:5000"))
	assert.True(t, req("[::1]:5000"))
	assert.True(t, req("127.8.9.10:1"))
	assert.False(t, req("192.168.1.10:5000"))
	assert.False(t, req("172.17.0.1:5000"), "the Docker bridge gateway is not localhost")
	assert.False(t, req(""))
	assert.False(t, req("127.0.0.1:5000", "X-Forwarded-For", "203.0.113.9"), "a proxy on the same host")
	assert.False(t, req("127.0.0.1:5000", "Forwarded", "for=203.0.113.9"))
}

func TestLocalhostAuth(t *testing.T) {
	ctx := context.Background()
	call := func(procedure, addr string) (any, error) {
		r := httptest.NewRequestWithContext(ctx, "POST", procedure, nil)
		r.RemoteAddr = addr
		return LocalhostAuth{}.Handle(ctx, r)
	}

	info, err := call(clientapiconnect.StencilBoxApiServiceStartBuildProcedure, "127.0.0.1:1")
	require.NoError(t, err)
	au := info.(*armlayer.AuthenticatedUser)
	assert.Equal(t, LocalhostUsername, au.User.Username)
	assert.True(t, au.HasPermission(PermissionDataFilesEdit))
	assert.False(t, au.RBAC.IsSuperuser)

	_, err = call(clientapiconnect.StencilBoxApiServiceGetUserProcedure, "127.0.0.1:1")
	require.Error(t, err, "IAM needs a database")

	_, err = call(clientapiconnect.StencilBoxApiServiceStartBuildProcedure, "10.0.0.5:1")
	require.Error(t, err, "remote clients are refused")

	info, err = call(clientapiconnect.StencilBoxApiServiceGetStatusProcedure, "10.0.0.5:1")
	require.NoError(t, err, "GetStatus stays open so the web UI can explain")
	assert.Nil(t, info)
}
