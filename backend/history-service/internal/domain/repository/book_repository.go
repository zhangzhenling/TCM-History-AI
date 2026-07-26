package repository

import (
	"context"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/pagination"
)

// BookRepository is the port for history_book persistence.
type BookRepository interface {
	Create(ctx context.Context, b *entity.Book) error
	Update(ctx context.Context, b *entity.Book) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*entity.Book, error)
	List(ctx context.Context, p pagination.Params) ([]entity.Book, int, error)
	Search(ctx context.Context, keyword string, p pagination.Params) ([]entity.Book, int, error)
}
