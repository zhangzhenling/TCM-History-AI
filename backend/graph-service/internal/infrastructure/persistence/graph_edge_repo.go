package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/graph-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// GraphEdgeRepo implements repository.GraphEdgeRepository with GORM.
type GraphEdgeRepo struct {
	baseRepo
}

// NewGraphEdgeRepo constructs a GraphEdgeRepo.
func NewGraphEdgeRepo(db *gorm.DB) *GraphEdgeRepo {
	return &GraphEdgeRepo{baseRepo{db: db}}
}

// Ensure GraphEdgeRepo satisfies the interface at compile time.
var _ repository.GraphEdgeRepository = (*GraphEdgeRepo)(nil)

// Create inserts a new graph_edge row.
func (r *GraphEdgeRepo) Create(ctx context.Context, e *entity.GraphEdge) error {
	if err := txFrom(ctx, r.db).Create(e).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create graph edge", err)
	}
	return nil
}

// Update saves changes to an existing graph_edge row.
func (r *GraphEdgeRepo) Update(ctx context.Context, e *entity.GraphEdge) error {
	res := txFrom(ctx, r.db).Save(e)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update graph edge", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "graph edge not found")
	}
	return nil
}

// Delete soft-deletes a graph_edge by uid.
func (r *GraphEdgeRepo) Delete(ctx context.Context, uid string) error {
	res := txFrom(ctx, r.db).Where("uid = ?", uid).Delete(&entity.GraphEdge{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete graph edge", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "graph edge not found")
	}
	return nil
}

// FindByUID fetches a single graph_edge by uid; returns (nil, nil) when not found.
func (r *GraphEdgeRepo) FindByUID(ctx context.Context, uid string) (*entity.GraphEdge, error) {
	var e entity.GraphEdge
	err := txFrom(ctx, r.db).Where("uid = ?", uid).First(&e).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find graph edge", err)
	}
	return &e, nil
}

// ListBySource returns a paginated list of edges whose source_uid matches.
func (r *GraphEdgeRepo) ListBySource(ctx context.Context, sourceUID string, p pagination.Params) ([]entity.GraphEdge, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.GraphEdge
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.GraphEdge{}).Where("source_uid = ?", sourceUID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count graph edges by source", err)
	}
	if err := db.Order("created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list graph edges by source", err)
	}
	return items, int(total), nil
}

// ListByTarget returns a paginated list of edges whose target_uid matches.
func (r *GraphEdgeRepo) ListByTarget(ctx context.Context, targetUID string, p pagination.Params) ([]entity.GraphEdge, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.GraphEdge
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.GraphEdge{}).Where("target_uid = ?", targetUID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count graph edges by target", err)
	}
	if err := db.Order("created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list graph edges by target", err)
	}
	return items, int(total), nil
}

// ListByType returns a paginated list of edges filtered by type.
// When edgeType is empty, all edges are returned.
func (r *GraphEdgeRepo) ListByType(ctx context.Context, edgeType string, p pagination.Params) ([]entity.GraphEdge, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.GraphEdge
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.GraphEdge{})
	if edgeType != "" {
		db = db.Where("type = ?", edgeType)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count graph edges by type", err)
	}
	if err := db.Order("created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list graph edges by type", err)
	}
	return items, int(total), nil
}
