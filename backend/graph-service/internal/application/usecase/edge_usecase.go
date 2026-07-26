package usecase

import (
	"context"
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

// EdgeUseCase implements CRUD operations on graph edges.
// 写入路径同时更新 PostgreSQL graph_edges 镜像表与 Neo4j 图谱（GraphStore），
// 并发布 EdgeUpserted 事件供下游服务消费。
type EdgeUseCase struct {
	repo  repository.GraphEdgeRepository
	store service.GraphStore
	pub   event.EventPublisher
}

// NewEdgeUseCase constructs an EdgeUseCase.
func NewEdgeUseCase(repo repository.GraphEdgeRepository, store service.GraphStore, pub event.EventPublisher) *EdgeUseCase {
	return &EdgeUseCase{repo: repo, store: store, pub: pub}
}

// Create persists a new graph edge, mirrors it to Neo4j, and publishes an event.
func (uc *EdgeUseCase) Create(ctx context.Context, in *dto.EdgeRequest) (*dto.EdgeResponse, error) {
	if err := validateEdgeRequest(in); err != nil {
		return nil, err
	}
	props := normalisePropsJSON(in.PropertiesJSON)
	e := &entity.GraphEdge{
		UID:            in.UID,
		Type:           in.Type,
		SourceUID:      in.SourceUID,
		TargetUID:      in.TargetUID,
		PropertiesJSON: props,
	}
	e.ID = idgen.Next()
	if err := uc.repo.Create(ctx, e); err != nil {
		return nil, err
	}
	_ = uc.store.UpsertEdge(ctx, service.EdgePayload{
		UID:        e.UID,
		Type:       e.Type,
		SourceUID:  e.SourceUID,
		TargetUID:  e.TargetUID,
		Properties: propsToMap(props),
	})
	_ = uc.pub.Publish(ctx, event.EdgeUpserted{
		UID:       e.UID,
		Type:      e.Type,
		SourceUID: e.SourceUID,
		TargetUID: e.TargetUID,
	})
	return toEdgeResponse(e), nil
}

// Update modifies an existing graph edge identified by uid.
func (uc *EdgeUseCase) Update(ctx context.Context, uid string, in *dto.EdgeRequest) (*dto.EdgeResponse, error) {
	if uid == "" {
		return nil, errno.New(errno.InvalidParams, "uid is required")
	}
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	if in.Type != "" && !entity.IsValidEdgeType(in.Type) {
		return nil, errno.New(errno.InvalidParams, "unknown edge type: "+in.Type)
	}
	e, err := uc.repo.FindByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errno.New(errno.NotFound, "edge not found: "+uid)
	}
	if in.Type != "" {
		e.Type = in.Type
	}
	if in.SourceUID != "" {
		e.SourceUID = in.SourceUID
	}
	if in.TargetUID != "" {
		e.TargetUID = in.TargetUID
	}
	if in.PropertiesJSON != nil {
		e.PropertiesJSON = normalisePropsJSON(in.PropertiesJSON)
	}
	if err := uc.repo.Update(ctx, e); err != nil {
		return nil, err
	}
	_ = uc.store.UpsertEdge(ctx, service.EdgePayload{
		UID:        e.UID,
		Type:       e.Type,
		SourceUID:  e.SourceUID,
		TargetUID:  e.TargetUID,
		Properties: propsToMap(e.PropertiesJSON),
	})
	_ = uc.pub.Publish(ctx, event.EdgeUpserted{
		UID:       e.UID,
		Type:      e.Type,
		SourceUID: e.SourceUID,
		TargetUID: e.TargetUID,
	})
	return toEdgeResponse(e), nil
}

// Delete removes an edge by uid from both PostgreSQL and Neo4j.
func (uc *EdgeUseCase) Delete(ctx context.Context, uid string) error {
	if uid == "" {
		return errno.New(errno.InvalidParams, "uid is required")
	}
	if err := uc.repo.Delete(ctx, uid); err != nil {
		return err
	}
	_ = uc.store.DeleteEdge(ctx, uid)
	return nil
}

// Get retrieves a single edge by uid.
func (uc *EdgeUseCase) Get(ctx context.Context, uid string) (*dto.EdgeResponse, error) {
	if uid == "" {
		return nil, errno.New(errno.InvalidParams, "uid is required")
	}
	e, err := uc.repo.FindByUID(ctx, uid)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errno.New(errno.NotFound, "edge not found: "+uid)
	}
	return toEdgeResponse(e), nil
}

// List returns a paginated list of edges filtered by source_uid, target_uid,
// and/or type. At least one filter is recommended to keep result sets bounded.
func (uc *EdgeUseCase) List(ctx context.Context, sourceUID, targetUID, edgeType string, p pagination.Params) (dto.ListResponse[dto.EdgeResponse], error) {
	if edgeType != "" && !entity.IsValidEdgeType(edgeType) {
		return dto.ListResponse[dto.EdgeResponse]{}, errno.New(errno.InvalidParams, "unknown edge type: "+edgeType)
	}
	var (
		items []entity.GraphEdge
		total int
		err   error
	)
	switch {
	case sourceUID != "":
		items, total, err = uc.repo.ListBySource(ctx, sourceUID, p)
	case targetUID != "":
		items, total, err = uc.repo.ListByTarget(ctx, targetUID, p)
	case edgeType != "":
		items, total, err = uc.repo.ListByType(ctx, edgeType, p)
	default:
		// No filter provided; fall back to listing by an empty type filter
		// via ListByType which returns all edges paginated.
		items, total, err = uc.repo.ListByType(ctx, "", p)
	}
	if err != nil {
		return dto.ListResponse[dto.EdgeResponse]{}, err
	}
	resp := make([]dto.EdgeResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toEdgeResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// validateEdgeRequest validates the create payload.
func validateEdgeRequest(in *dto.EdgeRequest) error {
	if in == nil {
		return errno.New(errno.InvalidParams, "body is required")
	}
	if in.UID == "" {
		return errno.New(errno.InvalidParams, "uid is required")
	}
	if in.Type == "" {
		return errno.New(errno.InvalidParams, "type is required")
	}
	if !entity.IsValidEdgeType(in.Type) {
		return errno.New(errno.InvalidParams, "unknown edge type: "+in.Type)
	}
	if in.SourceUID == "" || in.TargetUID == "" {
		return errno.New(errno.InvalidParams, "source_uid and target_uid are required")
	}
	return nil
}

// toEdgeResponse maps the entity to its wire DTO.
func toEdgeResponse(e *entity.GraphEdge) *dto.EdgeResponse {
	if e == nil {
		return nil
	}
	resp := &dto.EdgeResponse{
		ID:             e.ID,
		UID:            e.UID,
		Type:           e.Type,
		SourceUID:      e.SourceUID,
		TargetUID:      e.TargetUID,
		PropertiesJSON: normalisePropsJSON(e.PropertiesJSON),
	}
	if !e.SyncedAt.IsZero() {
		resp.SyncedAt = e.SyncedAt.Format(time.RFC3339)
	}
	if !e.CreatedAt.IsZero() {
		resp.CreatedAt = e.CreatedAt.Format(time.RFC3339)
	}
	if !e.UpdatedAt.IsZero() {
		resp.UpdatedAt = e.UpdatedAt.Format(time.RFC3339)
	}
	return resp
}
