-- +migrate Up

INSERT INTO rbac_permissions (name, description) VALUES
('datafiles.edit', 'Replace the contents of data files used by build configs');

INSERT OR IGNORE INTO rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM rbac_roles r CROSS JOIN rbac_permissions p
WHERE r.name IN ('member', 'superuser') AND p.name = 'datafiles.edit';

-- +migrate Down

DELETE FROM rbac_permissions WHERE name = 'datafiles.edit';
