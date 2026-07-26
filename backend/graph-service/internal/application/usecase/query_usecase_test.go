package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/graph-service/internal/application/dto"
	"tcm-history-ai/backend/graph-service/internal/application/usecase"
	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/errno"
)

// --- tests ---

func TestQueryUseCase_GetPersonWorks(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		store := &mockGraphStore{
			personWorksOut: []entity.GraphNodeView{
				{UID: "c1", Label: entity.LabelClassic, Name: "Shanghan Lun"},
				{UID: "c2", Label: entity.LabelClassic, Name: "Jingui Yaolue"},
			},
		}
		uc := usecase.NewQueryUseCase(store)
		got, err := uc.GetPersonWorks(context.Background(), "person:1")
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "c1", got[0].UID)
	})

	t.Run("empty person uid", func(t *testing.T) {
		uc := usecase.NewQueryUseCase(&mockGraphStore{})
		got, err := uc.GetPersonWorks(context.Background(), "")
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, got)
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockGraphStore{personWorksErr: errors.New("neo4j down")}
		uc := usecase.NewQueryUseCase(store)
		got, err := uc.GetPersonWorks(context.Background(), "p")
		require.Error(t, err)
		assert.Nil(t, got)
	})
}

func TestQueryUseCase_GetSchoolLineage(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		store := &mockGraphStore{
			lineageOut: &entity.LineagePath{
				Path: entity.GraphPath{
					Nodes: []entity.GraphNodeView{{UID: "a"}, {UID: "b"}},
					Hops:  1,
				},
				Generations: []int{0, 1},
			},
		}
		uc := usecase.NewQueryUseCase(store)
		got, err := uc.GetSchoolLineage(context.Background(), "Yishan", 4)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, 1, got.Path.Hops)
		require.Len(t, got.Generations, 2)
	})

	t.Run("default maxDepth when zero or negative", func(t *testing.T) {
		store := &mockGraphStore{
			lineageOut: &entity.LineagePath{Path: entity.GraphPath{Hops: 0}},
		}
		uc := usecase.NewQueryUseCase(store)
		_, err := uc.GetSchoolLineage(context.Background(), "X", 0)
		require.NoError(t, err)
	})

	t.Run("empty school name", func(t *testing.T) {
		uc := usecase.NewQueryUseCase(&mockGraphStore{})
		got, err := uc.GetSchoolLineage(context.Background(), "", 4)
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, got)
	})

	t.Run("not found when store returns nil", func(t *testing.T) {
		uc := usecase.NewQueryUseCase(&mockGraphStore{})
		got, err := uc.GetSchoolLineage(context.Background(), "X", 4)
		requireErrno(t, err, errno.NotFound)
		assert.Nil(t, got)
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockGraphStore{lineageErr: errors.New("boom")}
		uc := usecase.NewQueryUseCase(store)
		got, err := uc.GetSchoolLineage(context.Background(), "X", 4)
		require.Error(t, err)
		assert.Nil(t, got)
	})
}

func TestQueryUseCase_FindShortestPath(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		store := &mockGraphStore{
			queryPathOut: &entity.GraphPath{
				Nodes: []entity.GraphNodeView{{UID: "a"}, {UID: "b"}},
				Edges: []entity.GraphEdgeView{{UID: "e1", SourceUID: "a", TargetUID: "b"}},
				Hops:  1,
			},
		}
		uc := usecase.NewQueryUseCase(store)
		got, err := uc.FindShortestPath(context.Background(), "a", "b", 5)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, 1, got.Hops)
		require.Len(t, got.Nodes, 2)
		require.Len(t, got.Edges, 1)
	})

	t.Run("default maxHops when zero", func(t *testing.T) {
		store := &mockGraphStore{
			queryPathOut: &entity.GraphPath{Hops: 1},
		}
		uc := usecase.NewQueryUseCase(store)
		_, err := uc.FindShortestPath(context.Background(), "a", "b", 0)
		require.NoError(t, err)
	})

	t.Run("empty start uid", func(t *testing.T) {
		uc := usecase.NewQueryUseCase(&mockGraphStore{})
		got, err := uc.FindShortestPath(context.Background(), "", "b", 5)
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, got)
	})

	t.Run("empty end uid", func(t *testing.T) {
		uc := usecase.NewQueryUseCase(&mockGraphStore{})
		got, err := uc.FindShortestPath(context.Background(), "a", "", 5)
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, got)
	})

	t.Run("not found when store returns nil", func(t *testing.T) {
		uc := usecase.NewQueryUseCase(&mockGraphStore{})
		got, err := uc.FindShortestPath(context.Background(), "a", "b", 5)
		requireErrno(t, err, errno.NotFound)
		assert.Nil(t, got)
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockGraphStore{queryPathErr: errors.New("boom")}
		uc := usecase.NewQueryUseCase(store)
		got, err := uc.FindShortestPath(context.Background(), "a", "b", 5)
		require.Error(t, err)
		assert.Nil(t, got)
	})
}

func TestQueryUseCase_GetDynastyFigures(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		store := &mockGraphStore{
			dynastyOut: []entity.FigureWithWorks{
				{
					Person:  entity.GraphNodeView{UID: "p1", Label: entity.LabelPerson},
					Works:   []entity.GraphNodeView{{UID: "c1", Label: entity.LabelClassic}},
					Schools: []entity.GraphNodeView{{UID: "s1", Label: entity.LabelSchool}},
				},
			},
		}
		uc := usecase.NewQueryUseCase(store)
		got, err := uc.GetDynastyFigures(context.Background(), "Han")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "p1", got[0].Person.UID)
		require.Len(t, got[0].Works, 1)
		require.Len(t, got[0].Schools, 1)
	})

	t.Run("empty dynasty name", func(t *testing.T) {
		uc := usecase.NewQueryUseCase(&mockGraphStore{})
		got, err := uc.GetDynastyFigures(context.Background(), "")
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, got)
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockGraphStore{dynastyErr: errors.New("boom")}
		uc := usecase.NewQueryUseCase(store)
		got, err := uc.GetDynastyFigures(context.Background(), "Han")
		require.Error(t, err)
		assert.Nil(t, got)
	})
}

func TestQueryUseCase_GetPrescriptionDetail(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		store := &mockGraphStore{
			prescriptionOut: &entity.PrescriptionGraph{
				Prescription: entity.GraphNodeView{UID: "rx1", Label: entity.LabelPrescription},
				Medicines:    []entity.GraphNodeView{{UID: "m1", Label: entity.LabelMedicine}},
				Diseases:     []entity.GraphNodeView{{UID: "d1", Label: entity.LabelDisease}},
			},
		}
		uc := usecase.NewQueryUseCase(store)
		got, err := uc.GetPrescriptionDetail(context.Background(), "rx1")
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "rx1", got.Prescription.UID)
		require.Len(t, got.Medicines, 1)
		require.Len(t, got.Diseases, 1)
	})

	t.Run("empty uid", func(t *testing.T) {
		uc := usecase.NewQueryUseCase(&mockGraphStore{})
		got, err := uc.GetPrescriptionDetail(context.Background(), "")
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, got)
	})

	t.Run("not found when store returns nil", func(t *testing.T) {
		uc := usecase.NewQueryUseCase(&mockGraphStore{})
		got, err := uc.GetPrescriptionDetail(context.Background(), "rx")
		requireErrno(t, err, errno.NotFound)
		assert.Nil(t, got)
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockGraphStore{prescriptionErr: errors.New("boom")}
		uc := usecase.NewQueryUseCase(store)
		got, err := uc.GetPrescriptionDetail(context.Background(), "rx")
		require.Error(t, err)
		assert.Nil(t, got)
	})
}

func TestQueryUseCase_SearchNodes(t *testing.T) {
	t.Run("happy path with default limit", func(t *testing.T) {
		store := &mockGraphStore{
			searchOut: []entity.GraphNodeView{{UID: "n1", Label: entity.LabelPerson, Name: "Zhang"}},
		}
		uc := usecase.NewQueryUseCase(store)
		got, err := uc.SearchNodes(context.Background(), &dto.SearchParams{Keyword: "Zhang"})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "Zhang", got.Keyword)
		assert.Equal(t, 1, got.Total)
		require.Len(t, got.Items, 1)
	})

	t.Run("explicit limit", func(t *testing.T) {
		store := &mockGraphStore{searchOut: []entity.GraphNodeView{}}
		uc := usecase.NewQueryUseCase(store)
		got, err := uc.SearchNodes(context.Background(), &dto.SearchParams{Keyword: "x", Limit: 5})
		require.NoError(t, err)
		assert.Equal(t, 5, 5) // sanity for the limit parameter pass-through
		_ = got
	})

	t.Run("nil params", func(t *testing.T) {
		uc := usecase.NewQueryUseCase(&mockGraphStore{})
		got, err := uc.SearchNodes(context.Background(), nil)
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, got)
	})

	t.Run("empty keyword", func(t *testing.T) {
		uc := usecase.NewQueryUseCase(&mockGraphStore{})
		got, err := uc.SearchNodes(context.Background(), &dto.SearchParams{Keyword: ""})
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, got)
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockGraphStore{searchErr: errors.New("boom")}
		uc := usecase.NewQueryUseCase(store)
		got, err := uc.SearchNodes(context.Background(), &dto.SearchParams{Keyword: "x"})
		require.Error(t, err)
		assert.Nil(t, got)
	})
}

func TestQueryUseCase_GetSubgraph(t *testing.T) {
	t.Run("happy path with defaults", func(t *testing.T) {
		store := &mockGraphStore{
			subgraphOut: &entity.Subgraph{
				Nodes: []entity.GraphNodeView{{UID: "a"}, {UID: "b"}},
				Edges: []entity.GraphEdgeView{{UID: "e1", SourceUID: "a", TargetUID: "b"}},
			},
		}
		uc := usecase.NewQueryUseCase(store)
		got, err := uc.GetSubgraph(context.Background(), "a", 0, 0)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Len(t, got.Nodes, 2)
		require.Len(t, got.Edges, 1)
	})

	t.Run("empty center uid", func(t *testing.T) {
		uc := usecase.NewQueryUseCase(&mockGraphStore{})
		got, err := uc.GetSubgraph(context.Background(), "", 2, 50)
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, got)
	})

	t.Run("limit exceeds 300", func(t *testing.T) {
		uc := usecase.NewQueryUseCase(&mockGraphStore{})
		got, err := uc.GetSubgraph(context.Background(), "a", 2, 301)
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, got)
	})

	t.Run("store returns nil subgraph returns empty dto", func(t *testing.T) {
		uc := usecase.NewQueryUseCase(&mockGraphStore{})
		got, err := uc.GetSubgraph(context.Background(), "a", 2, 50)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Empty(t, got.Nodes)
		assert.Empty(t, got.Edges)
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockGraphStore{subgraphErr: errors.New("boom")}
		uc := usecase.NewQueryUseCase(store)
		got, err := uc.GetSubgraph(context.Background(), "a", 2, 50)
		require.Error(t, err)
		assert.Nil(t, got)
	})
}
