package database

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequiredMigration(t *testing.T) {
	assert.Equal(t, "4.buildconfigs-manage.sql", RequiredMigration, "bump deliberately alongside a new migration")

	_, err := migrationsFS.ReadFile("migrations/" + RequiredMigration)
	require.NoError(t, err, "RequiredMigration must name an embedded migration")
}

func TestOpenAppliesMigrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", DefaultFilename)
	ctx := context.Background()

	db, err := Open(ctx, path)
	require.NoError(t, err)

	var n int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM rbac_roles WHERE name IN ('superuser', 'member')`).Scan(&n))
	assert.Equal(t, 2, n)
	require.NoError(t, db.Close())

	// Re-opening an up-to-date database is a no-op.
	db, err = Open(ctx, path)
	require.NoError(t, err)
	require.NoError(t, db.Close())
}

func TestPath(t *testing.T) {
	t.Setenv("STENCILBOX_DB_PATH", "")
	assert.Equal(t, filepath.Join("/config", DefaultFilename), Path("/config"))
	assert.Equal(t, DefaultFilename, Path(""))

	t.Setenv("STENCILBOX_DB_PATH", "/data/sb.db")
	assert.Equal(t, "/data/sb.db", Path("/config"))
}
