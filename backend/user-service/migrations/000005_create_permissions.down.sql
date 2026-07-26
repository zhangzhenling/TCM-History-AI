-- 000005_create_permissions.down.sql

DROP INDEX IF EXISTS idx_permissions_resource_action;
DROP INDEX IF EXISTS uk_permissions_code;
DROP TABLE IF EXISTS permissions;
