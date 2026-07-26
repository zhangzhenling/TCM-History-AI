-- graph_sync_log: 记录 PostgreSQL → Neo4j 的 ETL 同步状态（doc/05 §5.6）。
-- 支撑增量同步与失败重试，与 entity.GraphSyncLog 字段对齐。
CREATE TABLE IF NOT EXISTS graph_sync_log (
    id            BIGINT       NOT NULL PRIMARY KEY,
    source_table  VARCHAR(64)  NOT NULL,
    source_uid    VARCHAR(64)  NOT NULL,
    operation     VARCHAR(32)  NOT NULL,
    status        VARCHAR(16)  NOT NULL DEFAULT 'pending',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_graph_sync_log_source ON graph_sync_log(source_table, source_uid);
CREATE INDEX IF NOT EXISTS idx_graph_sync_log_status ON graph_sync_log(status);
