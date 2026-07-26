package usecase

import (
	"context"
	"encoding/json"

	"tcm-history-ai/backend/graph-service/internal/application/dto"
	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/graph-service/internal/domain/event"
	"tcm-history-ai/backend/graph-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// NodeUseCase implements CRUD operations and search on graph nodes.
type NodeUseCase struct {
	repo repository.GraphRepository
	pub  event.EventPublisher
}

// NewNodeUseCase constructs a NodeUseCase.
func NewNodeUseCase(repo repository.GraphRepository, pub event.EventPublisher) *NodeUseCase {
	return &NodeUseCase{repo: repo, pub: pub}
}

// Create upserts a node (MERGE semantics by uid). The label must be one of the
// 8 known node labels. On success a NodeUpserted event is published.
func (uc *NodeUseCase) Create(ctx context.Context, in *dto.NodeRequest) (*dto.NodeResponse, error) {
	if in == nil || in.UID == "" {
		return nil, errno.New(errno.InvalidParams, "uid is required")
	}
	if in.Label == "" {
		return nil, errno.New(errno.InvalidParams, "label is required")
	}
	if !entity.IsValidLabel(in.Label) {
		return nil, errno.New(errno.InvalidParams, "unknown node label: "+in.Label)
	}
	props, err := propsToMap(in.Properties)
	if err != nil {
		return nil, errno.Wrap(errno.InvalidParams, "invalid properties", err)
	}
	if err := uc.repo.MergeNode(ctx, in.Label, in.UID, props); err != nil {
		return nil, err
	}
	// 发布节点 upsert 事件，供 AI Service 在 RAG 上下文中补充图谱关联。
	_ = uc.pub.Publish(ctx, event.NodeUpserted{UID: in.UID, Label: in.Label})
	return &dto.NodeResponse{UID: in.UID, Label: in.Label, Properties: mapToProps(props)}, nil
}

// Get retrieves a single node by uid.
func (uc *NodeUseCase) Get(ctx context.Context, uid string) (*dto.NodeResponse, error) {
	if uid == "" {
		return nil, errno.New(errno.InvalidParams, "uid is required")
	}
	n, err := uc.repo.GetNode(ctx, uid)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, errno.New(errno.NotFound, "node not found: "+uid)
	}
	return toNodeResponse(n), nil
}

// Delete removes a node by uid.
func (uc *NodeUseCase) Delete(ctx context.Context, uid string) error {
	if uid == "" {
		return errno.New(errno.InvalidParams, "uid is required")
	}
	return uc.repo.DeleteNode(ctx, uid)
}

// List returns a paginated list of nodes, optionally filtered by label.
func (uc *NodeUseCase) List(ctx context.Context, label string, p pagination.Params) (dto.ListResponse[dto.NodeResponse], error) {
	if label != "" && !entity.IsValidLabel(label) {
		return dto.ListResponse[dto.NodeResponse]{}, errno.New(errno.InvalidParams, "unknown node label: "+label)
	}
	items, total, err := uc.repo.ListNodes(ctx, label, p)
	if err != nil {
		return dto.ListResponse[dto.NodeResponse]{}, err
	}
	resp := make([]dto.NodeResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toNodeResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// Search runs a keyword search over nodes, optionally restricted to a label.
func (uc *NodeUseCase) Search(ctx context.Context, in *dto.SearchNodesRequest) ([]dto.NodeResponse, error) {
	if in == nil || in.Keyword == "" {
		return nil, errno.New(errno.InvalidParams, "keyword is required")
	}
	if in.Label != "" && !entity.IsValidLabel(in.Label) {
		return nil, errno.New(errno.InvalidParams, "unknown node label: "+in.Label)
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	nodes, err := uc.repo.SearchNodes(ctx, in.Keyword, in.Label, limit)
	if err != nil {
		return nil, err
	}
	resp := make([]dto.NodeResponse, 0, len(nodes))
	for i := range nodes {
		resp = append(resp, *toNodeResponse(&nodes[i]))
	}
	return resp, nil
}

// toNodeResponse maps the entity to its wire DTO.
func toNodeResponse(n *entity.GraphNode) *dto.NodeResponse {
	if n == nil {
		return nil
	}
	return &dto.NodeResponse{
		UID:        n.UID,
		Label:      n.Label,
		Properties: mapToProps(n.Properties),
	}
}

// propsToMap unmarshals a JSON properties blob into a map. An empty blob
// yields an empty (non-nil) map so downstream code can safely write into it.
func propsToMap(raw json.RawMessage) (map[string]any, error) {
	props := map[string]any{}
	if len(raw) == 0 || string(raw) == "null" {
		return props, nil
	}
	if err := json.Unmarshal(raw, &props); err != nil {
		return nil, err
	}
	return props, nil
}

// mapToProps marshals a properties map into a JSON blob. A nil map yields "{}".
func mapToProps(props map[string]any) json.RawMessage {
	if props == nil {
		return json.RawMessage("{}")
	}
	out, err := json.Marshal(props)
	if err != nil {
		return json.RawMessage("{}")
	}
	return out
}
