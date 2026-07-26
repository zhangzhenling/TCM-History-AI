package persistence

import (
	"context"

	"gorm.io/gorm"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/history-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
)

// MedicinePrescriptionRepo implements repository.MedicinePrescriptionRepository.
type MedicinePrescriptionRepo struct {
	baseRepo
}

// NewMedicinePrescriptionRepo constructs a MedicinePrescriptionRepo.
func NewMedicinePrescriptionRepo(db *gorm.DB) *MedicinePrescriptionRepo {
	return &MedicinePrescriptionRepo{baseRepo{db: db}}
}

var _ repository.MedicinePrescriptionRepository = (*MedicinePrescriptionRepo)(nil)

// AddRelation inserts a new medicine_prescription row.
func (r *MedicinePrescriptionRepo) AddRelation(ctx context.Context, rel *entity.MedicinePrescription) error {
	if rel.ID == 0 {
		rel.ID = idgen.Next()
	}
	if err := txFrom(ctx, r.db).Create(rel).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create medicine_prescription", err)
	}
	return nil
}

// RemoveRelation deletes a medicine_prescription row by (prescription_id, medicine_id).
func (r *MedicinePrescriptionRepo) RemoveRelation(ctx context.Context, prescriptionID, medicineID int64) error {
	res := txFrom(ctx, r.db).Where("prescription_id = ? AND medicine_id = ?", prescriptionID, medicineID).
		Delete(&entity.MedicinePrescription{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete medicine_prescription", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "medicine_prescription relation not found")
	}
	return nil
}

// ListByPrescription returns all medicines in the given prescription.
func (r *MedicinePrescriptionRepo) ListByPrescription(ctx context.Context, prescriptionID int64) ([]entity.MedicinePrescription, error) {
	var items []entity.MedicinePrescription
	if err := txFrom(ctx, r.db).Where("prescription_id = ?", prescriptionID).
		Order("sort_order ASC, id ASC").Find(&items).Error; err != nil {
		return nil, errno.Wrap(errno.InternalError, "list medicine_prescription by prescription", err)
	}
	return items, nil
}

// ListByMedicine returns all prescriptions containing the given medicine.
func (r *MedicinePrescriptionRepo) ListByMedicine(ctx context.Context, medicineID int64) ([]entity.MedicinePrescription, error) {
	var items []entity.MedicinePrescription
	if err := txFrom(ctx, r.db).Where("medicine_id = ?", medicineID).Order("id ASC").Find(&items).Error; err != nil {
		return nil, errno.Wrap(errno.InternalError, "list medicine_prescription by medicine", err)
	}
	return items, nil
}
