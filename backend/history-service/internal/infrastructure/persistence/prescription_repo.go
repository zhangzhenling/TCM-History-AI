package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/history-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// PrescriptionRepo implements repository.PrescriptionRepository with GORM.
type PrescriptionRepo struct {
	baseRepo
}

// NewPrescriptionRepo constructs a PrescriptionRepo.
func NewPrescriptionRepo(db *gorm.DB) *PrescriptionRepo {
	return &PrescriptionRepo{baseRepo{db: db}}
}

var _ repository.PrescriptionRepository = (*PrescriptionRepo)(nil)

// Create inserts a new prescription row.
func (r *PrescriptionRepo) Create(ctx context.Context, p *entity.Prescription) error {
	if err := txFrom(ctx, r.db).Create(p).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create prescription", err)
	}
	return nil
}

// Update saves changes to an existing prescription row.
func (r *PrescriptionRepo) Update(ctx context.Context, p *entity.Prescription) error {
	res := txFrom(ctx, r.db).Save(p)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update prescription", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "prescription not found")
	}
	return nil
}

// Delete soft-deletes a prescription by id.
func (r *PrescriptionRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Where("id = ?", id).Delete(&entity.Prescription{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete prescription", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "prescription not found")
	}
	return nil
}

// FindByID fetches a single prescription by id.
func (r *PrescriptionRepo) FindByID(ctx context.Context, id int64) (*entity.Prescription, error) {
	var p entity.Prescription
	err := txFrom(ctx, r.db).First(&p, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find prescription", err)
	}
	return &p, nil
}

// List returns a paginated list of prescription rows.
func (r *PrescriptionRepo) List(ctx context.Context, p pagination.Params) ([]entity.Prescription, int, error) {
	_, pageSize, offset := p.Normalise()
	var items []entity.Prescription
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Prescription{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count prescription", err)
	}
	if err := db.Order("id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list prescription", err)
	}
	return items, int(total), nil
}

// Search keyword-matches prescription rows on name, pinyin, composition, indications.
func (r *PrescriptionRepo) Search(ctx context.Context, keyword string, p pagination.Params) ([]entity.Prescription, int, error) {
	_, pageSize, offset := p.Normalise()
	pattern := "%" + keyword + "%"
	var items []entity.Prescription
	var total int64
	db := txFrom(ctx, r.db).Model(&entity.Prescription{}).
		Where("name ILIKE ? OR pinyin ILIKE ? OR composition ILIKE ? OR indications ILIKE ?",
			pattern, pattern, pattern, pattern)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count prescription search", err)
	}
	if err := db.Order("id DESC").Limit(pageSize).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "search prescription", err)
	}
	return items, int(total), nil
}
