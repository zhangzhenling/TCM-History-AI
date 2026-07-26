package repository

import (
	"context"

	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// StudyPlanRepository is the port for study plans persistence.
type StudyPlanRepository interface {
	Create(ctx context.Context, s *entity.StudyPlan) error
	Update(ctx context.Context, s *entity.StudyPlan) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*entity.StudyPlan, error)
	ListByUser(ctx context.Context, userID int64, p pagination.Params) ([]entity.StudyPlan, int, error)
	FindActive(ctx context.Context, userID int64) ([]entity.StudyPlan, error)
}
