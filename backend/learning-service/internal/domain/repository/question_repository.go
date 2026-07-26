package repository

import (
	"context"

	"tcm-history-ai/backend/learning-service/internal/domain/entity"
)

// QuestionRepository is the port for questions persistence.
type QuestionRepository interface {
	Create(ctx context.Context, q *entity.Question) error
	Update(ctx context.Context, q *entity.Question) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*entity.Question, error)
	ListByExam(ctx context.Context, examID int64) ([]entity.Question, error)
	UpdateExamCount(ctx context.Context, examID int64) error
}
