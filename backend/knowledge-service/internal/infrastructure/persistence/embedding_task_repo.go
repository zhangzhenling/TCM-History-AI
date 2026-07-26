package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
	"tcm-history-ai/backend/knowledge-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// EmbeddingTaskRepo implements repository.EmbeddingTaskRepository with GORM.
type EmbeddingTaskRepo struct {
	baseRepo
}

// NewEmbeddingTaskRepo constructs an EmbeddingTaskRepo.
func NewEmbeddingTaskRepo(db *gorm.DB) *EmbeddingTaskRepo {
	return &EmbeddingTaskRepo{baseRepo{db: db}}
}

// Ensure EmbeddingTaskRepo satisfies the interface at compile time.
var _ repository.EmbeddingTaskRepository = (*EmbeddingTaskRepo)(nil)

// Create inserts a new task row.
func (r *EmbeddingTaskRepo) Create(ctx context.Context, t *entity.EmbeddingTask) error {
	if err := txFrom(ctx, r.db).Create(t).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create embedding task", err)
	}
	return nil
}

// Update saves changes to an existing task row.
func (r *EmbeddingTaskRepo) Update(ctx context.Context, t *entity.EmbeddingTask) error {
	res := txFrom(ctx, r.db).Save(t)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update embedding task", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "embedding task not found")
	}
	return nil
}

// FindByID fetches a single task by id.
func (r *EmbeddingTaskRepo) FindByID(ctx context.Context, id int64) (*entity.EmbeddingTask, error) {
	var t entity.EmbeddingTask
	err := txFrom(ctx, r.db).First(&t, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find embedding task", err)
	}
	return &t, nil
}

// FindByDocumentID returns all tasks for a document.
func (r *EmbeddingTaskRepo) FindByDocumentID(ctx context.Context, documentID int64) ([]entity.EmbeddingTask, error) {
	var items []entity.EmbeddingTask
	if err := txFrom(ctx, r.db).Where("document_id = ?", documentID).Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, errno.Wrap(errno.InternalError, "find embedding tasks by document", err)
	}
	return items, nil
}

// List returns a paginated list ordered by created_at DESC.
func (r *EmbeddingTaskRepo) List(ctx context.Context, p pagination.Params) ([]entity.EmbeddingTask, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.EmbeddingTask
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.EmbeddingTask{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count embedding tasks", err)
	}
	if err := db.Order("created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list embedding tasks", err)
	}
	return items, int(total), nil
}

// ListByStatus filters tasks by status.
func (r *EmbeddingTaskRepo) ListByStatus(ctx context.Context, status string, p pagination.Params) ([]entity.EmbeddingTask, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.EmbeddingTask
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.EmbeddingTask{}).Where("status = ?", status)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count embedding tasks by status", err)
	}
	if err := db.Order("created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list embedding tasks by status", err)
	}
	return items, int(total), nil
}
