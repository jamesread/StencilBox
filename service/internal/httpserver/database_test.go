package httpserver

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jamesread/StencilBox/internal/config"
	"github.com/jamesread/StencilBox/internal/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenDatabaseModes(t *testing.T) {
	ctx := context.Background()
	off := false

	t.Setenv("STENCILBOX_DB_PATH", filepath.Join(t.TempDir(), "sb.db"))
	assert.Nil(t, openDatabase(ctx, &config.Config{Database: config.DatabaseConfig{Enabled: &off}}), "turned off in config.yaml")

	t.Setenv("STENCILBOX_DB_DISABLED", "true")
	assert.Nil(t, openDatabase(ctx, &config.Config{}), "turned off by environment")
	t.Setenv("STENCILBOX_DB_DISABLED", "")

	db := openDatabase(ctx, &config.Config{})
	require.NotNil(t, db, "on by default")
	authenticator, layer := setupAuth(db, &config.Config{})
	assert.NotNil(t, layer)
	assert.IsType(t, &iam.AuthLayer{}, authenticator)
	require.NoError(t, db.Close())

	// A database that cannot be opened degrades to localhost-only access instead of exiting.
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600))
	t.Setenv("STENCILBOX_DB_PATH", filepath.Join(blocker, "sb.db"))
	assert.Nil(t, openDatabase(ctx, &config.Config{}))
	authenticator, layer = setupAuth(nil, &config.Config{})
	assert.Nil(t, layer)
	assert.IsType(t, iam.LocalhostAuth{}, authenticator)
}
