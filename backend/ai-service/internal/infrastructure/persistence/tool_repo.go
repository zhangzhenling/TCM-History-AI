package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/ai-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// ToolRepo implements repository.ToolRepository with GORM.
type ToolRepo struct {
	baseRepo
}

// NewToolRepo constructs a ToolRepo.
func NewToolRepo(db *gorm.DB) *ToolRepo {
	return &ToolRepo{baseRepo{db: db}}
}

// Ensure ToolRepo satisfies the interface at compile time.
var _ repository.ToolRepository = (*ToolRepo)(nil)

// Create inserts a new tool row.
func (r *ToolRepo) Create(ctx context.Context, t *entity.Tool) error {
	if err := txFrom(ctx, r.db).Create(t).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create tool", err)
	}
	return nil
}

// Update saves changes to an existing tool row.
func (r *ToolRepo) Update(ctx context.Context, t *entity.Tool) error {
	res := txFrom(ctx, r.db).Save(t)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update tool", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "tool not found")
	}
	return nil
}

// Delete soft-deletes a tool by id.
func (r *ToolRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Where("id = ?", id).Delete(&entity.Tool{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete tool", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "tool not found")
	}
	return nil
}

// FindByID fetches a single tool by id.
func (r *ToolRepo) FindByID(ctx context.Context, id int64) (*entity.Tool, error) {
	var t entity.Tool
	err := txFrom(ctx, r.db).First(&t, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find tool", err)
	}
	return &t, nil
}

// FindByName looks up a tool by name.
func (r *ToolRepo) FindByName(ctx context.Context, name string) (*entity.Tool, error) {
	var t entity.Tool
	err := txFrom(ctx, r.db).Where("name = ?", name).First(&t).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find tool by name", err)
	}
	return &t, nil
}

// ListEnabled returns paginated enabled tools.
func (r *ToolRepo) ListEnabled(ctx context.Context, p pagination.Params) ([]entity.Tool, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.Tool
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Tool{}).Where("is_enabled = ?", true)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count enabled tools", err)
	}
	if err := db.Order("updated_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list enabled tools", err)
	}
	return items, int(total), nil
}

// List returns paginated tools (including disabled).
func (r *ToolRepo) List(ctx context.Context, p pagination.Params) ([]entity.Tool, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.Tool
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Tool{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count tools", err)
	}
	if err := db.Order("updated_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list tools", err)
	}
	return items, int(total), nil
}
