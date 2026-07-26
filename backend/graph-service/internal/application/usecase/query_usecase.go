package usecase

import (
	"context"

	"tcm-history-ai/backend/graph-service/internal/application/dto"
	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/graph-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
)

// QueryUseCase implements the complex graph query operations defined in
// doc/05 §5.5 and §5.7. Each method maps to a REST endpoint exposed by
// the query controller.
type QueryUseCase struct {
	repo repository.GraphRepository
}

// NewQueryUseCase constructs a QueryUseCase.
func NewQueryUseCase(repo repository.GraphRepository) *QueryUseCase {
	return &QueryUseCase{repo: repo}
}

// GetPersonWorks returns the classics authored by a person (doc/05 §5.5.1).
func (uc *QueryUseCase) GetPersonWorks(ctx context.Context, personUID string) ([]dto.NodeResponse, error) {
	if personUID == "" {
		return nil, errno.New(errno.InvalidParams, "person uid is required")
	}
	nodes, err := uc.repo.GetPersonWorks(ctx, personUID)
	if err != nil {
		return nil, err
	}
	resp := make([]dto.NodeResponse, 0, len(nodes))
	for i := range nodes {
		resp = append(resp, *toNodeResponse(&nodes[i]))
	}
	return resp, nil
}

// GetSchoolLineage returns the discipled lineage of a school up to maxDepth
// hops (doc/05 §5.5.2).
func (uc *QueryUseCase) GetSchoolLineage(ctx context.Context, schoolName string, maxDepth int) (*dto.LineageResponse, error) {
	if schoolName == "" {
		return nil, errno.New(errno.InvalidParams, "school name is required")
	}
	if maxDepth <= 0 {
		maxDepth = 6
	}
	lineage, err := uc.repo.GetSchoolLineage(ctx, schoolName, maxDepth)
	if err != nil {
		return nil, err
	}
	if lineage == nil {
		return nil, errno.New(errno.NotFound, "school lineage not found: "+schoolName)
	}
	return &dto.LineageResponse{
		Path:        toPathResponse(&lineage.Path),
		Generations: lineage.Generations,
	}, nil
}

// FindShortestPath returns the shortest path between two nodes, bounded by
// maxHops (doc/05 §5.5.3).
func (uc *QueryUseCase) FindShortestPath(ctx context.Context, startUID, endUID string, maxHops int) (*dto.PathResponse, error) {
	if startUID == "" || endUID == "" {
		return nil, errno.New(errno.InvalidParams, "start_uid and end_uid are required")
	}
	if maxHops <= 0 {
		maxHops = 8
	}
	path, err := uc.repo.FindShortestPath(ctx, startUID, endUID, maxHops)
	if err != nil {
		return nil, err
	}
	if path == nil {
		// 路径不可达：返回 400202（doc/10 §2.4 错误码分类表）
		return nil, errno.New(errno.NotFound, "path not reachable")
	}
	resp := toPathResponse(path)
	return &resp, nil
}

// GetDynastyFigures returns the representative figures and their works for a
// dynasty (doc/05 §5.5.4).
func (uc *QueryUseCase) GetDynastyFigures(ctx context.Context, dynastyName string) ([]dto.FigureWithWorksResponse, error) {
	if dynastyName == "" {
		return nil, errno.New(errno.InvalidParams, "dynasty name is required")
	}
	figures, err := uc.repo.GetDynastyFigures(ctx, dynastyName)
	if err != nil {
		return nil, err
	}
	resp := make([]dto.FigureWithWorksResponse, 0, len(figures))
	for i := range figures {
		f := figures[i]
		resp = append(resp, dto.FigureWithWorksResponse{
			Person:  *toNodeResponse(&f.Person),
			Works:   toNodeResponses(f.Works),
			Schools: toNodeResponses(f.Schools),
		})
	}
	return resp, nil
}

// GetPrescriptionDetail returns the medicines and diseases associated with a
// prescription (doc/05 §5.5.5).
func (uc *QueryUseCase) GetPrescriptionDetail(ctx context.Context, prescriptionUID string) (*dto.PrescriptionDetailResponse, error) {
	if prescriptionUID == "" {
		return nil, errno.New(errno.InvalidParams, "prescription uid is required")
	}
	g, err := uc.repo.GetPrescriptionDetail(ctx, prescriptionUID)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, errno.New(errno.NotFound, "prescription not found: "+prescriptionUID)
	}
	return &dto.PrescriptionDetailResponse{
		Prescription: *toNodeResponse(&g.Prescription),
		Medicines:    toNodeResponses(g.Medicines),
		Diseases:     toNodeResponses(g.Diseases),
	}, nil
}

// GetSubgraph returns the subgraph centred on a node, bounded by depth and a
// node limit, for front-end visualisation (doc/05 §5.9).
func (uc *QueryUseCase) GetSubgraph(ctx context.Context, centerUID string, depth, limit int) (*dto.SubgraphResponse, error) {
	if centerUID == "" {
		return nil, errno.New(errno.InvalidParams, "center_uid is required")
	}
	if depth <= 0 {
		depth = 2
	}
	if limit <= 0 {
		limit = 100
	}
	// 节点上限 300（doc/05 §5.9.2），超出阈值应由前端聚类折叠，后端拒绝过大数据量。
	if limit > 300 {
		return nil, errno.New(errno.InvalidParams, "subgraph limit exceeds 300 nodes")
	}
	sub, err := uc.repo.GetSubgraph(ctx, centerUID, depth, limit)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return &dto.SubgraphResponse{}, nil
	}
	return &dto.SubgraphResponse{
		Nodes: toNodeResponses(sub.Nodes),
		Edges: toRelationshipResponses(sub.Relationships),
	}, nil
}

// toNodeResponses maps a slice of node entities to DTOs.
func toNodeResponses(nodes []entity.GraphNode) []dto.NodeResponse {
	resp := make([]dto.NodeResponse, 0, len(nodes))
	for i := range nodes {
		resp = append(resp, *toNodeResponse(&nodes[i]))
	}
	return resp
}

// toRelationshipResponses maps a slice of relationship entities to DTOs.
func toRelationshipResponses(rels []entity.GraphRelationship) []dto.RelationshipResponse {
	resp := make([]dto.RelationshipResponse, 0, len(rels))
	for i := range rels {
		resp = append(resp, *toRelationshipResponse(&rels[i]))
	}
	return resp
}

// toPathResponse maps a graph path entity to its wire DTO.
func toPathResponse(p *entity.GraphPath) dto.PathResponse {
	if p == nil {
		return dto.PathResponse{}
	}
	return dto.PathResponse{
		Nodes:         toNodeResponses(p.Nodes),
		Relationships: toRelationshipResponses(p.Relationships),
		Length:        p.Length,
	}
}
