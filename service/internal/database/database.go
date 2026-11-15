// Package database opens the StencilBox SQLite database and applies its schema migrations.
//
// Migrations are sql-migrate scripts embedded in the binary and applied in-process at startup,
// so the container image and the standalone binary need no separate migration step.
package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	migrate "github.com/rubenv/sql-migrate"
	log "github.com/sirupsen/logrus"

	// Pure Go SQLite driver (registered as "sqlite"); release builds use CGO_ENABLED=0.
	_ "modernc.org/sqlite"
)

// RequiredMigration is the newest sql-migrate id this binary depends on. Bump it in the same
// change that adds a migration the code needs.
const RequiredMigration = "4.buildconfigs-manage.sql"

// DefaultFilename is the database file created next to config.yaml.
const DefaultFilename = "stencilbox.db"

const migrationsTable = "migrations"

//go:embed migrations/*.sql
var migrationsFS embed.FS

// maxOpenConns allows concurrent readers under WAL; busy_timeout covers writer contention.
const maxOpenConns = 4

// Path returns the database file path: $STENCILBOX_DB_PATH, or DefaultFilename in configDir.
func Path(configDir string) string {
	if p := strings.TrimSpace(os.Getenv("STENCILBOX_DB_PATH")); p != "" {
		return p
	}
	if configDir == "" {
		configDir = "."
	}
	return filepath.Join(configDir, DefaultFilename)
}

// Open opens (creating if needed) the SQLite database at path, applies pending migrations and
// checks RequiredMigration is present.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}
	db, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, fmt.Errorf("open database %s: %w", path, err)
	}
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxOpenConns)

	if err := setup(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	log.WithField("path", path).Info("Database ready")
	return db, nil
}

// OpenMemory opens a migrated in-memory database, for tests.
func OpenMemory(ctx context.Context) (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file::memory:?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	// Each connection to :memory: is its own database, so keep exactly one.
	db.SetMaxOpenConns(1)
	if err := setup(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func dsn(path string) string {
	return "file:" + path +
		"?_pragma=foreign_keys(1)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=busy_timeout(5000)"
}

func setup(ctx context.Context, db *sql.DB) error {
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	if err := Migrate(db); err != nil {
		return err
	}
	return assertMigration(ctx, db, RequiredMigration)
}

// Migrate applies all pending embedded migrations.
func Migrate(db *sql.DB) error {
	migrate.SetTable(migrationsTable)
	source := &migrate.EmbedFileSystemMigrationSource{FileSystem: migrationsFS, Root: "migrations"}
	n, err := migrate.Exec(db, "sqlite3", source, migrate.Up)
	if err != nil {
		return fmt.Errorf("apply database migrations: %w", err)
	}
	if n > 0 {
		log.WithField("count", n).Info("Applied database migrations")
	}
	return nil
}

func assertMigration(ctx context.Context, db *sql.DB, required string) error {
	var n int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+migrationsTable+` WHERE id = ?`, required).Scan(&n)
	if err != nil {
		return fmt.Errorf("query the migrations table: %w", err)
	}
	if n > 0 {
		return nil
	}
	latest, err := latestMigration(ctx, db)
	if err != nil {
		return err
	}
	return fmt.Errorf("requires database version %s but the database is at version %s", required, latest)
}

func latestMigration(ctx context.Context, db *sql.DB) (string, error) {
	var id sql.NullString
	err := db.QueryRowContext(ctx,
		`SELECT id FROM `+migrationsTable+` ORDER BY applied_at DESC, id DESC LIMIT 1`).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !id.Valid) {
		return "null", nil
	}
	if err != nil {
		return "", fmt.Errorf("query latest migration: %w", err)
	}
	return id.String, nil
}
