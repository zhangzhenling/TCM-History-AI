package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/history-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
)

// PrescriptionDiseaseRepo implements repository.PrescriptionDiseaseRepository.
type PrescriptionDiseaseRepo struct {
	baseRepo
}

// NewPrescriptionDiseaseRepo constructs a PrescriptionDiseaseRepo.
func NewPrescriptionDiseaseRepo(db *gorm.DB) *PrescriptionDiseaseRepo {
	return &PrescriptionDiseaseRepo{baseRepo{db: db}}
}

var _ repository.PrescriptionDiseaseRepository = (*PrescriptionDiseaseRepo)(nil)

// AddRelation inserts a new prescription_disease row.
func (r *PrescriptionDiseaseRepo) AddRelation(ctx context.Context, rel *entity.PrescriptionDisease) error {
	if rel.ID == 0 {
		rel.ID = idgen.Next()
	}
	if err := txFrom(ctx, r.db).Create(rel).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create prescription_disease", err)
	}
	return nil
}

// RemoveRelation deletes a prescription_disease row by (prescription_id, disease_id).
func (r *PrescriptionDiseaseRepo) RemoveRelation(ctx context.Context, prescriptionID, diseaseID int64) error {
	res := txFrom(ctx, r.db).Where("prescription_id = ? AND disease_id = ?", prescriptionID, diseaseID).
		Delete(&entity.PrescriptionDisease{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete prescription_disease", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "prescription_disease relation not found")
	}
	return nil
}

// ListByPrescription returns all diseases treated by the given prescription.
func (r *PrescriptionDiseaseRepo) ListByPrescription(ctx context.Context, prescriptionID int64) ([]entity.PrescriptionDisease, error) {
	var items []entity.PrescriptionDisease
	if err := txFrom(ctx, r.db).Where("prescription_id = ?", prescriptionID).
		Order("is_primary DESC, id ASC").Find(&items).Error; err != nil {
		return nil, errno.Wrap(errno.InternalError, "list prescription_disease by prescription", err)
	}
	return items, nil
}

// ListByDisease returns all prescriptions treating the given disease.
func (r *PrescriptionDiseaseRepo) ListByDisease(ctx context.Context, diseaseID int64) ([]entity.PrescriptionDisease, error) {
	var items []entity.PrescriptionDisease
	if err := txFrom(ctx, r.db).Where("disease_id = ?", diseaseID).Order("id ASC").Find(&items).Error; err != nil {
		return nil, errno.Wrap(errno.InternalError, "list prescription_disease by disease", err)
	}
	return items, nil
}
