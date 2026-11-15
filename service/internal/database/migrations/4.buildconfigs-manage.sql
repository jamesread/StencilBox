-- +migrate Up

-- Creating build configs and adding data files writes into the build configs directory, so
-- only superusers have it by default. Give it to other roles from the IAM pages.
INSERT INTO rbac_permissions (name, description) VALUES
('buildconfigs.manage', 'Create build configs and add data files to them');

INSERT OR IGNORE INTO rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM rbac_roles r CROSS JOIN rbac_permissions p
WHERE r.name = 'superuser' AND p.name = 'buildconfigs.manage';

-- +migrate Down

DELETE FROM rbac_permissions WHERE name = 'buildconfigs.manage';
