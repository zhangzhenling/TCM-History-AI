package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/history-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// PersonRepo implements repository.PersonRepository with GORM.
type PersonRepo struct {
	baseRepo
}

// NewPersonRepo constructs a PersonRepo.
func NewPersonRepo(db *gorm.DB) *PersonRepo {
	return &PersonRepo{baseRepo{db: db}}
}

var _ repository.PersonRepository = (*PersonRepo)(nil)

// Create inserts a new history_person row.
func (r *PersonRepo) Create(ctx context.Context, p *entity.Person) error {
	if err := txFrom(ctx, r.db).Create(p).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create person", err)
	}
	return nil
}

// Update saves changes to an existing history_person row.
func (r *PersonRepo) Update(ctx context.Context, p *entity.Person) error {
	res := txFrom(ctx, r.db).Save(p)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update person", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "person not found")
	}
	return nil
}

// Delete soft-deletes a history_person by id.
func (r *PersonRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Where("id = ?", id).Delete(&entity.Person{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete person", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "person not found")
	}
	return nil
}

// FindByID fetches a single history_person by id.
func (r *PersonRepo) FindByID(ctx context.Context, id int64) (*entity.Person, error) {
	var p entity.Person
	err := txFrom(ctx, r.db).First(&p, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find person", err)
	}
	return &p, nil
}

// List returns a paginated list of history_person rows.
func (r *PersonRepo) List(ctx context.Context, p pagination.Params) ([]entity.Person, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.Person
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Person{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count person", err)
	}
	if err := db.Order("id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list person", err)
	}
	return items, int(total), nil
}

// Search keyword-matches history_person rows on name, alias_name, title.
func (r *PersonRepo) Search(ctx context.Context, keyword string, p pagination.Params) ([]entity.Person, int, error) {
	_, pageSize, offset := p.Normalise()
	pattern := "%" + keyword + "%"
	var items []entity.Person
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Person{}).
		Where("name ILIKE ? OR alias_name ILIKE ? OR courtesy_name ILIKE ? OR title ILIKE ?",
			pattern, pattern, pattern, pattern)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count person search", err)
	}
	if err := db.Order("id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "search person", err)
	}
	return items, int(total), nil
}
