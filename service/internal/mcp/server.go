// Package mcp serves an MCP (Model Context Protocol) endpoint at /mcp, so AI assistants and
// agents can work with StencilBox: list and build build configs, and read and update data files.
//
// Every tool calls the Connect API in-process, through the same handler chain as /api, with the
// caller's Authorization header. Tools therefore get exactly the API's authentication, RBAC and
// API key scopes, and cannot do anything the caller's key could not do over /api.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"connectrpc.com/connect"
	pb "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1"
	clientapiconnect "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1/clientapi_pbconnect"
	"github.com/jamesread/StencilBox/internal/buildinfo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Path is where the MCP endpoint is mounted.
const Path = "/mcp"

// inProcessBaseURL is never dialled: requests go straight to the API handler.
const inProcessBaseURL = "http://stencilbox.in-process/api"

type authHeaderKey struct{}

// remoteAddrKey carries the MCP caller's address, so in-process API calls are checked as coming
// from the same client (localhost-only access depends on it).
type remoteAddrKey struct{}

// NewHandler returns the MCP Streamable HTTP handler. apiHandler is the handler mounted at /api
// (it must accept paths that still have the /api prefix).
func NewHandler(apiHandler http.Handler) http.Handler {
	client := clientapiconnect.NewStencilBoxApiServiceClient(
		&http.Client{Transport: inProcessTransport{handler: apiHandler}},
		inProcessBaseURL,
		connect.WithInterceptors(forwardAuthorization{}),
	)

	s := server.NewMCPServer(
		"StencilBox",
		buildinfo.Version,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
		server.WithInstructions("StencilBox builds static sites from templates and YAML data files. "+
			"A build config names a template and the data files it reads. To change a site, read a data "+
			"file, edit the YAML, and update it: every build config that uses the file is rebuilt."),
	)
	registerTools(s, client)

	return server.NewStreamableHTTPServer(s,
		server.WithEndpointPath(Path),
		server.WithStateLess(true),
		server.WithDisableLocalhostProtection(true),
		server.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			ctx = context.WithValue(ctx, remoteAddrKey{}, r.RemoteAddr)
			return context.WithValue(ctx, authHeaderKey{}, r.Header.Get("Authorization"))
		}),
	)
}

// inProcessTransport serves Connect client requests with the API handler, without a network hop.
type inProcessTransport struct {
	handler http.Handler
}

func (t inProcessTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if addr, ok := req.Context().Value(remoteAddrKey{}).(string); ok {
		req.RemoteAddr = addr
	}
	rec := httptest.NewRecorder()
	t.handler.ServeHTTP(rec, req)
	res := rec.Result()
	res.Request = req
	return res, nil
}

// forwardAuthorization copies the MCP caller's Authorization header onto each API call.
type forwardAuthorization struct{}

func authHeader(ctx context.Context) string {
	h, _ := ctx.Value(authHeaderKey{}).(string)
	return h
}

func (forwardAuthorization) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if h := authHeader(ctx); h != "" {
			req.Header().Set("Authorization", h)
		}
		return next(ctx, req)
	}
}

func (forwardAuthorization) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		if h := authHeader(ctx); h != "" {
			conn.RequestHeader().Set("Authorization", h)
		}
		return conn
	}
}

func (forwardAuthorization) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

var jsonOut = protojson.MarshalOptions{Multiline: true, Indent: "  ", EmitUnpopulated: true}

func protoResult(m proto.Message) (*mcp.CallToolResult, error) {
	b, err := jsonOut.Marshal(m)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

func errorResult(err error) (*mcp.CallToolResult, error) {
	var ce *connect.Error
	if errors.As(err, &ce) {
		return mcp.NewToolResultError(fmt.Sprintf("%s: %s", ce.Code(), ce.Message())), nil
	}
	return mcp.NewToolResultError(err.Error()), nil
}

func registerTools(s *server.MCPServer, c clientapiconnect.StencilBoxApiServiceClient) {
	s.AddTool(mcp.NewTool("stencilbox_whoami",
		mcp.WithDescription("Show the user this API key belongs to and the permissions it has."),
		mcp.WithReadOnlyHintAnnotation(true),
	), func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := c.GetStatus(ctx, connect.NewRequest(&pb.GetStatusRequest{}))
		if err != nil {
			return errorResult(err)
		}
		b, err := json.MarshalIndent(map[string]any{
			"username":    res.Msg.GetUsername(),
			"permissions": res.Msg.GetRbacPermissions(),
			"superuser":   res.Msg.GetRbacIsSuperuser(),
		}, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(string(b)), nil
	})

	s.AddTool(mcp.NewTool("stencilbox_list_build_configs",
		mcp.WithDescription("List build configs: each names a template, an output directory and the data files it reads."),
		mcp.WithReadOnlyHintAnnotation(true),
	), func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := c.GetBuildConfigs(ctx, connect.NewRequest(&pb.GetBuildConfigsRequest{}))
		if err != nil {
			return errorResult(err)
		}
		return protoResult(res.Msg)
	})

	s.AddTool(mcp.NewTool("stencilbox_get_build_config",
		mcp.WithDescription("Get one build config by name."),
		mcp.WithString("build_config", mcp.Required(), mcp.Description("Build config name")),
		mcp.WithReadOnlyHintAnnotation(true),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("build_config")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res, err := c.GetBuildConfig(ctx, connect.NewRequest(&pb.GetBuildConfigRequest{ConfigName: name}))
		if err != nil {
			return errorResult(err)
		}
		return protoResult(res.Msg)
	})

	s.AddTool(mcp.NewTool("stencilbox_list_templates",
		mcp.WithDescription("List the site templates available to build configs."),
		mcp.WithReadOnlyHintAnnotation(true),
	), func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := c.GetTemplates(ctx, connect.NewRequest(&pb.GetTemplatesRequest{}))
		if err != nil {
			return errorResult(err)
		}
		return protoResult(res.Msg)
	})

	s.AddTool(mcp.NewTool("stencilbox_start_build",
		mcp.WithDescription("Build a site from its build config and wait for the build to finish. Returns the final status and the build's progress messages."),
		mcp.WithString("build_config", mcp.Required(), mcp.Description("Build config name")),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("build_config")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return startBuild(ctx, c, name)
	})

	s.AddTool(mcp.NewTool("stencilbox_get_build_history",
		mcp.WithDescription("Recent builds of a build config: when, how long, whether they failed, and the output size."),
		mcp.WithString("build_config", mcp.Required(), mcp.Description("Build config name")),
		mcp.WithReadOnlyHintAnnotation(true),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("build_config")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res, err := c.GetBuildHistory(ctx, connect.NewRequest(&pb.GetBuildHistoryRequest{ConfigName: name}))
		if err != nil {
			return errorResult(err)
		}
		return protoResult(res.Msg)
	})

	s.AddTool(mcp.NewTool("stencilbox_clear_build_cache",
		mcp.WithDescription("Clear a build config's cache (downloaded repos and icons), so the next build fetches them again."),
		mcp.WithString("build_config", mcp.Required(), mcp.Description("Build config name")),
		mcp.WithDestructiveHintAnnotation(true),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("build_config")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res, err := c.ClearBuildCache(ctx, connect.NewRequest(&pb.ClearBuildCacheRequest{ConfigName: name}))
		if err != nil {
			return errorResult(err)
		}
		return protoResult(res.Msg)
	})

	s.AddTool(mcp.NewTool("stencilbox_list_data_files",
		mcp.WithDescription("List the YAML data files that build configs read."),
		mcp.WithReadOnlyHintAnnotation(true),
	), func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := c.ListDataFiles(ctx, connect.NewRequest(&pb.ListDataFilesRequest{}))
		if err != nil {
			return errorResult(err)
		}
		return protoResult(res.Msg)
	})

	s.AddTool(mcp.NewTool("stencilbox_get_data_file",
		mcp.WithDescription("Read a data file. The result includes sha256: pass it to stencilbox_update_data_file as expected_sha256."),
		mcp.WithString("build_config", mcp.Required(), mcp.Description("Build config name")),
		mcp.WithString("data_file", mcp.Required(), mcp.Description("Data file name, as listed in the build config")),
		mcp.WithReadOnlyHintAnnotation(true),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		bc, df, errResult := buildConfigAndDataFile(req)
		if errResult != nil {
			return errResult, nil
		}
		res, err := c.GetDataFile(ctx, connect.NewRequest(&pb.GetDataFileRequest{BuildConfigName: bc, DatafileName: df}))
		if err != nil {
			return errorResult(err)
		}
		return protoResult(res.Msg)
	})

	s.AddTool(mcp.NewTool("stencilbox_create_build_config",
		mcp.WithDescription("Create a build config: a site built from a template into an output directory. "+
			"Then add data files with stencilbox_create_data_file and build it with stencilbox_start_build."),
		mcp.WithString("build_config", mcp.Required(), mcp.Description("Name for the new build config")),
		mcp.WithString("template", mcp.Required(), mcp.Description("Template name, from stencilbox_list_templates")),
		mcp.WithString("output_dir", mcp.Required(), mcp.Description("Output directory, relative to the output root, eg \"team-links\"")),
		mcp.WithBoolean("build_on_startup", mcp.Description("Build this site whenever StencilBox starts")),
		mcp.WithDestructiveHintAnnotation(false),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("build_config")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		template, err := req.RequireString("template")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		out, err := req.RequireString("output_dir")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res, err := c.CreateBuildConfig(ctx, connect.NewRequest(&pb.CreateBuildConfigRequest{
			ConfigName: name, Template: template, OutputDir: out, BuildOnStartup: req.GetBool("build_on_startup", false),
		}))
		if err != nil {
			return errorResult(err)
		}
		return protoResult(res.Msg)
	})

	s.AddTool(mcp.NewTool("stencilbox_create_data_file",
		mcp.WithDescription("Add a data file to a build config. Templates read it by its name. "+
			"Give content to create a new YAML file, or a path to an existing file (and no content) to share it."),
		mcp.WithString("build_config", mcp.Required(), mcp.Description("Build config name")),
		mcp.WithString("data_file", mcp.Required(), mcp.Description("Name templates use for the data, eg \"links\" (letters, digits, underscores)")),
		mcp.WithString("content", mcp.Description("YAML content for a new file")),
		mcp.WithString("path", mcp.Description("Optional path relative to the build configs directory, eg data/links.yaml")),
		mcp.WithDestructiveHintAnnotation(false),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		bc, df, errResult := buildConfigAndDataFile(req)
		if errResult != nil {
			return errResult, nil
		}
		res, err := c.CreateDataFile(ctx, connect.NewRequest(&pb.CreateDataFileRequest{
			BuildConfigName: bc, DatafileName: df,
			Content: req.GetString("content", ""), Path: req.GetString("path", ""),
		}))
		if err != nil {
			return errorResult(err)
		}
		return protoResult(res.Msg)
	})

	s.AddTool(mcp.NewTool("stencilbox_update_data_file",
		mcp.WithDescription("Replace a data file with new YAML content. Every build config that uses the file is rebuilt. "+
			"Read the file first and pass its sha256 as expected_sha256, so a change made in the meantime is not overwritten."),
		mcp.WithString("build_config", mcp.Required(), mcp.Description("Build config name")),
		mcp.WithString("data_file", mcp.Required(), mcp.Description("Data file name, as listed in the build config")),
		mcp.WithString("content", mcp.Required(), mcp.Description("The complete new file content, as YAML")),
		mcp.WithString("expected_sha256", mcp.Description("sha256 from stencilbox_get_data_file; the update is refused if the file has changed since")),
		mcp.WithDestructiveHintAnnotation(true),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		bc, df, errResult := buildConfigAndDataFile(req)
		if errResult != nil {
			return errResult, nil
		}
		content, err := req.RequireString("content")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res, err := c.UpdateDataFile(ctx, connect.NewRequest(&pb.UpdateDataFileRequest{
			BuildConfigName: bc,
			DatafileName:    df,
			Content:         content,
			ExpectedSha256:  req.GetString("expected_sha256", ""),
		}))
		if err != nil {
			return errorResult(err)
		}
		return protoResult(res.Msg)
	})
}

func buildConfigAndDataFile(req mcp.CallToolRequest) (string, string, *mcp.CallToolResult) {
	bc, err := req.RequireString("build_config")
	if err != nil {
		return "", "", mcp.NewToolResultError(err.Error())
	}
	df, err := req.RequireString("data_file")
	if err != nil {
		return "", "", mcp.NewToolResultError(err.Error())
	}
	return bc, df, nil
}

func startBuild(ctx context.Context, c clientapiconnect.StencilBoxApiServiceClient, name string) (*mcp.CallToolResult, error) {
	stream, err := c.StartBuild(ctx, connect.NewRequest(&pb.BuildRequest{ConfigName: name}))
	if err != nil {
		return errorResult(err)
	}
	defer func() { _ = stream.Close() }()

	var last *pb.BuildUpdateResponse
	progress := []string{}
	for stream.Receive() {
		last = stream.Msg()
		progress = append(progress, last.GetStatus())
	}
	if err := stream.Err(); err != nil {
		return errorResult(err)
	}
	if last == nil {
		return mcp.NewToolResultError("the build returned no status"), nil
	}

	b, err := jsonOut.Marshal(last)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	text := string(b) + "\n\nProgress:\n- " + strings.Join(progress, "\n- ")
	if last.GetIsError() {
		return mcp.NewToolResultError("Build failed.\n" + text), nil
	}
	return mcp.NewToolResultText(text), nil
}
