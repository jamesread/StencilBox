package iam

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jamesread/armature-iam/rbac"
)

// StencilBox permissions, on top of the armature-iam core catalog. Names must match
// rbac_permissions rows (migration 2.api-key-scopes.sql).
const (
	PermissionBuildConfigsView    = "buildconfigs.view"
	PermissionBuildsRun           = "builds.run"
	PermissionBuildConfigsGitPull = "buildconfigs.git-pull"
	PermissionDataFilesEdit       = "datafiles.edit"
	PermissionBuildConfigsManage  = "buildconfigs.manage"
)

// APIKeyScope narrows what an API key can do. Its owner's current permissions are always the
// ceiling: a scope can only take access away.
type APIKeyScope struct {
	// When RestrictPermissions is false the key has every permission its owner has.
	RestrictPermissions bool
	Permissions         []string

	// When RestrictBuildConfigs is false the key can use every build config.
	RestrictBuildConfigs bool
	BuildConfigs         []string
}

// AllowsPermission reports whether the scope lets the key use a permission its owner holds.
// A nil scope (not an API key) allows everything.
func (s *APIKeyScope) AllowsPermission(p string) bool {
	if s == nil || !s.RestrictPermissions {
		return true
	}
	for _, have := range s.Permissions {
		if have == p {
			return true
		}
	}
	return false
}

// AllowsBuildConfig reports whether the scope lets the key use a build config.
func (s *APIKeyScope) AllowsBuildConfig(name string) bool {
	if s == nil || !s.RestrictBuildConfigs {
		return true
	}
	for _, have := range s.BuildConfigs {
		if have == name {
			return true
		}
	}
	return false
}

// ErrAPIKeyNotFound is returned when a key does not exist or belongs to another user.
var ErrAPIKeyNotFound = errors.New("API key not found")

// LoadAPIKeyScopeByValue returns the scope of the key with this secret, or nil when the token is
// not an API key.
func LoadAPIKeyScopeByValue(ctx context.Context, db *sql.DB, keyValue string) (*APIKeyScope, error) {
	if !strings.HasPrefix(keyValue, APIKeyPrefix) {
		return nil, nil
	}
	var id int
	err := db.QueryRowContext(ctx, `SELECT id FROM api_keys WHERE key_value = ?`, keyValue).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load api key: %w", err)
	}
	return LoadAPIKeyScope(ctx, db, id)
}

// LoadAPIKeyScope returns the scope of an API key by ID.
func LoadAPIKeyScope(ctx context.Context, db *sql.DB, keyID int) (*APIKeyScope, error) {
	scope := &APIKeyScope{Permissions: []string{}, BuildConfigs: []string{}}
	err := db.QueryRowContext(ctx,
		`SELECT restrict_permissions, restrict_build_configs FROM api_keys WHERE id = ?`, keyID,
	).Scan(&scope.RestrictPermissions, &scope.RestrictBuildConfigs)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAPIKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load api key scope: %w", err)
	}
	if scope.Permissions, err = queryStrings(ctx, db,
		`SELECT p.name FROM api_key_permissions akp
		 INNER JOIN rbac_permissions p ON p.id = akp.permission_id
		 WHERE akp.api_key_id = ? ORDER BY p.name`, keyID); err != nil {
		return nil, err
	}
	if scope.BuildConfigs, err = queryStrings(ctx, db,
		`SELECT build_config_name FROM api_key_build_configs
		 WHERE api_key_id = ? ORDER BY build_config_name`, keyID); err != nil {
		return nil, err
	}
	return scope, nil
}

// SetAPIKeyScope replaces the scope of one of userID's API keys. Unknown permission names are
// refused, so a typo cannot silently produce a key with fewer rights than intended.
func SetAPIKeyScope(ctx context.Context, db *sql.DB, keyID, userID int, scope APIKeyScope) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("set api key scope begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		`UPDATE api_keys SET restrict_permissions = ?, restrict_build_configs = ?, updated_at = datetime('now')
		 WHERE id = ? AND user_account_id = ?`,
		scope.RestrictPermissions, scope.RestrictBuildConfigs, keyID, userID)
	if err != nil {
		return fmt.Errorf("update api key: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrAPIKeyNotFound
	}

	if err := replacePermissions(ctx, tx, keyID, scope); err != nil {
		return err
	}
	if err := replaceBuildConfigs(ctx, tx, keyID, scope); err != nil {
		return err
	}
	return tx.Commit()
}

func replacePermissions(ctx context.Context, tx *sql.Tx, keyID int, scope APIKeyScope) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM api_key_permissions WHERE api_key_id = ?`, keyID); err != nil {
		return fmt.Errorf("clear api key permissions: %w", err)
	}
	if !scope.RestrictPermissions {
		return nil
	}
	for _, name := range uniqueSorted(scope.Permissions) {
		res, err := tx.ExecContext(ctx,
			`INSERT INTO api_key_permissions (api_key_id, permission_id)
			 SELECT ?, id FROM rbac_permissions WHERE name = ?`, keyID, name)
		if err != nil {
			return fmt.Errorf("add api key permission %s: %w", name, err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("unknown permission %q", name)
		}
	}
	return nil
}

func replaceBuildConfigs(ctx context.Context, tx *sql.Tx, keyID int, scope APIKeyScope) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM api_key_build_configs WHERE api_key_id = ?`, keyID); err != nil {
		return fmt.Errorf("clear api key build configs: %w", err)
	}
	if !scope.RestrictBuildConfigs {
		return nil
	}
	for _, name := range uniqueSorted(scope.BuildConfigs) {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO api_key_build_configs (api_key_id, build_config_name) VALUES (?, ?)`, keyID, name); err != nil {
			return fmt.Errorf("add api key build config %s: %w", name, err)
		}
	}
	return nil
}

// NarrowRBAC returns the permissions an API key request really has: the owner's effective
// permissions limited by the key's scope. A restricted key never counts as superuser.
func NarrowRBAC(owner *rbac.EffectiveRBAC, allPermissions []string, scope *APIKeyScope) *rbac.EffectiveRBAC {
	if scope == nil || !scope.RestrictPermissions || owner == nil {
		return owner
	}
	out := &rbac.EffectiveRBAC{Permissions: map[string]bool{}, RoleNames: owner.RoleNames}
	candidates := allPermissions
	if !owner.IsSuperuser {
		candidates = candidates[:0:0]
		for p, ok := range owner.Permissions {
			if ok {
				candidates = append(candidates, p)
			}
		}
	}
	for _, p := range candidates {
		if scope.AllowsPermission(p) {
			out.Permissions[p] = true
		}
	}
	return out
}

func queryStrings(ctx context.Context, db *sql.DB, q string, args ...any) ([]string, error) {
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func uniqueSorted(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}
