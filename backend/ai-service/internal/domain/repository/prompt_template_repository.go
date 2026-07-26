package repository

import (
	"context"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// PromptTemplateRepository is the port for ai_prompt_templates persistence.
type PromptTemplateRepository interface {
	Create(ctx context.Context, p *entity.PromptTemplate) error
	Update(ctx context.Context, p *entity.PromptTemplate) error
	FindByID(ctx context.Context, id int64) (*entity.PromptTemplate, error)
	FindByNameAndScene(ctx context.Context, name, scene string) (*entity.PromptTemplate, error)
	ListByScene(ctx context.Context, scene string, p pagination.Params) ([]entity.PromptTemplate, int, error)
	FindActive(ctx context.Context, scene string) (*entity.PromptTemplate, error)
	List(ctx context.Context, p pagination.Params) ([]entity.PromptTemplate, int, error)
}
