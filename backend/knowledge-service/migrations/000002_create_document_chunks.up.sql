-- document_chunks: 文献切片表，与 Milvus 中的向量一一对应。
CREATE TABLE IF NOT EXISTS document_chunks (
    id               BIGINT      NOT NULL PRIMARY KEY,
    document_id      BIGINT      NOT NULL,
    chunk_id         VARCHAR(64) NOT NULL,
    chunk_index      INTEGER     NOT NULL,
    classic_code     VARCHAR(32),
    volume           VARCHAR(64),
    clause_no        INTEGER,
    content_type     VARCHAR(16),
    content          TEXT        NOT NULL,
    text_original    TEXT,
    text_translation TEXT,
    token_count      INTEGER,
    embedding_id     VARCHAR(128),
    embedding_model  VARCHAR(64),
    metadata_json    JSONB       NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_document_chunks_doc_index ON document_chunks(document_id, chunk_index);
CREATE UNIQUE INDEX IF NOT EXISTS uk_document_chunks_chunk_id ON document_chunks(chunk_id);
CREATE INDEX IF NOT EXISTS idx_document_chunks_embedding_id ON document_chunks(embedding_id);
CREATE INDEX IF NOT EXISTS idx_document_chunks_classic ON document_chunks(classic_code);
CREATE INDEX IF NOT EXISTS idx_document_chunks_doc ON document_chunks(document_id);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_document_chunks_document'
    ) THEN
        ALTER TABLE document_chunks
            ADD CONSTRAINT fk_document_chunks_document
            FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE;
    END IF;
END $$;
