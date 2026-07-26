package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/ai-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// MessageRepo implements repository.MessageRepository with GORM.
type MessageRepo struct {
	baseRepo
}

// NewMessageRepo constructs a MessageRepo.
func NewMessageRepo(db *gorm.DB) *MessageRepo {
	return &MessageRepo{baseRepo{db: db}}
}

// Ensure MessageRepo satisfies the interface at compile time.
var _ repository.MessageRepository = (*MessageRepo)(nil)

// Create inserts a new message row.
func (r *MessageRepo) Create(ctx context.Context, m *entity.Message) error {
	if err := txFrom(ctx, r.db).Create(m).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create message", err)
	}
	return nil
}

// FindByConversation returns all messages for a conversation ordered by id ASC.
func (r *MessageRepo) FindByConversation(ctx context.Context, conversationID int64) ([]entity.Message, error) {
	var items []entity.Message
	if err := txFrom(ctx, r.db).Where("conversation_id = ?", conversationID).Order("id ASC").Find(&items).Error; err != nil {
		return nil, errno.Wrap(errno.InternalError, "find messages", err)
	}
	return items, nil
}

// ListByConversation returns paginated messages for a conversation.
func (r *MessageRepo) ListByConversation(ctx context.Context, conversationID int64, p pagination.Params) ([]entity.Message, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.Message
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Message{}).Where("conversation_id = ?", conversationID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count messages", err)
	}
	if err := db.Order("id ASC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list messages", err)
	}
	return items, int(total), nil
}
