package repository

import (
	"context"

	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// ExamRepository is the port for exams persistence.
type ExamRepository interface {
	Create(ctx context.Context, e *entity.Exam) error
	Update(ctx context.Context, e *entity.Exam) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*entity.Exam, error)
	List(ctx context.Context, p pagination.Params) ([]entity.Exam, int, error)
	ListByCourse(ctx context.Context, courseID int64, p pagination.Params) ([]entity.Exam, int, error)
	ListPublished(ctx context.Context, p pagination.Params) ([]entity.Exam, int, error)
	ListAllWithDuration(ctx context.Context) ([]entity.Exam, error)
}
