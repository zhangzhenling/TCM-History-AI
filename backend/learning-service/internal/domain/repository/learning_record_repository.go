package repository

import (
	"context"

	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// LearningRecordRepository is the port for learning records persistence.
type LearningRecordRepository interface {
	Upsert(ctx context.Context, r *entity.LearningRecord) error
	FindByID(ctx context.Context, id int64) (*entity.LearningRecord, error)
	FindByUserAndLesson(ctx context.Context, userID, lessonID int64) (*entity.LearningRecord, error)
	ListByUser(ctx context.Context, userID int64, p pagination.Params) ([]entity.LearningRecord, int, error)
	ListByUserAndCourse(ctx context.Context, userID, courseID int64, p pagination.Params) ([]entity.LearningRecord, int, error)
	MarkCompleted(ctx context.Context, id int64) error
}
