package controller_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route/param"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/knowledge-service/internal/application/usecase"
	"tcm-history-ai/backend/knowledge-service/internal/controller"
	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
	"tcm-history-ai/backend/knowledge-service/internal/domain/event"
	"tcm-history-ai/backend/knowledge-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/gormutil"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

func init() {
	idgen.Init(1)
}

// ---------------------------------------------------------------------------
// Mock: DocumentRepository
// ---------------------------------------------------------------------------

type mockDocRepo struct {
	items  map[int64]*entity.Document
	byHash map[string]*entity.Document

	createErr error
	updateErr error
	deleteErr error
	listErr   error
	findErr   error
}

func newMockDocRepo() *mockDocRepo {
	return &mockDocRepo{
		items:  map[int64]*entity.Document{},
		byHash: map[string]*entity.Document{},
	}
}

func (m *mockDocRepo) Create(_ context.Context, d *entity.Document) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.items[d.ID] = d
	if d.ContentHash != "" {
		m.byHash[d.ContentHash] = d
	}
	return nil
}

func (m *mockDocRepo) Update(_ context.Context, d *entity.Document) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.items[d.ID] = d
	if d.ContentHash != "" {
		m.byHash[d.ContentHash] = d
	}
	return nil
}

func (m *mockDocRepo) Delete(_ context.Context, id int64) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if d, ok := m.items[id]; ok {
		if d.ContentHash != "" {
			delete(m.byHash, d.ContentHash)
		}
		delete(m.items, id)
	}
	return nil
}

func (m *mockDocRepo) FindByID(_ context.Context, id int64) (*entity.Document, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	if d, ok := m.items[id]; ok {
		clone := *d
		return &clone, nil
	}
	return nil, nil
}

func (m *mockDocRepo) FindByContentHash(_ context.Context, hash string) (*entity.Document, error) {
	if d, ok := m.byHash[hash]; ok {
		clone := *d
		return &clone, nil
	}
	return nil, nil
}

func (m *mockDocRepo) List(_ context.Context, p pagination.Params) ([]entity.Document, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	all := make([]entity.Document, 0, len(m.items))
	for _, d := range m.items {
		all = append(all, *d)
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

func (m *mockDocRepo) ListByClassic(_ context.Context, classicCode string, p pagination.Params) ([]entity.Document, int, error) {
	all := make([]entity.Document, 0, len(m.items))
	for _, d := range m.items {
		if d.ClassicCode == classicCode {
			all = append(all, *d)
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

// ---------------------------------------------------------------------------
// Mock: DocumentChunkRepository
// ---------------------------------------------------------------------------

type mockChunkRepo struct {
	items      map[int64]*entity.DocumentChunk
	itemsByCID map[string]*entity.DocumentChunk

	createErr      error
	batchCreateErr error
	updateErr      error
	deleteErr      error
	findErr        error
	findByCIDErr   error
	listErr        error
	listByIDsErr   error
}

func newMockChunkRepo() *mockChunkRepo {
	return &mockChunkRepo{
		items:      map[int64]*entity.DocumentChunk{},
		itemsByCID: map[string]*entity.DocumentChunk{},
	}
}

func (m *mockChunkRepo) store(c *entity.DocumentChunk) {
	m.items[c.ID] = c
	m.itemsByCID[c.ChunkID] = c
}

func (m *mockChunkRepo) Create(_ context.Context, c *entity.DocumentChunk) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.store(c)
	return nil
}

func (m *mockChunkRepo) BatchCreate(_ context.Context, chunks []entity.DocumentChunk) error {
	if m.batchCreateErr != nil {
		return m.batchCreateErr
	}
	for i := range chunks {
		c := chunks[i]
		m.store(&c)
	}
	return nil
}

func (m *mockChunkRepo) Update(_ context.Context, c *entity.DocumentChunk) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.store(c)
	return nil
}

func (m *mockChunkRepo) DeleteByDocument(_ context.Context, documentID int64) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for id, c := range m.items {
		if c.DocumentID == documentID {
			delete(m.items, id)
			delete(m.itemsByCID, c.ChunkID)
		}
	}
	return nil
}

func (m *mockChunkRepo) FindByID(_ context.Context, id int64) (*entity.DocumentChunk, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	if c, ok := m.items[id]; ok {
		clone := *c
		return &clone, nil
	}
	return nil, nil
}

func (m *mockChunkRepo) FindByChunkID(_ context.Context, chunkID string) (*entity.DocumentChunk, error) {
	if m.findByCIDErr != nil {
		return nil, m.findByCIDErr
	}
	if c, ok := m.itemsByCID[chunkID]; ok {
		clone := *c
		return &clone, nil
	}
	return nil, nil
}

func (m *mockChunkRepo) ListByDocument(_ context.Context, documentID int64, p pagination.Params) ([]entity.DocumentChunk, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	all := make([]entity.DocumentChunk, 0, len(m.items))
	for _, c := range m.items {
		if c.DocumentID == documentID {
			all = append(all, *c)
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

func (m *mockChunkRepo) ListByIDs(_ context.Context, ids []int64) ([]entity.DocumentChunk, error) {
	if m.listByIDsErr != nil {
		return nil, m.listByIDsErr
	}
	out := make([]entity.DocumentChunk, 0, len(ids))
	for _, id := range ids {
		if c, ok := m.items[id]; ok {
			out = append(out, *c)
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Mock: EmbeddingTaskRepository
// ---------------------------------------------------------------------------

type mockTaskRepo struct {
	items map[int64]*entity.EmbeddingTask

	createErr       error
	updateErr       error
	findErr         error
	findByDocErr    error
	listErr         error
	listByStatusErr error
}

func newMockTaskRepo() *mockTaskRepo {
	return &mockTaskRepo{items: map[int64]*entity.EmbeddingTask{}}
}

func (m *mockTaskRepo) Create(_ context.Context, t *entity.EmbeddingTask) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.items[t.ID] = t
	return nil
}

func (m *mockTaskRepo) Update(_ context.Context, t *entity.EmbeddingTask) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.items[t.ID] = t
	return nil
}

func (m *mockTaskRepo) FindByID(_ context.Context, id int64) (*entity.EmbeddingTask, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	if t, ok := m.items[id]; ok {
		clone := *t
		return &clone, nil
	}
	return nil, nil
}

func (m *mockTaskRepo) FindByDocumentID(_ context.Context, documentID int64) ([]entity.EmbeddingTask, error) {
	if m.findByDocErr != nil {
		return nil, m.findByDocErr
	}
	out := make([]entity.EmbeddingTask, 0)
	for _, t := range m.items {
		if t.DocumentID == documentID {
			out = append(out, *t)
		}
	}
	return out, nil
}

func (m *mockTaskRepo) List(_ context.Context, p pagination.Params) ([]entity.EmbeddingTask, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	all := make([]entity.EmbeddingTask, 0, len(m.items))
	for _, t := range m.items {
		all = append(all, *t)
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

func (m *mockTaskRepo) ListByStatus(_ context.Context, status string, p pagination.Params) ([]entity.EmbeddingTask, int, error) {
	if m.listByStatusErr != nil {
		return nil, 0, m.listByStatusErr
	}
	all := make([]entity.EmbeddingTask, 0, len(m.items))
	for _, t := range m.items {
		if status == "" || t.Status == status {
			all = append(all, *t)
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

// ---------------------------------------------------------------------------
// Mock: RagQueryRepository
// ---------------------------------------------------------------------------

type mockRagQueryRepo struct {
	items map[int64]*entity.RagQuery

	createErr error
	updateErr error
	findErr   error
}

func newMockRagQueryRepo() *mockRagQueryRepo {
	return &mockRagQueryRepo{items: map[int64]*entity.RagQuery{}}
}

func (m *mockRagQueryRepo) Create(_ context.Context, q *entity.RagQuery) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.items[q.ID] = q
	return nil
}

func (m *mockRagQueryRepo) Update(_ context.Context, q *entity.RagQuery) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.items[q.ID] = q
	return nil
}

func (m *mockRagQueryRepo) FindByID(_ context.Context, id int64) (*entity.RagQuery, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	if q, ok := m.items[id]; ok {
		clone := *q
		return &clone, nil
	}
	return nil, nil
}

func (m *mockRagQueryRepo) ListByUser(_ context.Context, _ int64, _ pagination.Params) ([]entity.RagQuery, int, error) {
	return nil, 0, nil
}

func (m *mockRagQueryRepo) ListBySession(_ context.Context, _ string, _ pagination.Params) ([]entity.RagQuery, int, error) {
	return nil, 0, nil
}

// ---------------------------------------------------------------------------
// Mock: EventPublisher
// ---------------------------------------------------------------------------

type mockEventPub struct {
	published  []event.Event
	publishErr error
}

func (m *mockEventPub) Publish(_ context.Context, evt event.Event) error {
	if m.publishErr != nil {
		return m.publishErr
	}
	m.published = append(m.published, evt)
	return nil
}

// ---------------------------------------------------------------------------
// Mock: VectorStore
// ---------------------------------------------------------------------------

type mockVectorStore struct {
	ensureErr error
	insertErr error
	deleteErr error
	searchErr error
	results   []service.VectorSearchResult
}

func (m *mockVectorStore) EnsureCollection(_ context.Context) error {
	return m.ensureErr
}
func (m *mockVectorStore) Insert(_ context.Context, _ []service.VectorRecord) error {
	return m.insertErr
}
func (m *mockVectorStore) DeleteByDoc(_ context.Context, _ int64) error {
	return m.deleteErr
}
func (m *mockVectorStore) Search(_ context.Context, _ []float32, _ int, _ service.SearchFilter) ([]service.VectorSearchResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.results, nil
}

// ---------------------------------------------------------------------------
// Mock: FullTextSearcher
// ---------------------------------------------------------------------------

type mockFullTextSearcher struct {
	searchErr error
	indexErr  error
	hits      []service.FullTextHit
}

func (m *mockFullTextSearcher) Search(_ context.Context, _ string, _ int, _ service.SearchFilter) ([]service.FullTextHit, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.hits, nil
}

func (m *mockFullTextSearcher) Index(_ context.Context, _ []service.FullTextDoc) error {
	return m.indexErr
}

// ---------------------------------------------------------------------------
// Mock: EmbeddingProvider
// ---------------------------------------------------------------------------

type mockEmbedder struct {
	embedErr error
	vectors  [][]float32
	modelStr string
	dimInt   int
}

func (m *mockEmbedder) Embed(_ context.Context, _ []string) ([][]float32, error) {
	if m.embedErr != nil {
		return nil, m.embedErr
	}
	return m.vectors, nil
}

func (m *mockEmbedder) Model() string { return m.modelStr }
func (m *mockEmbedder) Dim() int      { return m.dimInt }

// ---------------------------------------------------------------------------
// Mock: Reranker
// ---------------------------------------------------------------------------

type mockReranker struct {
	rerankErr error
	out       []service.RerankCandidate
}

func (m *mockReranker) Rerank(_ context.Context, _ string, candidates []service.RerankCandidate, topK int) ([]service.RerankCandidate, error) {
	if m.rerankErr != nil {
		return nil, m.rerankErr
	}
	if m.out != nil {
		return m.out, nil
	}
	if len(candidates) > topK {
		out := make([]service.RerankCandidate, topK)
		copy(out, candidates[:topK])
		return out, nil
	}
	out := make([]service.RerankCandidate, len(candidates))
	copy(out, candidates)
	return out, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestContext() *app.RequestContext {
	return app.NewContext(0)
}

func setPathParam(c *app.RequestContext, key, value string) {
	c.Params = param.Params{{Key: key, Value: value}}
}

func assertJSONError(t *testing.T, c *app.RequestContext, expectedCode int, expectedErrno errno.Errno) {
	t.Helper()
	resp := c.GetResponse()
	assert.Equal(t, expectedCode, resp.StatusCode())

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Body(), &body))
	assert.Equal(t, float64(int(expectedErrno)), body["code"])
}

func assertJSONSuccess(t *testing.T, c *app.RequestContext, expectedStatus int) map[string]interface{} {
	t.Helper()
	resp := c.GetResponse()
	assert.Equal(t, expectedStatus, resp.StatusCode())

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Body(), &body))
	assert.Equal(t, float64(0), body["code"])
	return body
}

func makeDoc(id int64, classicCode, title, status string) *entity.Document {
	return &entity.Document{
		BaseModel:  gormutil.BaseModel{ID: id},
		ClassicCode: classicCode,
		Title:       title,
		Status:      status,
	}
}

func makeChunk(id, docID int64, chunkID, content string) *entity.DocumentChunk {
	return &entity.DocumentChunk{
		ID:         id,
		DocumentID: docID,
		ChunkID:    chunkID,
		Content:    content,
	}
}

func makeTask(id, docID int64, status, stage string) *entity.EmbeddingTask {
	return &entity.EmbeddingTask{
		BaseModel:  gormutil.BaseModel{ID: id},
		DocumentID: docID,
		Status:     status,
		Stage:      stage,
	}
}

// ===========================================================================
// DocumentController Tests
// ===========================================================================

func TestDocumentController_List_Success(t *testing.T) {
	docRepo := newMockDocRepo()
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	docRepo.items[1] = makeDoc(1, "X", "T1", entity.DocumentStatusPending)
	docRepo.items[2] = makeDoc(2, "X", "T2", entity.DocumentStatusPending)

	c := newTestContext()
	c.Request.SetQueryString("page=1&page_size=10")
	ctx := context.Background()
	ctrl.List(ctx, c)

	body := assertJSONSuccess(t, c, consts.StatusOK)
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(2), data["total"])
}

func TestDocumentController_List_WithClassicCodeFilter(t *testing.T) {
	docRepo := newMockDocRepo()
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	docRepo.items[1] = makeDoc(1, "A", "T1", entity.DocumentStatusPending)
	docRepo.items[2] = makeDoc(2, "B", "T2", entity.DocumentStatusPending)

	c := newTestContext()
	c.Request.SetQueryString("classic_code=A&page=1&page_size=10")
	ctx := context.Background()
	ctrl.List(ctx, c)

	body := assertJSONSuccess(t, c, consts.StatusOK)
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), data["total"])
}

func TestDocumentController_List_Pagination(t *testing.T) {
	docRepo := newMockDocRepo()
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	for i := int64(1); i <= 5; i++ {
		docRepo.items[i] = makeDoc(i, "X", "T", entity.DocumentStatusPending)
	}

	c := newTestContext()
	c.Request.SetQueryString("page=1&page_size=2")
	ctx := context.Background()
	ctrl.List(ctx, c)

	body := assertJSONSuccess(t, c, consts.StatusOK)
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(5), data["total"])
	assert.Equal(t, float64(1), data["page"])

	items, ok := data["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 2)
}

func TestDocumentController_List_EmptyResult(t *testing.T) {
	docRepo := newMockDocRepo()
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	c := newTestContext()
	c.Request.SetQueryString("page=1&page_size=10")
	ctx := context.Background()
	ctrl.List(ctx, c)

	body := assertJSONSuccess(t, c, consts.StatusOK)
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(0), data["total"])
}

func TestDocumentController_List_RepoError(t *testing.T) {
	docRepo := newMockDocRepo()
	docRepo.listErr = errors.New("db error")
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	c := newTestContext()
	c.Request.SetQueryString("page=1&page_size=10")
	ctx := context.Background()
	ctrl.List(ctx, c)

	assertJSONError(t, c, consts.StatusInternalServerError, errno.InternalError)
}

func TestDocumentController_Create_Success(t *testing.T) {
	docRepo := newMockDocRepo()
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	c := newTestContext()
	c.Request.SetBodyString(`{"classic_code":"HuangDiNeiJing","title":"黄帝内经","version":"v1"}`)
	ctx := context.Background()
	ctrl.Create(ctx, c)

	body := assertJSONSuccess(t, c, consts.StatusCreated)
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, data["id"])
	assert.Equal(t, "黄帝内经", data["title"])
}

func TestDocumentController_Create_InvalidJSON(t *testing.T) {
	docRepo := newMockDocRepo()
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	c := newTestContext()
	c.Request.SetBodyString(`{invalid json}`)
	ctx := context.Background()
	ctrl.Create(ctx, c)

	assertJSONError(t, c, consts.StatusBadRequest, errno.InvalidParams)
}

func TestDocumentController_Create_MissingTitle(t *testing.T) {
	docRepo := newMockDocRepo()
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	c := newTestContext()
	c.Request.SetBodyString(`{"classic_code":"X"}`)
	ctx := context.Background()
	ctrl.Create(ctx, c)

	assertJSONError(t, c, consts.StatusBadRequest, errno.InvalidParams)
}

func TestDocumentController_Create_MissingClassicCode(t *testing.T) {
	docRepo := newMockDocRepo()
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	c := newTestContext()
	c.Request.SetBodyString(`{"title":"T"}`)
	ctx := context.Background()
	ctrl.Create(ctx, c)

	assertJSONError(t, c, consts.StatusBadRequest, errno.InvalidParams)
}

func TestDocumentController_Get_Success(t *testing.T) {
	docRepo := newMockDocRepo()
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	docRepo.items[1] = makeDoc(1, "X", "T", entity.DocumentStatusPending)

	c := newTestContext()
	setPathParam(c, "id", "1")
	ctx := context.Background()
	ctrl.Get(ctx, c)

	body := assertJSONSuccess(t, c, consts.StatusOK)
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), data["id"])
	assert.Equal(t, "T", data["title"])
}

func TestDocumentController_Get_NotFound(t *testing.T) {
	docRepo := newMockDocRepo()
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	c := newTestContext()
	setPathParam(c, "id", "9999")
	ctx := context.Background()
	ctrl.Get(ctx, c)

	assertJSONError(t, c, consts.StatusNotFound, errno.NotFound)
}

func TestDocumentController_Get_InvalidID(t *testing.T) {
	docRepo := newMockDocRepo()
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	c := newTestContext()
	setPathParam(c, "id", "abc")
	ctx := context.Background()
	ctrl.Get(ctx, c)

	assertJSONError(t, c, consts.StatusBadRequest, errno.InvalidParams)
}

func TestDocumentController_Get_RepoError(t *testing.T) {
	docRepo := newMockDocRepo()
	docRepo.findErr = errors.New("db error")
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	c := newTestContext()
	setPathParam(c, "id", "1")
	ctx := context.Background()
	ctrl.Get(ctx, c)

	assertJSONError(t, c, consts.StatusInternalServerError, errno.InternalError)
}

func TestDocumentController_Update_Success(t *testing.T) {
	docRepo := newMockDocRepo()
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	docRepo.items[1] = makeDoc(1, "X", "Old", entity.DocumentStatusPending)

	c := newTestContext()
	setPathParam(c, "id", "1")
	c.Request.SetBodyString(`{"classic_code":"Y","title":"New"}`)
	ctx := context.Background()
	ctrl.Update(ctx, c)

	body := assertJSONSuccess(t, c, consts.StatusOK)
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "New", data["title"])
	assert.Equal(t, "Y", data["classic_code"])
}

func TestDocumentController_Update_InvalidID(t *testing.T) {
	docRepo := newMockDocRepo()
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	c := newTestContext()
	setPathParam(c, "id", "bad")
	c.Request.SetBodyString(`{"title":"X"}`)
	ctx := context.Background()
	ctrl.Update(ctx, c)

	assertJSONError(t, c, consts.StatusBadRequest, errno.InvalidParams)
}

func TestDocumentController_Update_NotFound(t *testing.T) {
	docRepo := newMockDocRepo()
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	c := newTestContext()
	setPathParam(c, "id", "9999")
	c.Request.SetBodyString(`{"title":"X"}`)
	ctx := context.Background()
	ctrl.Update(ctx, c)

	assertJSONError(t, c, consts.StatusNotFound, errno.NotFound)
}

func TestDocumentController_Update_InvalidJSON(t *testing.T) {
	docRepo := newMockDocRepo()
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	docRepo.items[1] = makeDoc(1, "X", "T", entity.DocumentStatusPending)

	c := newTestContext()
	setPathParam(c, "id", "1")
	c.Request.SetBodyString(`not-json`)
	ctx := context.Background()
	ctrl.Update(ctx, c)

	assertJSONError(t, c, consts.StatusBadRequest, errno.InvalidParams)
}

func TestDocumentController_Update_RepoError(t *testing.T) {
	docRepo := newMockDocRepo()
	docRepo.items[1] = makeDoc(1, "X", "T", entity.DocumentStatusPending)
	docRepo.updateErr = errors.New("db error")
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	c := newTestContext()
	setPathParam(c, "id", "1")
	c.Request.SetBodyString(`{"title":"New"}`)
	ctx := context.Background()
	ctrl.Update(ctx, c)

	assertJSONError(t, c, consts.StatusInternalServerError, errno.InternalError)
}

func TestDocumentController_Delete_Success(t *testing.T) {
	docRepo := newMockDocRepo()
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	docRepo.items[1] = makeDoc(1, "X", "T", entity.DocumentStatusPending)

	c := newTestContext()
	setPathParam(c, "id", "1")
	ctx := context.Background()
	ctrl.Delete(ctx, c)

	assert.Equal(t, consts.StatusNoContent, c.GetResponse().StatusCode())
}

func TestDocumentController_Delete_NonExistent(t *testing.T) {
	docRepo := newMockDocRepo()
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	c := newTestContext()
	setPathParam(c, "id", "9999")
	ctx := context.Background()
	ctrl.Delete(ctx, c)

	assert.Equal(t, consts.StatusNoContent, c.GetResponse().StatusCode())
}

func TestDocumentController_Delete_InvalidID(t *testing.T) {
	docRepo := newMockDocRepo()
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	c := newTestContext()
	setPathParam(c, "id", "abc")
	ctx := context.Background()
	ctrl.Delete(ctx, c)

	assertJSONError(t, c, consts.StatusBadRequest, errno.InvalidParams)
}

func TestDocumentController_Delete_RepoError(t *testing.T) {
	docRepo := newMockDocRepo()
	docRepo.deleteErr = errors.New("delete fail")
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	docRepo.items[1] = makeDoc(1, "X", "T", entity.DocumentStatusPending)

	c := newTestContext()
	setPathParam(c, "id", "1")
	ctx := context.Background()
	ctrl.Delete(ctx, c)

	assertJSONError(t, c, consts.StatusInternalServerError, errno.InternalError)
}

func TestDocumentController_Delete_ZeroID(t *testing.T) {
	docRepo := newMockDocRepo()
	uc := usecase.NewDocumentUseCase(docRepo, &mockEventPub{}, nil)
	ctrl := controller.NewDocumentController(uc)

	c := newTestContext()
	setPathParam(c, "id", "0")
	ctx := context.Background()
	ctrl.Delete(ctx, c)

	assertJSONError(t, c, consts.StatusBadRequest, errno.InvalidParams)
}

// ===========================================================================
// ChunkController Tests
// ===========================================================================

func TestChunkController_ListByDocument_Success(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	docRepo := newMockDocRepo()
	uc := usecase.NewChunkUseCase(chunkRepo, docRepo)
	ctrl := controller.NewChunkController(uc)

	chunkRepo.store(makeChunk(1, 10, "c1", "原文1"))
	chunkRepo.store(makeChunk(2, 10, "c2", "原文2"))
	chunkRepo.store(makeChunk(3, 20, "c3", "other"))

	c := newTestContext()
	setPathParam(c, "id", "10")
	c.Request.SetQueryString("page=1&page_size=10")
	ctx := context.Background()
	ctrl.ListByDocument(ctx, c)

	body := assertJSONSuccess(t, c, consts.StatusOK)
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(2), data["total"])

	items, ok := data["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 2)
}

func TestChunkController_ListByDocument_Pagination(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	docRepo := newMockDocRepo()
	uc := usecase.NewChunkUseCase(chunkRepo, docRepo)
	ctrl := controller.NewChunkController(uc)

	for i := int64(1); i <= 5; i++ {
		chunkRepo.store(makeChunk(i, 10, "c"+string(rune('0'+i)), "x"))
	}

	c := newTestContext()
	setPathParam(c, "id", "10")
	c.Request.SetQueryString("page=1&page_size=2")
	ctx := context.Background()
	ctrl.ListByDocument(ctx, c)

	body := assertJSONSuccess(t, c, consts.StatusOK)
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(5), data["total"])
	assert.Equal(t, float64(1), data["page"])
}

func TestChunkController_ListByDocument_EmptyResult(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	docRepo := newMockDocRepo()
	uc := usecase.NewChunkUseCase(chunkRepo, docRepo)
	ctrl := controller.NewChunkController(uc)

	c := newTestContext()
	setPathParam(c, "id", "999")
	c.Request.SetQueryString("page=1&page_size=10")
	ctx := context.Background()
	ctrl.ListByDocument(ctx, c)

	body := assertJSONSuccess(t, c, consts.StatusOK)
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(0), data["total"])
}

func TestChunkController_ListByDocument_InvalidDocID(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	docRepo := newMockDocRepo()
	uc := usecase.NewChunkUseCase(chunkRepo, docRepo)
	ctrl := controller.NewChunkController(uc)

	c := newTestContext()
	setPathParam(c, "id", "bad")
	ctx := context.Background()
	ctrl.ListByDocument(ctx, c)

	assertJSONError(t, c, consts.StatusBadRequest, errno.InvalidParams)
}

func TestChunkController_ListByDocument_RepoError(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	chunkRepo.listErr = errors.New("boom")
	docRepo := newMockDocRepo()
	uc := usecase.NewChunkUseCase(chunkRepo, docRepo)
	ctrl := controller.NewChunkController(uc)

	c := newTestContext()
	setPathParam(c, "id", "10")
	c.Request.SetQueryString("page=1&page_size=10")
	ctx := context.Background()
	ctrl.ListByDocument(ctx, c)

	assertJSONError(t, c, consts.StatusInternalServerError, errno.InternalError)
}

func TestChunkController_Create_Success(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	docRepo := newMockDocRepo()
	uc := usecase.NewChunkUseCase(chunkRepo, docRepo)
	ctrl := controller.NewChunkController(uc)

	c := newTestContext()
	setPathParam(c, "id", "10")
	c.Request.SetBodyString(`{"content":"新的chunk内容","classic_code":"X"}`)
	ctx := context.Background()
	ctrl.Create(ctx, c)

	body := assertJSONSuccess(t, c, consts.StatusCreated)
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, data["id"])
	assert.Equal(t, "新的chunk内容", data["content"])
}

func TestChunkController_Create_InvalidDocID(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	docRepo := newMockDocRepo()
	uc := usecase.NewChunkUseCase(chunkRepo, docRepo)
	ctrl := controller.NewChunkController(uc)

	c := newTestContext()
	setPathParam(c, "id", "abc")
	c.Request.SetBodyString(`{"content":"test"}`)
	ctx := context.Background()
	ctrl.Create(ctx, c)

	assertJSONError(t, c, consts.StatusBadRequest, errno.InvalidParams)
}

func TestChunkController_Create_InvalidJSON(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	docRepo := newMockDocRepo()
	uc := usecase.NewChunkUseCase(chunkRepo, docRepo)
	ctrl := controller.NewChunkController(uc)

	c := newTestContext()
	setPathParam(c, "id", "10")
	c.Request.SetBodyString(`invalid`)
	ctx := context.Background()
	ctrl.Create(ctx, c)

	assertJSONError(t, c, consts.StatusBadRequest, errno.InvalidParams)
}

func TestChunkController_Create_EmptyContent(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	docRepo := newMockDocRepo()
	uc := usecase.NewChunkUseCase(chunkRepo, docRepo)
	ctrl := controller.NewChunkController(uc)

	c := newTestContext()
	setPathParam(c, "id", "10")
	c.Request.SetBodyString(`{"classic_code":"X"}`)
	ctx := context.Background()
	ctrl.Create(ctx, c)

	assertJSONError(t, c, consts.StatusBadRequest, errno.InvalidParams)
}

func TestChunkController_Create_RepoError(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	chunkRepo.createErr = errors.New("boom")
	docRepo := newMockDocRepo()
	uc := usecase.NewChunkUseCase(chunkRepo, docRepo)
	ctrl := controller.NewChunkController(uc)

	c := newTestContext()
	setPathParam(c, "id", "10")
	c.Request.SetBodyString(`{"content":"test"}`)
	ctx := context.Background()
	ctrl.Create(ctx, c)

	assertJSONError(t, c, consts.StatusInternalServerError, errno.InternalError)
}

func TestChunkController_Get_Success(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	docRepo := newMockDocRepo()
	uc := usecase.NewChunkUseCase(chunkRepo, docRepo)
	ctrl := controller.NewChunkController(uc)

	chunkRepo.store(&entity.DocumentChunk{
		ID: 1, DocumentID: 10, ChunkID: "c1", Content: "原文内容",
		ClassicCode: "X", ContentType: entity.ContentOriginal,
	})

	c := newTestContext()
	setPathParam(c, "id", "1")
	ctx := context.Background()
	ctrl.Get(ctx, c)

	body := assertJSONSuccess(t, c, consts.StatusOK)
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), data["id"])
	assert.Equal(t, "原文内容", data["content"])
}

func TestChunkController_Get_NotFound(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	docRepo := newMockDocRepo()
	uc := usecase.NewChunkUseCase(chunkRepo, docRepo)
	ctrl := controller.NewChunkController(uc)

	c := newTestContext()
	setPathParam(c, "id", "9999")
	ctx := context.Background()
	ctrl.Get(ctx, c)

	assertJSONError(t, c, consts.StatusNotFound, errno.NotFound)
}

func TestChunkController_Get_InvalidID(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	docRepo := newMockDocRepo()
	uc := usecase.NewChunkUseCase(chunkRepo, docRepo)
	ctrl := controller.NewChunkController(uc)

	c := newTestContext()
	setPathParam(c, "id", "abc")
	ctx := context.Background()
	ctrl.Get(ctx, c)

	assertJSONError(t, c, consts.StatusBadRequest, errno.InvalidParams)
}

// ===========================================================================
// TaskController Tests
// ===========================================================================

func TestTaskController_List_Success(t *testing.T) {
	taskRepo := newMockTaskRepo()
	uc := usecase.NewTaskUseCase(taskRepo)
	ctrl := controller.NewTaskController(uc)

	taskRepo.items[1] = makeTask(1, 10, entity.TaskStatusDone, entity.StageEmbed)
	taskRepo.items[2] = makeTask(2, 11, entity.TaskStatusRunning, entity.StageChunk)

	c := newTestContext()
	c.Request.SetQueryString("page=1&page_size=10")
	ctx := context.Background()
	ctrl.List(ctx, c)

	body := assertJSONSuccess(t, c, consts.StatusOK)
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(2), data["total"])
}

func TestTaskController_List_WithStatusFilter(t *testing.T) {
	taskRepo := newMockTaskRepo()
	uc := usecase.NewTaskUseCase(taskRepo)
	ctrl := controller.NewTaskController(uc)

	taskRepo.items[1] = makeTask(1, 10, entity.TaskStatusDone, "")
	taskRepo.items[2] = makeTask(2, 11, entity.TaskStatusRunning, "")

	c := newTestContext()
	c.Request.SetQueryString("status=done&page=1&page_size=10")
	ctx := context.Background()
	ctrl.List(ctx, c)

	body := assertJSONSuccess(t, c, consts.StatusOK)
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), data["total"])
}

func TestTaskController_List_Pagination(t *testing.T) {
	taskRepo := newMockTaskRepo()
	uc := usecase.NewTaskUseCase(taskRepo)
	ctrl := controller.NewTaskController(uc)

	for i := int64(1); i <= 4; i++ {
		taskRepo.items[i] = makeTask(i, 10, entity.TaskStatusQueued, "")
	}

	c := newTestContext()
	c.Request.SetQueryString("page=1&page_size=2")
	ctx := context.Background()
	ctrl.List(ctx, c)

	body := assertJSONSuccess(t, c, consts.StatusOK)
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(4), data["total"])

	items, ok := data["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 2)
}

func TestTaskController_List_DefaultPagination(t *testing.T) {
	taskRepo := newMockTaskRepo()
	for i := int64(1); i <= 3; i++ {
		taskRepo.items[i] = makeTask(i, 10, entity.TaskStatusDone, "")
	}
	uc := usecase.NewTaskUseCase(taskRepo)
	ctrl := controller.NewTaskController(uc)

	c := newTestContext()
	ctx := context.Background()
	ctrl.List(ctx, c)

	body := assertJSONSuccess(t, c, consts.StatusOK)
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(3), data["total"])

	items, ok := data["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 3)
}

func TestTaskController_List_RepoError(t *testing.T) {
	taskRepo := newMockTaskRepo()
	taskRepo.listErr = errors.New("boom")
	uc := usecase.NewTaskUseCase(taskRepo)
	ctrl := controller.NewTaskController(uc)

	c := newTestContext()
	c.Request.SetQueryString("page=1&page_size=10")
	ctx := context.Background()
	ctrl.List(ctx, c)

	assertJSONError(t, c, consts.StatusInternalServerError, errno.InternalError)
}

func TestTaskController_Get_Success(t *testing.T) {
	taskRepo := newMockTaskRepo()
	uc := usecase.NewTaskUseCase(taskRepo)
	ctrl := controller.NewTaskController(uc)

	taskRepo.items[1] = makeTask(1, 10, entity.TaskStatusDone, entity.StageEmbed)
	taskRepo.items[1].Progress = 100

	c := newTestContext()
	setPathParam(c, "id", "1")
	ctx := context.Background()
	ctrl.Get(ctx, c)

	body := assertJSONSuccess(t, c, consts.StatusOK)
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), data["id"])
	assert.Equal(t, entity.TaskStatusDone, data["status"])
}

func TestTaskController_Get_NotFound(t *testing.T) {
	taskRepo := newMockTaskRepo()
	uc := usecase.NewTaskUseCase(taskRepo)
	ctrl := controller.NewTaskController(uc)

	c := newTestContext()
	setPathParam(c, "id", "9999")
	ctx := context.Background()
	ctrl.Get(ctx, c)

	assertJSONError(t, c, consts.StatusNotFound, errno.NotFound)
}

func TestTaskController_Get_InvalidID(t *testing.T) {
	taskRepo := newMockTaskRepo()
	uc := usecase.NewTaskUseCase(taskRepo)
	ctrl := controller.NewTaskController(uc)

	c := newTestContext()
	setPathParam(c, "id", "abc")
	ctx := context.Background()
	ctrl.Get(ctx, c)

	assertJSONError(t, c, consts.StatusBadRequest, errno.InvalidParams)
}

func TestTaskController_Get_RepoError(t *testing.T) {
	taskRepo := newMockTaskRepo()
	taskRepo.findErr = errors.New("boom")
	uc := usecase.NewTaskUseCase(taskRepo)
	ctrl := controller.NewTaskController(uc)

	c := newTestContext()
	setPathParam(c, "id", "1")
	ctx := context.Background()
	ctrl.Get(ctx, c)

	assertJSONError(t, c, consts.StatusInternalServerError, errno.InternalError)
}

func TestTaskController_ListByDocument_Success(t *testing.T) {
	taskRepo := newMockTaskRepo()
	uc := usecase.NewTaskUseCase(taskRepo)
	ctrl := controller.NewTaskController(uc)

	taskRepo.items[1] = makeTask(1, 10, entity.TaskStatusDone, "")
	taskRepo.items[2] = makeTask(2, 10, entity.TaskStatusFailed, "")
	taskRepo.items[3] = makeTask(3, 20, entity.TaskStatusDone, "")

	c := newTestContext()
	setPathParam(c, "id", "10")
	ctx := context.Background()
	ctrl.ListByDocument(ctx, c)

	body := assertJSONSuccess(t, c, consts.StatusOK)
	data, ok := body["data"].([]interface{})
	require.True(t, ok)
	assert.Len(t, data, 2)
}

func TestTaskController_ListByDocument_Empty(t *testing.T) {
	taskRepo := newMockTaskRepo()
	uc := usecase.NewTaskUseCase(taskRepo)
	ctrl := controller.NewTaskController(uc)

	c := newTestContext()
	setPathParam(c, "id", "999")
	ctx := context.Background()
	ctrl.ListByDocument(ctx, c)

	body := assertJSONSuccess(t, c, consts.StatusOK)
	data, ok := body["data"].([]interface{})
	require.True(t, ok)
	assert.Empty(t, data)
}

func TestTaskController_ListByDocument_InvalidID(t *testing.T) {
	taskRepo := newMockTaskRepo()
	uc := usecase.NewTaskUseCase(taskRepo)
	ctrl := controller.NewTaskController(uc)

	c := newTestContext()
	setPathParam(c, "id", "bad")
	ctx := context.Background()
	ctrl.ListByDocument(ctx, c)

	assertJSONError(t, c, consts.StatusBadRequest, errno.InvalidParams)
}

func TestTaskController_ListByDocument_RepoError(t *testing.T) {
	taskRepo := newMockTaskRepo()
	taskRepo.findByDocErr = errors.New("boom")
	uc := usecase.NewTaskUseCase(taskRepo)
	ctrl := controller.NewTaskController(uc)

	c := newTestContext()
	setPathParam(c, "id", "10")
	ctx := context.Background()
	ctrl.ListByDocument(ctx, c)

	assertJSONError(t, c, consts.StatusInternalServerError, errno.InternalError)
}

// ===========================================================================
// RetrievalController Tests
// ===========================================================================

func TestRetrievalController_Retrieve_Success(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	queryRepo := newMockRagQueryRepo()
	vectorStore := &mockVectorStore{
		results: []service.VectorSearchResult{
			{ChunkID: "c1", Score: 0.9, DocID: 100},
		},
	}
	fullText := &mockFullTextSearcher{
		hits: []service.FullTextHit{
			{ChunkID: "c1", Score: 5.0, DocID: 100},
		},
	}
	embedder := &mockEmbedder{vectors: [][]float32{{0.1, 0.2}}, modelStr: "bge-large-zh"}
	reranker := &mockReranker{}

	chunkRepo.store(&entity.DocumentChunk{
		ID: 1, ChunkID: "c1", Content: "原文内容", DocumentID: 100,
		ClassicCode: "X", ContentType: entity.ContentOriginal,
	})

	uc := usecase.NewRetrievalUseCase(chunkRepo, queryRepo, vectorStore, fullText, embedder, reranker, 5, 60, 3)
	ctrl := controller.NewRetrievalController(uc)

	c := newTestContext()
	c.Request.SetBodyString(`{"query":"辨證","topk":3}`)
	c.Request.Header.Set("X-User-ID", "42")
	ctx := context.Background()
	ctrl.Retrieve(ctx, c)

	body := assertJSONSuccess(t, c, consts.StatusOK)
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "辨證", data["query"])
	assert.NotZero(t, data["query_log_id"])
}

func TestRetrievalController_Retrieve_EmptyQuery(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	queryRepo := newMockRagQueryRepo()
	vectorStore := &mockVectorStore{}
	fullText := &mockFullTextSearcher{}
	embedder := &mockEmbedder{}
	reranker := &mockReranker{}

	uc := usecase.NewRetrievalUseCase(chunkRepo, queryRepo, vectorStore, fullText, embedder, reranker, 5, 60, 3)
	ctrl := controller.NewRetrievalController(uc)

	c := newTestContext()
	c.Request.SetBodyString(`{"query":""}`)
	ctx := context.Background()
	ctrl.Retrieve(ctx, c)

	assertJSONError(t, c, consts.StatusBadRequest, errno.InvalidParams)
}

func TestRetrievalController_Retrieve_InvalidJSON(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	queryRepo := newMockRagQueryRepo()
	vectorStore := &mockVectorStore{}
	fullText := &mockFullTextSearcher{}
	embedder := &mockEmbedder{}
	reranker := &mockReranker{}

	uc := usecase.NewRetrievalUseCase(chunkRepo, queryRepo, vectorStore, fullText, embedder, reranker, 5, 60, 3)
	ctrl := controller.NewRetrievalController(uc)

	c := newTestContext()
	c.Request.SetBodyString(`not-json`)
	ctx := context.Background()
	ctrl.Retrieve(ctx, c)

	assertJSONError(t, c, consts.StatusBadRequest, errno.InvalidParams)
}

func TestRetrievalController_Retrieve_EmbedFailure(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	queryRepo := newMockRagQueryRepo()
	vectorStore := &mockVectorStore{}
	fullText := &mockFullTextSearcher{}
	embedder := &mockEmbedder{embedErr: errors.New("model down")}
	reranker := &mockReranker{}

	uc := usecase.NewRetrievalUseCase(chunkRepo, queryRepo, vectorStore, fullText, embedder, reranker, 5, 60, 3)
	ctrl := controller.NewRetrievalController(uc)

	c := newTestContext()
	c.Request.SetBodyString(`{"query":"test"}`)
	ctx := context.Background()
	ctrl.Retrieve(ctx, c)

	assertJSONError(t, c, consts.StatusServiceUnavailable, errno.DependencyUnavailable)
}

func TestRetrievalController_Feedback_Good(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	queryRepo := newMockRagQueryRepo()
	queryRepo.items[10] = &entity.RagQuery{ID: 10, QueryText: "test query"}

	uc := usecase.NewRetrievalUseCase(chunkRepo, queryRepo, nil, nil, nil, nil, 5, 60, 3)
	ctrl := controller.NewRetrievalController(uc)

	c := newTestContext()
	setPathParam(c, "id", "10")
	c.Request.SetBodyString(`{"feedback":"good"}`)
	ctx := context.Background()
	ctrl.Feedback(ctx, c)

	assert.Equal(t, consts.StatusNoContent, c.GetResponse().StatusCode())
	assert.Equal(t, entity.FeedbackGood, queryRepo.items[10].Feedback)
}

func TestRetrievalController_Feedback_Bad(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	queryRepo := newMockRagQueryRepo()
	queryRepo.items[10] = &entity.RagQuery{ID: 10, QueryText: "test query"}

	uc := usecase.NewRetrievalUseCase(chunkRepo, queryRepo, nil, nil, nil, nil, 5, 60, 3)
	ctrl := controller.NewRetrievalController(uc)

	c := newTestContext()
	setPathParam(c, "id", "10")
	c.Request.SetBodyString(`{"feedback":"bad"}`)
	ctx := context.Background()
	ctrl.Feedback(ctx, c)

	assert.Equal(t, consts.StatusNoContent, c.GetResponse().StatusCode())
	assert.Equal(t, entity.FeedbackBad, queryRepo.items[10].Feedback)
}

func TestRetrievalController_Feedback_InvalidValue(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	queryRepo := newMockRagQueryRepo()
	queryRepo.items[10] = &entity.RagQuery{ID: 10, QueryText: "test query"}

	uc := usecase.NewRetrievalUseCase(chunkRepo, queryRepo, nil, nil, nil, nil, 5, 60, 3)
	ctrl := controller.NewRetrievalController(uc)

	c := newTestContext()
	setPathParam(c, "id", "10")
	c.Request.SetBodyString(`{"feedback":"invalid"}`)
	ctx := context.Background()
	ctrl.Feedback(ctx, c)

	assertJSONError(t, c, consts.StatusBadRequest, errno.InvalidParams)
}

func TestRetrievalController_Feedback_NotFound(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	queryRepo := newMockRagQueryRepo()

	uc := usecase.NewRetrievalUseCase(chunkRepo, queryRepo, nil, nil, nil, nil, 5, 60, 3)
	ctrl := controller.NewRetrievalController(uc)

	c := newTestContext()
	setPathParam(c, "id", "9999")
	c.Request.SetBodyString(`{"feedback":"good"}`)
	ctx := context.Background()
	ctrl.Feedback(ctx, c)

	assertJSONError(t, c, consts.StatusNotFound, errno.NotFound)
}

func TestRetrievalController_Feedback_InvalidID(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	queryRepo := newMockRagQueryRepo()

	uc := usecase.NewRetrievalUseCase(chunkRepo, queryRepo, nil, nil, nil, nil, 5, 60, 3)
	ctrl := controller.NewRetrievalController(uc)

	c := newTestContext()
	setPathParam(c, "id", "abc")
	c.Request.SetBodyString(`{"feedback":"good"}`)
	ctx := context.Background()
	ctrl.Feedback(ctx, c)

	assertJSONError(t, c, consts.StatusBadRequest, errno.InvalidParams)
}

func TestRetrievalController_Feedback_InvalidJSON(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	queryRepo := newMockRagQueryRepo()
	queryRepo.items[10] = &entity.RagQuery{ID: 10, QueryText: "q"}

	uc := usecase.NewRetrievalUseCase(chunkRepo, queryRepo, nil, nil, nil, nil, 5, 60, 3)
	ctrl := controller.NewRetrievalController(uc)

	c := newTestContext()
	setPathParam(c, "id", "10")
	c.Request.SetBodyString(`{bad}`)
	ctx := context.Background()
	ctrl.Feedback(ctx, c)

	assertJSONError(t, c, consts.StatusBadRequest, errno.InvalidParams)
}

func TestRetrievalController_Feedback_UpdateError(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	queryRepo := newMockRagQueryRepo()
	queryRepo.items[10] = &entity.RagQuery{ID: 10, QueryText: "q"}
	queryRepo.updateErr = errors.New("update fail")

	uc := usecase.NewRetrievalUseCase(chunkRepo, queryRepo, nil, nil, nil, nil, 5, 60, 3)
	ctrl := controller.NewRetrievalController(uc)

	c := newTestContext()
	setPathParam(c, "id", "10")
	c.Request.SetBodyString(`{"feedback":"good"}`)
	ctx := context.Background()
	ctrl.Feedback(ctx, c)

	assertJSONError(t, c, consts.StatusInternalServerError, errno.InternalError)
}

// ===========================================================================
// IngestController Tests
// ===========================================================================

func TestIngestController_Ingest_Success(t *testing.T) {
	docRepo := newMockDocRepo()
	chunkRepo := newMockChunkRepo()
	taskRepo := newMockTaskRepo()
	vectorStore := &mockVectorStore{}
	embedder := &mockEmbedder{vectors: [][]float32{{0.1, 0.2}}, modelStr: "bge-large-zh"}
	pub := &mockEventPub{}

	docRepo.items[1] = makeDoc(1, "HuangDiNeiJing", "黄帝内经", entity.DocumentStatusPending)

	uc := usecase.NewIngestUseCase(docRepo, chunkRepo, taskRepo, vectorStore, embedder, nil, pub)
	ctrl := controller.NewIngestController(uc)

	c := newTestContext()
	setPathParam(c, "id", "1")
	c.Request.SetBodyString(`{"markdown_text":"辨證論治。"}`)
	ctx := context.Background()
	ctrl.Ingest(ctx, c)

	body := assertJSONSuccess(t, c, consts.StatusOK)
	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), data["document_id"])
	assert.Equal(t, "embedded", data["status"])

	assert.Len(t, pub.published, 2)
}

func TestIngestController_Ingest_DocumentNotFound(t *testing.T) {
	docRepo := newMockDocRepo()
	chunkRepo := newMockChunkRepo()
	taskRepo := newMockTaskRepo()
	vectorStore := &mockVectorStore{}
	embedder := &mockEmbedder{modelStr: "bge-large-zh"}
	pub := &mockEventPub{}

	uc := usecase.NewIngestUseCase(docRepo, chunkRepo, taskRepo, vectorStore, embedder, nil, pub)
	ctrl := controller.NewIngestController(uc)

	c := newTestContext()
	setPathParam(c, "id", "9999")
	c.Request.SetBodyString(`{"markdown_text":"test"}`)
	ctx := context.Background()
	ctrl.Ingest(ctx, c)

	assertJSONError(t, c, consts.StatusNotFound, errno.NotFound)
}

func TestIngestController_Ingest_InvalidDocID(t *testing.T) {
	docRepo := newMockDocRepo()
	chunkRepo := newMockChunkRepo()
	taskRepo := newMockTaskRepo()
	vectorStore := &mockVectorStore{}
	embedder := &mockEmbedder{}
	pub := &mockEventPub{}

	uc := usecase.NewIngestUseCase(docRepo, chunkRepo, taskRepo, vectorStore, embedder, nil, pub)
	ctrl := controller.NewIngestController(uc)

	c := newTestContext()
	setPathParam(c, "id", "abc")
	c.Request.SetBodyString(`{"markdown_text":"test"}`)
	ctx := context.Background()
	ctrl.Ingest(ctx, c)

	assertJSONError(t, c, consts.StatusBadRequest, errno.InvalidParams)
}

func TestIngestController_Ingest_EmptyMarkdownNoObjectKey(t *testing.T) {
	docRepo := newMockDocRepo()
	chunkRepo := newMockChunkRepo()
	taskRepo := newMockTaskRepo()
	vectorStore := &mockVectorStore{}
	embedder := &mockEmbedder{modelStr: "bge-large-zh"}
	pub := &mockEventPub{}

	docRepo.items[1] = makeDoc(1, "X", "T", entity.DocumentStatusPending)

	uc := usecase.NewIngestUseCase(docRepo, chunkRepo, taskRepo, vectorStore, embedder, nil, pub)
	ctrl := controller.NewIngestController(uc)

	c := newTestContext()
	setPathParam(c, "id", "1")
	c.Request.SetBodyString(`{}`)
	ctx := context.Background()
	ctrl.Ingest(ctx, c)

	assertJSONError(t, c, consts.StatusBadRequest, errno.InvalidParams)
}

func TestIngestController_Ingest_EmbedFailure(t *testing.T) {
	docRepo := newMockDocRepo()
	chunkRepo := newMockChunkRepo()
	taskRepo := newMockTaskRepo()
	vectorStore := &mockVectorStore{}
	embedder := &mockEmbedder{embedErr: errors.New("model down")}
	pub := &mockEventPub{}

	docRepo.items[1] = makeDoc(1, "X", "T", entity.DocumentStatusPending)

	uc := usecase.NewIngestUseCase(docRepo, chunkRepo, taskRepo, vectorStore, embedder, nil, pub)
	ctrl := controller.NewIngestController(uc)

	c := newTestContext()
	setPathParam(c, "id", "1")
	c.Request.SetBodyString(`{"markdown_text":"辨證論治。"}`)
	ctx := context.Background()
	ctrl.Ingest(ctx, c)

	assertJSONError(t, c, consts.StatusServiceUnavailable, errno.DependencyUnavailable)
}

func TestIngestController_Ingest_VectorInsertFailure(t *testing.T) {
	docRepo := newMockDocRepo()
	chunkRepo := newMockChunkRepo()
	taskRepo := newMockTaskRepo()
	vectorStore := &mockVectorStore{insertErr: errors.New("milvus down")}
	embedder := &mockEmbedder{vectors: [][]float32{{0.1, 0.2}}, modelStr: "bge-large-zh"}
	pub := &mockEventPub{}

	docRepo.items[1] = makeDoc(1, "X", "T", entity.DocumentStatusPending)

	uc := usecase.NewIngestUseCase(docRepo, chunkRepo, taskRepo, vectorStore, embedder, nil, pub)
	ctrl := controller.NewIngestController(uc)

	c := newTestContext()
	setPathParam(c, "id", "1")
	c.Request.SetBodyString(`{"markdown_text":"辨證論治。"}`)
	ctx := context.Background()
	ctrl.Ingest(ctx, c)

	assertJSONError(t, c, consts.StatusServiceUnavailable, errno.DependencyUnavailable)
}