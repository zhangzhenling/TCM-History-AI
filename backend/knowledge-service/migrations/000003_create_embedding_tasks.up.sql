-- embedding_tasks: 文献处理任务状态表，支撑前端进度追踪与失败重试。
CREATE TABLE IF NOT EXISTS embedding_tasks (
    id            BIGINT      NOT NULL PRIMARY KEY,
    document_id   BIGINT,
    chunk_id      BIGINT,
    task_type     VARCHAR(32) NOT NULL,
    stage         VARCHAR(32),
    status        VARCHAR(32) NOT NULL DEFAULT 'queued',
    progress      INTEGER     NOT NULL DEFAULT 0,
    model         VARCHAR(64),
    chunk_count   INTEGER     NOT NULL DEFAULT 0,
    vector_count  INTEGER     NOT NULL DEFAULT 0,
    error_message TEXT,
    retry_count   INTEGER     NOT NULL DEFAULT 0,
    started_at    TIMESTAMPTZ,
    finished_at   TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_embedding_tasks_status ON embedding_tasks(status);
CREATE INDEX IF NOT EXISTS idx_embedding_tasks_document_id ON embedding_tasks(document_id);
CREATE INDEX IF NOT EXISTS idx_embedding_tasks_status_created ON embedding_tasks(status, created_at);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_embedding_tasks_document'
    ) THEN
        ALTER TABLE embedding_tasks
            ADD CONSTRAINT fk_embedding_tasks_document
            FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_embedding_tasks_chunk'
    ) THEN
        ALTER TABLE embedding_tasks
            ADD CONSTRAINT fk_embedding_tasks_chunk
            FOREIGN KEY (chunk_id) REFERENCES document_chunks(id) ON DELETE CASCADE;
    END IF;
END $$;
