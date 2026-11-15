package clientapi

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	pb "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1"
	"github.com/jamesread/StencilBox/internal/buildconfigs"
	"github.com/jamesread/StencilBox/internal/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAndUpdateDataFile(t *testing.T) {
	newClient, api := newIamTestClient(t)
	ctx := context.Background()

	dir := t.TempDir()
	dataPath := filepath.Join(dir, "data", "links.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(dataPath), 0o750))
	require.NoError(t, os.WriteFile(dataPath, []byte("links: []\n"), 0o640))

	api.buildCtx, api.buildCancel = context.WithCancel(ctx)
	api.BaseOutputDir = t.TempDir()
	api.historyPath = filepath.Join(t.TempDir(), "history.json")
	api.buildHistory = map[string][]*pb.BuildHistoryEntry{}
	api.buildConfigs = map[string]*buildconfigs.BuildConfig{
		"homepage": {Name: "homepage", Path: filepath.Join(dir, "homepage.yaml"), Datafiles: map[string]string{"links": "data/links.yaml"}},
		"other":    {Name: "other", Path: filepath.Join(dir, "other.yaml"), Datafiles: map[string]string{"links": "data/links.yaml"}},
	}
	t.Cleanup(func() { api.buildCancel(); api.builds.Wait() })

	admin := newClient()
	require.True(t, login(t, admin, iam.DefaultAdminUsername, iam.DefaultAdminPassword))

	got, err := admin.GetDataFile(ctx, connect.NewRequest(&pb.GetDataFileRequest{BuildConfigName: "homepage", DatafileName: "links"}))
	require.NoError(t, err)
	assert.Equal(t, "links: []\n", got.Msg.GetContent())
	original := got.Msg.GetSha256()

	_, err = admin.UpdateDataFile(ctx, connect.NewRequest(&pb.UpdateDataFileRequest{
		BuildConfigName: "homepage", DatafileName: "links", Content: "links: [unclosed",
	}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "invalid YAML is refused")

	updated, err := admin.UpdateDataFile(ctx, connect.NewRequest(&pb.UpdateDataFileRequest{
		BuildConfigName: "homepage", DatafileName: "links", Content: "links:\n  - name: GitHub\n", ExpectedSha256: original,
	}))
	require.NoError(t, err)
	assert.Equal(t, []string{"homepage", "other"}, updated.Msg.GetRebuildingBuildConfigs(), "every user of the file is rebuilt")

	onDisk, err := os.ReadFile(dataPath)
	require.NoError(t, err)
	assert.Equal(t, "links:\n  - name: GitHub\n", string(onDisk))
	info, err := os.Stat(dataPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o640), info.Mode().Perm(), "the file mode is kept")

	_, err = admin.UpdateDataFile(ctx, connect.NewRequest(&pb.UpdateDataFileRequest{
		BuildConfigName: "homepage", DatafileName: "links", Content: "links: []\n", ExpectedSha256: original,
	}))
	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err), "a stale hash is refused")

	_, err = admin.UpdateDataFile(ctx, connect.NewRequest(&pb.UpdateDataFileRequest{
		BuildConfigName: "homepage", DatafileName: "../../etc/passwd", Content: "x: 1",
	}))
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "only data files a build config names")

	// A key that may edit only homepage's data files.
	created, err := admin.CreateApiKey(ctx, connect.NewRequest(&pb.CreateApiKeyRequest{
		Name: "editor",
		Scope: &pb.ApiKeyScope{
			RestrictPermissions: true, Permissions: []string{"datafiles.edit"},
			RestrictBuildConfigs: true, BuildConfigs: []string{"homepage"},
		},
	}))
	require.NoError(t, err)
	keyed := func(m *pb.UpdateDataFileRequest) *connect.Request[pb.UpdateDataFileRequest] {
		r := connect.NewRequest(m)
		r.Header().Set("Authorization", "Bearer "+created.Msg.GetKey())
		return r
	}
	c := newClient()
	_, err = c.UpdateDataFile(ctx, keyed(&pb.UpdateDataFileRequest{BuildConfigName: "other", DatafileName: "links", Content: "a: 1"}))
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	_, err = c.UpdateDataFile(ctx, keyed(&pb.UpdateDataFileRequest{BuildConfigName: "homepage", DatafileName: "links", Content: "a: 1"}))
	require.NoError(t, err)
}
