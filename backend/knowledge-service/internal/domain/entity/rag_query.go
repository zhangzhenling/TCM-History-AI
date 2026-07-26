package entity

import (
	"encoding/json"
	"time"
)

// RagQuery corresponds to the rag_queries table.
// 记录每次 RAG 查询的召回结果、耗时与用户反馈，支撑检索质量评估闭环。
type RagQuery struct {
	ID                int64           `gorm:"column:id;type:bigint;primaryKey;autoIncrement:false" json:"id"`
	SessionID         string          `gorm:"column:session_id;type:varchar(64);index:idx_rag_queries_session_id" json:"session_id"`
	UserID            int64           `gorm:"column:user_id;type:bigint;index:idx_rag_queries_user_id" json:"user_id"`
	QueryText         string          `gorm:"column:query_text;type:text;not null" json:"query_text"`
	QueryEmbedding    json.RawMessage `gorm:"column:query_embedding;type:jsonb" json:"query_embedding,omitempty"`
	TopK              int             `gorm:"column:top_k;type:integer;not null;default:5" json:"top_k"`
	RetrievedChunkIDs json.RawMessage `gorm:"column:retrieved_chunk_ids;type:jsonb;not null;default:'[]'" json:"retrieved_chunk_ids"`
	LatencyMs         int             `gorm:"column:latency_ms;type:integer" json:"latency_ms"`
	Feedback          string          `gorm:"column:feedback;type:varchar(16)" json:"feedback"`
	CreatedAt         time.Time       `gorm:"column:created_at;type:timestamptz;not null;default:now();index:idx_rag_queries_created_at" json:"created_at"`
}

// TableName overrides the default GORM table name.
func (RagQuery) TableName() string { return "rag_queries" }

// Feedback 枚举。
const (
	FeedbackGood = "good"
	FeedbackBad  = "bad"
)
