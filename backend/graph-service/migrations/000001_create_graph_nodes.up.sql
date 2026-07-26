-- graph_nodes: 图节点元数据镜像表，与 Neo4j 节点保持最终一致（doc/05 §5.6）。
-- 业务主键 uid（UUID v7 风格字符串），与 Neo4j 节点 uid 一致，保证跨数据源
-- 可追溯。PropertiesJSON 承载各类节点的异构属性。
CREATE TABLE IF NOT EXISTS graph_nodes (
    id              BIGINT       NOT NULL PRIMARY KEY,
    uid             VARCHAR(64)  NOT NULL,
    label           VARCHAR(32)  NOT NULL,
    name            VARCHAR(255) NOT NULL,
    properties_json JSONB        NOT NULL DEFAULT '{}',
    synced_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_graph_nodes_uid ON graph_nodes(uid) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_graph_nodes_label ON graph_nodes(label);
CREATE INDEX IF NOT EXISTS idx_graph_nodes_name ON graph_nodes(name);
CREATE INDEX IF NOT EXISTS idx_graph_nodes_deleted_at ON graph_nodes(deleted_at);
