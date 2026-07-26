-- 000005_create_history_event.up.sql
-- history_event: 历史事件表。外键 dynasty_id ON DELETE RESTRICT。

CREATE TABLE IF NOT EXISTS history_event (
    id            BIGINT       PRIMARY KEY,
    title         VARCHAR(255) NOT NULL,
    dynasty_id    BIGINT,
    occurred_year SMALLINT,
    event_type    VARCHAR(32)  NOT NULL,
    description   TEXT,
    impact        TEXT,
    location      VARCHAR(128),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ,
    CONSTRAINT fk_history_event_dynasty
        FOREIGN KEY (dynasty_id) REFERENCES history_dynasty(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_history_event_dynasty_id
    ON history_event (dynasty_id);
CREATE INDEX IF NOT EXISTS idx_history_event_occurred_year
    ON history_event (occurred_year);
CREATE INDEX IF NOT EXISTS idx_history_event_type
    ON history_event (event_type);
CREATE INDEX IF NOT EXISTS idx_history_event_deleted_at
    ON history_event (deleted_at);

DROP TRIGGER IF EXISTS trg_history_event_updated_at ON history_event;
CREATE TRIGGER trg_history_event_updated_at
    BEFORE UPDATE ON history_event
    FOR EACH ROW
    EXECUTE FUNCTION tcm_set_updated_at();
