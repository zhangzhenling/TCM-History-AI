package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/history-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// SchoolRepo implements repository.SchoolRepository with GORM.
type SchoolRepo struct {
	baseRepo
}

// NewSchoolRepo constructs a SchoolRepo.
func NewSchoolRepo(db *gorm.DB) *SchoolRepo {
	return &SchoolRepo{baseRepo{db: db}}
}

var _ repository.SchoolRepository = (*SchoolRepo)(nil)

// Create inserts a new history_school row.
func (r *SchoolRepo) Create(ctx context.Context, s *entity.School) error {
	if err := txFrom(ctx, r.db).Create(s).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create school", err)
	}
	return nil
}

// Update saves changes to an existing history_school row.
func (r *SchoolRepo) Update(ctx context.Context, s *entity.School) error {
	res := txFrom(ctx, r.db).Save(s)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update school", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "school not found")
	}
	return nil
}

// Delete soft-deletes a history_school by id.
func (r *SchoolRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Where("id = ?", id).Delete(&entity.School{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete school", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "school not found")
	}
	return nil
}

// FindByID fetches a single history_school by id.
func (r *SchoolRepo) FindByID(ctx context.Context, id int64) (*entity.School, error) {
	var s entity.School
	err := txFrom(ctx, r.db).First(&s, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find school", err)
	}
	return &s, nil
}

// List returns a paginated list of history_school rows.
func (r *SchoolRepo) List(ctx context.Context, p pagination.Params) ([]entity.School, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.School
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.School{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count school", err)
	}
	if err := db.Order("id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list school", err)
	}
	return items, int(total), nil
}

// Search keyword-matches history_school rows on name and summary.
func (r *SchoolRepo) Search(ctx context.Context, keyword string, p pagination.Params) ([]entity.School, int, error) {
	_, pageSize, offset := p.Normalise()
	pattern := "%" + keyword + "%"
	var items []entity.School
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.School{}).
		Where("name ILIKE ? OR summary ILIKE ?", pattern, pattern)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count school search", err)
	}
	if err := db.Order("id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "search school", err)
	}
	return items, int(total), nil
}
