package usecase

import (
	"context"

	"tcm-history-ai/backend/graph-service/internal/application/dto"
	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/graph-service/internal/domain/event"
	"tcm-history-ai/backend/graph-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
)

// RelationshipUseCase implements CRUD operations on graph relationships.
type RelationshipUseCase struct {
	repo repository.GraphRepository
	pub  event.EventPublisher
}

// NewRelationshipUseCase constructs a RelationshipUseCase.
func NewRelationshipUseCase(repo repository.GraphRepository, pub event.EventPublisher) *RelationshipUseCase {
	return &RelationshipUseCase{repo: repo, pub: pub}
}

// Create upserts a relationship (MERGE semantics by uid). The type must be one
// of the 9 known relationship types. On success a RelationshipUpserted event is
// published.
func (uc *RelationshipUseCase) Create(ctx context.Context, in *dto.RelationshipRequest) (*dto.RelationshipResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	if in.UID == "" {
		return nil, errno.New(errno.InvalidParams, "uid is required")
	}
	if in.Type == "" {
		return nil, errno.New(errno.InvalidParams, "type is required")
	}
	if !entity.IsValidRelationshipType(in.Type) {
		return nil, errno.New(errno.InvalidParams, "unknown relationship type: "+in.Type)
	}
	if in.FromUID == "" || in.ToUID == "" {
		return nil, errno.New(errno.InvalidParams, "from_uid and to_uid are required")
	}
	props, err := propsToMap(in.Properties)
	if err != nil {
		return nil, errno.Wrap(errno.InvalidParams, "invalid properties", err)
	}
	if err := uc.repo.MergeRelationship(ctx, in.Type, in.FromUID, in.ToUID, in.UID, props); err != nil {
		return nil, err
	}
	_ = uc.pub.Publish(ctx, event.RelationshipUpserted{
		UID:    in.UID,
		Type:   in.Type,
		FromUID: in.FromUID,
		ToUID:   in.ToUID,
	})
	return &dto.RelationshipResponse{
		UID:        in.UID,
		Type:       in.Type,
		FromUID:    in.FromUID,
		ToUID:      in.ToUID,
		Properties: mapToProps(props),
	}, nil
}

// Get retrieves a single relationship by uid.
func (uc *RelationshipUseCase) Get(ctx context.Context, uid string) (*dto.RelationshipResponse, error) {
	if uid == "" {
		return nil, errno.New(errno.InvalidParams, "uid is required")
	}
	r, err := uc.repo.GetRelationship(ctx, uid)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, errno.New(errno.NotFound, "relationship not found: "+uid)
	}
	return toRelationshipResponse(r), nil
}

// Delete removes a relationship by uid.
func (uc *RelationshipUseCase) Delete(ctx context.Context, uid string) error {
	if uid == "" {
		return errno.New(errno.InvalidParams, "uid is required")
	}
	return uc.repo.DeleteRelationship(ctx, uid)
}

// toRelationshipResponse maps the entity to its wire DTO.
func toRelationshipResponse(r *entity.GraphRelationship) *dto.RelationshipResponse {
	if r == nil {
		return nil
	}
	return &dto.RelationshipResponse{
		UID:        r.UID,
		Type:       r.Type,
		FromUID:    r.FromUID,
		ToUID:      r.ToUID,
		Properties: mapToProps(r.Properties),
	}
}
