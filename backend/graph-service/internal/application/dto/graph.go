package dto

import "encoding/json"

// NodeRequest is the create/update payload for a graph node.
// UID 是业务主键；Label 必须为 8 类已知节点 Label 之一；Name 为节点显示名。
type NodeRequest struct {
	UID           string          `json:"uid,required"`
	Label         string          `json:"label,required"`
	Name          string          `json:"name,required"`
	PropertiesJSON json.RawMessage `json:"properties_json,optional"`
}

// NodeResponse is the wire representation of a graph node.
type NodeResponse struct {
	ID             int64           `json:"id"`
	UID            string          `json:"uid"`
	Label          string          `json:"label"`
	Name           string          `json:"name"`
	PropertiesJSON json.RawMessage `json:"properties_json"`
	SyncedAt       string          `json:"synced_at"`
	CreatedAt      string          `json:"created_at"`
	UpdatedAt      string          `json:"updated_at"`
}

// EdgeRequest is the create/update payload for a graph edge.
// Type 必须为 9 类已知关系 Type 之一；SourceUID / TargetUID 引用已存在的节点 uid。
type EdgeRequest struct {
	UID            string          `json:"uid,required"`
	Type           string          `json:"type,required"`
	SourceUID      string          `json:"source_uid,required"`
	TargetUID      string          `json:"target_uid,required"`
	PropertiesJSON json.RawMessage `json:"properties_json,optional"`
}

// EdgeResponse is the wire representation of a graph edge.
type EdgeResponse struct {
	ID             int64           `json:"id"`
	UID            string          `json:"uid"`
	Type           string          `json:"type"`
	SourceUID      string          `json:"source_uid"`
	TargetUID      string          `json:"target_uid"`
	PropertiesJSON json.RawMessage `json:"properties_json"`
	SyncedAt       string          `json:"synced_at"`
	CreatedAt      string          `json:"created_at"`
	UpdatedAt      string          `json:"updated_at"`
}

// NodeView is the lightweight node view returned by graph queries (no sync metadata).
type NodeView struct {
	UID        string          `json:"uid"`
	Label      string          `json:"label"`
	Name       string          `json:"name"`
	Properties json.RawMessage `json:"properties"`
}

// EdgeView is the lightweight edge view returned by graph queries.
type EdgeView struct {
	UID        string          `json:"uid"`
	Type       string          `json:"type"`
	SourceUID  string          `json:"source_uid"`
	TargetUID  string          `json:"target_uid"`
	Properties json.RawMessage `json:"properties"`
}

// GraphPath is the wire representation of a graph path query result (doc/05 §5.5.3).
type GraphPath struct {
	Nodes []NodeView `json:"nodes"`
	Edges []EdgeView `json:"edges"`
	Hops  int        `json:"hops"`
}

// Subgraph is the wire representation of a subgraph query result (doc/05 §5.9).
type Subgraph struct {
	Nodes []NodeView `json:"nodes"`
	Edges []EdgeView `json:"edges"`
}

// LineageResponse is the wire representation of a school lineage query (doc/05 §5.5.2).
type LineageResponse struct {
	Path        GraphPath `json:"path"`
	Generations []int     `json:"generations"`
}

// FigureWithWorksResponse is the wire representation of a dynasty figures query (doc/05 §5.5.4).
type FigureWithWorksResponse struct {
	Person  NodeView   `json:"person"`
	Works   []NodeView `json:"works"`
	Schools []NodeView `json:"schools"`
}

// PrescriptionDetailResponse is the wire representation of a prescription detail query (doc/05 §5.5.5).
type PrescriptionDetailResponse struct {
	Prescription NodeView   `json:"prescription"`
	Medicines    []NodeView `json:"medicines"`
	Diseases     []NodeView `json:"diseases"`
}

// PersonWorksRequest captures the body of POST /query/person-works.
type PersonWorksRequest struct {
	PersonUID string `json:"person_uid,required"`
}

// SchoolLineageRequest captures the body of POST /query/school-lineage.
type SchoolLineageRequest struct {
	SchoolName string `json:"school_name,required"`
	MaxDepth   int    `json:"max_depth,optional"`
}

// ShortestPathRequest captures the body of POST /query/shortest-path.
type ShortestPathRequest struct {
	StartUID string `json:"start_uid,required"`
	EndUID   string `json:"end_uid,required"`
	MaxHops  int    `json:"max_hops,optional"`
}

// DynastyFiguresRequest captures the body of POST /query/dynasty-figures.
type DynastyFiguresRequest struct {
	DynastyName string `json:"dynasty_name,required"`
}

// PrescriptionDetailRequest captures the body of POST /query/prescription-detail.
type PrescriptionDetailRequest struct {
	PrescriptionUID string `json:"prescription_uid,required"`
}

// SubgraphRequest captures the body of POST /query/subgraph.
type SubgraphRequest struct {
	CenterUID string `json:"center_uid,required"`
	Depth     int    `json:"depth,optional"`
	Limit     int    `json:"limit,optional"`
}

// SearchParams captures the query string of GET /search.
type SearchParams struct {
	Keyword string `json:"keyword"`
	Label   string `json:"label"`
	Limit   int    `json:"limit"`
}

// SearchResponse is the wire representation of a node search result.
type SearchResponse struct {
	Keyword string     `json:"keyword"`
	Label   string     `json:"label"`
	Total   int        `json:"total"`
	Items   []NodeView `json:"items"`
}

// SyncResponse is returned by POST /sync to report the outcome of the
// manually triggered ETL re-run (doc/05 §5.6).
type SyncResponse struct {
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
	Pending   int `json:"pending"`
}
