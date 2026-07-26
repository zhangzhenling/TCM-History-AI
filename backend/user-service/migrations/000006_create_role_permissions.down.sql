-- 000006_create_role_permissions.down.sql

DROP INDEX IF EXISTS uk_role_permissions_role_permission;
DROP TABLE IF EXISTS role_permissions;
