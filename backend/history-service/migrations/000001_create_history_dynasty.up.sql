-- 000001_create_history_dynasty.up.sql
-- history_dynasty: 朝代表。无外键，是其他实体表的依赖根。

CREATE TABLE IF NOT EXISTS history_dynasty (
    id          BIGINT       PRIMARY KEY,
    name        VARCHAR(64)  NOT NULL,
    start_year  SMALLINT,
    end_year    SMALLINT,
    sort_order  INTEGER      NOT NULL DEFAULT 0,
    description TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_history_dynasty_name
    ON history_dynasty (name)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_history_dynasty_sort_order
    ON history_dynasty (sort_order);

CREATE INDEX IF NOT EXISTS idx_history_dynasty_deleted_at
    ON history_dynasty (deleted_at);

-- updated_at 自动维护触发器函数。该函数在最末 down 文件中删除。
CREATE OR REPLACE FUNCTION tcm_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_history_dynasty_updated_at ON history_dynasty;
CREATE TRIGGER trg_history_dynasty_updated_at
    BEFORE UPDATE ON history_dynasty
    FOR EACH ROW
    EXECUTE FUNCTION tcm_set_updated_at();
