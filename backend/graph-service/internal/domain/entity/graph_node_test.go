package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
)

// TestGraphNode_TableName verifies the GORM table name override.
func TestGraphNode_TableName(t *testing.T) {
	assert.Equal(t, "graph_nodes", entity.GraphNode{}.TableName())
}

// TestNodeLabels_ContainsAllKnownLabels pins the 8 known node labels and
// guarantees NodeLabels contains exactly them (no drift on add/remove).
func TestNodeLabels_ContainsAllKnownLabels(t *testing.T) {
	want := []string{
		entity.LabelPerson,
		entity.LabelClassic,
		entity.LabelSchool,
		entity.LabelPrescription,
		entity.LabelMedicine,
		entity.LabelDisease,
		entity.LabelDynasty,
		entity.LabelHistoricalEvent,
	}
	assert.ElementsMatch(t, want, entity.NodeLabels)
}

// TestIsValidLabel_TableDriven covers every known label plus the rejection
// of bogus/empty inputs.
func TestIsValidLabel_TableDriven(t *testing.T) {
	cases := []struct {
		name  string
		label string
		want  bool
	}{
		{"Person", entity.LabelPerson, true},
		{"Classic", entity.LabelClassic, true},
		{"School", entity.LabelSchool, true},
		{"Prescription", entity.LabelPrescription, true},
		{"Medicine", entity.LabelMedicine, true},
		{"Disease", entity.LabelDisease, true},
		{"Dynasty", entity.LabelDynasty, true},
		{"HistoricalEvent", entity.LabelHistoricalEvent, true},
		{"empty string", "", false},
		{"unknown label", "Robot", false},
		{"lowercase variant rejected", "person", false},
		{"whitespace rejected", " ", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, entity.IsValidLabel(tc.label))
		})
	}
}

// TestGraphNode_Construction mirrors how NodeUseCase.Create builds a node.
func TestGraphNode_Construction(t *testing.T) {
	n := entity.GraphNode{
		UID:            "person:1",
		Label:          entity.LabelPerson,
		Name:           "Zhang Zhongjing",
		PropertiesJSON: []byte(`{"dynasty":"Han"}`),
	}
	assert.Equal(t, "person:1", n.UID)
	assert.Equal(t, entity.LabelPerson, n.Label)
	assert.Equal(t, "Zhang Zhongjing", n.Name)
	assert.JSONEq(t, `{"dynasty":"Han"}`, string(n.PropertiesJSON))
}
