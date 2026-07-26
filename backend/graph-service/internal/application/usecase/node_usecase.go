package usecase

import (
	"context"
	"encoding/json"
	"time"

	"tcm-history-ai/backend/graph-service/internal/application/dto"
	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/graph-service/internal/domain/event"
	"tcm-history-ai/backend/graph-service/internal/domain/repository"
	"tcm-history-ai/backend/graph-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

// NodeUseCase implements CRUD operations and search on graph nodes.
// 写入路径同时更新 PostgreSQL graph_nodes 镜像表与 Neo4j 图谱（GraphStore），
// 并发布 NodeUpserted 事件供下游服务（AI Service）补充图谱关联。
type NodeUseCase struct {
	repo  repository.GraphNodeRepository
	store service.GraphStore
	pub   event.EventPublisher
}

// NewNodeUseCase constructs a NodeUseCase.
func NewNodeUseCase(repo repository.GraphNodeRepository, store service.GraphStore, pub event.EventPublisher) *NodeUseCase {
	return &NodeUseCase{repo: repo, store: store, pub: pub}
}

// Create persists a new graph node, mirrors it to Neo4j, and publishes an event.
func (uc *NodeUseCase) Create(ctx context.Context, in *dto.NodeRequest) (*dto.NodeResponse, error) {
	if in == nil || in.UID == "" {
		return nil, errno.New(errno.InvalidParams, "uid is required")
	}
	if in.Name == "" {
		return nil, errno.New(errno.InvalidParams, "name is required")
	}
	if in.Label == "" {
		return nil, errno.New(errno.InvalidParams, "label is required")
	}
	if !entity.IsValidLabel(in.Label) {
		return nil, errno.New(errno.InvalidParams, "unknown node label: "+in.Label)
	}
	props := normalisePropsJSON(in.PropertiesJSON)
	n := &entity.GraphNode{
		UID:            in.UID,
		Label:          in.Label,
		Name:           in.Name,
		PropertiesJSON: props,
	}
	n.ID = idgen.Next()
	if err := uc.repo.Create(ctx, n); err != nil {
		return nil, err
	}
	// 镜像同步到 Neo4j（stub 模式下 no-op，不阻塞主流程）。
	_ = uc.store.UpsertNode(ctx, service.NodePayload{
		UID:        n.UID,
		Label:      n.Label,
		Name:       n.Name,
		Properties: propsToMap(props),
	})
	_ = uc.pub.Publish(ctx, event.NodeUpserted{UID: n.UID, Label: n.Label, Name: n.Name})
	return toNodeResponse(n), nil
}

// Update modifies an existing graph node identified by uid.
func (uc *NodeUseCase) Update(ctx context.Context, uid string, in *dto.NodeRequest) (*dto.NodeResponse, error) {
	if uid == "" {
		return nil, errno.New(errno.InvalidParams, "uid is required")
	}
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	if in.Label != "" && !entity.IsValidLabel(in.Label) {
		return nil, errno.New(errno.InvalidParams, "unknown node label: "+in.Label)
	}
	n, err := uc.repo.FindByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, errno.New(errno.NotFound, "node not found: "+uid)
	}
	if in.Label != "" {
		n.Label = in.Label
	}
	if in.Name != "" {
		n.Name = in.Name
	}
	if in.PropertiesJSON != nil {
		n.PropertiesJSON = normalisePropsJSON(in.PropertiesJSON)
	}
	if err := uc.repo.Update(ctx, n); err != nil {
		return nil, err
	}
	_ = uc.store.UpsertNode(ctx, service.NodePayload{
		UID:        n.UID,
		Label:      n.Label,
		Name:       n.Name,
		Properties: propsToMap(n.PropertiesJSON),
	})
	_ = uc.pub.Publish(ctx, event.NodeUpserted{UID: n.UID, Label: n.Label, Name: n.Name})
	return toNodeResponse(n), nil
}

// Delete removes a node by uid from both PostgreSQL and Neo4j.
func (uc *NodeUseCase) Delete(ctx context.Context, uid string) error {
	if uid == "" {
		return errno.New(errno.InvalidParams, "uid is required")
	}
	if err := uc.repo.Delete(ctx, uid); err != nil {
		return err
	}
	_ = uc.store.DeleteNode(ctx, uid)
	return nil
}

// Get retrieves a single node by uid.
func (uc *NodeUseCase) Get(ctx context.Context, uid string) (*dto.NodeResponse, error) {
	if uid == "" {
		return nil, errno.New(errno.InvalidParams, "uid is required")
	}
	n, err := uc.repo.FindByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, errno.New(errno.NotFound, "node not found: "+uid)
	}
	return toNodeResponse(n), nil
}

// List returns a paginated list of nodes. When keyword is non-empty, performs
// a name search; otherwise lists by label (or all when label is empty).
func (uc *NodeUseCase) List(ctx context.Context, label, keyword string, p pagination.Params) (dto.ListResponse[dto.NodeResponse], error) {
	if label != "" && !entity.IsValidLabel(label) {
		return dto.ListResponse[dto.NodeResponse]{}, errno.New(errno.InvalidParams, "unknown node label: "+label)
	}
	var (
		items []entity.GraphNode
		total int
		err   error
	)
	if keyword != "" {
		items, total, err = uc.repo.SearchByName(ctx, keyword, label, p)
	} else {
		items, total, err = uc.repo.ListByLabel(ctx, label, p)
	}
	if err != nil {
		return dto.ListResponse[dto.NodeResponse]{}, err
	}
	resp := make([]dto.NodeResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toNodeResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// toNodeResponse maps the entity to its wire DTO.
func toNodeResponse(n *entity.GraphNode) *dto.NodeResponse {
	if n == nil {
		return nil
	}
	resp := &dto.NodeResponse{
		ID:             n.ID,
		UID:            n.UID,
		Label:          n.Label,
		Name:           n.Name,
		PropertiesJSON: normalisePropsJSON(n.PropertiesJSON),
	}
	if !n.SyncedAt.IsZero() {
		resp.SyncedAt = n.SyncedAt.Format(time.RFC3339)
	}
	if !n.CreatedAt.IsZero() {
		resp.CreatedAt = n.CreatedAt.Format(time.RFC3339)
	}
	if !n.UpdatedAt.IsZero() {
		resp.UpdatedAt = n.UpdatedAt.Format(time.RFC3339)
	}
	return resp
}

// normalisePropsJSON returns "{}" when raw is empty or null, otherwise raw as-is.
func normalisePropsJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || string(raw) == "null" {
		return json.RawMessage("{}")
	}
	return raw
}

// propsToMap unmarshals a JSON properties blob into a map for the GraphStore payload.
func propsToMap(raw json.RawMessage) map[string]any {
	props := map[string]any{}
	if len(raw) == 0 || string(raw) == "null" {
		return props
	}
	_ = json.Unmarshal(raw, &props)
	return props
}
