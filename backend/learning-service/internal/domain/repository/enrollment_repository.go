package repository

import (
	"context"

	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// EnrollmentRepository is the port for enrollments persistence.
type EnrollmentRepository interface {
	Create(ctx context.Context, e *entity.Enrollment) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*entity.Enrollment, error)
	FindByUserAndCourse(ctx context.Context, userID, courseID int64) (*entity.Enrollment, error)
	ListByUser(ctx context.Context, userID int64, p pagination.Params) ([]entity.Enrollment, int, error)
	UpdateProgress(ctx context.Context, id, lastLessonID int64, progressPercent int) error
	MarkCompleted(ctx context.Context, id int64) error
}
