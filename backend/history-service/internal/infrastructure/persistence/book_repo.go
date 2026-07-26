package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/history-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// BookRepo implements repository.BookRepository with GORM.
type BookRepo struct {
	baseRepo
}

// NewBookRepo constructs a BookRepo.
func NewBookRepo(db *gorm.DB) *BookRepo {
	return &BookRepo{baseRepo{db: db}}
}

var _ repository.BookRepository = (*BookRepo)(nil)

// Create inserts a new history_book row.
func (r *BookRepo) Create(ctx context.Context, b *entity.Book) error {
	if err := txFrom(ctx, r.db).Create(b).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create book", err)
	}
	return nil
}

// Update saves changes to an existing history_book row.
func (r *BookRepo) Update(ctx context.Context, b *entity.Book) error {
	res := txFrom(ctx, r.db).Save(b)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update book", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "book not found")
	}
	return nil
}

// Delete soft-deletes a history_book by id.
func (r *BookRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Where("id = ?", id).Delete(&entity.Book{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete book", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "book not found")
	}
	return nil
}

// FindByID fetches a single history_book by id.
func (r *BookRepo) FindByID(ctx context.Context, id int64) (*entity.Book, error) {
	var b entity.Book
	err := txFrom(ctx, r.db).First(&b, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find book", err)
	}
	return &b, nil
}

// List returns a paginated list of history_book rows.
func (r *BookRepo) List(ctx context.Context, p pagination.Params) ([]entity.Book, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.Book
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Book{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count book", err)
	}
	if err := db.Order("id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list book", err)
	}
	return items, int(total), nil
}

// Search keyword-matches history_book rows on title and summary.
func (r *BookRepo) Search(ctx context.Context, keyword string, p pagination.Params) ([]entity.Book, int, error) {
	_, pageSize, offset := p.Normalise()
	pattern := "%" + keyword + "%"
	var items []entity.Book
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Book{}).
		Where("title ILIKE ? OR summary ILIKE ?", pattern, pattern)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count book search", err)
	}
	if err := db.Order("id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "search book", err)
	}
	return items, int(total), nil
}
