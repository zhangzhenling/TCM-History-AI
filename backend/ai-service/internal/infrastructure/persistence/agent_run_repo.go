package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/ai-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// AgentRunRepo implements repository.AgentRunRepository with GORM.
type AgentRunRepo struct {
	baseRepo
}

// NewAgentRunRepo constructs an AgentRunRepo.
func NewAgentRunRepo(db *gorm.DB) *AgentRunRepo {
	return &AgentRunRepo{baseRepo{db: db}}
}

// Ensure AgentRunRepo satisfies the interface at compile time.
var _ repository.AgentRunRepository = (*AgentRunRepo)(nil)

// Create inserts a new agent run row.
func (r *AgentRunRepo) Create(ctx context.Context, a *entity.AgentRun) error {
	if err := txFrom(ctx, r.db).Create(a).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create agent run", err)
	}
	return nil
}

// Update saves changes to an existing agent run row.
func (r *AgentRunRepo) Update(ctx context.Context, a *entity.AgentRun) error {
	res := txFrom(ctx, r.db).Save(a)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update agent run", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "agent run not found")
	}
	return nil
}

// FindByID fetches a single agent run by id.
func (r *AgentRunRepo) FindByID(ctx context.Context, id int64) (*entity.AgentRun, error) {
	var a entity.AgentRun
	err := txFrom(ctx, r.db).First(&a, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find agent run", err)
	}
	return &a, nil
}

// ListByConversation returns paginated agent runs for a conversation.
func (r *AgentRunRepo) ListByConversation(ctx context.Context, conversationID int64, p pagination.Params) ([]entity.AgentRun, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.AgentRun
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.AgentRun{}).Where("conversation_id = ?", conversationID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count agent runs", err)
	}
	if err := db.Order("id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list agent runs", err)
	}
	return items, int(total), nil
}

// List returns paginated agent runs.
func (r *AgentRunRepo) List(ctx context.Context, p pagination.Params) ([]entity.AgentRun, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.AgentRun
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.AgentRun{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count agent runs", err)
	}
	if err := db.Order("id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list agent runs", err)
	}
	return items, int(total), nil
}
