package dto

import "encoding/json"

// DocumentRequest is the create/update payload for documents.
type DocumentRequest struct {
	ClassicCode       string          `json:"classic_code,required"`
	Title             string          `json:"title,required"`
	Version           string          `json:"version,optional"`
	Dynasty           string          `json:"dynasty,optional"`
	School            string          `json:"school,optional"`
	Author            string          `json:"author,optional"`
	SourceType        string          `json:"source_type,optional"`
	SourceRef         string          `json:"source_ref,optional"`
	FileURL           string          `json:"file_url,optional"`
	PDFObjectKey      string          `json:"pdf_object_key,optional"`
	MarkdownObjectKey string          `json:"markdown_object_key,optional"`
	MimeType          string          `json:"mime_type,optional"`
	ContentHash       string          `json:"content_hash,optional"`
	VolumeCount       int             `json:"volume_count,optional"`
	ClauseCount       int             `json:"clause_count,optional"`
	MetadataJSON      json.RawMessage `json:"metadata_json,optional"`
}

// DocumentResponse is the wire representation of a document.
type DocumentResponse struct {
	ID                int64           `json:"id"`
	ClassicCode       string          `json:"classic_code"`
	Title             string          `json:"title"`
	Version           string          `json:"version"`
	Dynasty           string          `json:"dynasty"`
	School            string          `json:"school"`
	Author            string          `json:"author"`
	SourceType        string          `json:"source_type"`
	SourceRef         string          `json:"source_ref"`
	FileURL           string          `json:"file_url"`
	PDFObjectKey      string          `json:"pdf_object_key"`
	MarkdownObjectKey string          `json:"markdown_object_key"`
	MimeType          string          `json:"mime_type"`
	ContentHash       string          `json:"content_hash"`
	Status            string          `json:"status"`
	ChunkCount        int             `json:"chunk_count"`
	VolumeCount       int             `json:"volume_count"`
	ClauseCount       int             `json:"clause_count"`
	MetadataJSON      json.RawMessage `json:"metadata_json"`
	CreatedAt         string          `json:"created_at"`
	UpdatedAt         string          `json:"updated_at"`
}

// ChunkResponse is the wire representation of a document chunk.
type ChunkResponse struct {
	ID              int64  `json:"id"`
	DocumentID      int64  `json:"document_id"`
	ChunkID         string `json:"chunk_id"`
	ChunkIndex      int    `json:"chunk_index"`
	ClassicCode     string `json:"classic_code"`
	Volume          string `json:"volume"`
	ClauseNo        int    `json:"clause_no"`
	ContentType     string `json:"content_type"`
	Content         string `json:"content"`
	TextOriginal    string `json:"text_original"`
	TextTranslation string `json:"text_translation"`
	TokenCount      int    `json:"token_count"`
	EmbeddingID     string `json:"embedding_id"`
	EmbeddingModel  string `json:"embedding_model"`
}

// TaskResponse is the wire representation of an embedding task.
type TaskResponse struct {
	ID           int64  `json:"id"`
	DocumentID   int64  `json:"document_id"`
	ChunkID      int64  `json:"chunk_id"`
	TaskType     string `json:"task_type"`
	Stage        string `json:"stage"`
	Status       string `json:"status"`
	Progress     int    `json:"progress"`
	Model        string `json:"model"`
	ChunkCount   int    `json:"chunk_count"`
	VectorCount  int    `json:"vector_count"`
	ErrorMessage string `json:"error_message,omitempty"`
	RetryCount   int    `json:"retry_count"`
	StartedAt    string `json:"started_at,omitempty"`
	FinishedAt   string `json:"finished_at,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// RetrieveRequest is the payload for RAG retrieval.
type RetrieveRequest struct {
	Query        string   `json:"query,required"`
	TopK         int      `json:"topk,optional"`
	ClassicCodes []string `json:"classic_codes,optional"`
	Dynasties    []string `json:"dynasties,optional"`
	Schools      []string `json:"schools,optional"`
	ContentTypes []string `json:"content_types,optional"`
	SessionID    string   `json:"session_id,optional"`
}

// RetrievedChunk is a single hit in retrieval results.
type RetrievedChunk struct {
	ChunkID         string  `json:"chunk_id"`
	DocumentID      int64   `json:"document_id"`
	ClassicCode     string  `json:"classic_code"`
	Volume          string  `json:"volume"`
	ClauseNo        int     `json:"clause_no"`
	ContentType     string  `json:"content_type"`
	Content         string  `json:"content"`
	TextOriginal    string  `json:"text_original"`
	TextTranslation string  `json:"text_translation"`
	Score           float32 `json:"score"`
	Source          string  `json:"source"` // vector | bm25 | rerank
}

// RetrieveResponse is the result of a RAG retrieval.
type RetrieveResponse struct {
	Query       string           `json:"query"`
	TopK        int              `json:"topk"`
	LatencyMs   int              `json:"latency_ms"`
	Total       int              `json:"total"`
	Chunks      []RetrievedChunk `json:"chunks"`
	QueryLogID  int64            `json:"query_log_id,omitempty"`
}

// FeedbackRequest updates a rag_query's feedback field.
type FeedbackRequest struct {
	Feedback string `json:"feedback,required"` // good | bad
}
