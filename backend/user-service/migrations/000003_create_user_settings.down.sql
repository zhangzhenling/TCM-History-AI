-- 000003_create_user_settings.down.sql

DROP TRIGGER IF EXISTS trg_user_settings_updated_at ON user_settings;
DROP INDEX IF EXISTS uk_user_settings_user_id;
DROP TABLE IF EXISTS user_settings;
