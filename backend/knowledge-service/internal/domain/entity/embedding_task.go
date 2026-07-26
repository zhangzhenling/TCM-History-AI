package entity

import (
	"time"

	"tcm-history-ai/backend/pkg/gormutil"
)

// EmbeddingTask corresponds to the embedding_tasks table.
// 记录文献处理任务的状态与进度，支撑前端进度追踪与失败重试。
type EmbeddingTask struct {
	gormutil.BaseModel
	DocumentID   int64      `gorm:"column:document_id;type:bigint;index:idx_embedding_tasks_document_id" json:"document_id"`
	ChunkID      int64      `gorm:"column:chunk_id;type:bigint" json:"chunk_id"`
	TaskType     string     `gorm:"column:task_type;type:varchar(32);not null" json:"task_type"`
	Stage        string     `gorm:"column:stage;type:varchar(32)" json:"stage"`
	Status       string     `gorm:"column:status;type:varchar(32);not null;default:queued;index:idx_embedding_tasks_status" json:"status"`
	Progress     int        `gorm:"column:progress;type:integer;not null;default:0" json:"progress"`
	Model        string     `gorm:"column:model;type:varchar(64)" json:"model"`
	ChunkCount   int        `gorm:"column:chunk_count;type:integer;not null;default:0" json:"chunk_count"`
	VectorCount  int        `gorm:"column:vector_count;type:integer;not null;default:0" json:"vector_count"`
	ErrorMessage string     `gorm:"column:error_message;type:text" json:"error_message"`
	RetryCount   int        `gorm:"column:retry_count;type:integer;not null;default:0" json:"retry_count"`
	StartedAt    *time.Time `gorm:"column:started_at;type:timestamptz" json:"started_at,omitempty"`
	FinishedAt   *time.Time `gorm:"column:finished_at;type:timestamptz" json:"finished_at,omitempty"`
}

// TableName overrides the default GORM table name.
func (EmbeddingTask) TableName() string { return "embedding_tasks" }

// TaskType 枚举。
const (
	TaskTypeDocument = "document"
	TaskTypeChunk    = "chunk"
	TaskTypeRetry    = "retry"
)

// Stage 枚举：处理流水线的各阶段。
const (
	StageUpload   = "upload"
	StageOCR      = "ocr"
	StageMarkdown = "markdown"
	StageChunk    = "chunk"
	StageEmbed    = "embed"
	StageMilvus   = "milvus"
)

// Task status 枚举。
const (
	TaskStatusQueued  = "queued"
	TaskStatusRunning = "running"
	TaskStatusDone    = "done"
	TaskStatusFailed  = "failed"
)
