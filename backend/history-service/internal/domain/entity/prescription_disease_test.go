package entity_test

import (
	"testing"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
)

// TestPrescriptionDiseaseTableName verifies the GORM table override for the
// junction between prescription and disease.
func TestPrescriptionDiseaseTableName(t *testing.T) {
	if got, want := (entity.PrescriptionDisease{}).TableName(), "prescription_disease"; got != want {
		t.Errorf("PrescriptionDisease.TableName() = %q; want %q", got, want)
	}
}

// TestPrescriptionDiseaseFields exercises struct field assignment.
func TestPrescriptionDiseaseFields(t *testing.T) {
	pd := entity.PrescriptionDisease{
		PrescriptionID: 1,
		DiseaseID:      2,
		EfficacyNote:   "primary treatment",
		IsPrimary:      true,
	}
	if pd.PrescriptionID != 1 || pd.DiseaseID != 2 {
		t.Errorf("unexpected IDs: prescription=%d disease=%d",
			pd.PrescriptionID, pd.DiseaseID)
	}
	if !pd.IsPrimary {
		t.Error("IsPrimary should be true")
	}
	if pd.EfficacyNote != "primary treatment" {
		t.Errorf("EfficacyNote = %q; want \"primary treatment\"", pd.EfficacyNote)
	}
}
