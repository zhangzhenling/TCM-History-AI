package dto

import "encoding/json"

// NodeRequest is the create/update payload for a graph node (MERGE semantics).
// UID is the business key; Label must be one of the 8 known node labels.
type NodeRequest struct {
	UID        string          `json:"uid,required"`
	Label      string          `json:"label,required"`
	Properties json.RawMessage `json:"properties,optional"`
}

// NodeResponse is the wire representation of a graph node.
type NodeResponse struct {
	UID        string          `json:"uid"`
	Label      string          `json:"label"`
	Properties json.RawMessage `json:"properties"`
}

// RelationshipRequest is the create payload for a graph relationship.
type RelationshipRequest struct {
	UID        string          `json:"uid,required"`
	Type       string          `json:"type,required"`
	FromUID    string          `json:"from_uid,required"`
	ToUID      string          `json:"to_uid,required"`
	Properties json.RawMessage `json:"properties,optional"`
}

// RelationshipResponse is the wire representation of a graph relationship.
type RelationshipResponse struct {
	UID        string          `json:"uid"`
	Type       string          `json:"type"`
	FromUID    string          `json:"from_uid"`
	ToUID      string          `json:"to_uid"`
	Properties json.RawMessage `json:"properties"`
}

// SearchNodesRequest is the payload for keyword-based node search.
type SearchNodesRequest struct {
	Keyword string `json:"keyword,required"`
	Label   string `json:"label,optional"`
	Limit   int    `json:"limit,optional"`
}

// PathResponse is the wire representation of a graph path query result.
type PathResponse struct {
	Nodes         []NodeResponse         `json:"nodes"`
	Relationships []RelationshipResponse `json:"relationships"`
	Length        int                    `json:"length"`
}

// SubgraphResponse is the wire representation of a subgraph query result.
// 字段命名对齐 doc/05 §5.9.1 的前端数据契约（nodes / edges）。
type SubgraphResponse struct {
	Nodes []NodeResponse         `json:"nodes"`
	Edges []RelationshipResponse `json:"edges"`
}

// LineageResponse is the wire representation of a school lineage query.
type LineageResponse struct {
	Path        PathResponse `json:"path"`
	Generations []int        `json:"generations"`
}

// FigureWithWorksResponse is the wire representation of a dynasty figures query.
type FigureWithWorksResponse struct {
	Person  NodeResponse   `json:"person"`
	Works   []NodeResponse `json:"works"`
	Schools []NodeResponse `json:"schools"`
}

// PrescriptionDetailResponse is the wire representation of a prescription detail query.
type PrescriptionDetailResponse struct {
	Prescription NodeResponse   `json:"prescription"`
	Medicines    []NodeResponse `json:"medicines"`
	Diseases     []NodeResponse `json:"diseases"`
}

// ShortestPathRequest captures the query parameters for the shortest path endpoint.
type ShortestPathRequest struct {
	StartUID string `query:"start_uid,required"`
	EndUID   string `query:"end_uid,required"`
	MaxHops  int    `query:"max_hops,optional"`
}

// SubgraphRequest captures the query parameters for the subgraph endpoint.
type SubgraphRequest struct {
	CenterUID string `query:"center_uid,required"`
	Depth     int    `query:"depth,optional"`
	Limit     int    `query:"limit,optional"`
}
