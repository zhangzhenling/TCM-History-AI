package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/history-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// DynastyRepo implements repository.DynastyRepository with GORM.
type DynastyRepo struct {
	baseRepo
}

// NewDynastyRepo constructs a DynastyRepo.
func NewDynastyRepo(db *gorm.DB) *DynastyRepo {
	return &DynastyRepo{baseRepo{db: db}}
}

// Ensure DynastyRepo satisfies the interface at compile time.
var _ repository.DynastyRepository = (*DynastyRepo)(nil)

// Create inserts a new dynasty row.
func (r *DynastyRepo) Create(ctx context.Context, d *entity.Dynasty) error {
	if err := txFrom(ctx, r.db).Create(d).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create dynasty", err)
	}
	return nil
}

// Update saves changes to an existing dynasty row.
func (r *DynastyRepo) Update(ctx context.Context, d *entity.Dynasty) error {
	res := txFrom(ctx, r.db).Save(d)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update dynasty", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "dynasty not found")
	}
	return nil
}

// Delete soft-deletes a dynasty by id.
func (r *DynastyRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Where("id = ?", id).Delete(&entity.Dynasty{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete dynasty", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "dynasty not found")
	}
	return nil
}

// FindByID fetches a single dynasty by id; returns (nil, nil) when not found.
func (r *DynastyRepo) FindByID(ctx context.Context, id int64) (*entity.Dynasty, error) {
	var d entity.Dynasty
	err := txFrom(ctx, r.db).First(&d, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find dynasty", err)
	}
	return &d, nil
}

// List returns a paginated list ordered by sort_order, id.
func (r *DynastyRepo) List(ctx context.Context, p pagination.Params) ([]entity.Dynasty, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.Dynasty
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Dynasty{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count dynasty", err)
	}
	if err := db.Order("sort_order ASC, id ASC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list dynasty", err)
	}
	return items, int(total), nil
}

// Search keyword-matches dynasties via ILIKE on name and description.
func (r *DynastyRepo) Search(ctx context.Context, keyword string, p pagination.Params) ([]entity.Dynasty, int, error) {
	_, pageSize, offset := p.Normalise()
	pattern := "%" + keyword + "%"
	var items []entity.Dynasty
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Dynasty{}).
		Where("name ILIKE ? OR description ILIKE ?", pattern, pattern)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count dynasty search", err)
	}
	if err := db.Order("sort_order ASC, id ASC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "search dynasty", err)
	}
	return items, int(total), nil
}
