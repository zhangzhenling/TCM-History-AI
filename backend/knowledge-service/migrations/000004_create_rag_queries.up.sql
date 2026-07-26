-- rag_queries: RAG 检索日志表，记录每次查询的召回结果、耗时与反馈。
CREATE TABLE IF NOT EXISTS rag_queries (
    id                  BIGINT      NOT NULL PRIMARY KEY,
    session_id          VARCHAR(64),
    user_id             BIGINT,
    query_text          TEXT        NOT NULL,
    query_embedding     JSONB,
    top_k               INTEGER     NOT NULL DEFAULT 5,
    retrieved_chunk_ids JSONB       NOT NULL DEFAULT '[]',
    latency_ms          INTEGER,
    feedback            VARCHAR(16),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_rag_queries_user_id ON rag_queries(user_id);
CREATE INDEX IF NOT EXISTS idx_rag_queries_created_at ON rag_queries(created_at);
CREATE INDEX IF NOT EXISTS idx_rag_queries_session_id ON rag_queries(session_id);
