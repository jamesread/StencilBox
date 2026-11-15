-- +migrate Up

-- Build permissions, split out of app.access so API keys (and roles) can be limited to them.
INSERT INTO rbac_permissions (name, description) VALUES
('buildconfigs.view', 'View build configs, build history, templates and data files'),
('builds.run', 'Start builds and clear build caches'),
('buildconfigs.git-pull', 'Git pull the build configs directory');

-- member keeps everything it could do when these were part of app.access.
INSERT OR IGNORE INTO rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM rbac_roles r CROSS JOIN rbac_permissions p
WHERE r.name IN ('member', 'superuser')
AND p.name IN ('buildconfigs.view', 'builds.run', 'buildconfigs.git-pull');

-- An API key acts as its owner, optionally narrowed to some permissions and build configs.
-- The owner's current access is always the ceiling.
ALTER TABLE api_keys ADD COLUMN restrict_permissions INTEGER NOT NULL DEFAULT 0;
ALTER TABLE api_keys ADD COLUMN restrict_build_configs INTEGER NOT NULL DEFAULT 0;

CREATE TABLE api_key_permissions (
  api_key_id INTEGER NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
  permission_id INTEGER NOT NULL REFERENCES rbac_permissions(id) ON DELETE CASCADE,
  PRIMARY KEY (api_key_id, permission_id)
);

CREATE TABLE api_key_build_configs (
  api_key_id INTEGER NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
  build_config_name TEXT NOT NULL,
  PRIMARY KEY (api_key_id, build_config_name)
);

-- +migrate Down

DROP TABLE IF EXISTS api_key_build_configs;
DROP TABLE IF EXISTS api_key_permissions;
ALTER TABLE api_keys DROP COLUMN restrict_build_configs;
ALTER TABLE api_keys DROP COLUMN restrict_permissions;
DELETE FROM rbac_permissions WHERE name IN ('buildconfigs.view', 'builds.run', 'buildconfigs.git-pull');
