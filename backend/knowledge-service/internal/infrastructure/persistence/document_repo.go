package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
	"tcm-history-ai/backend/knowledge-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// DocumentRepo implements repository.DocumentRepository with GORM.
type DocumentRepo struct {
	baseRepo
}

// NewDocumentRepo constructs a DocumentRepo.
func NewDocumentRepo(db *gorm.DB) *DocumentRepo {
	return &DocumentRepo{baseRepo{db: db}}
}

// Ensure DocumentRepo satisfies the interface at compile time.
var _ repository.DocumentRepository = (*DocumentRepo)(nil)

// Create inserts a new document row.
func (r *DocumentRepo) Create(ctx context.Context, d *entity.Document) error {
	if err := txFrom(ctx, r.db).Create(d).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create document", err)
	}
	return nil
}

// Update saves changes to an existing document row.
func (r *DocumentRepo) Update(ctx context.Context, d *entity.Document) error {
	res := txFrom(ctx, r.db).Save(d)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update document", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "document not found")
	}
	return nil
}

// Delete soft-deletes a document by id.
func (r *DocumentRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Where("id = ?", id).Delete(&entity.Document{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete document", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "document not found")
	}
	return nil
}

// FindByID fetches a single document by id; returns (nil, nil) when not found.
func (r *DocumentRepo) FindByID(ctx context.Context, id int64) (*entity.Document, error) {
	var d entity.Document
	err := txFrom(ctx, r.db).First(&d, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find document", err)
	}
	return &d, nil
}

// FindByContentHash looks up a document by its content hash for dedup.
func (r *DocumentRepo) FindByContentHash(ctx context.Context, hash string) (*entity.Document, error) {
	if hash == "" {
		return nil, nil
	}
	var d entity.Document
	err := txFrom(ctx, r.db).Where("content_hash = ?", hash).First(&d).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find document by hash", err)
	}
	return &d, nil
}

// List returns a paginated list ordered by created_at DESC.
func (r *DocumentRepo) List(ctx context.Context, p pagination.Params) ([]entity.Document, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.Document
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Document{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count documents", err)
	}
	if err := db.Order("created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list documents", err)
	}
	return items, int(total), nil
}

// ListByClassic filters documents by classic_code.
func (r *DocumentRepo) ListByClassic(ctx context.Context, classicCode string, p pagination.Params) ([]entity.Document, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.Document
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Document{}).Where("classic_code = ?", classicCode)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count documents by classic", err)
	}
	if err := db.Order("created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list documents by classic", err)
	}
	return items, int(total), nil
}
