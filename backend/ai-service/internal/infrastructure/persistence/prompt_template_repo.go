package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/ai-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// PromptTemplateRepo implements repository.PromptTemplateRepository with GORM.
type PromptTemplateRepo struct {
	baseRepo
}

// NewPromptTemplateRepo constructs a PromptTemplateRepo.
func NewPromptTemplateRepo(db *gorm.DB) *PromptTemplateRepo {
	return &PromptTemplateRepo{baseRepo{db: db}}
}

// Ensure PromptTemplateRepo satisfies the interface at compile time.
var _ repository.PromptTemplateRepository = (*PromptTemplateRepo)(nil)

// Create inserts a new prompt template row.
func (r *PromptTemplateRepo) Create(ctx context.Context, p *entity.PromptTemplate) error {
	if err := txFrom(ctx, r.db).Create(p).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create prompt template", err)
	}
	return nil
}

// Update saves changes to an existing prompt template row.
func (r *PromptTemplateRepo) Update(ctx context.Context, p *entity.PromptTemplate) error {
	res := txFrom(ctx, r.db).Save(p)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update prompt template", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "prompt template not found")
	}
	return nil
}

// Delete soft-deletes a prompt template by id. GORM Delete on a model with
// a DeletedAt column issues UPDATE ... SET deleted_at=now() WHERE id=? AND
// deleted_at IS NULL; rows already soft-deleted are not matched.
func (r *PromptTemplateRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Delete(&entity.PromptTemplate{}, id)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete prompt template", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "prompt template not found")
	}
	return nil
}

// DeactivateByScene sets is_active=false for every (non-deleted) template in
// the given scene, returning the number of rows updated. Used by Activate to
// ensure at most one active template per scene.
func (r *PromptTemplateRepo) DeactivateByScene(ctx context.Context, scene string) (int64, error) {
	res := txFrom(ctx, r.db).Model(&entity.PromptTemplate{}).
		Where("scene = ? AND is_active = ? AND deleted_at IS NULL", scene, true).
		Update("is_active", false)
	if res.Error != nil {
		return 0, errno.Wrap(errno.InternalError, "deactivate prompt templates by scene", res.Error)
	}
	return res.RowsAffected, nil
}

// FindByID fetches a single prompt template by id.
func (r *PromptTemplateRepo) FindByID(ctx context.Context, id int64) (*entity.PromptTemplate, error) {
	var p entity.PromptTemplate
	err := txFrom(ctx, r.db).First(&p, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find prompt template", err)
	}
	return &p, nil
}

// FindByNameAndScene looks up a prompt template by name + scene.
func (r *PromptTemplateRepo) FindByNameAndScene(ctx context.Context, name, scene string) (*entity.PromptTemplate, error) {
	var p entity.PromptTemplate
	err := txFrom(ctx, r.db).Where("name = ? AND scene = ?", name, scene).First(&p).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find prompt template by name", err)
	}
	return &p, nil
}

// ListByScene returns paginated prompt templates for a scene.
func (r *PromptTemplateRepo) ListByScene(ctx context.Context, scene string, p pagination.Params) ([]entity.PromptTemplate, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.PromptTemplate
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.PromptTemplate{})
	if scene != "" {
		db = db.Where("scene = ?", scene)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count prompt templates", err)
	}
	if err := db.Order("updated_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list prompt templates", err)
	}
	return items, int(total), nil
}

// FindActive returns the active template for a scene (most recently updated).
func (r *PromptTemplateRepo) FindActive(ctx context.Context, scene string) (*entity.PromptTemplate, error) {
	var p entity.PromptTemplate
	err := txFrom(ctx, r.db).Where("scene = ? AND is_active = ?", scene, true).
		Order("version DESC, updated_at DESC").First(&p).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find active prompt template", err)
	}
	return &p, nil
}

// List returns paginated prompt templates ordered by updated_at DESC.
func (r *PromptTemplateRepo) List(ctx context.Context, p pagination.Params) ([]entity.PromptTemplate, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.PromptTemplate
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.PromptTemplate{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count prompt templates", err)
	}
	if err := db.Order("updated_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list prompt templates", err)
	}
	return items, int(total), nil
}
