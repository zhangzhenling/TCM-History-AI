package entity_test

import (
	"testing"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
)

// TestDiseaseTableName verifies the GORM table override.
func TestDiseaseTableName(t *testing.T) {
	if got, want := (entity.Disease{}).TableName(), "disease"; got != want {
		t.Errorf("Disease.TableName() = %q; want %q", got, want)
	}
}

// TestDiseaseCategoryConstants asserts the wire values for the disease
// category constants used as enum strings.
func TestDiseaseCategoryConstants(t *testing.T) {
	want := map[string]string{
		"external_contraction": entity.DiseaseCategoryExternalContraction,
		"internal_injury":      entity.DiseaseCategoryInternalInjury,
		"miscellaneous":        entity.DiseaseCategoryMiscellaneous,
	}
	for expected, got := range want {
		if expected != got {
			t.Errorf("category constant mismatch: want=%s got=%s", expected, got)
		}
	}
}

// TestDiseaseFields exercises struct field assignment.
func TestDiseaseFields(t *testing.T) {
	d := entity.Disease{
		Name:            "Cold Damage",
		Pinyin:          "shanghan",
		Category:        entity.DiseaseCategoryExternalContraction,
		Description:     "Exterior-releasing disorder",
		Symptoms:        "fever, chills",
		TCMPathogenesis: "wind-cold invasion",
	}
	if d.Name != "Cold Damage" {
		t.Errorf("Name = %q", d.Name)
	}
	if d.Category != entity.DiseaseCategoryExternalContraction {
		t.Errorf("Category = %q", d.Category)
	}
	if d.TCMPathogenesis != "wind-cold invasion" {
		t.Errorf("TCMPathogenesis = %q", d.TCMPathogenesis)
	}
}
