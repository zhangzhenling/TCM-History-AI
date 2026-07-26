package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/history-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// MedicineRepo implements repository.MedicineRepository with GORM.
type MedicineRepo struct {
	baseRepo
}

// NewMedicineRepo constructs a MedicineRepo.
func NewMedicineRepo(db *gorm.DB) *MedicineRepo {
	return &MedicineRepo{baseRepo{db: db}}
}

var _ repository.MedicineRepository = (*MedicineRepo)(nil)

// Create inserts a new medicine row.
func (r *MedicineRepo) Create(ctx context.Context, m *entity.Medicine) error {
	if err := txFrom(ctx, r.db).Create(m).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create medicine", err)
	}
	return nil
}

// Update saves changes to an existing medicine row.
func (r *MedicineRepo) Update(ctx context.Context, m *entity.Medicine) error {
	res := txFrom(ctx, r.db).Save(m)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update medicine", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "medicine not found")
	}
	return nil
}

// Delete soft-deletes a medicine by id.
func (r *MedicineRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Where("id = ?", id).Delete(&entity.Medicine{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete medicine", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "medicine not found")
	}
	return nil
}

// FindByID fetches a single medicine by id.
func (r *MedicineRepo) FindByID(ctx context.Context, id int64) (*entity.Medicine, error) {
	var m entity.Medicine
	err := txFrom(ctx, r.db).First(&m, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find medicine", err)
	}
	return &m, nil
}

// List returns a paginated list of medicine rows.
func (r *MedicineRepo) List(ctx context.Context, p pagination.Params) ([]entity.Medicine, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.Medicine
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Medicine{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count medicine", err)
	}
	if err := db.Order("id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list medicine", err)
	}
	return items, int(total), nil
}

// Search keyword-matches medicine rows on name, pinyin, efficacy, meridian.
func (r *MedicineRepo) Search(ctx context.Context, keyword string, p pagination.Params) ([]entity.Medicine, int, error) {
	_, pageSize, offset := p.Normalise()
	pattern := "%" + keyword + "%"
	var items []entity.Medicine
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Medicine{}).
		Where("name ILIKE ? OR pinyin ILIKE ? OR efficacy ILIKE ? OR meridian ILIKE ?",
			pattern, pattern, pattern, pattern)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count medicine search", err)
	}
	if err := db.Order("id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "search medicine", err)
	}
	return items, int(total), nil
}
