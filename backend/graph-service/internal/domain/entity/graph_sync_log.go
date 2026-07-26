package entity

import (
	"tcm-history-ai/backend/pkg/gormutil"
)

// 同步来源 source_type 枚举。
const (
	SourceHistory   = "history"   // 来自 history-service
	SourceKnowledge = "knowledge" // 来自 knowledge-service
)

// 同步动作 action 枚举。
const (
	ActionUpsert = "upsert"
	ActionDelete = "delete"
)

// 同步状态 status 枚举。
const (
	SyncStatusPending = "pending"
	SyncStatusDone    = "done"
	SyncStatusFailed  = "failed"
)

// GraphSyncLog 对应 graph_sync_logs 表，记录 PostgreSQL 主数据 → Neo4j 的
// ETL 同步状态，支撑增量同步与失败重试（doc/05 §5.6）。
// source_type 标识来源服务（history / knowledge），source_id 为来源实体的业务
// 主键，entity_type 为映射到的图节点/关系类型，action 标识 upsert/delete，
// status 标识处理状态，error_msg 在失败时记录原因。
type GraphSyncLog struct {
	gormutil.BaseModel
	SourceType string `gorm:"column:source_type;type:varchar(32);not null;index:idx_graph_sync_logs_source" json:"source_type"`
	SourceID   string `gorm:"column:source_id;type:varchar(64);not null;index:idx_graph_sync_logs_source" json:"source_id"`
	EntityType string `gorm:"column:entity_type;type:varchar(64);not null" json:"entity_type"`
	Action     string `gorm:"column:action;type:varchar(16);not null;default:upsert" json:"action"`
	Status     string `gorm:"column:status;type:varchar(16);not null;default:pending;index:idx_graph_sync_logs_status" json:"status"`
	ErrorMsg   string `gorm:"column:error_msg;type:text" json:"error_msg"`
}

// TableName overrides the default GORM table name.
func (GraphSyncLog) TableName() string { return "graph_sync_logs" }
