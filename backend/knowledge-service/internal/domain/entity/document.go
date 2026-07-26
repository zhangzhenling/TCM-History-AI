// Package entity defines the GORM-mapped domain entities for Knowledge Service.
//
// Each entity file maps a database table from the Knowledge Service schema
// (see doc/04-数据库设计.md §4) and exposes typed constants for enumerations.
package entity

import (
	"encoding/json"

	"tcm-history-ai/backend/pkg/gormutil"
)

// Document corresponds to the documents table.
// 一部经典的每个版本一行，记录从 PDF 上传到向量化的完整状态。
type Document struct {
	gormutil.BaseModel
	ClassicCode        string          `gorm:"column:classic_code;type:varchar(32);not null;index:idx_documents_classic" json:"classic_code"`
	Title              string          `gorm:"column:title;type:varchar(255);not null" json:"title"`
	Version            string          `gorm:"column:version;type:varchar(32)" json:"version"`
	Dynasty            string          `gorm:"column:dynasty;type:varchar(16)" json:"dynasty"`
	School             string          `gorm:"column:school;type:varchar(32)" json:"school"`
	Author             string          `gorm:"column:author;type:varchar(64)" json:"author"`
	SourceType         string          `gorm:"column:source_type;type:varchar(32);not null;default:book" json:"source_type"`
	SourceRef          string          `gorm:"column:source_ref;type:varchar(255)" json:"source_ref"`
	FileURL            string          `gorm:"column:file_url;type:varchar(512)" json:"file_url"`
	PDFObjectKey       string          `gorm:"column:pdf_object_key;type:varchar(256)" json:"pdf_object_key"`
	MarkdownObjectKey  string          `gorm:"column:markdown_object_key;type:varchar(256)" json:"markdown_object_key"`
	MimeType           string          `gorm:"column:mime_type;type:varchar(64)" json:"mime_type"`
	ContentHash        string          `gorm:"column:content_hash;type:varchar(64);uniqueIndex:uk_documents_content_hash" json:"content_hash"`
	Status             string          `gorm:"column:status;type:varchar(32);not null;default:pending;index:idx_documents_status" json:"status"`
	ChunkCount         int             `gorm:"column:chunk_count;type:integer;not null;default:0" json:"chunk_count"`
	VolumeCount        int             `gorm:"column:volume_count;type:integer" json:"volume_count"`
	ClauseCount        int             `gorm:"column:clause_count;type:integer" json:"clause_count"`
	MetadataJSON       json.RawMessage `gorm:"column:metadata_json;type:jsonb;not null;default:'{}'" json:"metadata_json"`
}

// TableName overrides the default GORM table name.
func (Document) TableName() string { return "documents" }

// Document status 枚举，对应 documents.status 字段。
const (
	DocumentStatusPending   = "pending"   // 已上传，待处理
	DocumentStatusOCRed     = "ocr_done"  // OCR 完成
	DocumentStatusMarked    = "markdown_done"
	DocumentStatusChunked   = "chunked"   // 分块完成
	DocumentStatusEmbedded  = "embedded"  // 向量化完成
	DocumentStatusOnline    = "online"    // 上线可检索
	DocumentStatusFailed    = "failed"
)

// SourceType 枚举。
const (
	SourceBook   = "book"
	SourceUpload = "upload"
	SourceAPI    = "api"
)
