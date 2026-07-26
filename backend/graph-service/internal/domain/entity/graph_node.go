// Package entity defines the GORM-mapped domain entities for Graph Service.
//
// Graph Service 采用 Neo4j 作为查询引擎，PostgreSQL 仅承载图谱同步元数据
// （graph_nodes / graph_edges / graph_sync_logs 三张表），与 doc/05 §5.6
// 的 ETL 流程对齐。本包定义三类实体及其常量。
package entity

import (
	"encoding/json"
	"time"

	"tcm-history-ai/backend/pkg/gormutil"
)

// 节点 Label 枚举，对应 doc/05 §5.2 的 8 类节点。
const (
	LabelPerson          = "Person"          // 人物
	LabelClassic         = "Classic"         // 经典（著作/理论）
	LabelSchool          = "School"          // 学派
	LabelPrescription    = "Prescription"    // 方剂
	LabelMedicine        = "Medicine"        // 药物
	LabelDisease         = "Disease"         // 疾病
	LabelDynasty         = "Dynasty"         // 朝代
	LabelHistoricalEvent = "HistoricalEvent" // 历史事件
)

// NodeLabels 罗列全部节点 Label，供校验与索引建立复用。
var NodeLabels = []string{
	LabelPerson,
	LabelClassic,
	LabelSchool,
	LabelPrescription,
	LabelMedicine,
	LabelDisease,
	LabelDynasty,
	LabelHistoricalEvent,
}

// IsValidLabel reports whether label is one of the 8 known node labels.
func IsValidLabel(label string) bool {
	for _, l := range NodeLabels {
		if l == label {
			return true
		}
	}
	return false
}

// GraphNode 对应 graph_nodes 表，记录图节点的元数据镜像。
// 业务主键为 uid（UUID v7 风格字符串），与 Neo4j 节点 uid 一致，
// 保证跨数据源可追溯。PropertiesJSON 承载各类节点的异构属性。
type GraphNode struct {
	gormutil.BaseModel
	UID           string          `gorm:"column:uid;type:varchar(64);not null;uniqueIndex:uk_graph_nodes_uid" json:"uid"`
	Label         string          `gorm:"column:label;type:varchar(32);not null;index:idx_graph_nodes_label" json:"label"`
	Name          string          `gorm:"column:name;type:varchar(255);not null;index:idx_graph_nodes_name" json:"name"`
	PropertiesJSON json.RawMessage `gorm:"column:properties_json;type:jsonb;not null;default:'{}'" json:"properties_json"`
	SyncedAt      time.Time       `gorm:"column:synced_at;type:timestamptz;not null;default:now()" json:"synced_at"`
}

// TableName overrides the default GORM table name.
func (GraphNode) TableName() string { return "graph_nodes" }
