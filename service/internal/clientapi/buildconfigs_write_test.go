package clientapi

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	pb "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1"
	"github.com/jamesread/StencilBox/internal/buildconfigs"
	"github.com/jamesread/StencilBox/internal/config"
	"github.com/jamesread/StencilBox/internal/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateBuildConfigAndDataFile(t *testing.T) {
	newClient, api := newIamTestClient(t)
	ctx := context.Background()

	configDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(configDir, "buildconfigs"), 0o750))
	config.SetConfigDir(configDir)
	t.Cleanup(func() { config.SetConfigDir("") })

	templates := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(templates, "my-template"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(templates, "my-template", "index.html"), []byte("<html></html>"), 0o600))
	t.Setenv("STENCILBOX_TEMPLATES_DIR", templates)

	api.buildCtx, api.buildCancel = context.WithCancel(ctx)
	api.BaseOutputDir = t.TempDir()
	api.historyPath = filepath.Join(t.TempDir(), "history.json")
	api.buildHistory = map[string][]*pb.BuildHistoryEntry{}
	api.buildConfigs = map[string]*buildconfigs.BuildConfig{}
	t.Cleanup(func() { api.buildCancel(); api.builds.Wait() })

	admin := newClient()
	require.True(t, login(t, admin, iam.DefaultAdminUsername, iam.DefaultAdminPassword))

	create := func(name, template, out string) error {
		_, err := admin.CreateBuildConfig(ctx, connect.NewRequest(&pb.CreateBuildConfigRequest{
			ConfigName: name, Template: template, OutputDir: out,
		}))
		return err
	}
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(create("Site", "no-such-template", "site")))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(create("Site", "my-template", "../escape")))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(create("Site", "my-template", "webui")))
	require.NoError(t, create("Team Links", "my-template", "team"))
	assert.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(create("Team Links", "my-template", "team2")))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(create("Other", "my-template", "team")), "output overlaps")

	got, err := admin.GetBuildConfig(ctx, connect.NewRequest(&pb.GetBuildConfigRequest{ConfigName: "Team Links"}))
	require.NoError(t, err)
	assert.Equal(t, "team-links.yaml", got.Msg.GetBuildConfig().GetFilename())

	addData := func(m *pb.CreateDataFileRequest) error {
		_, addErr := admin.CreateDataFile(ctx, connect.NewRequest(m))
		return addErr
	}
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(addData(&pb.CreateDataFileRequest{
		BuildConfigName: "Team Links", DatafileName: "links", Path: "../../evil.yaml", Content: "a: 1",
	})), "paths outside the build configs directory are refused")
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(addData(&pb.CreateDataFileRequest{
		BuildConfigName: "Team Links", DatafileName: "bad-name", Content: "a: 1",
	})))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(addData(&pb.CreateDataFileRequest{
		BuildConfigName: "Team Links", DatafileName: "links", Content: "links: [unclosed",
	})))

	res, err := admin.CreateDataFile(ctx, connect.NewRequest(&pb.CreateDataFileRequest{
		BuildConfigName: "Team Links", DatafileName: "links", Content: "links:\n  - title: GitHub\n",
	}))
	require.NoError(t, err)
	wantPath := filepath.Join(configDir, "buildconfigs", "data", "team-links-links.yaml")
	assert.Equal(t, wantPath, res.Msg.GetPath())
	assert.Equal(t, []string{"Team Links"}, res.Msg.GetRebuildingBuildConfigs())

	assert.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(addData(&pb.CreateDataFileRequest{
		BuildConfigName: "Team Links", DatafileName: "links", Content: "a: 1",
	})), "the name is taken")

	// The new data file can be read and updated like any other.
	df, err := admin.GetDataFile(ctx, connect.NewRequest(&pb.GetDataFileRequest{BuildConfigName: "Team Links", DatafileName: "links"}))
	require.NoError(t, err)
	assert.Equal(t, "links:\n  - title: GitHub\n", df.Msg.GetContent())

	// A second data file reusing an existing file (no content).
	require.NoError(t, addData(&pb.CreateDataFileRequest{
		BuildConfigName: "Team Links", DatafileName: "shared", Path: "data/team-links-links.yaml",
	}))

	// A key limited to another build config cannot create or extend this one.
	key, err := admin.CreateApiKey(ctx, connect.NewRequest(&pb.CreateApiKeyRequest{
		Name: "scoped", Scope: &pb.ApiKeyScope{RestrictBuildConfigs: true, BuildConfigs: []string{"Something Else"}},
	}))
	require.NoError(t, err)
	keyed := connect.NewRequest(&pb.CreateDataFileRequest{BuildConfigName: "Team Links", DatafileName: "more", Content: "a: 1"})
	keyed.Header().Set("Authorization", "Bearer "+key.Msg.GetKey())
	_, err = newClient().CreateDataFile(ctx, keyed)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	keyedCreate := connect.NewRequest(&pb.CreateBuildConfigRequest{ConfigName: "Sneaky", Template: "my-template", OutputDir: "sneaky"})
	keyedCreate.Header().Set("Authorization", "Bearer "+key.Msg.GetKey())
	_, err = newClient().CreateBuildConfig(ctx, keyedCreate)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

func TestUpdateDataFileOutsideConfigDirRefused(t *testing.T) {
	newClient, api := newIamTestClient(t)
	ctx := context.Background()
	root := t.TempDir()
	configDir := filepath.Join(root, "buildconfigs")
	require.NoError(t, os.MkdirAll(configDir, 0o750))
	outside := filepath.Join(root, "outside.yaml")
	require.NoError(t, os.WriteFile(outside, []byte("a: 1\n"), 0o600))
	api.buildConfigs = map[string]*buildconfigs.BuildConfig{
		"hand": {Name: "hand", Path: filepath.Join(configDir, "hand.yaml"), Datafiles: map[string]string{"x": "../outside.yaml"}},
	}
	admin := newClient()
	require.True(t, login(t, admin, iam.DefaultAdminUsername, iam.DefaultAdminPassword))

	_, err := admin.UpdateDataFile(ctx, connect.NewRequest(&pb.UpdateDataFileRequest{BuildConfigName: "hand", DatafileName: "x", Content: "b: 2"}))
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	raw, _ := os.ReadFile(outside)
	assert.Equal(t, "a: 1\n", string(raw))
}
