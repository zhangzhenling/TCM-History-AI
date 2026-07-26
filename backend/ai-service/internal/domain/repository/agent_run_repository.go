package repository

import (
	"context"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// AgentRunRepository is the port for ai_agent_runs persistence.
type AgentRunRepository interface {
	Create(ctx context.Context, a *entity.AgentRun) error
	Update(ctx context.Context, a *entity.AgentRun) error
	FindByID(ctx context.Context, id int64) (*entity.AgentRun, error)
	ListByConversation(ctx context.Context, conversationID int64, p pagination.Params) ([]entity.AgentRun, int, error)
	List(ctx context.Context, p pagination.Params) ([]entity.AgentRun, int, error)
}
