package usecase

import (
	"context"

	"tcm-history-ai/backend/graph-service/internal/application/dto"
	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/graph-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
)

// QueryUseCase implements the complex graph query operations defined in
// doc/05 §5.5 and §5.7. Each method maps to a REST endpoint exposed by the
// query controller. 查询通过 GraphStore（Neo4j 适配器）执行；在 stub 模式下
// 返回空结果，不阻塞接口契约。
type QueryUseCase struct {
	store service.GraphStore
}

// NewQueryUseCase constructs a QueryUseCase.
func NewQueryUseCase(store service.GraphStore) *QueryUseCase {
	return &QueryUseCase{store: store}
}

// GetPersonWorks returns the classics authored by a person (doc/05 §5.5.1).
func (uc *QueryUseCase) GetPersonWorks(ctx context.Context, personUID string) ([]dto.NodeView, error) {
	if personUID == "" {
		return nil, errno.New(errno.InvalidParams, "person_uid is required")
	}
	nodes, err := uc.store.GetPersonWorks(ctx, personUID)
	if err != nil {
		return nil, err
	}
	return toNodeViews(nodes), nil
}

// GetSchoolLineage returns the discipled lineage of a school up to maxDepth
// hops (doc/05 §5.5.2).
func (uc *QueryUseCase) GetSchoolLineage(ctx context.Context, schoolName string, maxDepth int) (*dto.LineageResponse, error) {
	if schoolName == "" {
		return nil, errno.New(errno.InvalidParams, "school_name is required")
	}
	if maxDepth <= 0 {
		maxDepth = 6
	}
	lineage, err := uc.store.GetSchoolLineage(ctx, schoolName, maxDepth)
	if err != nil {
		return nil, err
	}
	if lineage == nil {
		return nil, errno.New(errno.NotFound, "school lineage not found: "+schoolName)
	}
	return &dto.LineageResponse{
		Path:        toGraphPath(&lineage.Path),
		Generations: lineage.Generations,
	}, nil
}

// FindShortestPath returns the shortest path between two nodes, bounded by
// maxHops (doc/05 §5.5.3).
func (uc *QueryUseCase) FindShortestPath(ctx context.Context, startUID, endUID string, maxHops int) (*dto.GraphPath, error) {
	if startUID == "" || endUID == "" {
		return nil, errno.New(errno.InvalidParams, "start_uid and end_uid are required")
	}
	if maxHops <= 0 {
		maxHops = 8
	}
	path, err := uc.store.QueryPath(ctx, startUID, endUID, maxHops)
	if err != nil {
		return nil, err
	}
	if path == nil {
		return nil, errno.New(errno.NotFound, "path not reachable")
	}
	resp := toGraphPath(path)
	return &resp, nil
}

// GetDynastyFigures returns the representative figures and their works for a
// dynasty (doc/05 §5.5.4).
func (uc *QueryUseCase) GetDynastyFigures(ctx context.Context, dynastyName string) ([]dto.FigureWithWorksResponse, error) {
	if dynastyName == "" {
		return nil, errno.New(errno.InvalidParams, "dynasty_name is required")
	}
	figures, err := uc.store.GetDynastyFigures(ctx, dynastyName)
	if err != nil {
		return nil, err
	}
	resp := make([]dto.FigureWithWorksResponse, 0, len(figures))
	for i := range figures {
		f := figures[i]
		resp = append(resp, dto.FigureWithWorksResponse{
			Person:  toNodeView(&f.Person),
			Works:   toNodeViews(f.Works),
			Schools: toNodeViews(f.Schools),
		})
	}
	return resp, nil
}

// GetPrescriptionDetail returns the medicines and diseases associated with a
// prescription (doc/05 §5.5.5).
func (uc *QueryUseCase) GetPrescriptionDetail(ctx context.Context, prescriptionUID string) (*dto.PrescriptionDetailResponse, error) {
	if prescriptionUID == "" {
		return nil, errno.New(errno.InvalidParams, "prescription_uid is required")
	}
	g, err := uc.store.GetPrescriptionDetail(ctx, prescriptionUID)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, errno.New(errno.NotFound, "prescription not found: "+prescriptionUID)
	}
	return &dto.PrescriptionDetailResponse{
		Prescription: toNodeView(&g.Prescription),
		Medicines:    toNodeViews(g.Medicines),
		Diseases:     toNodeViews(g.Diseases),
	}, nil
}

// SearchNodes runs a keyword search over nodes, optionally restricted to a label.
// Unlike NodeUseCase.List which queries the PostgreSQL mirror, this method
// uses the Neo4j fulltext index for richer semantic matching (doc/05 §5.8.3).
func (uc *QueryUseCase) SearchNodes(ctx context.Context, params *dto.SearchParams) (*dto.SearchResponse, error) {
	if params == nil || params.Keyword == "" {
		return nil, errno.New(errno.InvalidParams, "keyword is required")
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	nodes, err := uc.store.SearchNodes(ctx, params.Keyword, params.Label, limit)
	if err != nil {
		return nil, err
	}
	return &dto.SearchResponse{
		Keyword: params.Keyword,
		Label:   params.Label,
		Total:   len(nodes),
		Items:   toNodeViews(nodes),
	}, nil
}

// GetSubgraph returns the subgraph centred on a node, bounded by depth and a
// node limit, for front-end visualisation (doc/05 §5.9).
func (uc *QueryUseCase) GetSubgraph(ctx context.Context, centerUID string, depth, limit int) (*dto.Subgraph, error) {
	if centerUID == "" {
		return nil, errno.New(errno.InvalidParams, "center_uid is required")
	}
	if depth <= 0 {
		depth = 2
	}
	if limit <= 0 {
		limit = 100
	}
	// 节点上限 300（doc/05 §5.9.2），超出阈值应由前端聚类折叠。
	if limit > 300 {
		return nil, errno.New(errno.InvalidParams, "subgraph limit exceeds 300 nodes")
	}
	sub, err := uc.store.GetSubgraph(ctx, centerUID, depth, limit)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return &dto.Subgraph{}, nil
	}
	return &dto.Subgraph{
		Nodes: toNodeViews(sub.Nodes),
		Edges: toEdgeViews(sub.Edges),
	}, nil
}

// toNodeView maps a domain node view to its wire DTO.
func toNodeView(n *entity.GraphNodeView) dto.NodeView {
	if n == nil {
		return dto.NodeView{}
	}
	return dto.NodeView{
		UID:        n.UID,
		Label:      n.Label,
		Name:       n.Name,
		Properties: normalisePropsJSON(n.Properties),
	}
}

// toNodeViews maps a slice of domain node views to DTOs.
func toNodeViews(nodes []entity.GraphNodeView) []dto.NodeView {
	resp := make([]dto.NodeView, 0, len(nodes))
	for i := range nodes {
		resp = append(resp, toNodeView(&nodes[i]))
	}
	return resp
}

// toEdgeView maps a domain edge view to its wire DTO.
func toEdgeView(e *entity.GraphEdgeView) dto.EdgeView {
	if e == nil {
		return dto.EdgeView{}
	}
	return dto.EdgeView{
		UID:        e.UID,
		Type:       e.Type,
		SourceUID:  e.SourceUID,
		TargetUID:  e.TargetUID,
		Properties: normalisePropsJSON(e.Properties),
	}
}

// toEdgeViews maps a slice of domain edge views to DTOs.
func toEdgeViews(edges []entity.GraphEdgeView) []dto.EdgeView {
	resp := make([]dto.EdgeView, 0, len(edges))
	for i := range edges {
		resp = append(resp, toEdgeView(&edges[i]))
	}
	return resp
}

// toGraphPath maps a domain graph path to its wire DTO.
func toGraphPath(p *entity.GraphPath) dto.GraphPath {
	if p == nil {
		return dto.GraphPath{}
	}
	return dto.GraphPath{
		Nodes: toNodeViews(p.Nodes),
		Edges: toEdgeViews(p.Edges),
		Hops:  p.Hops,
	}
}
