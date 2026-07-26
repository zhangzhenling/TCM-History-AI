// Package repository defines the domain repository interfaces (ports) for
// Learning Service. Each entity has its own interface file; infrastructure/
// persistence provides the GORM adapters.
package repository

import (
	"context"

	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// CourseRepository is the port for courses persistence.
type CourseRepository interface {
	Create(ctx context.Context, c *entity.Course) error
	Update(ctx context.Context, c *entity.Course) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*entity.Course, error)
	List(ctx context.Context, p pagination.Params) ([]entity.Course, int, error)
	ListByCategory(ctx context.Context, category string, p pagination.Params) ([]entity.Course, int, error)
	ListPublished(ctx context.Context, p pagination.Params) ([]entity.Course, int, error)
}
