package repository

import (
	"context"

	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// LessonRepository is the port for lessons persistence.
type LessonRepository interface {
	Create(ctx context.Context, l *entity.Lesson) error
	Update(ctx context.Context, l *entity.Lesson) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*entity.Lesson, error)
	ListByCourse(ctx context.Context, courseID int64, p pagination.Params) ([]entity.Lesson, int, error)
	FindPublished(ctx context.Context, id int64) (*entity.Lesson, error)
	CountByCourse(ctx context.Context, courseID int64) (int, error)
	// UpdateCourseLessonCount recounts lessons under a course and writes the
	// result onto learning_courses.lesson_count.
	UpdateCourseLessonCount(ctx context.Context, courseID int64) error
}
