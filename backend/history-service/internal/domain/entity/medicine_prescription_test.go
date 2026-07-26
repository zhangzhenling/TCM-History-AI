package entity_test

import (
	"testing"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
)

// TestMedicinePrescriptionTableName verifies the GORM table override for the
// junction between medicine and prescription.
func TestMedicinePrescriptionTableName(t *testing.T) {
	if got, want := (entity.MedicinePrescription{}).TableName(), "medicine_prescription"; got != want {
		t.Errorf("MedicinePrescription.TableName() = %q; want %q", got, want)
	}
}

// TestMedicinePrescriptionFields exercises struct field assignment and the
// 君臣佐使 role constants exposed by the prescription entity.
func TestMedicinePrescriptionFields(t *testing.T) {
	mp := entity.MedicinePrescription{
		PrescriptionID: 1,
		MedicineID:     2,
		Role:           entity.PrescriptionRoleJun,
		Dosage:         "9g",
		SortOrder:      0,
	}
	if mp.PrescriptionID != 1 || mp.MedicineID != 2 {
		t.Errorf("unexpected IDs: prescription=%d medicine=%d",
			mp.PrescriptionID, mp.MedicineID)
	}
	if mp.Role != entity.PrescriptionRoleJun {
		t.Errorf("Role = %q; want %q", mp.Role, entity.PrescriptionRoleJun)
	}
	if mp.Dosage != "9g" {
		t.Errorf("Dosage = %q; want 9g", mp.Dosage)
	}
}
