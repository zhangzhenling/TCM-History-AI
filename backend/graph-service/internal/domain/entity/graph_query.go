package entity

import "encoding/json"

// GraphNodeView 是图查询结果中的节点视图，与 GraphNode 实体解耦：
// 查询场景仅需 uid/label/name/properties，不需要 synced_at 等同步元数据。
type GraphNodeView struct {
	UID        string          `json:"uid"`
	Label      string          `json:"label"`
	Name       string          `json:"name"`
	Properties json.RawMessage `json:"properties"`
}

// GraphEdgeView 是图查询结果中的边视图，与 GraphEdge 实体解耦。
type GraphEdgeView struct {
	UID        string          `json:"uid"`
	Type       string          `json:"type"`
	SourceUID  string          `json:"source_uid"`
	TargetUID  string          `json:"target_uid"`
	Properties json.RawMessage `json:"properties"`
}

// GraphPath 表示图上的一条路径，由节点序列与边序列组成。
// 用于最短路径、师承链等查询场景（doc/05 §5.5.3）。
type GraphPath struct {
	Nodes []GraphNodeView `json:"nodes"`
	Edges []GraphEdgeView `json:"edges"`
	Hops  int             `json:"hops"`
}

// Subgraph 是以指定节点为中心、限定深度展开的子图，供前端可视化渲染（doc/05 §5.9）。
type Subgraph struct {
	Nodes []GraphNodeView `json:"nodes"`
	Edges []GraphEdgeView `json:"edges"`
}

// LineagePath 承载学派师承传承链查询结果，附带每个节点的代际深度（doc/05 §5.5.2）。
type LineagePath struct {
	Path        GraphPath `json:"path"`
	Generations []int     `json:"generations"`
}

// FigureWithWorks 是朝代代表人物与著作的聚合查询结果（doc/05 §5.5.4）。
type FigureWithWorks struct {
	Person  GraphNodeView   `json:"person"`
	Works   []GraphNodeView `json:"works"`
	Schools []GraphNodeView `json:"schools"`
}

// PrescriptionGraph 是方剂全貌查询结果，包含组成药物与主治疾病（doc/05 §5.5.5）。
type PrescriptionGraph struct {
	Prescription GraphNodeView   `json:"prescription"`
	Medicines    []GraphNodeView `json:"medicines"`
	Diseases     []GraphNodeView `json:"diseases"`
}
