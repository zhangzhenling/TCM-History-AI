package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
	"tcm-history-ai/backend/knowledge-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// DocumentChunkRepo implements repository.DocumentChunkRepository with GORM.
type DocumentChunkRepo struct {
	baseRepo
}

// NewDocumentChunkRepo constructs a DocumentChunkRepo.
func NewDocumentChunkRepo(db *gorm.DB) *DocumentChunkRepo {
	return &DocumentChunkRepo{baseRepo{db: db}}
}

// Ensure DocumentChunkRepo satisfies the interface at compile time.
var _ repository.DocumentChunkRepository = (*DocumentChunkRepo)(nil)

// Create inserts a single chunk row.
func (r *DocumentChunkRepo) Create(ctx context.Context, c *entity.DocumentChunk) error {
	if err := txFrom(ctx, r.db).Create(c).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create chunk", err)
	}
	return nil
}

// BatchCreate inserts multiple chunk rows in a single statement.
func (r *DocumentChunkRepo) BatchCreate(ctx context.Context, chunks []entity.DocumentChunk) error {
	if len(chunks) == 0 {
		return nil
	}
	if err := txFrom(ctx, r.db).CreateInBatches(chunks, 200).Error; err != nil {
		return errno.Wrap(errno.InternalError, "batch create chunks", err)
	}
	return nil
}

// Update saves changes to an existing chunk row.
func (r *DocumentChunkRepo) Update(ctx context.Context, c *entity.DocumentChunk) error {
	res := txFrom(ctx, r.db).Save(c)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update chunk", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "chunk not found")
	}
	return nil
}

// DeleteByDocument removes all chunks belonging to a document.
func (r *DocumentChunkRepo) DeleteByDocument(ctx context.Context, documentID int64) error {
	res := txFrom(ctx, r.db).Where("document_id = ?", documentID).Delete(&entity.DocumentChunk{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete chunks by document", res.Error)
	}
	return nil
}

// FindByID fetches a single chunk by id; returns (nil, nil) when not found.
func (r *DocumentChunkRepo) FindByID(ctx context.Context, id int64) (*entity.DocumentChunk, error) {
	var c entity.DocumentChunk
	err := txFrom(ctx, r.db).First(&c, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find chunk", err)
	}
	return &c, nil
}

// FindByChunkID fetches a single chunk by its chunk_id (Milvus PK).
func (r *DocumentChunkRepo) FindByChunkID(ctx context.Context, chunkID string) (*entity.DocumentChunk, error) {
	var c entity.DocumentChunk
	err := txFrom(ctx, r.db).Where("chunk_id = ?", chunkID).First(&c).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find chunk by chunk_id", err)
	}
	return &c, nil
}

// ListByDocument returns paginated chunks for a document.
func (r *DocumentChunkRepo) ListByDocument(ctx context.Context, documentID int64, p pagination.Params) ([]entity.DocumentChunk, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.DocumentChunk
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.DocumentChunk{}).Where("document_id = ?", documentID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count chunks", err)
	}
	if err := db.Order("chunk_index ASC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list chunks", err)
	}
	return items, int(total), nil
}

// ListByIDs returns chunks matching any of the provided ids.
func (r *DocumentChunkRepo) ListByIDs(ctx context.Context, ids []int64) ([]entity.DocumentChunk, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var items []entity.DocumentChunk
	if err := txFrom(ctx, r.db).Where("id IN ?", ids).Order("chunk_index ASC").Find(&items).Error; err != nil {
		return nil, errno.Wrap(errno.InternalError, "list chunks by ids", err)
	}
	return items, nil
}
