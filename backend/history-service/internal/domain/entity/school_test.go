package entity_test

import (
	"testing"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
)

// TestSchoolTableName verifies the GORM table override.
func TestSchoolTableName(t *testing.T) {
	if got, want := (entity.School{}).TableName(), "history_school"; got != want {
		t.Errorf("School.TableName() = %q; want %q", got, want)
	}
}

// TestSchoolFields exercises struct field assignment.
func TestSchoolFields(t *testing.T) {
	s := entity.School{
		Name:            "Yishui School",
		DynastyID:       1,
		FounderPersonID: 2,
		Summary:         "Cold and cooling school",
		EstablishedYear: 1200,
	}
	if s.Name != "Yishui School" {
		t.Errorf("Name = %q", s.Name)
	}
	if s.FounderPersonID != 2 {
		t.Errorf("FounderPersonID = %d; want 2", s.FounderPersonID)
	}
	if s.EstablishedYear != 1200 {
		t.Errorf("EstablishedYear = %d; want 1200", s.EstablishedYear)
	}
}
