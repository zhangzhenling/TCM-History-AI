package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
)

// TestGraphEdge_TableName verifies the GORM table name override.
func TestGraphEdge_TableName(t *testing.T) {
	assert.Equal(t, "graph_edges", entity.GraphEdge{}.TableName())
}

// TestEdgeTypes_ContainsAllKnownTypes pins the 9 known edge types.
func TestEdgeTypes_ContainsAllKnownTypes(t *testing.T) {
	want := []string{
		entity.RelAuthored,
		entity.RelDiscipled,
		entity.RelInfluenced,
		entity.RelBelongsTo,
		entity.RelOccurredIn,
		entity.RelCited,
		entity.RelProposed,
		entity.RelOpposed,
		entity.RelInherited,
	}
	assert.ElementsMatch(t, want, entity.EdgeTypes)
}

// TestIsValidEdgeType_TableDriven covers every known edge type plus
// rejection of bogus/empty inputs.
func TestIsValidEdgeType_TableDriven(t *testing.T) {
	cases := []struct {
		name    string
		edgeT   string
		want    bool
	}{
		{"AUTHORED", entity.RelAuthored, true},
		{"DISCIPLED", entity.RelDiscipled, true},
		{"INFLUENCED", entity.RelInfluenced, true},
		{"BELONGS_TO", entity.RelBelongsTo, true},
		{"OCCURRED_IN", entity.RelOccurredIn, true},
		{"CITED", entity.RelCited, true},
		{"PROPOSED", entity.RelProposed, true},
		{"OPPOSED", entity.RelOpposed, true},
		{"INHERITED", entity.RelInherited, true},
		{"empty string", "", false},
		{"unknown edge type", "FRIEND_OF", false},
		{"lowercase variant rejected", "authored", false},
		{"whitespace rejected", " ", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, entity.IsValidEdgeType(tc.edgeT))
		})
	}
}

// TestGraphEdge_Construction mirrors how EdgeUseCase.Create builds an edge.
func TestGraphEdge_Construction(t *testing.T) {
	e := entity.GraphEdge{
		UID:            "edge:1",
		Type:           entity.RelAuthored,
		SourceUID:      "person:1",
		TargetUID:      "classic:1",
		PropertiesJSON: []byte(`{"year":200}`),
	}
	assert.Equal(t, "edge:1", e.UID)
	assert.Equal(t, entity.RelAuthored, e.Type)
	assert.Equal(t, "person:1", e.SourceUID)
	assert.Equal(t, "classic:1", e.TargetUID)
	assert.JSONEq(t, `{"year":200}`, string(e.PropertiesJSON))
}
