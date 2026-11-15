-- +migrate Up

CREATE TABLE user_accounts (
  id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  username TEXT NOT NULL COLLATE NOCASE,
  password_hash TEXT NOT NULL DEFAULT '',
  created_by TEXT NOT NULL DEFAULT 'admin-created'
);

CREATE UNIQUE INDEX idx_user_accounts_username ON user_accounts(username);

CREATE TABLE sessions (
  id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  sid TEXT NOT NULL,
  user_account_id INTEGER NOT NULL REFERENCES user_accounts(id) ON DELETE CASCADE,
  impersonator_user_id INTEGER REFERENCES user_accounts(id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX idx_sessions_sid ON sessions(sid);
CREATE INDEX idx_sessions_user ON sessions(user_account_id);

CREATE TABLE api_keys (
  id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  user_account_id INTEGER NOT NULL REFERENCES user_accounts(id) ON DELETE CASCADE,
  name TEXT NOT NULL DEFAULT '',
  key_value TEXT NOT NULL,
  read_only INTEGER NOT NULL DEFAULT 0,
  last_used_at TEXT
);

CREATE UNIQUE INDEX idx_api_keys_key_value ON api_keys(key_value);
CREATE INDEX idx_api_keys_user ON api_keys(user_account_id);

CREATE TABLE user_groups (
  id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  name TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_user_groups_name ON user_groups(name);

CREATE TABLE user_group_memberships (
  id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  user_account_id INTEGER NOT NULL REFERENCES user_accounts(id) ON DELETE CASCADE,
  user_group_id INTEGER NOT NULL REFERENCES user_groups(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_user_group_membership ON user_group_memberships(user_account_id, user_group_id);
CREATE INDEX idx_ugm_group ON user_group_memberships(user_group_id);

CREATE TABLE rbac_permissions (
  id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX idx_rbac_permissions_name ON rbac_permissions(name);

CREATE TABLE rbac_roles (
  id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX idx_rbac_roles_name ON rbac_roles(name);

CREATE TABLE rbac_role_permissions (
  role_id INTEGER NOT NULL REFERENCES rbac_roles(id) ON DELETE CASCADE,
  permission_id INTEGER NOT NULL REFERENCES rbac_permissions(id) ON DELETE CASCADE,
  PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX idx_rrp_perm ON rbac_role_permissions(permission_id);

CREATE TABLE rbac_group_roles (
  user_group_id INTEGER NOT NULL REFERENCES user_groups(id) ON DELETE CASCADE,
  role_id INTEGER NOT NULL REFERENCES rbac_roles(id) ON DELETE CASCADE,
  PRIMARY KEY (user_group_id, role_id)
);

CREATE INDEX idx_rgr_role ON rbac_group_roles(role_id);

INSERT INTO rbac_permissions (name, description) VALUES
('app.access', 'Use the application (non-admin APIs)'),
('users.view', 'List and view user accounts'),
('users.create', 'Create user accounts'),
('users.delete', 'Delete user accounts'),
('users.reset-password', 'Reset passwords for other users'),
('usergroups.view', 'List user groups and view membership'),
('usergroups.manage', 'Create and delete user groups; manage membership'),
('rbac.view', 'View roles and permissions'),
('rbac.manage', 'Create, update, and delete roles; assign roles to groups'),
('system.settings', 'View and modify system settings'),
('system.logs', 'View logs, job status, and run maintenance'),
('system.impersonate', 'Impersonate other users');

INSERT INTO rbac_roles (name, description) VALUES
('superuser', 'All permissions (system role)'),
('member', 'Standard application access');

INSERT INTO rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM rbac_roles r CROSS JOIN rbac_permissions p WHERE r.name = 'superuser';

INSERT INTO rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM rbac_roles r JOIN rbac_permissions p ON p.name = 'app.access' WHERE r.name = 'member';

INSERT INTO user_groups (name) VALUES ('Everyone'), ('Administrators');

INSERT INTO rbac_group_roles (user_group_id, role_id)
SELECT g.id, r.id FROM user_groups g CROSS JOIN rbac_roles r
WHERE g.name = 'Everyone' AND r.name = 'member';

INSERT INTO rbac_group_roles (user_group_id, role_id)
SELECT g.id, r.id FROM user_groups g CROSS JOIN rbac_roles r
WHERE g.name = 'Administrators' AND r.name = 'superuser';

-- +migrate Down

DROP TABLE IF EXISTS rbac_group_roles;
DROP TABLE IF EXISTS rbac_role_permissions;
DROP TABLE IF EXISTS rbac_roles;
DROP TABLE IF EXISTS rbac_permissions;
DROP TABLE IF EXISTS user_group_memberships;
DROP TABLE IF EXISTS user_groups;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS user_accounts;
