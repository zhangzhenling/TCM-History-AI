package controller_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route/param"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/graph-service/internal/application/usecase"
	"tcm-history-ai/backend/graph-service/internal/controller"
	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/graph-service/internal/domain/event"
	"tcm-history-ai/backend/graph-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
	"tcm-history-ai/backend/pkg/response"
)

func init() { idgen.Init(3) }

func newRC() *app.RequestContext { return app.NewContext(0) }

func setParam(rc *app.RequestContext, key, value string) {
	rc.Params = param.Params{{Key: key, Value: value}}
}

func decodeBody(t *testing.T, rc *app.RequestContext) response.Body {
	t.Helper()
	var body response.Body
	require.NoError(t, json.Unmarshal(rc.Response.Body(), &body))
	return body
}

func assertStatusCode(t *testing.T, rc *app.RequestContext, want int) response.Body {
	t.Helper()
	require.Equal(t, want, rc.Response.StatusCode())
	if rc.Response.StatusCode() == http.StatusNoContent {
		return response.Body{}
	}
	return decodeBody(t, rc)
}

func ctx() context.Context { return context.Background() }

// ============================================================================
// Mock GraphNodeRepository
// ============================================================================

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

// ============================================================================
// Mock GraphEdgeRepository
// ============================================================================

type mockEdgeRepo struct {
	items         map[string]*entity.GraphEdge
	createErr     error
	updateErr     error
	deleteErr     error
	findErr       error
	listBySrcErr  error
	listByTgtErr  error
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

// ============================================================================
// Mock GraphSyncLogRepository
// ============================================================================

type mockSyncLogRepo struct {
	items         map[int64]*entity.GraphSyncLog
	createErr     error
	updateErr     error
	listPendingErr error
	pendingOut    []entity.GraphSyncLog
}

func newMockSyncLogRepo() *mockSyncLogRepo {
	return &mockSyncLogRepo{items: map[int64]*entity.GraphSyncLog{}}
}

func (m *mockSyncLogRepo) Create(_ context.Context, log *entity.GraphSyncLog) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.items[log.ID] = log
	return nil
}

func (m *mockSyncLogRepo) UpdateStatus(_ context.Context, id int64, status, errorMsg string) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if log, ok := m.items[id]; ok {
		log.Status = status
		log.ErrorMsg = errorMsg
	}
	return nil
}

func (m *mockSyncLogRepo) ListPending(_ context.Context, limit int) ([]entity.GraphSyncLog, error) {
	if m.listPendingErr != nil {
		return nil, m.listPendingErr
	}
	if m.pendingOut != nil {
		out := make([]entity.GraphSyncLog, 0, len(m.pendingOut))
		for _, l := range m.pendingOut {
			if limit > 0 && len(out) >= limit {
				break
			}
			out = append(out, l)
		}
		return out, nil
	}
	out := make([]entity.GraphSyncLog, 0)
	for _, l := range m.items {
		if l.Status == entity.SyncStatusPending {
			out = append(out, *l)
		}
	}
	return out, nil
}

// ============================================================================
// Mock GraphStore
// ============================================================================

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

// ============================================================================
// Mock EventPublisher
// ============================================================================

type mockEventPublisher struct {
	published  []event.Event
	publishErr error
}

func (m *mockEventPublisher) Publish(_ context.Context, evt event.Event) error {
	if m.publishErr != nil {
		return m.publishErr
	}
	m.published = append(m.published, evt)
	return nil
}

// ============================================================================
// Helper: build controllers
// ============================================================================

func newNodeController() (*controller.NodeController, *mockNodeRepo, *mockGraphStore, *mockEventPublisher) {
	repo := newMockNodeRepo()
	store := &mockGraphStore{}
	pub := &mockEventPublisher{}
	uc := usecase.NewNodeUseCase(repo, store, pub)
	return controller.NewNodeController(uc), repo, store, pub
}

func newEdgeController() (*controller.EdgeController, *mockEdgeRepo, *mockGraphStore, *mockEventPublisher) {
	repo := newMockEdgeRepo()
	store := &mockGraphStore{}
	pub := &mockEventPublisher{}
	uc := usecase.NewEdgeUseCase(repo, store, pub)
	return controller.NewEdgeController(uc), repo, store, pub
}

func newQueryController(store *mockGraphStore) *controller.QueryController {
	uc := usecase.NewQueryUseCase(store)
	return controller.NewQueryController(uc)
}

func newSyncController(store *mockGraphStore, logRepo *mockSyncLogRepo) *controller.SyncController {
	uc := usecase.NewSyncUseCase(store, logRepo)
	return controller.NewSyncController(uc)
}

// ============================================================================
// NodeController Tests
// ============================================================================

func TestNodeController_List(t *testing.T) {
	t.Run("happy path with label filter", func(t *testing.T) {
		ctrl, repo, _, _ := newNodeController()
		repo.items["person:1"] = &entity.GraphNode{UID: "person:1", Label: entity.LabelPerson, Name: "Zhang Zhongjing"}
		repo.items["person:2"] = &entity.GraphNode{UID: "person:2", Label: entity.LabelPerson, Name: "Hua Tuo"}

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/nodes?label=Person&page=1&page_size=10")

		ctrl.List(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)
	})

	t.Run("search by keyword", func(t *testing.T) {
		ctrl, repo, _, _ := newNodeController()
		repo.items["person:1"] = &entity.GraphNode{UID: "person:1", Label: entity.LabelPerson, Name: "Zhang Zhongjing"}
		repo.items["person:2"] = &entity.GraphNode{UID: "person:2", Label: entity.LabelPerson, Name: "Hua Tuo"}

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/nodes?keyword=Zhang")

		ctrl.List(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)
	})

	t.Run("empty list returns 200", func(t *testing.T) {
		ctrl, _, _, _ := newNodeController()

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/nodes")

		ctrl.List(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("repo error returns 500", func(t *testing.T) {
		ctrl, repo, _, _ := newNodeController()
		repo.listErr = errors.New("db down")

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/nodes")

		ctrl.List(ctx(), rc)
		assertStatusCode(t, rc, http.StatusInternalServerError)
	})
}

func TestNodeController_Create(t *testing.T) {
	t.Run("happy path returns 201", func(t *testing.T) {
		ctrl, _, _, _ := newNodeController()

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/graph/nodes")
		rc.Request.SetBody([]byte(`{"uid":"person:1","label":"Person","name":"Zhang Zhongjing"}`))

		ctrl.Create(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusCreated)
		assert.Equal(t, "created", body.Message)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		ctrl, _, _, _ := newNodeController()

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/graph/nodes")
		rc.Request.SetBody([]byte(`{not-json`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("missing uid returns 400", func(t *testing.T) {
		ctrl, _, _, _ := newNodeController()

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/graph/nodes")
		rc.Request.SetBody([]byte(`{"label":"Person","name":"X"}`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("invalid label returns 400", func(t *testing.T) {
		ctrl, _, _, _ := newNodeController()

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/graph/nodes")
		rc.Request.SetBody([]byte(`{"uid":"u","label":"Robot","name":"n"}`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("repo error returns 500", func(t *testing.T) {
		ctrl, repo, _, _ := newNodeController()
		repo.createErr = errors.New("db down")

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/graph/nodes")
		rc.Request.SetBody([]byte(`{"uid":"u","label":"Person","name":"n"}`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusInternalServerError)
	})
}

func TestNodeController_Get(t *testing.T) {
	t.Run("found returns 200 with data", func(t *testing.T) {
		ctrl, repo, _, _ := newNodeController()
		repo.items["person:1"] = &entity.GraphNode{UID: "person:1", Label: entity.LabelPerson, Name: "Zhang Zhongjing"}

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/nodes/person:1")
		setParam(rc, "uid", "person:1")

		ctrl.Get(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)

		data, ok := body.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "person:1", data["uid"])
		assert.Equal(t, "Zhang Zhongjing", data["name"])
	})

	t.Run("missing uid param returns 400", func(t *testing.T) {
		ctrl, _, _, _ := newNodeController()

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/nodes/")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		ctrl, _, _, _ := newNodeController()

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/nodes/missing")
		setParam(rc, "uid", "missing")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})
}

func TestNodeController_Update(t *testing.T) {
	t.Run("happy path returns 200", func(t *testing.T) {
		ctrl, repo, _, _ := newNodeController()
		repo.items["person:1"] = &entity.GraphNode{UID: "person:1", Label: entity.LabelPerson, Name: "Old Name"}

		rc := newRC()
		rc.Request.SetMethod("PUT")
		rc.Request.SetRequestURI("/api/v1/graph/nodes/person:1")
		setParam(rc, "uid", "person:1")
		rc.Request.SetBody([]byte(`{"label":"Person","name":"New Name"}`))

		ctrl.Update(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)
	})

	t.Run("missing uid param returns 400", func(t *testing.T) {
		ctrl, _, _, _ := newNodeController()

		rc := newRC()
		rc.Request.SetMethod("PUT")
		rc.Request.SetRequestURI("/api/v1/graph/nodes/")
		rc.Request.SetBody([]byte(`{"name":"x"}`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		ctrl, _, _, _ := newNodeController()

		rc := newRC()
		rc.Request.SetMethod("PUT")
		rc.Request.SetRequestURI("/api/v1/graph/nodes/person:1")
		setParam(rc, "uid", "person:1")
		rc.Request.SetBody([]byte(`{not-json`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		ctrl, _, _, _ := newNodeController()

		rc := newRC()
		rc.Request.SetMethod("PUT")
		rc.Request.SetRequestURI("/api/v1/graph/nodes/missing")
		setParam(rc, "uid", "missing")
		rc.Request.SetBody([]byte(`{"name":"x"}`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})
}

func TestNodeController_Delete(t *testing.T) {
	t.Run("happy path returns 204", func(t *testing.T) {
		ctrl, repo, _, _ := newNodeController()
		repo.items["person:1"] = &entity.GraphNode{UID: "person:1", Label: entity.LabelPerson, Name: "Zhang Zhongjing"}

		rc := newRC()
		rc.Request.SetMethod("DELETE")
		rc.Request.SetRequestURI("/api/v1/graph/nodes/person:1")
		setParam(rc, "uid", "person:1")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNoContent)
	})

	t.Run("missing uid param returns 400", func(t *testing.T) {
		ctrl, _, _, _ := newNodeController()

		rc := newRC()
		rc.Request.SetMethod("DELETE")
		rc.Request.SetRequestURI("/api/v1/graph/nodes/")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("repo error returns 500", func(t *testing.T) {
		ctrl, repo, _, _ := newNodeController()
		repo.deleteErr = errors.New("db down")

		rc := newRC()
		rc.Request.SetMethod("DELETE")
		rc.Request.SetRequestURI("/api/v1/graph/nodes/person:1")
		setParam(rc, "uid", "person:1")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusInternalServerError)
	})
}

// ============================================================================
// EdgeController Tests
// ============================================================================

func TestEdgeController_List(t *testing.T) {
	t.Run("happy path with source_uid filter", func(t *testing.T) {
		ctrl, repo, _, _ := newEdgeController()
		repo.items["edge:1"] = &entity.GraphEdge{UID: "edge:1", Type: entity.RelAuthored, SourceUID: "person:1", TargetUID: "classic:1"}
		repo.items["edge:2"] = &entity.GraphEdge{UID: "edge:2", Type: entity.RelAuthored, SourceUID: "person:1", TargetUID: "classic:2"}

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/edges?source_uid=person:1&page=1&page_size=10")

		ctrl.List(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)
	})

	t.Run("list by target_uid", func(t *testing.T) {
		ctrl, repo, _, _ := newEdgeController()
		repo.items["edge:1"] = &entity.GraphEdge{UID: "edge:1", Type: entity.RelAuthored, SourceUID: "person:1", TargetUID: "classic:1"}

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/edges?target_uid=classic:1")

		ctrl.List(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("list by type", func(t *testing.T) {
		ctrl, repo, _, _ := newEdgeController()
		repo.items["edge:1"] = &entity.GraphEdge{UID: "edge:1", Type: entity.RelAuthored, SourceUID: "person:1", TargetUID: "classic:1"}

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/edges?type=AUTHORED")

		ctrl.List(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("empty list returns 200", func(t *testing.T) {
		ctrl, _, _, _ := newEdgeController()

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/edges")

		ctrl.List(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
	})

	t.Run("repo error returns 500", func(t *testing.T) {
		ctrl, repo, _, _ := newEdgeController()
		repo.listBySrcErr = errors.New("db down")

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/edges?source_uid=p:1")

		ctrl.List(ctx(), rc)
		assertStatusCode(t, rc, http.StatusInternalServerError)
	})
}

func TestEdgeController_Create(t *testing.T) {
	t.Run("happy path returns 201", func(t *testing.T) {
		ctrl, _, _, _ := newEdgeController()

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/graph/edges")
		rc.Request.SetBody([]byte(`{"uid":"edge:1","type":"AUTHORED","source_uid":"person:1","target_uid":"classic:1"}`))

		ctrl.Create(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusCreated)
		assert.Equal(t, "created", body.Message)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		ctrl, _, _, _ := newEdgeController()

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/graph/edges")
		rc.Request.SetBody([]byte(`{not-json`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("missing type returns 400", func(t *testing.T) {
		ctrl, _, _, _ := newEdgeController()

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/graph/edges")
		rc.Request.SetBody([]byte(`{"uid":"e","source_uid":"s","target_uid":"t"}`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("invalid type returns 400", func(t *testing.T) {
		ctrl, _, _, _ := newEdgeController()

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/graph/edges")
		rc.Request.SetBody([]byte(`{"uid":"e","type":"FRIEND_OF","source_uid":"s","target_uid":"t"}`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("repo error returns 500", func(t *testing.T) {
		ctrl, repo, _, _ := newEdgeController()
		repo.createErr = errors.New("db down")

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/graph/edges")
		rc.Request.SetBody([]byte(`{"uid":"e","type":"AUTHORED","source_uid":"s","target_uid":"t"}`))

		ctrl.Create(ctx(), rc)
		assertStatusCode(t, rc, http.StatusInternalServerError)
	})
}

func TestEdgeController_Get(t *testing.T) {
	t.Run("found returns 200 with data", func(t *testing.T) {
		ctrl, repo, _, _ := newEdgeController()
		repo.items["edge:1"] = &entity.GraphEdge{UID: "edge:1", Type: entity.RelAuthored, SourceUID: "person:1", TargetUID: "classic:1"}

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/edges/edge:1")
		setParam(rc, "uid", "edge:1")

		ctrl.Get(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)

		data, ok := body.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "edge:1", data["uid"])
		assert.Equal(t, "AUTHORED", data["type"])
	})

	t.Run("missing uid param returns 400", func(t *testing.T) {
		ctrl, _, _, _ := newEdgeController()

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/edges/")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		ctrl, _, _, _ := newEdgeController()

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/edges/missing")
		setParam(rc, "uid", "missing")

		ctrl.Get(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})
}

func TestEdgeController_Update(t *testing.T) {
	t.Run("happy path returns 200", func(t *testing.T) {
		ctrl, repo, _, _ := newEdgeController()
		repo.items["edge:1"] = &entity.GraphEdge{UID: "edge:1", Type: entity.RelAuthored, SourceUID: "person:1", TargetUID: "classic:1"}

		rc := newRC()
		rc.Request.SetMethod("PUT")
		rc.Request.SetRequestURI("/api/v1/graph/edges/edge:1")
		setParam(rc, "uid", "edge:1")
		rc.Request.SetBody([]byte(`{"type":"DISCIPLED","source_uid":"person:2","target_uid":"person:3"}`))

		ctrl.Update(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)
	})

	t.Run("missing uid param returns 400", func(t *testing.T) {
		ctrl, _, _, _ := newEdgeController()

		rc := newRC()
		rc.Request.SetMethod("PUT")
		rc.Request.SetRequestURI("/api/v1/graph/edges/")
		rc.Request.SetBody([]byte(`{"type":"AUTHORED"}`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("invalid JSON returns 400", func(t *testing.T) {
		ctrl, _, _, _ := newEdgeController()

		rc := newRC()
		rc.Request.SetMethod("PUT")
		rc.Request.SetRequestURI("/api/v1/graph/edges/edge:1")
		setParam(rc, "uid", "edge:1")
		rc.Request.SetBody([]byte(`{not-json`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		ctrl, _, _, _ := newEdgeController()

		rc := newRC()
		rc.Request.SetMethod("PUT")
		rc.Request.SetRequestURI("/api/v1/graph/edges/missing")
		setParam(rc, "uid", "missing")
		rc.Request.SetBody([]byte(`{"type":"AUTHORED"}`))

		ctrl.Update(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})
}

func TestEdgeController_Delete(t *testing.T) {
	t.Run("happy path returns 204", func(t *testing.T) {
		ctrl, repo, _, _ := newEdgeController()
		repo.items["edge:1"] = &entity.GraphEdge{UID: "edge:1", Type: entity.RelAuthored, SourceUID: "person:1", TargetUID: "classic:1"}

		rc := newRC()
		rc.Request.SetMethod("DELETE")
		rc.Request.SetRequestURI("/api/v1/graph/edges/edge:1")
		setParam(rc, "uid", "edge:1")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNoContent)
	})

	t.Run("missing uid param returns 400", func(t *testing.T) {
		ctrl, _, _, _ := newEdgeController()

		rc := newRC()
		rc.Request.SetMethod("DELETE")
		rc.Request.SetRequestURI("/api/v1/graph/edges/")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("repo error returns 500", func(t *testing.T) {
		ctrl, repo, _, _ := newEdgeController()
		repo.deleteErr = errors.New("db down")

		rc := newRC()
		rc.Request.SetMethod("DELETE")
		rc.Request.SetRequestURI("/api/v1/graph/edges/edge:1")
		setParam(rc, "uid", "edge:1")

		ctrl.Delete(ctx(), rc)
		assertStatusCode(t, rc, http.StatusInternalServerError)
	})
}

// ============================================================================
// QueryController Tests
// ============================================================================

func TestQueryController_GetPersonWorks(t *testing.T) {
	t.Run("happy path returns 200 with works", func(t *testing.T) {
		store := &mockGraphStore{
			personWorksOut: []entity.GraphNodeView{
				{UID: "c1", Label: entity.LabelClassic, Name: "Shanghan Lun"},
				{UID: "c2", Label: entity.LabelClassic, Name: "Jingui Yaolue"},
			},
		}
		ctrl := newQueryController(store)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/persons/person:1/works")
		setParam(rc, "uid", "person:1")

		ctrl.GetPersonWorks(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)

		items, ok := body.Data.([]interface{})
		require.True(t, ok)
		assert.Len(t, items, 2)
	})

	t.Run("missing uid param returns 400", func(t *testing.T) {
		ctrl := newQueryController(&mockGraphStore{})

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/persons//works")

		ctrl.GetPersonWorks(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockGraphStore{personWorksErr: errors.New("neo4j down")}
		ctrl := newQueryController(store)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/persons/p:1/works")
		setParam(rc, "uid", "p:1")

		ctrl.GetPersonWorks(ctx(), rc)
		assertStatusCode(t, rc, http.StatusInternalServerError)
	})
}

func TestQueryController_GetSchoolLineage(t *testing.T) {
	t.Run("happy path returns 200", func(t *testing.T) {
		store := &mockGraphStore{
			lineageOut: &entity.LineagePath{
				Path: entity.GraphPath{
					Nodes: []entity.GraphNodeView{{UID: "a"}, {UID: "b"}},
					Hops:  1,
				},
				Generations: []int{0, 1},
			},
		}
		ctrl := newQueryController(store)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/schools/Yishan/lineage?max_depth=4")
		setParam(rc, "name", "Yishan")

		ctrl.GetSchoolLineage(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)
	})

	t.Run("missing name param returns 400", func(t *testing.T) {
		ctrl := newQueryController(&mockGraphStore{})

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/schools//lineage")

		ctrl.GetSchoolLineage(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		ctrl := newQueryController(&mockGraphStore{})

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/schools/Unknown/lineage")
		setParam(rc, "name", "Unknown")

		ctrl.GetSchoolLineage(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockGraphStore{lineageErr: errors.New("boom")}
		ctrl := newQueryController(store)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/schools/X/lineage")
		setParam(rc, "name", "X")

		ctrl.GetSchoolLineage(ctx(), rc)
		assertStatusCode(t, rc, http.StatusInternalServerError)
	})
}

func TestQueryController_FindShortestPath(t *testing.T) {
	t.Run("happy path returns 200", func(t *testing.T) {
		store := &mockGraphStore{
			queryPathOut: &entity.GraphPath{
				Nodes: []entity.GraphNodeView{{UID: "a"}, {UID: "b"}},
				Edges: []entity.GraphEdgeView{{UID: "e1", SourceUID: "a", TargetUID: "b"}},
				Hops:  1,
			},
		}
		ctrl := newQueryController(store)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/paths/shortest?start_uid=a&end_uid=b&max_hops=5")

		ctrl.FindShortestPath(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)

		data, ok := body.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(1), data["hops"])
	})

	t.Run("missing start_uid returns 400", func(t *testing.T) {
		ctrl := newQueryController(&mockGraphStore{})

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/paths/shortest?end_uid=b")

		ctrl.FindShortestPath(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("missing end_uid returns 400", func(t *testing.T) {
		ctrl := newQueryController(&mockGraphStore{})

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/paths/shortest?start_uid=a")

		ctrl.FindShortestPath(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not reachable returns 404", func(t *testing.T) {
		ctrl := newQueryController(&mockGraphStore{})

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/paths/shortest?start_uid=a&end_uid=b")

		ctrl.FindShortestPath(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockGraphStore{queryPathErr: errors.New("boom")}
		ctrl := newQueryController(store)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/paths/shortest?start_uid=a&end_uid=b")

		ctrl.FindShortestPath(ctx(), rc)
		assertStatusCode(t, rc, http.StatusInternalServerError)
	})
}

func TestQueryController_GetDynastyFigures(t *testing.T) {
	t.Run("happy path returns 200", func(t *testing.T) {
		store := &mockGraphStore{
			dynastyOut: []entity.FigureWithWorks{
				{
					Person:  entity.GraphNodeView{UID: "p1", Label: entity.LabelPerson},
					Works:   []entity.GraphNodeView{{UID: "c1", Label: entity.LabelClassic}},
					Schools: []entity.GraphNodeView{{UID: "s1", Label: entity.LabelSchool}},
				},
			},
		}
		ctrl := newQueryController(store)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/dynasties/Han/figures")
		setParam(rc, "name", "Han")

		ctrl.GetDynastyFigures(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)
	})

	t.Run("missing name param returns 400", func(t *testing.T) {
		ctrl := newQueryController(&mockGraphStore{})

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/dynasties//figures")

		ctrl.GetDynastyFigures(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockGraphStore{dynastyErr: errors.New("boom")}
		ctrl := newQueryController(store)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/dynasties/Han/figures")
		setParam(rc, "name", "Han")

		ctrl.GetDynastyFigures(ctx(), rc)
		assertStatusCode(t, rc, http.StatusInternalServerError)
	})
}

func TestQueryController_GetPrescriptionDetail(t *testing.T) {
	t.Run("happy path returns 200", func(t *testing.T) {
		store := &mockGraphStore{
			prescriptionOut: &entity.PrescriptionGraph{
				Prescription: entity.GraphNodeView{UID: "rx1", Label: entity.LabelPrescription},
				Medicines:    []entity.GraphNodeView{{UID: "m1", Label: entity.LabelMedicine}},
				Diseases:     []entity.GraphNodeView{{UID: "d1", Label: entity.LabelDisease}},
			},
		}
		ctrl := newQueryController(store)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/prescriptions/rx1/detail")
		setParam(rc, "uid", "rx1")

		ctrl.GetPrescriptionDetail(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)

		data, ok := body.Data.(map[string]interface{})
		require.True(t, ok)
		prescription, ok := data["prescription"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "rx1", prescription["uid"])
	})

	t.Run("missing uid param returns 400", func(t *testing.T) {
		ctrl := newQueryController(&mockGraphStore{})

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/prescriptions//detail")

		ctrl.GetPrescriptionDetail(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		ctrl := newQueryController(&mockGraphStore{})

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/prescriptions/missing/detail")
		setParam(rc, "uid", "missing")

		ctrl.GetPrescriptionDetail(ctx(), rc)
		assertStatusCode(t, rc, http.StatusNotFound)
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockGraphStore{prescriptionErr: errors.New("boom")}
		ctrl := newQueryController(store)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/prescriptions/rx1/detail")
		setParam(rc, "uid", "rx1")

		ctrl.GetPrescriptionDetail(ctx(), rc)
		assertStatusCode(t, rc, http.StatusInternalServerError)
	})
}

func TestQueryController_GetSubgraph(t *testing.T) {
	t.Run("happy path returns 200", func(t *testing.T) {
		store := &mockGraphStore{
			subgraphOut: &entity.Subgraph{
				Nodes: []entity.GraphNodeView{{UID: "a"}, {UID: "b"}},
				Edges: []entity.GraphEdgeView{{UID: "e1", SourceUID: "a", TargetUID: "b"}},
			},
		}
		ctrl := newQueryController(store)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/subgraph?center_uid=a&depth=2&limit=50")

		ctrl.GetSubgraph(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)
	})

	t.Run("missing center_uid returns 400", func(t *testing.T) {
		ctrl := newQueryController(&mockGraphStore{})

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/subgraph")

		ctrl.GetSubgraph(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("limit exceeds 300 returns 400", func(t *testing.T) {
		ctrl := newQueryController(&mockGraphStore{})

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/subgraph?center_uid=a&limit=301")

		ctrl.GetSubgraph(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("store returns nil subgraph returns empty 200", func(t *testing.T) {
		ctrl := newQueryController(&mockGraphStore{})

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/subgraph?center_uid=a")

		ctrl.GetSubgraph(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockGraphStore{subgraphErr: errors.New("boom")}
		ctrl := newQueryController(store)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/subgraph?center_uid=a")

		ctrl.GetSubgraph(ctx(), rc)
		assertStatusCode(t, rc, http.StatusInternalServerError)
	})
}

func TestQueryController_Search(t *testing.T) {
	t.Run("happy path returns 200 with results", func(t *testing.T) {
		store := &mockGraphStore{
			searchOut: []entity.GraphNodeView{
				{UID: "n1", Label: entity.LabelPerson, Name: "Zhang Zhongjing"},
				{UID: "n2", Label: entity.LabelPerson, Name: "Zhang Wei"},
			},
		}
		ctrl := newQueryController(store)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/search?keyword=Zhang&label=Person&limit=10")

		ctrl.Search(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)

		data, ok := body.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "Zhang", data["keyword"])
		assert.Equal(t, "Person", data["label"])
	})

	t.Run("missing keyword returns 400", func(t *testing.T) {
		ctrl := newQueryController(&mockGraphStore{})

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/search")

		ctrl.Search(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
	})

	t.Run("empty result returns 200 with zero total", func(t *testing.T) {
		store := &mockGraphStore{searchOut: []entity.GraphNodeView{}}
		ctrl := newQueryController(store)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/search?keyword=xyz")

		ctrl.Search(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)

		data, ok := body.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(0), data["total"])
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockGraphStore{searchErr: errors.New("boom")}
		ctrl := newQueryController(store)

		rc := newRC()
		rc.Request.SetMethod("GET")
		rc.Request.SetRequestURI("/api/v1/graph/search?keyword=test")

		ctrl.Search(ctx(), rc)
		assertStatusCode(t, rc, http.StatusInternalServerError)
	})
}

// ============================================================================
// SyncController Tests
// ============================================================================

func TestSyncController_TriggerSync(t *testing.T) {
	t.Run("happy path with default limit returns 200", func(t *testing.T) {
		store := &mockGraphStore{}
		logRepo := newMockSyncLogRepo()
		logRepo.pendingOut = []entity.GraphSyncLog{
			{
				SourceType: entity.SourceHistory,
				SourceID:   "person:1",
				EntityType: entity.LabelPerson,
				Action:     entity.ActionUpsert,
				Status:     entity.SyncStatusPending,
			},
		}
		logRepo.pendingOut[0].ID = 1
		ctrl := newSyncController(store, logRepo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/graph/sync")

		ctrl.TriggerSync(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)

		data, ok := body.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(1), data["succeeded"])
		assert.Equal(t, float64(0), data["failed"])
	})

	t.Run("happy path with explicit limit", func(t *testing.T) {
		store := &mockGraphStore{}
		logRepo := newMockSyncLogRepo()
		logRepo.pendingOut = []entity.GraphSyncLog{
			{
				SourceType: entity.SourceHistory,
				SourceID:   "person:1",
				EntityType: entity.LabelPerson,
				Action:     entity.ActionUpsert,
				Status:     entity.SyncStatusPending,
			},
			{
				SourceType: entity.SourceHistory,
				SourceID:   "person:2",
				EntityType: entity.LabelPerson,
				Action:     entity.ActionDelete,
				Status:     entity.SyncStatusPending,
			},
		}
		logRepo.pendingOut[0].ID = 1
		logRepo.pendingOut[1].ID = 2
		ctrl := newSyncController(store, logRepo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/graph/sync?limit=1")

		ctrl.TriggerSync(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)

		data, ok := body.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(1), data["succeeded"])
	})

	t.Run("empty pending logs returns zeros", func(t *testing.T) {
		store := &mockGraphStore{}
		logRepo := newMockSyncLogRepo()
		ctrl := newSyncController(store, logRepo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/graph/sync")

		ctrl.TriggerSync(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), body.Code)

		data, ok := body.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(0), data["succeeded"])
		assert.Equal(t, float64(0), data["failed"])
	})

	t.Run("upsert failure counts as failed", func(t *testing.T) {
		store := &mockGraphStore{upsertNodeErr: errors.New("neo4j down")}
		logRepo := newMockSyncLogRepo()
		logRepo.pendingOut = []entity.GraphSyncLog{
			{
				SourceType: entity.SourceHistory,
				SourceID:   "person:1",
				EntityType: entity.LabelPerson,
				Action:     entity.ActionUpsert,
				Status:     entity.SyncStatusPending,
			},
		}
		logRepo.pendingOut[0].ID = 1
		ctrl := newSyncController(store, logRepo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/graph/sync")

		ctrl.TriggerSync(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)

		data, ok := body.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(0), data["succeeded"])
		assert.Equal(t, float64(1), data["failed"])
	})

	t.Run("delete failure counts as failed", func(t *testing.T) {
		store := &mockGraphStore{deleteNodeErr: errors.New("neo4j down")}
		logRepo := newMockSyncLogRepo()
		logRepo.pendingOut = []entity.GraphSyncLog{
			{
				SourceType: entity.SourceHistory,
				SourceID:   "person:1",
				EntityType: entity.LabelPerson,
				Action:     entity.ActionDelete,
				Status:     entity.SyncStatusPending,
			},
		}
		logRepo.pendingOut[0].ID = 1
		ctrl := newSyncController(store, logRepo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/graph/sync")

		ctrl.TriggerSync(ctx(), rc)
		body := assertStatusCode(t, rc, http.StatusOK)

		data, ok := body.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(0), data["succeeded"])
		assert.Equal(t, float64(1), data["failed"])
	})

	t.Run("ListPending error returns 500", func(t *testing.T) {
		store := &mockGraphStore{}
		logRepo := newMockSyncLogRepo()
		logRepo.listPendingErr = errors.New("db down")
		ctrl := newSyncController(store, logRepo)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/graph/sync")

		ctrl.TriggerSync(ctx(), rc)
		assertStatusCode(t, rc, http.StatusInternalServerError)
	})
}

// ============================================================================
// Error Response Format Tests
// ============================================================================

func TestErrorResponseFormat_NotFound(t *testing.T) {
	ctrl, _, _, _ := newNodeController()

	rc := newRC()
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/graph/nodes/missing")
	setParam(rc, "uid", "missing")

	ctrl.Get(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusNotFound)
	assert.Equal(t, int(errno.NotFound), body.Code)
	assert.Contains(t, body.Message, "not found")
}

func TestErrorResponseFormat_InvalidParams(t *testing.T) {
	ctrl, _, _, _ := newNodeController()

	rc := newRC()
	rc.Request.SetMethod("POST")
	rc.Request.SetRequestURI("/api/v1/graph/nodes")
	rc.Request.SetBody([]byte(`{"uid":"u","label":"Robot","name":"n"}`))

	ctrl.Create(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusBadRequest)
	assert.Equal(t, int(errno.InvalidParams), body.Code)
}

func TestErrorResponseFormat_InvalidJSON(t *testing.T) {
	ctrl, _, _, _ := newNodeController()

	rc := newRC()
	rc.Request.SetMethod("POST")
	rc.Request.SetRequestURI("/api/v1/graph/nodes")
	rc.Request.SetBody([]byte(`{not-json`))

	ctrl.Create(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusBadRequest)
	assert.Equal(t, int(errno.InvalidParams), body.Code)
}

// ============================================================================
// Path Parameter Extraction Tests
// ============================================================================

func TestPathParamExtraction_NodeUID(t *testing.T) {
	ctrl, repo, _, _ := newNodeController()
	repo.items["person:123"] = &entity.GraphNode{UID: "person:123", Label: entity.LabelPerson, Name: "Test Person"}

	rc := newRC()
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/graph/nodes/person:123")
	setParam(rc, "uid", "person:123")

	ctrl.Get(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusOK)
	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "person:123", data["uid"])
	assert.Equal(t, "Test Person", data["name"])
}

func TestPathParamExtraction_EdgeUID(t *testing.T) {
	ctrl, repo, _, _ := newEdgeController()
	repo.items["edge:abc"] = &entity.GraphEdge{UID: "edge:abc", Type: entity.RelAuthored, SourceUID: "s", TargetUID: "t"}

	rc := newRC()
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/graph/edges/edge:abc")
	setParam(rc, "uid", "edge:abc")

	ctrl.Get(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusOK)
	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "edge:abc", data["uid"])
}

func TestPathParamExtraction_SchoolName(t *testing.T) {
	store := &mockGraphStore{
		lineageOut: &entity.LineagePath{
			Path: entity.GraphPath{Hops: 1},
		},
	}
	ctrl := newQueryController(store)

	rc := newRC()
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/graph/schools/Yishan/lineage")
	setParam(rc, "name", "Yishan")

	ctrl.GetSchoolLineage(ctx(), rc)
	assertStatusCode(t, rc, http.StatusOK)
}

func TestPathParamExtraction_DynastyName(t *testing.T) {
	store := &mockGraphStore{
		dynastyOut: []entity.FigureWithWorks{},
	}
	ctrl := newQueryController(store)

	rc := newRC()
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/graph/dynasties/Han/figures")
	setParam(rc, "name", "Han")

	ctrl.GetDynastyFigures(ctx(), rc)
	assertStatusCode(t, rc, http.StatusOK)
}

func TestPathParamExtraction_PrescriptionUID(t *testing.T) {
	store := &mockGraphStore{
		prescriptionOut: &entity.PrescriptionGraph{
			Prescription: entity.GraphNodeView{UID: "rx99", Label: entity.LabelPrescription},
		},
	}
	ctrl := newQueryController(store)

	rc := newRC()
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/graph/prescriptions/rx99/detail")
	setParam(rc, "uid", "rx99")

	ctrl.GetPrescriptionDetail(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusOK)
	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	prescription, ok := data["prescription"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "rx99", prescription["uid"])
}

func TestPathParamExtraction_PersonUID(t *testing.T) {
	store := &mockGraphStore{
		personWorksOut: []entity.GraphNodeView{
			{UID: "c1", Label: entity.LabelClassic, Name: "Test Classic"},
		},
	}
	ctrl := newQueryController(store)

	rc := newRC()
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/graph/persons/person:42/works")
	setParam(rc, "uid", "person:42")

	ctrl.GetPersonWorks(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusOK)
	assert.Equal(t, int(errno.OK), body.Code)
}

func TestPathParamExtraction_UpdateNodeUID(t *testing.T) {
	ctrl, repo, _, _ := newNodeController()
	repo.items["person:7"] = &entity.GraphNode{UID: "person:7", Label: entity.LabelPerson, Name: "Original"}

	rc := newRC()
	rc.Request.SetMethod("PUT")
	rc.Request.SetRequestURI("/api/v1/graph/nodes/person:7")
	setParam(rc, "uid", "person:7")
	rc.Request.SetBody([]byte(`{"label":"Person","name":"Updated"}`))

	ctrl.Update(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusOK)
	assert.Equal(t, int(errno.OK), body.Code)
}

func TestPathParamExtraction_DeleteNodeUID(t *testing.T) {
	ctrl, repo, _, _ := newNodeController()
	repo.items["person:8"] = &entity.GraphNode{UID: "person:8", Label: entity.LabelPerson, Name: "ToDelete"}

	rc := newRC()
	rc.Request.SetMethod("DELETE")
	rc.Request.SetRequestURI("/api/v1/graph/nodes/person:8")
	setParam(rc, "uid", "person:8")

	ctrl.Delete(ctx(), rc)
	assertStatusCode(t, rc, http.StatusNoContent)
}

func TestPathParamExtraction_UpdateEdgeUID(t *testing.T) {
	ctrl, repo, _, _ := newEdgeController()
	repo.items["edge:xyz"] = &entity.GraphEdge{UID: "edge:xyz", Type: entity.RelAuthored, SourceUID: "s", TargetUID: "t"}

	rc := newRC()
	rc.Request.SetMethod("PUT")
	rc.Request.SetRequestURI("/api/v1/graph/edges/edge:xyz")
	setParam(rc, "uid", "edge:xyz")
	rc.Request.SetBody([]byte(`{"type":"DISCIPLED","source_uid":"s2","target_uid":"t2"}`))

	ctrl.Update(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusOK)
	assert.Equal(t, int(errno.OK), body.Code)
}

func TestPathParamExtraction_DeleteEdgeUID(t *testing.T) {
	ctrl, repo, _, _ := newEdgeController()
	repo.items["edge:del"] = &entity.GraphEdge{UID: "edge:del", Type: entity.RelAuthored, SourceUID: "s", TargetUID: "t"}

	rc := newRC()
	rc.Request.SetMethod("DELETE")
	rc.Request.SetRequestURI("/api/v1/graph/edges/edge:del")
	setParam(rc, "uid", "edge:del")

	ctrl.Delete(ctx(), rc)
	assertStatusCode(t, rc, http.StatusNoContent)
}

// ============================================================================
// Query Parameter Extraction Tests
// ============================================================================

func TestQueryParamExtraction_NodeListLabelFilter(t *testing.T) {
	ctrl, repo, _, _ := newNodeController()
	repo.items["p1"] = &entity.GraphNode{UID: "p1", Label: entity.LabelPerson, Name: "Zhang"}
	repo.items["c1"] = &entity.GraphNode{UID: "c1", Label: entity.LabelClassic, Name: "Shanghan Lun"}

	rc := newRC()
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/graph/nodes?label=Person")

	ctrl.List(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusOK)
	assert.Equal(t, int(errno.OK), body.Code)
}

func TestQueryParamExtraction_NodeListPagination(t *testing.T) {
	ctrl, _, _, _ := newNodeController()

	rc := newRC()
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/graph/nodes?page=2&page_size=5")

	ctrl.List(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusOK)
	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(2), data["page"])
	assert.Equal(t, float64(5), data["page_size"])
}

func TestQueryParamExtraction_EdgeListSourceUID(t *testing.T) {
	ctrl, repo, _, _ := newEdgeController()
	repo.items["e1"] = &entity.GraphEdge{UID: "e1", Type: entity.RelAuthored, SourceUID: "person:1", TargetUID: "classic:1"}

	rc := newRC()
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/graph/edges?source_uid=person:1")

	ctrl.List(ctx(), rc)
	assertStatusCode(t, rc, http.StatusOK)
}

func TestQueryParamExtraction_EdgeListTypeFilter(t *testing.T) {
	ctrl, repo, _, _ := newEdgeController()
	repo.items["e1"] = &entity.GraphEdge{UID: "e1", Type: entity.RelAuthored, SourceUID: "s", TargetUID: "t"}

	rc := newRC()
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/graph/edges?type=AUTHORED")

	ctrl.List(ctx(), rc)
	assertStatusCode(t, rc, http.StatusOK)
}

func TestQueryParamExtraction_ShortestPathParams(t *testing.T) {
	store := &mockGraphStore{
		queryPathOut: &entity.GraphPath{
			Nodes: []entity.GraphNodeView{{UID: "a"}, {UID: "b"}, {UID: "c"}},
			Edges: []entity.GraphEdgeView{
				{UID: "e1", SourceUID: "a", TargetUID: "b"},
				{UID: "e2", SourceUID: "b", TargetUID: "c"},
			},
			Hops: 2,
		},
	}
	ctrl := newQueryController(store)

	rc := newRC()
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/graph/paths/shortest?start_uid=a&end_uid=c&max_hops=5")

	ctrl.FindShortestPath(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusOK)
	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(2), data["hops"])
}

func TestQueryParamExtraction_SubgraphDefaults(t *testing.T) {
	store := &mockGraphStore{
		subgraphOut: &entity.Subgraph{
			Nodes: []entity.GraphNodeView{{UID: "center"}},
		},
	}
	ctrl := newQueryController(store)

	rc := newRC()
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/graph/subgraph?center_uid=center")

	ctrl.GetSubgraph(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusOK)
	assert.Equal(t, int(errno.OK), body.Code)
}

func TestQueryParamExtraction_SearchKeywordAndLabel(t *testing.T) {
	store := &mockGraphStore{
		searchOut: []entity.GraphNodeView{
			{UID: "n1", Label: entity.LabelPerson, Name: "Zhang Zhongjing"},
		},
	}
	ctrl := newQueryController(store)

	rc := newRC()
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/graph/search?keyword=Zhang&label=Person&limit=5")

	ctrl.Search(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusOK)
	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Zhang", data["keyword"])
	assert.Equal(t, "Person", data["label"])
	assert.Equal(t, float64(1), data["total"])
}

func TestQueryParamExtraction_SyncLimit(t *testing.T) {
	store := &mockGraphStore{}
	logRepo := newMockSyncLogRepo()
	logRepo.pendingOut = []entity.GraphSyncLog{
		{SourceType: entity.SourceHistory, SourceID: "p1", EntityType: entity.LabelPerson, Action: entity.ActionUpsert, Status: entity.SyncStatusPending},
		{SourceType: entity.SourceHistory, SourceID: "p2", EntityType: entity.LabelPerson, Action: entity.ActionUpsert, Status: entity.SyncStatusPending},
		{SourceType: entity.SourceHistory, SourceID: "p3", EntityType: entity.LabelPerson, Action: entity.ActionUpsert, Status: entity.SyncStatusPending},
	}
	logRepo.pendingOut[0].ID = 1
	logRepo.pendingOut[1].ID = 2
	logRepo.pendingOut[2].ID = 3
	ctrl := newSyncController(store, logRepo)

	rc := newRC()
	rc.Request.SetMethod("POST")
	rc.Request.SetRequestURI("/api/v1/graph/sync?limit=2")

	ctrl.TriggerSync(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusOK)
	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(2), data["succeeded"])
	assert.Equal(t, float64(0), data["failed"])
}

// ============================================================================
// Additional Error Case Tests
// ============================================================================

func TestNodeController_Update_RepoError(t *testing.T) {
	ctrl, repo, _, _ := newNodeController()
	repo.items["person:1"] = &entity.GraphNode{UID: "person:1", Label: entity.LabelPerson, Name: "Existing"}
	repo.updateErr = errors.New("db down")

	rc := newRC()
	rc.Request.SetMethod("PUT")
	rc.Request.SetRequestURI("/api/v1/graph/nodes/person:1")
	setParam(rc, "uid", "person:1")
	rc.Request.SetBody([]byte(`{"name":"x"}`))

	ctrl.Update(ctx(), rc)
	assertStatusCode(t, rc, http.StatusInternalServerError)
}

func TestNodeController_Delete_NotFound_NoError(t *testing.T) {
	ctrl, _, _, _ := newNodeController()

	rc := newRC()
	rc.Request.SetMethod("DELETE")
	rc.Request.SetRequestURI("/api/v1/graph/nodes/nonexistent")
	setParam(rc, "uid", "nonexistent")

	ctrl.Delete(ctx(), rc)
	assertStatusCode(t, rc, http.StatusNoContent)
}

func TestEdgeController_Update_RepoError(t *testing.T) {
	ctrl, repo, _, _ := newEdgeController()
	repo.items["edge:1"] = &entity.GraphEdge{UID: "edge:1", Type: entity.RelAuthored, SourceUID: "s", TargetUID: "t"}
	repo.updateErr = errors.New("db down")

	rc := newRC()
	rc.Request.SetMethod("PUT")
	rc.Request.SetRequestURI("/api/v1/graph/edges/edge:1")
	setParam(rc, "uid", "edge:1")
	rc.Request.SetBody([]byte(`{"type":"DISCIPLED"}`))

	ctrl.Update(ctx(), rc)
	assertStatusCode(t, rc, http.StatusInternalServerError)
}

func TestEdgeController_Delete_NotFound_NoError(t *testing.T) {
	ctrl, _, _, _ := newEdgeController()

	rc := newRC()
	rc.Request.SetMethod("DELETE")
	rc.Request.SetRequestURI("/api/v1/graph/edges/nonexistent")
	setParam(rc, "uid", "nonexistent")

	ctrl.Delete(ctx(), rc)
	assertStatusCode(t, rc, http.StatusNoContent)
}

func TestQueryController_FindShortestPath_DefaultMaxHops(t *testing.T) {
	store := &mockGraphStore{
		queryPathOut: &entity.GraphPath{Hops: 3},
	}
	ctrl := newQueryController(store)

	rc := newRC()
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/graph/paths/shortest?start_uid=a&end_uid=b")

	ctrl.FindShortestPath(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusOK)
	assert.Equal(t, int(errno.OK), body.Code)
}

func TestQueryController_GetSubgraph_DefaultDepth(t *testing.T) {
	store := &mockGraphStore{
		subgraphOut: &entity.Subgraph{
			Nodes: []entity.GraphNodeView{{UID: "a"}},
			Edges: []entity.GraphEdgeView{{UID: "e1", SourceUID: "a", TargetUID: "b"}},
		},
	}
	ctrl := newQueryController(store)

	rc := newRC()
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/graph/subgraph?center_uid=a")

	ctrl.GetSubgraph(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusOK)
	assert.Equal(t, int(errno.OK), body.Code)
}

func TestQueryController_GetSchoolLineage_DefaultMaxDepth(t *testing.T) {
	store := &mockGraphStore{
		lineageOut: &entity.LineagePath{
			Path: entity.GraphPath{
				Nodes: []entity.GraphNodeView{{UID: "a"}, {UID: "b"}},
				Hops:  1,
			},
			Generations: []int{0, 1},
		},
	}
	ctrl := newQueryController(store)

	rc := newRC()
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/graph/schools/Yishan/lineage")
	setParam(rc, "name", "Yishan")

	ctrl.GetSchoolLineage(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusOK)
	assert.Equal(t, int(errno.OK), body.Code)
}

func TestSyncController_TriggerSync_DefaultLimit(t *testing.T) {
	store := &mockGraphStore{}
	logRepo := newMockSyncLogRepo()
	logRepo.pendingOut = []entity.GraphSyncLog{
		{SourceType: entity.SourceHistory, SourceID: "p1", EntityType: entity.LabelPerson, Action: entity.ActionUpsert, Status: entity.SyncStatusPending},
	}
	logRepo.pendingOut[0].ID = 1
	ctrl := newSyncController(store, logRepo)

	rc := newRC()
	rc.Request.SetMethod("POST")
	rc.Request.SetRequestURI("/api/v1/graph/sync")

	ctrl.TriggerSync(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusOK)
	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), data["succeeded"])
	assert.Equal(t, float64(0), data["failed"])
	assert.Equal(t, float64(0), data["pending"])
}

// ============================================================================
// Validation & Edge Case Tests
// ============================================================================

func TestNodeController_Create_EmptyName(t *testing.T) {
	ctrl, _, _, _ := newNodeController()

	rc := newRC()
	rc.Request.SetMethod("POST")
	rc.Request.SetRequestURI("/api/v1/graph/nodes")
	rc.Request.SetBody([]byte(`{"uid":"u","label":"Person","name":""}`))

	ctrl.Create(ctx(), rc)
	assertStatusCode(t, rc, http.StatusBadRequest)
}

func TestNodeController_Create_MissingAllFields(t *testing.T) {
	ctrl, _, _, _ := newNodeController()

	rc := newRC()
	rc.Request.SetMethod("POST")
	rc.Request.SetRequestURI("/api/v1/graph/nodes")
	rc.Request.SetBody([]byte(`{}`))

	ctrl.Create(ctx(), rc)
	assertStatusCode(t, rc, http.StatusBadRequest)
}

func TestEdgeController_Create_MissingSourceTarget(t *testing.T) {
	ctrl, _, _, _ := newEdgeController()

	rc := newRC()
	rc.Request.SetMethod("POST")
	rc.Request.SetRequestURI("/api/v1/graph/edges")
	rc.Request.SetBody([]byte(`{"uid":"e","type":"AUTHORED"}`))

	ctrl.Create(ctx(), rc)
	assertStatusCode(t, rc, http.StatusBadRequest)
}

func TestEdgeController_Create_InvalidType(t *testing.T) {
	ctrl, _, _, _ := newEdgeController()

	rc := newRC()
	rc.Request.SetMethod("POST")
	rc.Request.SetRequestURI("/api/v1/graph/edges")
	rc.Request.SetBody([]byte(`{"uid":"e","type":"INVALID_TYPE","source_uid":"s","target_uid":"t"}`))

	ctrl.Create(ctx(), rc)
	assertStatusCode(t, rc, http.StatusBadRequest)
}

func TestNodeController_List_InvalidLabel(t *testing.T) {
	ctrl, _, _, _ := newNodeController()

	rc := newRC()
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/graph/nodes?label=InvalidLabel")

	ctrl.List(ctx(), rc)
	assertStatusCode(t, rc, http.StatusBadRequest)
}

func TestEdgeController_List_InvalidType(t *testing.T) {
	ctrl, _, _, _ := newEdgeController()

	rc := newRC()
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/graph/edges?type=INVALID_TYPE")

	ctrl.List(ctx(), rc)
	assertStatusCode(t, rc, http.StatusBadRequest)
}

// ============================================================================
// Response Structure Verification
// ============================================================================

func TestResponseStructure_CreateNode(t *testing.T) {
	ctrl, _, _, _ := newNodeController()

	rc := newRC()
	rc.Request.SetMethod("POST")
	rc.Request.SetRequestURI("/api/v1/graph/nodes")
	rc.Request.SetBody([]byte(`{"uid":"person:1","label":"Person","name":"Zhang Zhongjing"}`))

	ctrl.Create(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusCreated)
	assert.Equal(t, 0, body.Code)
	assert.Equal(t, "created", body.Message)
	assert.NotNil(t, body.Data)
}

func TestResponseStructure_UpdateNode(t *testing.T) {
	ctrl, repo, _, _ := newNodeController()
	repo.items["person:1"] = &entity.GraphNode{UID: "person:1", Label: entity.LabelPerson, Name: "Old"}

	rc := newRC()
	rc.Request.SetMethod("PUT")
	rc.Request.SetRequestURI("/api/v1/graph/nodes/person:1")
	setParam(rc, "uid", "person:1")
	rc.Request.SetBody([]byte(`{"label":"Person","name":"New"}`))

	ctrl.Update(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusOK)
	assert.Equal(t, 0, body.Code)
	assert.Equal(t, "ok", body.Message)
	assert.NotNil(t, body.Data)
}

func TestResponseStructure_ListNode(t *testing.T) {
	ctrl, repo, _, _ := newNodeController()
	repo.items["p1"] = &entity.GraphNode{UID: "p1", Label: entity.LabelPerson, Name: "Zhang"}

	rc := newRC()
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/graph/nodes?label=Person")

	ctrl.List(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusOK)
	assert.Equal(t, 0, body.Code)
	assert.Equal(t, "ok", body.Message)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, data, "items")
	assert.Contains(t, data, "total")
	assert.Contains(t, data, "page")
	assert.Contains(t, data, "page_size")
}

func TestResponseStructure_CreateEdge(t *testing.T) {
	ctrl, _, _, _ := newEdgeController()

	rc := newRC()
	rc.Request.SetMethod("POST")
	rc.Request.SetRequestURI("/api/v1/graph/edges")
	rc.Request.SetBody([]byte(`{"uid":"edge:1","type":"AUTHORED","source_uid":"person:1","target_uid":"classic:1"}`))

	ctrl.Create(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusCreated)
	assert.Equal(t, 0, body.Code)
	assert.Equal(t, "created", body.Message)
}

func TestResponseStructure_SyncResponse(t *testing.T) {
	store := &mockGraphStore{}
	logRepo := newMockSyncLogRepo()
	logRepo.pendingOut = []entity.GraphSyncLog{
		{SourceType: entity.SourceHistory, SourceID: "p1", EntityType: entity.LabelPerson, Action: entity.ActionUpsert, Status: entity.SyncStatusPending},
		{SourceType: entity.SourceHistory, SourceID: "p2", EntityType: entity.LabelPerson, Action: entity.ActionDelete, Status: entity.SyncStatusPending},
	}
	logRepo.pendingOut[0].ID = 1
	logRepo.pendingOut[1].ID = 2
	ctrl := newSyncController(store, logRepo)

	rc := newRC()
	rc.Request.SetMethod("POST")
	rc.Request.SetRequestURI("/api/v1/graph/sync")

	ctrl.TriggerSync(ctx(), rc)
	body := assertStatusCode(t, rc, http.StatusOK)
	assert.Equal(t, 0, body.Code)
	assert.Equal(t, "ok", body.Message)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, data, "succeeded")
	assert.Contains(t, data, "failed")
	assert.Contains(t, data, "pending")
}