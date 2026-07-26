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
	"tcm-history-ai/backend/graph-service/internal/domain/event"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// --- mock GraphEdgeRepository ---

type mockEdgeRepo struct {
	items     map[string]*entity.GraphEdge
	createErr error
	updateErr error
	deleteErr error
	findErr   error
	listBySrcErr error
	listByTgtErr error
	listByTypeErr error
}

func newMockEdgeRepo() *mockEdgeRepo {
	return &mockEdgeRepo{items: map[string]*entity.GraphEdge{}}
}

func (m *mockEdgeRepo) Create(_ context.Context, e *entity.GraphEdge) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.items[e.UID] = e
	return nil
}

func (m *mockEdgeRepo) Update(_ context.Context, e *entity.GraphEdge) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if _, ok := m.items[e.UID]; !ok {
		return errno.New(errno.NotFound, "edge not found")
	}
	m.items[e.UID] = e
	return nil
}

func (m *mockEdgeRepo) Delete(_ context.Context, uid string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.items, uid)
	return nil
}

func (m *mockEdgeRepo) FindByUID(_ context.Context, uid string) (*entity.GraphEdge, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	if e, ok := m.items[uid]; ok {
		clone := *e
		return &clone, nil
	}
	return nil, nil
}

func (m *mockEdgeRepo) ListBySource(_ context.Context, sourceUID string, p pagination.Params) ([]entity.GraphEdge, int, error) {
	if m.listBySrcErr != nil {
		return nil, 0, m.listBySrcErr
	}
	all := make([]entity.GraphEdge, 0, len(m.items))
	for _, e := range m.items {
		if e.SourceUID == sourceUID {
			all = append(all, *e)
		}
	}
	return paginateEdges(all, p), len(all), nil
}

func (m *mockEdgeRepo) ListByTarget(_ context.Context, targetUID string, p pagination.Params) ([]entity.GraphEdge, int, error) {
	if m.listByTgtErr != nil {
		return nil, 0, m.listByTgtErr
	}
	all := make([]entity.GraphEdge, 0, len(m.items))
	for _, e := range m.items {
		if e.TargetUID == targetUID {
			all = append(all, *e)
		}
	}
	return paginateEdges(all, p), len(all), nil
}

func (m *mockEdgeRepo) ListByType(_ context.Context, edgeType string, p pagination.Params) ([]entity.GraphEdge, int, error) {
	if m.listByTypeErr != nil {
		return nil, 0, m.listByTypeErr
	}
	all := make([]entity.GraphEdge, 0, len(m.items))
	for _, e := range m.items {
		if edgeType == "" || e.Type == edgeType {
			all = append(all, *e)
		}
	}
	return paginateEdges(all, p), len(all), nil
}

func paginateEdges(all []entity.GraphEdge, p pagination.Params) []entity.GraphEdge {
	_, pageSize, offset := p.Normalise()
	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	return all[offset:end]
}

// --- tests ---

func TestEdgeUseCase_Create(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		repo := newMockEdgeRepo()
		store := &mockGraphStore{}
		pub := &mockEventPublisher{}
		uc := usecase.NewEdgeUseCase(repo, store, pub)

		resp, err := uc.Create(context.Background(), &dto.EdgeRequest{
			UID:       "edge:1",
			Type:      entity.RelAuthored,
			SourceUID: "person:1",
			TargetUID: "classic:1",
			PropertiesJSON: []byte(`{"year":200}`),
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "edge:1", resp.UID)
		assert.Equal(t, entity.RelAuthored, resp.Type)
		assert.NotZero(t, resp.ID)
		assert.JSONEq(t, `{"year":200}`, string(resp.PropertiesJSON))
		require.Len(t, store.upsertEdgeCalls, 1)
		require.Len(t, pub.published, 1)
		evt, ok := pub.published[0].(event.EdgeUpserted)
		require.True(t, ok)
		assert.Equal(t, "edge:1", evt.UID)
	})

	t.Run("default properties when nil", func(t *testing.T) {
		uc := usecase.NewEdgeUseCase(newMockEdgeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Create(context.Background(), &dto.EdgeRequest{
			UID: "e", Type: entity.RelAuthored, SourceUID: "s", TargetUID: "t",
		})
		require.NoError(t, err)
		assert.JSONEq(t, "{}", string(resp.PropertiesJSON))
	})

	t.Run("nil body", func(t *testing.T) {
		uc := usecase.NewEdgeUseCase(newMockEdgeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Create(context.Background(), nil)
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("missing uid", func(t *testing.T) {
		uc := usecase.NewEdgeUseCase(newMockEdgeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Create(context.Background(), &dto.EdgeRequest{
			Type: entity.RelAuthored, SourceUID: "s", TargetUID: "t",
		})
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("missing type", func(t *testing.T) {
		uc := usecase.NewEdgeUseCase(newMockEdgeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Create(context.Background(), &dto.EdgeRequest{
			UID: "e", SourceUID: "s", TargetUID: "t",
		})
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("invalid type", func(t *testing.T) {
		uc := usecase.NewEdgeUseCase(newMockEdgeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Create(context.Background(), &dto.EdgeRequest{
			UID: "e", Type: "FRIEND_OF", SourceUID: "s", TargetUID: "t",
		})
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("missing source", func(t *testing.T) {
		uc := usecase.NewEdgeUseCase(newMockEdgeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Create(context.Background(), &dto.EdgeRequest{
			UID: "e", Type: entity.RelAuthored, TargetUID: "t",
		})
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("missing target", func(t *testing.T) {
		uc := usecase.NewEdgeUseCase(newMockEdgeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Create(context.Background(), &dto.EdgeRequest{
			UID: "e", Type: entity.RelAuthored, SourceUID: "s",
		})
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := newMockEdgeRepo()
		repo.createErr = errors.New("boom")
		uc := usecase.NewEdgeUseCase(repo, &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Create(context.Background(), &dto.EdgeRequest{
			UID: "e", Type: entity.RelAuthored, SourceUID: "s", TargetUID: "t",
		})
		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestEdgeUseCase_Update(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		repo := newMockEdgeRepo()
		uc := usecase.NewEdgeUseCase(repo, &mockGraphStore{}, &mockEventPublisher{})
		_, err := uc.Create(context.Background(), &dto.EdgeRequest{
			UID: "e", Type: entity.RelAuthored, SourceUID: "s", TargetUID: "t",
		})
		require.NoError(t, err)
		resp, err := uc.Update(context.Background(), "e", &dto.EdgeRequest{
			Type: entity.RelDiscipled, SourceUID: "s2", TargetUID: "t2",
			PropertiesJSON: []byte(`{"k":"v"}`),
		})
		require.NoError(t, err)
		assert.Equal(t, entity.RelDiscipled, resp.Type)
		assert.Equal(t, "s2", resp.SourceUID)
		assert.Equal(t, "t2", resp.TargetUID)
	})

	t.Run("partial update keeps original fields", func(t *testing.T) {
		repo := newMockEdgeRepo()
		uc := usecase.NewEdgeUseCase(repo, &mockGraphStore{}, &mockEventPublisher{})
		_, err := uc.Create(context.Background(), &dto.EdgeRequest{
			UID: "e", Type: entity.RelAuthored, SourceUID: "s", TargetUID: "t",
		})
		require.NoError(t, err)
		// Only change Type — SourceUID/TargetUID should remain.
		resp, err := uc.Update(context.Background(), "e", &dto.EdgeRequest{Type: entity.RelCited})
		require.NoError(t, err)
		assert.Equal(t, entity.RelCited, resp.Type)
		assert.Equal(t, "s", resp.SourceUID)
		assert.Equal(t, "t", resp.TargetUID)
	})

	t.Run("empty uid", func(t *testing.T) {
		uc := usecase.NewEdgeUseCase(newMockEdgeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Update(context.Background(), "", &dto.EdgeRequest{Type: entity.RelAuthored})
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("nil body", func(t *testing.T) {
		uc := usecase.NewEdgeUseCase(newMockEdgeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Update(context.Background(), "e", nil)
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("invalid type", func(t *testing.T) {
		repo := newMockEdgeRepo()
		uc := usecase.NewEdgeUseCase(repo, &mockGraphStore{}, &mockEventPublisher{})
		_, err := uc.Create(context.Background(), &dto.EdgeRequest{
			UID: "e", Type: entity.RelAuthored, SourceUID: "s", TargetUID: "t",
		})
		require.NoError(t, err)
		resp, err := uc.Update(context.Background(), "e", &dto.EdgeRequest{Type: "FRIEND_OF"})
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("not found", func(t *testing.T) {
		uc := usecase.NewEdgeUseCase(newMockEdgeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Update(context.Background(), "missing", &dto.EdgeRequest{Type: entity.RelAuthored})
		requireErrno(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("repo error on update", func(t *testing.T) {
		repo := newMockEdgeRepo()
		uc := usecase.NewEdgeUseCase(repo, &mockGraphStore{}, &mockEventPublisher{})
		_, err := uc.Create(context.Background(), &dto.EdgeRequest{
			UID: "e", Type: entity.RelAuthored, SourceUID: "s", TargetUID: "t",
		})
		require.NoError(t, err)
		repo.updateErr = errors.New("boom")
		_, err = uc.Update(context.Background(), "e", &dto.EdgeRequest{Type: entity.RelCited})
		require.Error(t, err)
	})
}

func TestEdgeUseCase_Delete(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		repo := newMockEdgeRepo()
		store := &mockGraphStore{}
		uc := usecase.NewEdgeUseCase(repo, store, &mockEventPublisher{})
		_, err := uc.Create(context.Background(), &dto.EdgeRequest{
			UID: "e", Type: entity.RelAuthored, SourceUID: "s", TargetUID: "t",
		})
		require.NoError(t, err)
		require.NoError(t, uc.Delete(context.Background(), "e"))
		require.Len(t, store.deleteEdgeCalls, 1)
	})

	t.Run("empty uid", func(t *testing.T) {
		uc := usecase.NewEdgeUseCase(newMockEdgeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		err := uc.Delete(context.Background(), "")
		requireErrno(t, err, errno.InvalidParams)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := newMockEdgeRepo()
		repo.deleteErr = errors.New("boom")
		uc := usecase.NewEdgeUseCase(repo, &mockGraphStore{}, &mockEventPublisher{})
		err := uc.Delete(context.Background(), "e")
		require.Error(t, err)
	})
}

func TestEdgeUseCase_Get(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		uc := usecase.NewEdgeUseCase(newMockEdgeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		_, err := uc.Create(context.Background(), &dto.EdgeRequest{
			UID: "e", Type: entity.RelAuthored, SourceUID: "s", TargetUID: "t",
		})
		require.NoError(t, err)
		got, err := uc.Get(context.Background(), "e")
		require.NoError(t, err)
		assert.Equal(t, entity.RelAuthored, got.Type)
	})

	t.Run("empty uid", func(t *testing.T) {
		uc := usecase.NewEdgeUseCase(newMockEdgeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Get(context.Background(), "")
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("not found", func(t *testing.T) {
		uc := usecase.NewEdgeUseCase(newMockEdgeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Get(context.Background(), "missing")
		requireErrno(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})
}

func TestEdgeUseCase_List(t *testing.T) {
	repo := newMockEdgeRepo()
	uc := usecase.NewEdgeUseCase(repo, &mockGraphStore{}, &mockEventPublisher{})
	_, _ = uc.Create(context.Background(), &dto.EdgeRequest{UID: "e1", Type: entity.RelAuthored, SourceUID: "p1", TargetUID: "c1"})
	_, _ = uc.Create(context.Background(), &dto.EdgeRequest{UID: "e2", Type: entity.RelAuthored, SourceUID: "p2", TargetUID: "c1"})
	_, _ = uc.Create(context.Background(), &dto.EdgeRequest{UID: "e3", Type: entity.RelDiscipled, SourceUID: "p2", TargetUID: "p3"})

	t.Run("list by source", func(t *testing.T) {
		resp, err := uc.List(context.Background(), "p2", "", "", pagination.Params{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Total)
	})

	t.Run("list by target", func(t *testing.T) {
		resp, err := uc.List(context.Background(), "", "c1", "", pagination.Params{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Total)
	})

	t.Run("list by type", func(t *testing.T) {
		resp, err := uc.List(context.Background(), "", "", entity.RelDiscipled, pagination.Params{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Total)
	})

	t.Run("list all (no filter)", func(t *testing.T) {
		resp, err := uc.List(context.Background(), "", "", "", pagination.Params{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, 3, resp.Total)
	})

	t.Run("invalid type", func(t *testing.T) {
		resp, err := uc.List(context.Background(), "", "", "FRIEND_OF", pagination.Params{Page: 1, PageSize: 10})
		requireErrno(t, err, errno.InvalidParams)
		assert.Equal(t, 0, resp.Total)
	})

	t.Run("repo error on ListBySource", func(t *testing.T) {
		repo := newMockEdgeRepo()
		repo.listBySrcErr = errors.New("boom")
		uc := usecase.NewEdgeUseCase(repo, &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.List(context.Background(), "p", "", "", pagination.Params{Page: 1, PageSize: 10})
		require.Error(t, err)
		assert.Equal(t, 0, resp.Total)
	})

	t.Run("repo error on ListByTarget", func(t *testing.T) {
		repo := newMockEdgeRepo()
		repo.listByTgtErr = errors.New("boom")
		uc := usecase.NewEdgeUseCase(repo, &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.List(context.Background(), "", "t", "", pagination.Params{Page: 1, PageSize: 10})
		require.Error(t, err)
		assert.Equal(t, 0, resp.Total)
	})

	t.Run("repo error on ListByType", func(t *testing.T) {
		repo := newMockEdgeRepo()
		repo.listByTypeErr = errors.New("boom")
		uc := usecase.NewEdgeUseCase(repo, &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.List(context.Background(), "", "", entity.RelAuthored, pagination.Params{Page: 1, PageSize: 10})
		require.Error(t, err)
		assert.Equal(t, 0, resp.Total)
	})
}
