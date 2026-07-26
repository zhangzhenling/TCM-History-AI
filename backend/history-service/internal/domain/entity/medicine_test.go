package entity_test

import (
	"testing"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
)

// TestMedicineTableName verifies the GORM table override.
func TestMedicineTableName(t *testing.T) {
	if got, want := (entity.Medicine{}).TableName(), "medicine"; got != want {
		t.Errorf("Medicine.TableName() = %q; want %q", got, want)
	}
}

// TestMedicineNatureConstants asserts the wire values for the four-nature
// (四气) constants.
func TestMedicineNatureConstants(t *testing.T) {
	want := map[string]string{
		"cold": entity.MedicineNatureCold,
		"hot":  entity.MedicineNatureHot,
		"warm": entity.MedicineNatureWarm,
		"cool": entity.MedicineNatureCool,
		"flat": entity.MedicineNatureFlat,
	}
	for expected, got := range want {
		if expected != got {
			t.Errorf("nature constant mismatch: want=%s got=%s", expected, got)
		}
	}
	if len(entity.ValidMedicineNatures) != len(want) {
		t.Errorf("ValidMedicineNatures len = %d; want %d",
			len(entity.ValidMedicineNatures), len(want))
	}
}

// TestMedicineToxicityConstants asserts the wire values for the toxicity
// level constants.
func TestMedicineToxicityConstants(t *testing.T) {
	want := map[string]string{
		"none":     entity.MedicineToxicityNone,
		"mild":     entity.MedicineToxicityMild,
		"moderate": entity.MedicineToxicityModerate,
		"severe":   entity.MedicineToxicitySevere,
	}
	for expected, got := range want {
		if expected != got {
			t.Errorf("toxicity constant mismatch: want=%s got=%s", expected, got)
		}
	}
}

// TestIsValidMedicineNature covers the validator for both valid and invalid
// inputs.
func TestIsValidMedicineNature(t *testing.T) {
	for _, v := range entity.ValidMedicineNatures {
		if !entity.IsValidMedicineNature(v) {
			t.Errorf("IsValidMedicineNature(%q) = false; want true", v)
		}
	}
	if entity.IsValidMedicineNature("") {
		t.Error("IsValidMedicineNature(\"\") = true; want false")
	}
	if entity.IsValidMedicineNature("frozen") {
		t.Error("IsValidMedicineNature(\"frozen\") = true; want false")
	}
}

// TestMedicineFields exercises struct field assignment.
func TestMedicineFields(t *testing.T) {
	m := entity.Medicine{
		Name:      "Gui Zhi",
		Pinyin:    "guizhi",
		AliasJSON: []string{"Cinnamon Twig"},
		Nature:    entity.MedicineNatureWarm,
		Flavor:    "sweet, pungent",
		Meridian:  "heart, lung, bladder",
		Efficacy:  "releases exterior, warms channels",
		Dosage:    "3-9g",
		Toxicity:  entity.MedicineToxicityNone,
	}
	if m.Name != "Gui Zhi" {
		t.Errorf("Name = %q", m.Name)
	}
	if m.Nature != entity.MedicineNatureWarm {
		t.Errorf("Nature = %q", m.Nature)
	}
	if len(m.AliasJSON) != 1 {
		t.Errorf("AliasJSON len = %d; want 1", len(m.AliasJSON))
	}
}
