package repository

import (
	"context"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
)

// PrescriptionDiseaseRepository is the port for the prescription_disease junction table.
type PrescriptionDiseaseRepository interface {
	AddRelation(ctx context.Context, rel *entity.PrescriptionDisease) error
	RemoveRelation(ctx context.Context, prescriptionID, diseaseID int64) error
	ListByPrescription(ctx context.Context, prescriptionID int64) ([]entity.PrescriptionDisease, error)
	ListByDisease(ctx context.Context, diseaseID int64) ([]entity.PrescriptionDisease, error)
}
