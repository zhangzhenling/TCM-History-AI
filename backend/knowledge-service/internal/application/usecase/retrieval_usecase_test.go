package usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/knowledge-service/internal/application/dto"
	"tcm-history-ai/backend/knowledge-service/internal/application/usecase"
	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
	"tcm-history-ai/backend/knowledge-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// --- mock VectorStore ---

type mockVectorStore struct {
	ensureErr error
	insertErr error
	deleteErr error
	searchErr error
	results   []service.VectorSearchResult
}

func (m *mockVectorStore) EnsureCollection(_ context.Context) error { return m.ensureErr }
func (m *mockVectorStore) Insert(_ context.Context, _ []service.VectorRecord) error {
	return m.insertErr
}
func (m *mockVectorStore) DeleteByDoc(_ context.Context, _ int64) error { return m.deleteErr }
func (m *mockVectorStore) Search(_ context.Context, _ []float32, _ int, _ service.SearchFilter) ([]service.VectorSearchResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.results, nil
}

// --- mock FullTextSearcher ---

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
func (m *mockFullTextSearcher) Index(_ context.Context, _ []service.FullTextDoc) error { return m.indexErr }

// --- mock EmbeddingProvider ---

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

// --- mock Reranker ---

type mockReranker struct {
	rerankErr error
	in        []service.RerankCandidate
	out       []service.RerankCandidate
	called    int
	mu        sync.Mutex
}

func (m *mockReranker) Rerank(_ context.Context, _ string, candidates []service.RerankCandidate, topK int) ([]service.RerankCandidate, error) {
	m.mu.Lock()
	m.called++
	m.in = candidates
	m.mu.Unlock()
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

// --- mock RagQueryRepository ---

type mockRagQueryRepo struct {
	items       map[int64]*entity.RagQuery
	createErr   error
	updateErr   error
	findErr     error
	createCalls int
	mu          sync.Mutex
}

func newMockRagQueryRepo() *mockRagQueryRepo {
	return &mockRagQueryRepo{items: map[int64]*entity.RagQuery{}}
}

func (m *mockRagQueryRepo) Create(_ context.Context, q *entity.RagQuery) error {
	m.mu.Lock()
	m.createCalls++
	m.mu.Unlock()
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

// --- tests ---

func TestNewRetrievalUseCase_DefaultsWhenZero(t *testing.T) {
	uc := usecase.NewRetrievalUseCase(nil, nil, nil, nil, nil, nil, 0, 0, 0)
	require.NotNil(t, uc)
	// Smoke test: calling Retrieve with a nil embedder panics? No — embedder
	// is dereferenced. But we just need to ensure the constructor does not
	// blow up for invalid zero inputs.
}

func TestRetrievalUseCase_Retrieve_ValidationErrors(t *testing.T) {
	uc := usecase.NewRetrievalUseCase(nil, nil, nil, nil, nil, nil, 10, 60, 5)
	t.Run("nil request", func(t *testing.T) {
		resp, err := uc.Retrieve(context.Background(), nil, 0)
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})
	t.Run("empty query", func(t *testing.T) {
		resp, err := uc.Retrieve(context.Background(), &dto.RetrieveRequest{Query: ""}, 0)
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})
}

func TestRetrievalUseCase_Retrieve_EmbedFailure(t *testing.T) {
	embedder := &mockEmbedder{embedErr: errors.New("model down")}
	uc := usecase.NewRetrievalUseCase(
		newMockChunkRepo(), newMockRagQueryRepo(),
		&mockVectorStore{}, &mockFullTextSearcher{},
		embedder, &mockReranker{}, 5, 60, 3,
	)
	resp, err := uc.Retrieve(context.Background(), &dto.RetrieveRequest{Query: "q"}, 0)
	requireError(t, err, errno.DependencyUnavailable)
	assert.Nil(t, resp)
}

func TestRetrievalUseCase_Retrieve_BothRecallPathsFail(t *testing.T) {
	embedder := &mockEmbedder{vectors: [][]float32{{0.1, 0.2}}}
	vectorStore := &mockVectorStore{searchErr: errors.New("milvus down")}
	fullText := &mockFullTextSearcher{searchErr: errors.New("meili down")}
	uc := usecase.NewRetrievalUseCase(
		newMockChunkRepo(), newMockRagQueryRepo(),
		vectorStore, fullText,
		embedder, &mockReranker{}, 5, 60, 3,
	)
	resp, err := uc.Retrieve(context.Background(), &dto.RetrieveRequest{Query: "q"}, 0)
	requireError(t, err, errno.DependencyUnavailable)
	assert.Nil(t, resp)
}

func TestRetrievalUseCase_Retrieve_HappyPath(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	// Pre-seed chunks that both retrieval paths will return.
	chunks := []*entity.DocumentChunk{
		{ID: 1, ChunkID: "c1", Content: "原文1", DocumentID: 100},
		{ID: 2, ChunkID: "c2", Content: "原文2", DocumentID: 101},
	}
	for _, c := range chunks {
		chunkRepo.store(c)
	}

	embedder := &mockEmbedder{vectors: [][]float32{{0.1, 0.2}}}
	vectorStore := &mockVectorStore{
		results: []service.VectorSearchResult{
			{ChunkID: "c1", Score: 0.9, DocID: 100},
			{ChunkID: "c2", Score: 0.7, DocID: 101},
		},
	}
	fullText := &mockFullTextSearcher{
		hits: []service.FullTextHit{
			{ChunkID: "c1", Score: 5.0, DocID: 100},
			{ChunkID: "c2", Score: 3.0, DocID: 101},
		},
	}
	reranker := &mockReranker{}
	queryRepo := newMockRagQueryRepo()

	uc := usecase.NewRetrievalUseCase(
		chunkRepo, queryRepo,
		vectorStore, fullText,
		embedder, reranker, 5, 60, 3,
	)

	resp, err := uc.Retrieve(context.Background(), &dto.RetrieveRequest{
		Query:     "辨證",
		TopK:      2,
		SessionID: "sess-1",
	}, 7)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "辨證", resp.Query)
	assert.Equal(t, 2, resp.TopK)
	assert.NotZero(t, resp.QueryLogID)
	assert.GreaterOrEqual(t, resp.Total, 1)
	require.Equal(t, 1, reranker.called)
	// rag_queries log was persisted.
	require.Equal(t, 1, queryRepo.createCalls)
}

func TestRetrievalUseCase_Retrieve_RerankFailureFallback(t *testing.T) {
	// When the reranker fails, the usecase should fall back to RRF ordering
	// instead of erroring out.
	chunkRepo := newMockChunkRepo()
	chunkRepo.store(&entity.DocumentChunk{ID: 1, ChunkID: "c1", Content: "x", DocumentID: 1})

	vectorStore := &mockVectorStore{
		results: []service.VectorSearchResult{{ChunkID: "c1", Score: 0.9, DocID: 1}},
	}
	fullText := &mockFullTextSearcher{
		hits: []service.FullTextHit{{ChunkID: "c1", Score: 1.0, DocID: 1}},
	}
	reranker := &mockReranker{rerankErr: errors.New("cross-encoder down")}

	uc := usecase.NewRetrievalUseCase(
		chunkRepo, newMockRagQueryRepo(),
		vectorStore, fullText,
		&mockEmbedder{vectors: [][]float32{{0.1}}}, reranker, 5, 60, 3,
	)

	resp, err := uc.Retrieve(context.Background(), &dto.RetrieveRequest{Query: "q"}, 0)
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestRetrievalUseCase_Retrieve_TopKDefaults(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	uc := usecase.NewRetrievalUseCase(
		chunkRepo, newMockRagQueryRepo(),
		&mockVectorStore{}, &mockFullTextSearcher{},
		&mockEmbedder{vectors: [][]float32{{0.1}}}, &mockReranker{},
		5, 60, 4,
	)
	resp, err := uc.Retrieve(context.Background(), &dto.RetrieveRequest{Query: "q"}, 0)
	require.NoError(t, err)
	// topK defaulted to uc.rerankTopK = 4
	assert.Equal(t, 4, resp.TopK)
}

func TestRetrievalUseCase_Feedback(t *testing.T) {
	t.Run("happy path good", func(t *testing.T) {
		queryRepo := newMockRagQueryRepo()
		queryRepo.items[10] = &entity.RagQuery{ID: 10, QueryText: "q"}
		uc := usecase.NewRetrievalUseCase(
			newMockChunkRepo(), queryRepo,
			nil, nil, nil, nil, 5, 60, 3,
		)
		err := uc.Feedback(context.Background(), 10, entity.FeedbackGood)
		require.NoError(t, err)
		assert.Equal(t, entity.FeedbackGood, queryRepo.items[10].Feedback)
	})

	t.Run("happy path bad", func(t *testing.T) {
		queryRepo := newMockRagQueryRepo()
		queryRepo.items[11] = &entity.RagQuery{ID: 11, QueryText: "q"}
		uc := usecase.NewRetrievalUseCase(
			newMockChunkRepo(), queryRepo,
			nil, nil, nil, nil, 5, 60, 3,
		)
		err := uc.Feedback(context.Background(), 11, entity.FeedbackBad)
		require.NoError(t, err)
		assert.Equal(t, entity.FeedbackBad, queryRepo.items[11].Feedback)
	})

	t.Run("invalid feedback value", func(t *testing.T) {
		uc := usecase.NewRetrievalUseCase(
			newMockChunkRepo(), newMockRagQueryRepo(),
			nil, nil, nil, nil, 5, 60, 3,
		)
		err := uc.Feedback(context.Background(), 1, "maybe")
		requireError(t, err, errno.InvalidParams)
	})

	t.Run("query log not found", func(t *testing.T) {
		uc := usecase.NewRetrievalUseCase(
			newMockChunkRepo(), newMockRagQueryRepo(),
			nil, nil, nil, nil, 5, 60, 3,
		)
		err := uc.Feedback(context.Background(), 999, entity.FeedbackGood)
		requireError(t, err, errno.NotFound)
	})

	t.Run("find error", func(t *testing.T) {
		queryRepo := newMockRagQueryRepo()
		queryRepo.findErr = errors.New("db down")
		uc := usecase.NewRetrievalUseCase(
			newMockChunkRepo(), queryRepo,
			nil, nil, nil, nil, 5, 60, 3,
		)
		err := uc.Feedback(context.Background(), 1, entity.FeedbackGood)
		require.Error(t, err)
	})

	t.Run("update error", func(t *testing.T) {
		queryRepo := newMockRagQueryRepo()
		queryRepo.items[1] = &entity.RagQuery{ID: 1}
		queryRepo.updateErr = errors.New("update fail")
		uc := usecase.NewRetrievalUseCase(
			newMockChunkRepo(), queryRepo,
			nil, nil, nil, nil, 5, 60, 3,
		)
		err := uc.Feedback(context.Background(), 1, entity.FeedbackGood)
		require.Error(t, err)
	})
}
