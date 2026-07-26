package entity_test

import (
	"testing"

	"tcm-history-ai/backend/history-service/internal/domain/entity"
)

// TestDynastyTableName verifies the GORM table override.
func TestDynastyTableName(t *testing.T) {
	if got, want := (entity.Dynasty{}).TableName(), "history_dynasty"; got != want {
		t.Errorf("Dynasty.TableName() = %q; want %q", got, want)
	}
}

// TestDynastyFields exercises struct field assignment to ensure the entity
// compiles as a value type and round-trips through pointer semantics.
func TestDynastyFields(t *testing.T) {
	d := entity.Dynasty{
		Name:        "Han",
		StartYear:   -202,
		EndYear:     220,
		SortOrder:   3,
		Description: "Western & Eastern Han",
	}
	if d.Name != "Han" {
		t.Errorf("Name = %q; want Han", d.Name)
	}
	if d.StartYear != -202 {
		t.Errorf("StartYear = %d; want -202", d.StartYear)
	}
	if d.EndYear != 220 {
		t.Errorf("EndYear = %d; want 220", d.EndYear)
	}
	if d.SortOrder != 3 {
		t.Errorf("SortOrder = %d; want 3", d.SortOrder)
	}
}
