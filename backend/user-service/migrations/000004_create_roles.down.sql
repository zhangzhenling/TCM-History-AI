-- 000004_create_roles.down.sql

DROP TRIGGER IF EXISTS trg_roles_updated_at ON roles;
DROP INDEX IF EXISTS uk_roles_code;
DROP TABLE IF EXISTS roles;
