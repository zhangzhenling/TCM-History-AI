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
	"tcm-history-ai/backend/graph-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// --- mock GraphNodeRepository ---

type mockNodeRepo struct {
	items     map[string]*entity.GraphNode
	createErr error
	updateErr error
	deleteErr error
	findErr   error
	listErr   error
	searchErr error
}

func newMockNodeRepo() *mockNodeRepo {
	return &mockNodeRepo{items: map[string]*entity.GraphNode{}}
}

func (m *mockNodeRepo) Create(_ context.Context, n *entity.GraphNode) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.items[n.UID] = n
	return nil
}

func (m *mockNodeRepo) Update(_ context.Context, n *entity.GraphNode) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if _, ok := m.items[n.UID]; !ok {
		return errno.New(errno.NotFound, "node not found")
	}
	m.items[n.UID] = n
	return nil
}

func (m *mockNodeRepo) Delete(_ context.Context, uid string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.items, uid)
	return nil
}

func (m *mockNodeRepo) FindByUID(_ context.Context, uid string) (*entity.GraphNode, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	if n, ok := m.items[uid]; ok {
		clone := *n
		return &clone, nil
	}
	return nil, nil
}

func (m *mockNodeRepo) ListByLabel(_ context.Context, label string, p pagination.Params) ([]entity.GraphNode, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	all := make([]entity.GraphNode, 0, len(m.items))
	for _, n := range m.items {
		if label == "" || n.Label == label {
			all = append(all, *n)
		}
	}
	_, pageSize, offset := p.Normalise()
	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (m *mockNodeRepo) SearchByName(_ context.Context, keyword, label string, p pagination.Params) ([]entity.GraphNode, int, error) {
	if m.searchErr != nil {
		return nil, 0, m.searchErr
	}
	all := make([]entity.GraphNode, 0, len(m.items))
	for _, n := range m.items {
		if label != "" && n.Label != label {
			continue
		}
		if keyword == "" || containsStr(n.Name, keyword) {
			all = append(all, *n)
		}
	}
	_, pageSize, offset := p.Normalise()
	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

// containsStr reports whether haystack contains needle (case-sensitive).
func containsStr(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// --- mock GraphStore (subset used by NodeUseCase) ---

type mockGraphStore struct {
	upsertNodeErr  error
	upsertEdgeErr  error
	deleteNodeErr  error
	deleteEdgeErr  error
	getNodeOut     *entity.GraphNodeView
	getNodeErr     error
	getEdgeOut     *entity.GraphEdgeView
	getEdgeErr     error
	queryPathOut   *entity.GraphPath
	queryPathErr   error
	subgraphOut    *entity.Subgraph
	subgraphErr    error
	personWorksOut []entity.GraphNodeView
	personWorksErr error
	lineageOut     *entity.LineagePath
	lineageErr     error
	dynastyOut     []entity.FigureWithWorks
	dynastyErr     error
	prescriptionOut *entity.PrescriptionGraph
	prescriptionErr error
	searchOut      []entity.GraphNodeView
	searchErr      error
	ensureErr      error

	upsertNodeCalls []service.NodePayload
	deleteNodeCalls []string
	upsertEdgeCalls []service.EdgePayload
	deleteEdgeCalls []string
}

func (m *mockGraphStore) EnsureConstraints(_ context.Context) error { return m.ensureErr }
func (m *mockGraphStore) UpsertNode(_ context.Context, n service.NodePayload) error {
	m.upsertNodeCalls = append(m.upsertNodeCalls, n)
	return m.upsertNodeErr
}
func (m *mockGraphStore) GetNode(_ context.Context, _ string) (*entity.GraphNodeView, error) {
	return m.getNodeOut, m.getNodeErr
}
func (m *mockGraphStore) DeleteNode(_ context.Context, uid string) error {
	m.deleteNodeCalls = append(m.deleteNodeCalls, uid)
	return m.deleteNodeErr
}
func (m *mockGraphStore) UpsertEdge(_ context.Context, e service.EdgePayload) error {
	m.upsertEdgeCalls = append(m.upsertEdgeCalls, e)
	return m.upsertEdgeErr
}
func (m *mockGraphStore) GetEdge(_ context.Context, _ string) (*entity.GraphEdgeView, error) {
	return m.getEdgeOut, m.getEdgeErr
}
func (m *mockGraphStore) DeleteEdge(_ context.Context, uid string) error {
	m.deleteEdgeCalls = append(m.deleteEdgeCalls, uid)
	return m.deleteEdgeErr
}
func (m *mockGraphStore) QueryPath(_ context.Context, _, _ string, _ int) (*entity.GraphPath, error) {
	return m.queryPathOut, m.queryPathErr
}
func (m *mockGraphStore) GetSubgraph(_ context.Context, _ string, _, _ int) (*entity.Subgraph, error) {
	return m.subgraphOut, m.subgraphErr
}
func (m *mockGraphStore) GetPersonWorks(_ context.Context, _ string) ([]entity.GraphNodeView, error) {
	return m.personWorksOut, m.personWorksErr
}
func (m *mockGraphStore) GetSchoolLineage(_ context.Context, _ string, _ int) (*entity.LineagePath, error) {
	return m.lineageOut, m.lineageErr
}
func (m *mockGraphStore) GetDynastyFigures(_ context.Context, _ string) ([]entity.FigureWithWorks, error) {
	return m.dynastyOut, m.dynastyErr
}
func (m *mockGraphStore) GetPrescriptionDetail(_ context.Context, _ string) (*entity.PrescriptionGraph, error) {
	return m.prescriptionOut, m.prescriptionErr
}
func (m *mockGraphStore) SearchNodes(_ context.Context, _, _ string, _ int) ([]entity.GraphNodeView, error) {
	return m.searchOut, m.searchErr
}

// --- mock EventPublisher ---

type mockEventPublisher struct {
	published []event.Event
	publishErr error
}

func (m *mockEventPublisher) Publish(_ context.Context, evt event.Event) error {
	if m.publishErr != nil {
		return m.publishErr
	}
	m.published = append(m.published, evt)
	return nil
}

// --- helpers ---

// requireErrno asserts err is non-nil and (when possible) carries the expected
// errno code.
func requireErrno(t *testing.T, err error, code errno.Errno) {
	t.Helper()
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, code, e.Code)
	}
}

// --- tests ---

func TestNodeUseCase_Create(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		repo := newMockNodeRepo()
		store := &mockGraphStore{}
		pub := &mockEventPublisher{}
		uc := usecase.NewNodeUseCase(repo, store, pub)

		resp, err := uc.Create(context.Background(), &dto.NodeRequest{
			UID:   "person:1",
			Label: entity.LabelPerson,
			Name:  "Zhang Zhongjing",
			PropertiesJSON: []byte(`{"dynasty":"Han"}`),
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "person:1", resp.UID)
		assert.Equal(t, entity.LabelPerson, resp.Label)
		assert.NotZero(t, resp.ID)
		assert.JSONEq(t, `{"dynasty":"Han"}`, string(resp.PropertiesJSON))
		// Mirrored to GraphStore + event published.
		require.Len(t, store.upsertNodeCalls, 1)
		require.Len(t, pub.published, 1)
	})

	t.Run("default properties when nil", func(t *testing.T) {
		uc := usecase.NewNodeUseCase(newMockNodeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Create(context.Background(), &dto.NodeRequest{
			UID: "p:1", Label: entity.LabelPerson, Name: "X",
		})
		require.NoError(t, err)
		assert.JSONEq(t, "{}", string(resp.PropertiesJSON))
	})

	t.Run("nil request", func(t *testing.T) {
		uc := usecase.NewNodeUseCase(newMockNodeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Create(context.Background(), nil)
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("empty uid", func(t *testing.T) {
		uc := usecase.NewNodeUseCase(newMockNodeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Create(context.Background(), &dto.NodeRequest{Label: entity.LabelPerson, Name: "X"})
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("empty name", func(t *testing.T) {
		uc := usecase.NewNodeUseCase(newMockNodeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Create(context.Background(), &dto.NodeRequest{UID: "u", Label: entity.LabelPerson})
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("empty label", func(t *testing.T) {
		uc := usecase.NewNodeUseCase(newMockNodeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Create(context.Background(), &dto.NodeRequest{UID: "u", Name: "n"})
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("invalid label", func(t *testing.T) {
		uc := usecase.NewNodeUseCase(newMockNodeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Create(context.Background(), &dto.NodeRequest{UID: "u", Label: "Robot", Name: "n"})
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := newMockNodeRepo()
		repo.createErr = errors.New("db down")
		uc := usecase.NewNodeUseCase(repo, &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Create(context.Background(), &dto.NodeRequest{
			UID: "u", Label: entity.LabelPerson, Name: "n",
		})
		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestNodeUseCase_Update(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		repo := newMockNodeRepo()
		uc := usecase.NewNodeUseCase(repo, &mockGraphStore{}, &mockEventPublisher{})
		_, err := uc.Create(context.Background(), &dto.NodeRequest{
			UID: "u", Label: entity.LabelPerson, Name: "Old",
		})
		require.NoError(t, err)

		resp, err := uc.Update(context.Background(), "u", &dto.NodeRequest{
			Label: entity.LabelPerson, Name: "New",
			PropertiesJSON: []byte(`{"k":"v"}`),
		})
		require.NoError(t, err)
		assert.Equal(t, "New", resp.Name)
		assert.JSONEq(t, `{"k":"v"}`, string(resp.PropertiesJSON))
	})

	t.Run("empty uid", func(t *testing.T) {
		uc := usecase.NewNodeUseCase(newMockNodeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Update(context.Background(), "", &dto.NodeRequest{Name: "x"})
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("nil body", func(t *testing.T) {
		uc := usecase.NewNodeUseCase(newMockNodeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Update(context.Background(), "u", nil)
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("invalid label", func(t *testing.T) {
		repo := newMockNodeRepo()
		uc := usecase.NewNodeUseCase(repo, &mockGraphStore{}, &mockEventPublisher{})
		_, err := uc.Create(context.Background(), &dto.NodeRequest{
			UID: "u", Label: entity.LabelPerson, Name: "Old",
		})
		require.NoError(t, err)
		resp, err := uc.Update(context.Background(), "u", &dto.NodeRequest{Label: "Robot"})
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("not found", func(t *testing.T) {
		uc := usecase.NewNodeUseCase(newMockNodeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Update(context.Background(), "missing", &dto.NodeRequest{Name: "x"})
		requireErrno(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("repo error on find", func(t *testing.T) {
		repo := newMockNodeRepo()
		repo.findErr = errors.New("boom")
		uc := usecase.NewNodeUseCase(repo, &mockGraphStore{}, &mockEventPublisher{})
		_, err := uc.Update(context.Background(), "u", &dto.NodeRequest{Name: "x"})
		require.Error(t, err)
	})

	t.Run("repo error on update", func(t *testing.T) {
		repo := newMockNodeRepo()
		uc := usecase.NewNodeUseCase(repo, &mockGraphStore{}, &mockEventPublisher{})
		_, err := uc.Create(context.Background(), &dto.NodeRequest{
			UID: "u", Label: entity.LabelPerson, Name: "Old",
		})
		require.NoError(t, err)
		repo.updateErr = errors.New("update fail")
		_, err = uc.Update(context.Background(), "u", &dto.NodeRequest{Name: "New"})
		require.Error(t, err)
	})
}

func TestNodeUseCase_Delete(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		repo := newMockNodeRepo()
		store := &mockGraphStore{}
		uc := usecase.NewNodeUseCase(repo, store, &mockEventPublisher{})
		_, err := uc.Create(context.Background(), &dto.NodeRequest{
			UID: "u", Label: entity.LabelPerson, Name: "n",
		})
		require.NoError(t, err)

		require.NoError(t, uc.Delete(context.Background(), "u"))
		require.Len(t, store.deleteNodeCalls, 1)
		_, err = uc.Get(context.Background(), "u")
		requireErrno(t, err, errno.NotFound)
	})

	t.Run("empty uid", func(t *testing.T) {
		uc := usecase.NewNodeUseCase(newMockNodeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		err := uc.Delete(context.Background(), "")
		requireErrno(t, err, errno.InvalidParams)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := newMockNodeRepo()
		repo.deleteErr = errors.New("boom")
		uc := usecase.NewNodeUseCase(repo, &mockGraphStore{}, &mockEventPublisher{})
		err := uc.Delete(context.Background(), "u")
		require.Error(t, err)
	})
}

func TestNodeUseCase_Get(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		uc := usecase.NewNodeUseCase(newMockNodeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		_, err := uc.Create(context.Background(), &dto.NodeRequest{
			UID: "u", Label: entity.LabelPerson, Name: "n",
		})
		require.NoError(t, err)
		got, err := uc.Get(context.Background(), "u")
		require.NoError(t, err)
		assert.Equal(t, "n", got.Name)
	})

	t.Run("empty uid", func(t *testing.T) {
		uc := usecase.NewNodeUseCase(newMockNodeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Get(context.Background(), "")
		requireErrno(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("not found", func(t *testing.T) {
		uc := usecase.NewNodeUseCase(newMockNodeRepo(), &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.Get(context.Background(), "missing")
		requireErrno(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := newMockNodeRepo()
		repo.findErr = errors.New("boom")
		uc := usecase.NewNodeUseCase(repo, &mockGraphStore{}, &mockEventPublisher{})
		_, err := uc.Get(context.Background(), "u")
		require.Error(t, err)
	})
}

func TestNodeUseCase_List(t *testing.T) {
	repo := newMockNodeRepo()
	uc := usecase.NewNodeUseCase(repo, &mockGraphStore{}, &mockEventPublisher{})
	_, _ = uc.Create(context.Background(), &dto.NodeRequest{UID: "p1", Label: entity.LabelPerson, Name: "Zhang"})
	_, _ = uc.Create(context.Background(), &dto.NodeRequest{UID: "c1", Label: entity.LabelClassic, Name: "Neijing"})
	_, _ = uc.Create(context.Background(), &dto.NodeRequest{UID: "p2", Label: entity.LabelPerson, Name: "Hua"})

	t.Run("list all", func(t *testing.T) {
		resp, err := uc.List(context.Background(), "", "", pagination.Params{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, 3, resp.Total)
	})

	t.Run("list by label", func(t *testing.T) {
		resp, err := uc.List(context.Background(), entity.LabelPerson, "", pagination.Params{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Total)
	})

	t.Run("search by keyword", func(t *testing.T) {
		resp, err := uc.List(context.Background(), entity.LabelPerson, "Zhang", pagination.Params{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Total)
		require.Len(t, resp.Items, 1)
		assert.Equal(t, "Zhang", resp.Items[0].Name)
	})

	t.Run("invalid label", func(t *testing.T) {
		resp, err := uc.List(context.Background(), "Robot", "", pagination.Params{Page: 1, PageSize: 10})
		requireErrno(t, err, errno.InvalidParams)
		assert.Equal(t, 0, resp.Total)
	})

	t.Run("repo list error", func(t *testing.T) {
		repo := newMockNodeRepo()
		repo.listErr = errors.New("boom")
		uc := usecase.NewNodeUseCase(repo, &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.List(context.Background(), "", "", pagination.Params{Page: 1, PageSize: 10})
		require.Error(t, err)
		assert.Equal(t, 0, resp.Total)
	})

	t.Run("repo search error", func(t *testing.T) {
		repo := newMockNodeRepo()
		repo.searchErr = errors.New("boom")
		uc := usecase.NewNodeUseCase(repo, &mockGraphStore{}, &mockEventPublisher{})
		resp, err := uc.List(context.Background(), "", "kw", pagination.Params{Page: 1, PageSize: 10})
		require.Error(t, err)
		assert.Equal(t, 0, resp.Total)
	})
}
