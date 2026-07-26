-- documents: 文献元信息表，每部经典的每个版本一行。
CREATE TABLE IF NOT EXISTS documents (
    id              BIGINT       NOT NULL PRIMARY KEY,
    classic_code    VARCHAR(32)  NOT NULL,
    title           VARCHAR(255) NOT NULL,
    version         VARCHAR(32),
    dynasty         VARCHAR(16),
    school          VARCHAR(32),
    author          VARCHAR(64),
    source_type     VARCHAR(32)  NOT NULL DEFAULT 'book',
    source_ref      VARCHAR(255),
    file_url        VARCHAR(512),
    pdf_object_key  VARCHAR(256),
    markdown_object_key VARCHAR(256),
    mime_type       VARCHAR(64),
    content_hash    VARCHAR(64),
    status          VARCHAR(32)  NOT NULL DEFAULT 'pending',
    chunk_count     INTEGER      NOT NULL DEFAULT 0,
    volume_count    INTEGER,
    clause_count    INTEGER,
    metadata_json   JSONB        NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_documents_status ON documents(status);
CREATE INDEX IF NOT EXISTS idx_documents_source ON documents(source_type, source_ref);
CREATE INDEX IF NOT EXISTS idx_documents_classic ON documents(classic_code);
CREATE UNIQUE INDEX IF NOT EXISTS uk_documents_content_hash ON documents(content_hash) WHERE content_hash IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_documents_deleted_at ON documents(deleted_at);
