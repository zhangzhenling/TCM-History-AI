package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/graph-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// GraphNodeRepo implements repository.GraphNodeRepository with GORM.
type GraphNodeRepo struct {
	baseRepo
}

// NewGraphNodeRepo constructs a GraphNodeRepo.
func NewGraphNodeRepo(db *gorm.DB) *GraphNodeRepo {
	return &GraphNodeRepo{baseRepo{db: db}}
}

// Ensure GraphNodeRepo satisfies the interface at compile time.
var _ repository.GraphNodeRepository = (*GraphNodeRepo)(nil)

// Create inserts a new graph_node row.
func (r *GraphNodeRepo) Create(ctx context.Context, n *entity.GraphNode) error {
	if err := txFrom(ctx, r.db).Create(n).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create graph node", err)
	}
	return nil
}

// Update saves changes to an existing graph_node row.
func (r *GraphNodeRepo) Update(ctx context.Context, n *entity.GraphNode) error {
	res := txFrom(ctx, r.db).Save(n)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update graph node", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "graph node not found")
	}
	return nil
}

// Delete soft-deletes a graph_node by uid.
func (r *GraphNodeRepo) Delete(ctx context.Context, uid string) error {
	res := txFrom(ctx, r.db).Where("uid = ?", uid).Delete(&entity.GraphNode{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete graph node", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "graph node not found")
	}
	return nil
}

// FindByUID fetches a single graph_node by uid; returns (nil, nil) when not found.
func (r *GraphNodeRepo) FindByUID(ctx context.Context, uid string) (*entity.GraphNode, error) {
	var n entity.GraphNode
	err := txFrom(ctx, r.db).Where("uid = ?", uid).First(&n).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find graph node", err)
	}
	return &n, nil
}

// ListByLabel returns a paginated list of nodes filtered by label.
// When label is empty, all nodes are returned.
func (r *GraphNodeRepo) ListByLabel(ctx context.Context, label string, p pagination.Params) ([]entity.GraphNode, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.GraphNode
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.GraphNode{})
	if label != "" {
		db = db.Where("label = ?", label)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count graph nodes", err)
	}
	if err := db.Order("created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list graph nodes", err)
	}
	return items, int(total), nil
}

// SearchByName returns a paginated list of nodes whose name matches keyword,
// optionally restricted to a label.
func (r *GraphNodeRepo) SearchByName(ctx context.Context, keyword, label string, p pagination.Params) ([]entity.GraphNode, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.GraphNode
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.GraphNode{})
	if keyword != "" {
		db = db.Where("name ILIKE ?", "%"+keyword+"%")
	}
	if label != "" {
		db = db.Where("label = ?", label)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count graph nodes by keyword", err)
	}
	if err := db.Order("created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "search graph nodes", err)
	}
	return items, int(total), nil
}
