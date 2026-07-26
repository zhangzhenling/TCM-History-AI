package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/history-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// DiseaseRepo implements repository.DiseaseRepository with GORM.
type DiseaseRepo struct {
	baseRepo
}

// NewDiseaseRepo constructs a DiseaseRepo.
func NewDiseaseRepo(db *gorm.DB) *DiseaseRepo {
	return &DiseaseRepo{baseRepo{db: db}}
}

var _ repository.DiseaseRepository = (*DiseaseRepo)(nil)

// Create inserts a new disease row.
func (r *DiseaseRepo) Create(ctx context.Context, d *entity.Disease) error {
	if err := txFrom(ctx, r.db).Create(d).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create disease", err)
	}
	return nil
}

// Update saves changes to an existing disease row.
func (r *DiseaseRepo) Update(ctx context.Context, d *entity.Disease) error {
	res := txFrom(ctx, r.db).Save(d)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update disease", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "disease not found")
	}
	return nil
}

// Delete soft-deletes a disease by id.
func (r *DiseaseRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Where("id = ?", id).Delete(&entity.Disease{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete disease", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "disease not found")
	}
	return nil
}

// FindByID fetches a single disease by id.
func (r *DiseaseRepo) FindByID(ctx context.Context, id int64) (*entity.Disease, error) {
	var d entity.Disease
	err := txFrom(ctx, r.db).First(&d, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find disease", err)
	}
	return &d, nil
}

// List returns a paginated list of disease rows.
func (r *DiseaseRepo) List(ctx context.Context, p pagination.Params) ([]entity.Disease, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.Disease
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Disease{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count disease", err)
	}
	if err := db.Order("id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list disease", err)
	}
	return items, int(total), nil
}

// Search keyword-matches disease rows on name, pinyin, symptoms, description.
func (r *DiseaseRepo) Search(ctx context.Context, keyword string, p pagination.Params) ([]entity.Disease, int, error) {
	_, pageSize, offset := p.Normalise()
	pattern := "%" + keyword + "%"
	var items []entity.Disease
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Disease{}).
		Where("name ILIKE ? OR pinyin ILIKE ? OR symptoms ILIKE ? OR description ILIKE ?",
			pattern, pattern, pattern, pattern)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count disease search", err)
	}
	if err := db.Order("id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "search disease", err)
	}
	return items, int(total), nil
}
