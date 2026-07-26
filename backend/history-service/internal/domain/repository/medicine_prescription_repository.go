package repository

import (
	"context"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
)

// MedicinePrescriptionRepository is the port for the medicine_prescription junction table.
type MedicinePrescriptionRepository interface {
	AddRelation(ctx context.Context, rel *entity.MedicinePrescription) error
	RemoveRelation(ctx context.Context, prescriptionID, medicineID int64) error
	ListByPrescription(ctx context.Context, prescriptionID int64) ([]entity.MedicinePrescription, error)
	ListByMedicine(ctx context.Context, medicineID int64) ([]entity.MedicinePrescription, error)
}
