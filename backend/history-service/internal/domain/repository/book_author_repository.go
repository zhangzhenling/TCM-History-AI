package repository

import (
	"context"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
)

// BookAuthorRepository is the port for the book_author junction table.
type BookAuthorRepository interface {
	AddRelation(ctx context.Context, rel *entity.BookAuthor) error
	RemoveRelation(ctx context.Context, bookID, personID int64) error
	ListByBook(ctx context.Context, bookID int64) ([]entity.BookAuthor, error)
	ListByPerson(ctx context.Context, personID int64) ([]entity.BookAuthor, error)
}
