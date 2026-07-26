// Package entity defines the domain entities for Graph Service.
//
// Graph Service 采用 Neo4j 作为知识图谱存储，节点与关系均为通用结构，
// 通过 Label / Type 区分 8 类节点与 9 类关系（详见 doc/05-知识图谱设计.md）。
// 通用结构便于承载异构节点属性，避免为每类节点定义独立实体。
package entity

// GraphNode 是图谱节点的通用实体。
// 业务主键统一使用 uid（UUID v7 风格字符串），与 PostgreSQL 主键一致，
// 保证跨数据源可追溯。Properties 承载各类节点的异构属性。
type GraphNode struct {
	UID        string         `json:"uid"`
	Label      string         `json:"label"`
	Properties map[string]any `json:"properties"`
}

// 节点 Label 枚举，对应 doc/05 §5.2 的 8 类节点。
const (
	LabelPerson           = "Person"           // 人物
	LabelClassic          = "Classic"          // 经典（著作/理论）
	LabelSchool           = "School"           // 学派
	LabelPrescription     = "Prescription"     // 方剂
	LabelMedicine         = "Medicine"         // 药物
	LabelDisease          = "Disease"          // 疾病
	LabelDynasty          = "Dynasty"          // 朝代
	LabelHistoricalEvent  = "HistoricalEvent"  // 历史事件
)

// NodeLabels 罗列全部节点 Label，供约束/索引建立与校验复用。
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
