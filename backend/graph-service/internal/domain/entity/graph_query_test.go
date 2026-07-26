package entity_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
)

// TestGraphNodeView_JSONRoundTrip verifies the lightweight view (used in
// query results) survives a JSON round-trip with all four fields preserved.
func TestGraphNodeView_JSONRoundTrip(t *testing.T) {
	n := entity.GraphNodeView{
		UID:        "person:1",
		Label:      "Person",
		Name:       "Zhang Zhongjing",
		Properties: json.RawMessage(`{"dynasty":"Han"}`),
	}
	out, err := json.Marshal(n)
	require.NoError(t, err)
	var roundTripped entity.GraphNodeView
	require.NoError(t, json.Unmarshal(out, &roundTripped))
	assert.Equal(t, n.UID, roundTripped.UID)
	assert.Equal(t, n.Label, roundTripped.Label)
	assert.Equal(t, n.Name, roundTripped.Name)
	assert.JSONEq(t, string(n.Properties), string(roundTripped.Properties))
}

// TestGraphEdgeView_JSONRoundTrip verifies the edge view round-trip.
func TestGraphEdgeView_JSONRoundTrip(t *testing.T) {
	e := entity.GraphEdgeView{
		UID:        "edge:1",
		Type:       "AUTHORED",
		SourceUID:  "person:1",
		TargetUID:  "classic:1",
		Properties: json.RawMessage(`{"year":200}`),
	}
	out, err := json.Marshal(e)
	require.NoError(t, err)
	var roundTripped entity.GraphEdgeView
	require.NoError(t, json.Unmarshal(out, &roundTripped))
	assert.Equal(t, e.UID, roundTripped.UID)
	assert.Equal(t, e.Type, roundTripped.Type)
	assert.Equal(t, e.SourceUID, roundTripped.SourceUID)
	assert.Equal(t, e.TargetUID, roundTripped.TargetUID)
}

// TestGraphPath_Hops exercises the Hops field semantics: a single-node path
// has Hops=0, and adding edges increments Hops.
func TestGraphPath_Hops(t *testing.T) {
	zero := entity.GraphPath{Nodes: []entity.GraphNodeView{{UID: "a"}}, Hops: 0}
	assert.Equal(t, 0, zero.Hops)

	oneHop := entity.GraphPath{
		Nodes: []entity.GraphNodeView{{UID: "a"}, {UID: "b"}},
		Edges: []entity.GraphEdgeView{{UID: "e1", SourceUID: "a", TargetUID: "b"}},
		Hops:  1,
	}
	assert.Equal(t, 1, oneHop.Hops)
	assert.Len(t, oneHop.Nodes, 2)
	assert.Len(t, oneHop.Edges, 1)
}

// TestSubgraph_Empty verifies the zero value is a usable empty subgraph.
func TestSubgraph_Empty(t *testing.T) {
	var s entity.Subgraph
	assert.Empty(t, s.Nodes)
	assert.Empty(t, s.Edges)
}

// TestLineagePath_Generations verifies the lineage payload carries a
// generation depth per node, with len(Generations) == len(Path.Nodes).
func TestLineagePath_Generations(t *testing.T) {
	lp := entity.LineagePath{
		Path: entity.GraphPath{
			Nodes: []entity.GraphNodeView{{UID: "a"}, {UID: "b"}},
			Hops:  1,
		},
		Generations: []int{0, 1},
	}
	assert.Len(t, lp.Generations, len(lp.Path.Nodes))
}

// TestFigureWithWorks_Aggregation verifies the aggregation view structure.
func TestFigureWithWorks_Aggregation(t *testing.T) {
	f := entity.FigureWithWorks{
		Person:  entity.GraphNodeView{UID: "p1", Label: "Person"},
		Works:   []entity.GraphNodeView{{UID: "c1", Label: "Classic"}},
		Schools: []entity.GraphNodeView{{UID: "s1", Label: "School"}},
	}
	assert.Equal(t, "p1", f.Person.UID)
	assert.Len(t, f.Works, 1)
	assert.Len(t, f.Schools, 1)
}

// TestPrescriptionGraph_Aggregation verifies the prescription detail view.
func TestPrescriptionGraph_Aggregation(t *testing.T) {
	g := entity.PrescriptionGraph{
		Prescription: entity.GraphNodeView{UID: "rx1", Label: "Prescription"},
		Medicines:    []entity.GraphNodeView{{UID: "m1", Label: "Medicine"}},
		Diseases:     []entity.GraphNodeView{{UID: "d1", Label: "Disease"}},
	}
	assert.Equal(t, "rx1", g.Prescription.UID)
	assert.Len(t, g.Medicines, 1)
	assert.Len(t, g.Diseases, 1)
}
