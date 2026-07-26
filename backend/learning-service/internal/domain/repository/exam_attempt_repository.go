package repository

import (
	"context"

	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// ExamAttemptRepository is the port for exam attempts persistence.
type ExamAttemptRepository interface {
	Create(ctx context.Context, a *entity.ExamAttempt) error
	Update(ctx context.Context, a *entity.ExamAttempt) error
	FindByID(ctx context.Context, id int64) (*entity.ExamAttempt, error)
	ListByUserAndExam(ctx context.Context, userID, examID int64, p pagination.Params) ([]entity.ExamAttempt, int, error)
	FindLatest(ctx context.Context, userID, examID int64) (*entity.ExamAttempt, error)
}
