package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/ai-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// ConversationRepo implements repository.ConversationRepository with GORM.
type ConversationRepo struct {
	baseRepo
}

// NewConversationRepo constructs a ConversationRepo.
func NewConversationRepo(db *gorm.DB) *ConversationRepo {
	return &ConversationRepo{baseRepo{db: db}}
}

// Ensure ConversationRepo satisfies the interface at compile time.
var _ repository.ConversationRepository = (*ConversationRepo)(nil)

// Create inserts a new conversation row.
func (r *ConversationRepo) Create(ctx context.Context, c *entity.Conversation) error {
	if err := txFrom(ctx, r.db).Create(c).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create conversation", err)
	}
	return nil
}

// Update saves changes to an existing conversation row.
func (r *ConversationRepo) Update(ctx context.Context, c *entity.Conversation) error {
	res := txFrom(ctx, r.db).Save(c)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update conversation", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "conversation not found")
	}
	return nil
}

// Delete soft-deletes a conversation by id.
func (r *ConversationRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Where("id = ?", id).Delete(&entity.Conversation{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete conversation", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "conversation not found")
	}
	return nil
}

// FindByID fetches a single conversation by id; returns (nil, nil) when not found.
func (r *ConversationRepo) FindByID(ctx context.Context, id int64) (*entity.Conversation, error) {
	var c entity.Conversation
	err := txFrom(ctx, r.db).First(&c, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find conversation", err)
	}
	return &c, nil
}

// ListByUser returns paginated conversations for a user, ordered by updated_at DESC.
func (r *ConversationRepo) ListByUser(ctx context.Context, userID int64, p pagination.Params) ([]entity.Conversation, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.Conversation
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Conversation{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count conversations", err)
	}
	if err := db.Order("updated_at DESC, id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list conversations", err)
	}
	return items, int(total), nil
}
