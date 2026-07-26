package entity

import "time"

// GraphPath 表示图上的一条路径，由节点序列与关系序列组成。
// 用于最短路径、师承链等查询场景（doc/05 §5.5.3）。
type GraphPath struct {
	Nodes         []GraphNode         `json:"nodes"`
	Relationships []GraphRelationship `json:"relationships"`
	Length        int                 `json:"length"`
}

// Subgraph 是以指定节点为中心、限定深度展开的子图，供前端可视化渲染（doc/05 §5.9）。
type Subgraph struct {
	Nodes         []GraphNode         `json:"nodes"`
	Relationships []GraphRelationship `json:"edges"`
}

// LineagePath 承载学派师承传承链查询结果，附带每个节点的代际深度（doc/05 §5.5.2）。
type LineagePath struct {
	Path        GraphPath `json:"path"`
	Generations []int     `json:"generations"`
}

// FigureWithWorks 是朝代代表人物与著作的聚合查询结果（doc/05 §5.5.4）。
type FigureWithWorks struct {
	Person  GraphNode   `json:"person"`
	Works   []GraphNode `json:"works"`
	Schools []GraphNode `json:"schools"`
}

// PrescriptionGraph 是方剂全貌查询结果，包含组成药物与主治疾病（doc/05 §5.5.5）。
type PrescriptionGraph struct {
	Prescription GraphNode   `json:"prescription"`
	Medicines    []GraphNode `json:"medicines"`
	Diseases     []GraphNode `json:"diseases"`
}

// GraphSyncLog 记录 PostgreSQL → Neo4j 的 ETL 同步状态，支撑增量同步与失败重试。
// 字段对齐 migrations/000001_create_graph_sync_log.up.sql。
type GraphSyncLog struct {
	ID          int64     `gorm:"column:id;type:bigint;primaryKey;autoIncrement:false" json:"id"`
	SourceTable string    `gorm:"column:source_table;type:varchar(64);not null;index:idx_graph_sync_log_source" json:"source_table"`
	SourceUID   string    `gorm:"column:source_uid;type:varchar(64);not null;index:idx_graph_sync_log_source" json:"source_uid"`
	Operation   string    `gorm:"column:operation;type:varchar(32);not null" json:"operation"`
	Status      string    `gorm:"column:status;type:varchar(16);not null;default:pending;index:idx_graph_sync_log_status" json:"status"`
	CreatedAt   time.Time `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

// TableName overrides the default GORM table name.
func (GraphSyncLog) TableName() string { return "graph_sync_log" }

// Operation 枚举：同步操作类型。
const (
	OpNodeUpsert       = "node_upsert"
	OpNodeDelete       = "node_delete"
	OpRelationUpsert   = "relation_upsert"
	OpRelationDelete   = "relation_delete"
)

// Status 枚举：同步状态。
const (
	SyncStatusPending = "pending"
	SyncStatusDone    = "done"
	SyncStatusFailed  = "failed"
)
