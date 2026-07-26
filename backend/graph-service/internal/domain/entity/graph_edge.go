package entity

import (
	"encoding/json"
	"time"

	"tcm-history-ai/backend/pkg/gormutil"
)

// 关系 Type 枚举，对应 doc/05 §5.3 的 9 类关系。
const (
	RelAuthored   = "AUTHORED"   // Person → Classic 著作
	RelDiscipled  = "DISCIPLED"  // Person → Person 师承（弟子→师父）
	RelInfluenced = "INFLUENCED" // 任意 → 任意 影响
	RelBelongsTo  = "BELONGS_TO" // Person/Classic/Prescription → School 属于
	RelOccurredIn = "OCCURRED_IN" // Person/Classic/HistoricalEvent → Dynasty 发生于
	RelCited      = "CITED"       // Classic → Classic 引用
	RelProposed   = "PROPOSED"    // Person/Classic → Classic(理论) 提出
	RelOpposed    = "OPPOSED"     // Person/Classic → Person/Classic/School 反对
	RelInherited  = "INHERITED"   // Person/School → Person/Classic/School 继承
)

// EdgeTypes 罗列全部关系 Type，供校验与约束建立复用。
var EdgeTypes = []string{
	RelAuthored,
	RelDiscipled,
	RelInfluenced,
	RelBelongsTo,
	RelOccurredIn,
	RelCited,
	RelProposed,
	RelOpposed,
	RelInherited,
}

// IsValidEdgeType reports whether t is one of the 9 known edge types.
func IsValidEdgeType(t string) bool {
	for _, r := range EdgeTypes {
		if r == t {
			return true
		}
	}
	return false
}

// GraphEdge 对应 graph_edges 表，记录图关系的元数据镜像。
// 关系方向严格约束在领域语义合理范围内（详见 doc/05 §5.3）。
// source_uid / target_uid 引用 GraphNode.UID；PropertiesJSON 承载关系属性。
type GraphEdge struct {
	gormutil.BaseModel
	UID            string          `gorm:"column:uid;type:varchar(64);not null;uniqueIndex:uk_graph_edges_uid" json:"uid"`
	Type           string          `gorm:"column:type;type:varchar(32);not null;index:idx_graph_edges_lookup" json:"type"`
	SourceUID      string          `gorm:"column:source_uid;type:varchar(64);not null;index:idx_graph_edges_lookup" json:"source_uid"`
	TargetUID      string          `gorm:"column:target_uid;type:varchar(64);not null;index:idx_graph_edges_lookup" json:"target_uid"`
	PropertiesJSON json.RawMessage `gorm:"column:properties_json;type:jsonb;not null;default:'{}'" json:"properties_json"`
	SyncedAt       time.Time       `gorm:"column:synced_at;type:timestamptz;not null;default:now()" json:"synced_at"`
}

// TableName overrides the default GORM table name.
func (GraphEdge) TableName() string { return "graph_edges" }
