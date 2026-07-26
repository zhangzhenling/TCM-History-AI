package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/history-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
)

// BookAuthorRepo implements repository.BookAuthorRepository with GORM.
type BookAuthorRepo struct {
	baseRepo
}

// NewBookAuthorRepo constructs a BookAuthorRepo.
func NewBookAuthorRepo(db *gorm.DB) *BookAuthorRepo {
	return &BookAuthorRepo{baseRepo{db: db}}
}

var _ repository.BookAuthorRepository = (*BookAuthorRepo)(nil)

// AddRelation inserts a new book_author row.
func (r *BookAuthorRepo) AddRelation(ctx context.Context, rel *entity.BookAuthor) error {
	if rel.ID == 0 {
		rel.ID = idgen.Next()
	}
	if err := txFrom(ctx, r.db).Create(rel).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create book_author", err)
	}
	return nil
}

// RemoveRelation deletes a book_author row by (book_id, person_id).
func (r *BookAuthorRepo) RemoveRelation(ctx context.Context, bookID, personID int64) error {
	res := txFrom(ctx, r.db).Where("book_id = ? AND person_id = ?", bookID, personID).
		Delete(&entity.BookAuthor{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete book_author", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "book_author relation not found")
	}
	return nil
}

// ListByBook returns all authors of the given book.
func (r *BookAuthorRepo) ListByBook(ctx context.Context, bookID int64) ([]entity.BookAuthor, error) {
	var items []entity.BookAuthor
	if err := txFrom(ctx, r.db).Where("book_id = ?", bookID).Order("sort_order ASC, id ASC").Find(&items).Error; err != nil {
		return nil, errno.Wrap(errno.InternalError, "list book_author by book", err)
	}
	return items, nil
}

// ListByPerson returns all books authored by the given person.
func (r *BookAuthorRepo) ListByPerson(ctx context.Context, personID int64) ([]entity.BookAuthor, error) {
	var items []entity.BookAuthor
	if err := txFrom(ctx, r.db).Where("person_id = ?", personID).Order("id ASC").Find(&items).Error; err != nil {
		return nil, errno.Wrap(errno.InternalError, "list book_author by person", err)
	}
	return items, nil
}
