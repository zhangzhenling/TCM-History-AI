package entity

import (
	"encoding/json"
	"time"
)

// DocumentChunk corresponds to the document_chunks table.
// 与 Milvus 中的向量一一对应，chunk_id 同时作为 Milvus 主键。
type DocumentChunk struct {
	ID               int64           `gorm:"column:id;type:bigint;primaryKey;autoIncrement:false" json:"id"`
	DocumentID       int64           `gorm:"column:document_id;type:bigint;not null;uniqueIndex:uk_document_chunks_doc_index,priority:1;index:idx_document_chunks_doc" json:"document_id"`
	ChunkID          string          `gorm:"column:chunk_id;type:varchar(64);not null;uniqueIndex:uk_document_chunks_chunk_id" json:"chunk_id"`
	ChunkIndex       int             `gorm:"column:chunk_index;type:integer;not null;uniqueIndex:uk_document_chunks_doc_index,priority:2" json:"chunk_index"`
	ClassicCode      string          `gorm:"column:classic_code;type:varchar(32);index:idx_document_chunks_classic" json:"classic_code"`
	Volume           string          `gorm:"column:volume;type:varchar(64)" json:"volume"`
	ClauseNo         int             `gorm:"column:clause_no;type:integer" json:"clause_no"`
	ContentType      string          `gorm:"column:content_type;type:varchar(16)" json:"content_type"`
	Content          string          `gorm:"column:content;type:text;not null" json:"content"`
	TextOriginal     string          `gorm:"column:text_original;type:text" json:"text_original"`
	TextTranslation  string          `gorm:"column:text_translation;type:text" json:"text_translation"`
	TokenCount       int             `gorm:"column:token_count;type:integer" json:"token_count"`
	EmbeddingID      string          `gorm:"column:embedding_id;type:varchar(128);index:idx_document_chunks_embedding_id" json:"embedding_id"`
	EmbeddingModel   string          `gorm:"column:embedding_model;type:varchar(64)" json:"embedding_model"`
	MetadataJSON     json.RawMessage `gorm:"column:metadata_json;type:jsonb;not null;default:'{}'" json:"metadata_json"`
	CreatedAt        time.Time       `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
}

// TableName overrides the default GORM table name.
func (DocumentChunk) TableName() string { return "document_chunks" }

// ContentType 枚举：原文 / 夹注 / 方剂。
const (
	ContentOriginal   = "original"
	ContentAnnotation = "annotation"
	ContentFormula    = "formula"
)
