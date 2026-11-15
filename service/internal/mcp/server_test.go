package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	pb "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1"
	clientapiconnect "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1/clientapi_pbconnect"
	"github.com/jamesread/StencilBox/internal/clientapi"
	"github.com/jamesread/StencilBox/internal/database"
	"github.com/jamesread/StencilBox/internal/iam"
	armpassword "github.com/jamesread/armature-iam/password"
	iamsqlite "github.com/jamesread/armature-iam/store/sqlite"
	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAPI answers the data file RPCs the tools under test use.
type fakeAPI struct {
	clientapiconnect.UnimplementedStencilBoxApiServiceHandler
	content string
}

func (f *fakeAPI) GetStatus(ctx context.Context, _ *connect.Request[pb.GetStatusRequest]) (*connect.Response[pb.GetStatusResponse], error) {
	return connect.NewResponse(&pb.GetStatusResponse{Username: "admin", IsLoggedIn: true, OutputPath: "/secret/path"}), nil
}

func (f *fakeAPI) GetDataFile(ctx context.Context, req *connect.Request[pb.GetDataFileRequest]) (*connect.Response[pb.GetDataFileResponse], error) {
	return connect.NewResponse(&pb.GetDataFileResponse{Content: f.content, BuildConfigName: req.Msg.GetBuildConfigName()}), nil
}

func (f *fakeAPI) UpdateDataFile(ctx context.Context, req *connect.Request[pb.UpdateDataFileRequest]) (*connect.Response[pb.UpdateDataFileResponse], error) {
	f.content = req.Msg.GetContent()
	return connect.NewResponse(&pb.UpdateDataFileResponse{RebuildingBuildConfigs: []string{req.Msg.GetBuildConfigName()}}), nil
}

func TestMCPToolsUseAPIAuth(t *testing.T) {
	ctx := context.Background()
	db, err := database.OpenMemory(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	st := iamsqlite.New(db)
	require.NoError(t, iam.Bootstrap(ctx, st))
	layer, err := iam.NewAuthLayer(st, nil)
	require.NoError(t, err)

	admin, err := st.GetUserByUsername(ctx, iam.DefaultAdminUsername)
	require.NoError(t, err)
	fullKey, err := armpassword.GenerateAPIKey(iam.APIKeyPrefix)
	require.NoError(t, err)
	_, err = st.CreateAPIKey(ctx, admin.ID, "full", fullKey, false)
	require.NoError(t, err)
	readKey, err := armpassword.GenerateAPIKey(iam.APIKeyPrefix)
	require.NoError(t, err)
	readKeyID, err := st.CreateAPIKey(ctx, admin.ID, "read", readKey, false)
	require.NoError(t, err)
	require.NoError(t, iam.SetAPIKeyScope(ctx, db, readKeyID, admin.ID, iam.APIKeyScope{
		RestrictPermissions: true, Permissions: []string{iam.PermissionBuildConfigsView},
	}))

	api := &fakeAPI{content: "links: []\n"}
	// The same chain as production: auth layer, then the API key scope interceptor.
	_, connectHandler := clientapiconnect.NewStencilBoxApiServiceHandler(api,
		connect.WithInterceptors(clientapi.NewScopeInterceptor(layer, db)))
	apiHandler := http.StripPrefix("/api", layer.WrapHandler(connectHandler))

	mux := http.NewServeMux()
	mux.Handle(Path, layer.WrapMCPHandler(NewHandler(apiHandler)))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	connectWith := func(key string) *mcpclient.Client {
		headers := map[string]string{}
		if key != "" {
			headers["Authorization"] = "Bearer " + key
		}
		c, err := mcpclient.NewStreamableHttpClient(srv.URL+Path, transport.WithHTTPHeaders(headers))
		require.NoError(t, err)
		t.Cleanup(func() { _ = c.Close() })
		require.NoError(t, c.Start(ctx))
		return c
	}
	call := func(c *mcpclient.Client, tool string, args map[string]any) *mcp.CallToolResult {
		req := mcp.CallToolRequest{}
		req.Params.Name = tool
		req.Params.Arguments = args
		res, err := c.CallTool(ctx, req)
		require.NoError(t, err)
		return res
	}
	text := func(res *mcp.CallToolResult) string {
		var b strings.Builder
		for _, c := range res.Content {
			if tc, ok := c.(mcp.TextContent); ok {
				b.WriteString(tc.Text)
			}
		}
		return b.String()
	}

	// No key: refused before MCP is reached.
	anon, err := mcpclient.NewStreamableHttpClient(srv.URL + Path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = anon.Close() })
	require.NoError(t, anon.Start(ctx))
	_, err = anon.Initialize(ctx, mcp.InitializeRequest{})
	assert.Error(t, err)

	full := connectWith(fullKey)
	_, err = full.Initialize(ctx, mcp.InitializeRequest{})
	require.NoError(t, err)
	tools, err := full.ListTools(ctx, mcp.ListToolsRequest{})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tools.Tools), 10)

	who := call(full, "stencilbox_whoami", nil)
	assert.False(t, who.IsError)
	assert.Contains(t, text(who), `"username": "admin"`)
	assert.NotContains(t, text(who), "/secret/path", "whoami only reports identity")

	got := call(full, "stencilbox_get_data_file", map[string]any{"build_config": "homepage", "data_file": "links"})
	assert.False(t, got.IsError, text(got))
	assert.Contains(t, text(got), "links: []")

	upd := call(full, "stencilbox_update_data_file", map[string]any{"build_config": "homepage", "data_file": "links", "content": "links: [a]\n"})
	assert.False(t, upd.IsError, text(upd))
	assert.Equal(t, "links: [a]\n", api.content)

	// The read-only key's scope applies to MCP exactly as to the API.
	ro := connectWith(readKey)
	_, err = ro.Initialize(ctx, mcp.InitializeRequest{})
	require.NoError(t, err)
	denied := call(ro, "stencilbox_update_data_file", map[string]any{"build_config": "homepage", "data_file": "links", "content": "x: 1"})
	assert.True(t, denied.IsError)
	assert.Contains(t, text(denied), "datafiles.edit")
	assert.Equal(t, "links: [a]\n", api.content)
}

// Keep the in-process transport honest about the request path.
func TestInProcessTransportKeepsPath(t *testing.T) {
	var seen string
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { seen = r.URL.Path })
	req := httptest.NewRequest("POST", inProcessBaseURL+"/StencilBox.clientapi.v1.StencilBoxApiService/GetStatus", nil)
	_, err := inProcessTransport{handler: h}.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, "/api/StencilBox.clientapi.v1.StencilBoxApiService/GetStatus", seen)
}

// Without a database, MCP works from localhost with no key; the in-process API calls keep the
// caller's address, so they pass the same localhost check.
func TestMCPLocalhostWithoutDatabase(t *testing.T) {
	ctx := context.Background()
	api := &fakeAPI{content: "a: 1\n"}
	_, connectHandler := clientapiconnect.NewStencilBoxApiServiceHandler(api)
	apiHandler := http.StripPrefix("/api", iam.LocalhostAuth{}.WrapHandler(connectHandler))
	mux := http.NewServeMux()
	mux.Handle(Path, iam.LocalhostAuth{}.WrapMCPHandler(NewHandler(apiHandler)))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c, err := mcpclient.NewStreamableHttpClient(srv.URL + Path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })
	require.NoError(t, c.Start(ctx))
	_, err = c.Initialize(ctx, mcp.InitializeRequest{})
	require.NoError(t, err)

	req := mcp.CallToolRequest{}
	req.Params.Name = "stencilbox_update_data_file"
	req.Params.Arguments = map[string]any{"build_config": "homepage", "data_file": "links", "content": "b: 2\n"}
	res, err := c.CallTool(ctx, req)
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Equal(t, "b: 2\n", api.content)

	// From elsewhere (here: through a proxy) MCP is refused.
	remote, err := mcpclient.NewStreamableHttpClient(srv.URL+Path, transport.WithHTTPHeaders(map[string]string{"X-Forwarded-For": "203.0.113.9"}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = remote.Close() })
	require.NoError(t, remote.Start(ctx))
	_, err = remote.Initialize(ctx, mcp.InitializeRequest{})
	assert.Error(t, err)
}
