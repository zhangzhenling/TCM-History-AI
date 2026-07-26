package repository

import (
	"context"

	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// WrongQuestionRepository is the port for wrong questions persistence.
type WrongQuestionRepository interface {
	Create(ctx context.Context, w *entity.WrongQuestion) error
	Update(ctx context.Context, w *entity.WrongQuestion) error
	FindByID(ctx context.Context, id int64) (*entity.WrongQuestion, error)
	FindByUserAndQuestion(ctx context.Context, userID, questionID int64) (*entity.WrongQuestion, error)
	ListByUser(ctx context.Context, userID int64, p pagination.Params) ([]entity.WrongQuestion, int, error)
	ListByExam(ctx context.Context, userID, examID int64, p pagination.Params) ([]entity.WrongQuestion, int, error)
	MarkMastered(ctx context.Context, id int64) error
}
