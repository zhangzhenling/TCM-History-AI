-- graph_edges: 图关系元数据镜像表（doc/05 §5.3）。
-- 关系方向严格约束在领域语义合理范围内。source_uid / target_uid 引用
-- GraphNode.uid；PropertiesJSON 承载关系属性。
CREATE TABLE IF NOT EXISTS graph_edges (
    id              BIGINT       NOT NULL PRIMARY KEY,
    uid             VARCHAR(64)  NOT NULL,
    type            VARCHAR(32)  NOT NULL,
    source_uid      VARCHAR(64)  NOT NULL,
    target_uid      VARCHAR(64)  NOT NULL,
    properties_json JSONB        NOT NULL DEFAULT '{}',
    synced_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_graph_edges_uid ON graph_edges(uid) WHERE deleted_at IS NULL;
-- 复合索引覆盖 ListBySource / ListByTarget / ListByType 三种查询模式。
CREATE INDEX IF NOT EXISTS idx_graph_edges_lookup ON graph_edges(type, source_uid, target_uid);
CREATE INDEX IF NOT EXISTS idx_graph_edges_source ON graph_edges(source_uid);
CREATE INDEX IF NOT EXISTS idx_graph_edges_target ON graph_edges(target_uid);
CREATE INDEX IF NOT EXISTS idx_graph_edges_deleted_at ON graph_edges(deleted_at);
