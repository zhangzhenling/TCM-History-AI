-- graph_sync_logs: PostgreSQL 主数据 → Neo4j 的 ETL 同步状态表（doc/05 §5.6）。
-- source_type 标识来源服务（history / knowledge），source_id 为来源实体的业务
-- 主键，entity_type 为映射到的图节点/关系类型，action 标识 upsert/delete，
-- status 标识处理状态，error_msg 在失败时记录原因。支撑增量同步与失败重试。
CREATE TABLE IF NOT EXISTS graph_sync_logs (
    id           BIGINT      NOT NULL PRIMARY KEY,
    source_type  VARCHAR(32) NOT NULL,
    source_id    VARCHAR(64) NOT NULL,
    entity_type  VARCHAR(64) NOT NULL,
    action       VARCHAR(16) NOT NULL DEFAULT 'upsert',
    status       VARCHAR(16) NOT NULL DEFAULT 'pending',
    error_msg    TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

-- 同源同实体去重检索；用于幂等消费与失败重试定位。
CREATE INDEX IF NOT EXISTS idx_graph_sync_logs_source ON graph_sync_logs(source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_graph_sync_logs_status ON graph_sync_logs(status);
CREATE INDEX IF NOT EXISTS idx_graph_sync_logs_status_created ON graph_sync_logs(status, created_at);
CREATE INDEX IF NOT EXISTS idx_graph_sync_logs_deleted_at ON graph_sync_logs(deleted_at);
