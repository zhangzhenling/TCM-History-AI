package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
	"tcm-history-ai/backend/knowledge-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// RagQueryRepo implements repository.RagQueryRepository with GORM.
type RagQueryRepo struct {
	baseRepo
}

// NewRagQueryRepo constructs a RagQueryRepo.
func NewRagQueryRepo(db *gorm.DB) *RagQueryRepo {
	return &RagQueryRepo{baseRepo{db: db}}
}

// Ensure RagQueryRepo satisfies the interface at compile time.
var _ repository.RagQueryRepository = (*RagQueryRepo)(nil)

// Create inserts a new rag_query row.
func (r *RagQueryRepo) Create(ctx context.Context, q *entity.RagQuery) error {
	if err := txFrom(ctx, r.db).Create(q).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create rag query", err)
	}
	return nil
}

// Update saves changes to an existing rag_query row.
func (r *RagQueryRepo) Update(ctx context.Context, q *entity.RagQuery) error {
	res := txFrom(ctx, r.db).Save(q)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update rag query", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "rag query not found")
	}
	return nil
}

// FindByID fetches a single rag_query by id.
func (r *RagQueryRepo) FindByID(ctx context.Context, id int64) (*entity.RagQuery, error) {
	var q entity.RagQuery
	err := txFrom(ctx, r.db).First(&q, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find rag query", err)
	}
	return &q, nil
}

// ListByUser returns paginated rag_queries for a user.
func (r *RagQueryRepo) ListByUser(ctx context.Context, userID int64, p pagination.Params) ([]entity.RagQuery, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.RagQuery
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.RagQuery{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count rag queries by user", err)
	}
	if err := db.Order("created_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list rag queries by user", err)
	}
	return items, int(total), nil
}

// ListBySession returns paginated rag_queries for a session.
func (r *RagQueryRepo) ListBySession(ctx context.Context, sessionID string, p pagination.Params) ([]entity.RagQuery, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.RagQuery
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.RagQuery{}).Where("session_id = ?", sessionID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count rag queries by session", err)
	}
	if err := db.Order("created_at ASC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list rag queries by session", err)
	}
	return items, int(total), nil
}
