package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/knowledge-service/internal/application/dto"
	"tcm-history-ai/backend/knowledge-service/internal/application/usecase"
	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// --- mock DocumentChunkRepository ---

type mockChunkRepo struct {
	items        map[int64]*entity.DocumentChunk
	itemsByCID   map[string]*entity.DocumentChunk
	createErr   error
	batchCreateErr error
	updateErr   error
	deleteErr   error
	findErr     error
	findByCIDErr error
	listErr     error
	listByIDsErr error
}

func newMockChunkRepo() *mockChunkRepo {
	return &mockChunkRepo{items: map[int64]*entity.DocumentChunk{}, itemsByCID: map[string]*entity.DocumentChunk{}}
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

// --- tests ---

func TestChunkUseCase_Get(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	uc := usecase.NewChunkUseCase(chunkRepo, newMockDocumentRepo())

	t.Run("found", func(t *testing.T) {
		created, err := uc.Create(context.Background(), 42, &dto.ChunkResponse{
			Content: "原文", ClassicCode: "X",
		})
		require.NoError(t, err)
		got, err := uc.Get(context.Background(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "原文", got.Content)
	})

	t.Run("not found", func(t *testing.T) {
		resp, err := uc.Get(context.Background(), 99999)
		requireError(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("repo error", func(t *testing.T) {
		chunkRepo := newMockChunkRepo()
		chunkRepo.findErr = errors.New("boom")
		uc := usecase.NewChunkUseCase(chunkRepo, newMockDocumentRepo())
		resp, err := uc.Get(context.Background(), 1)
		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestChunkUseCase_Create(t *testing.T) {
	t.Run("happy path with explicit chunk_id", func(t *testing.T) {
		uc := usecase.NewChunkUseCase(newMockChunkRepo(), newMockDocumentRepo())
		resp, err := uc.Create(context.Background(), 7, &dto.ChunkResponse{
			ChunkID: "c-001", ChunkIndex: 1, Content: "hello",
			ContentType: entity.ContentAnnotation, ClassicCode: "X",
		})
		require.NoError(t, err)
		assert.NotZero(t, resp.ID)
		assert.Equal(t, int64(7), resp.DocumentID)
		assert.Equal(t, "c-001", resp.ChunkID)
		assert.Equal(t, entity.ContentAnnotation, resp.ContentType)
	})

	t.Run("defaults chunk_id and content_type when empty", func(t *testing.T) {
		uc := usecase.NewChunkUseCase(newMockChunkRepo(), newMockDocumentRepo())
		resp, err := uc.Create(context.Background(), 7, &dto.ChunkResponse{
			Content: "hello",
		})
		require.NoError(t, err)
		// ChunkID defaults to the string form of the snowflake ID.
		assert.NotEmpty(t, resp.ChunkID)
		assert.Equal(t, entity.ContentOriginal, resp.ContentType)
	})

	t.Run("nil body rejected", func(t *testing.T) {
		uc := usecase.NewChunkUseCase(newMockChunkRepo(), newMockDocumentRepo())
		resp, err := uc.Create(context.Background(), 7, nil)
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("empty content rejected", func(t *testing.T) {
		uc := usecase.NewChunkUseCase(newMockChunkRepo(), newMockDocumentRepo())
		resp, err := uc.Create(context.Background(), 7, &dto.ChunkResponse{Content: ""})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := newMockChunkRepo()
		repo.createErr = errors.New("boom")
		uc := usecase.NewChunkUseCase(repo, newMockDocumentRepo())
		resp, err := uc.Create(context.Background(), 7, &dto.ChunkResponse{Content: "x"})
		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestChunkUseCase_ListByDocument(t *testing.T) {
	chunkRepo := newMockChunkRepo()
	uc := usecase.NewChunkUseCase(chunkRepo, newMockDocumentRepo())

	// Seed three chunks for doc 1 and one chunk for doc 2.
	for _, docID := range []int64{1, 1, 1, 2} {
		_, err := uc.Create(context.Background(), docID, &dto.ChunkResponse{Content: "x"})
		require.NoError(t, err)
	}

	t.Run("happy path", func(t *testing.T) {
		resp, err := uc.ListByDocument(context.Background(), 1, pagination.Params{Page: 1, PageSize: 2})
		require.NoError(t, err)
		assert.Equal(t, 3, resp.Total)
		require.Len(t, resp.Items, 2)
	})

	t.Run("empty result", func(t *testing.T) {
		resp, err := uc.ListByDocument(context.Background(), 999, pagination.Params{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, 0, resp.Total)
		assert.Empty(t, resp.Items)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := newMockChunkRepo()
		repo.listErr = errors.New("boom")
		uc := usecase.NewChunkUseCase(repo, newMockDocumentRepo())
		resp, err := uc.ListByDocument(context.Background(), 1, pagination.Params{Page: 1, PageSize: 10})
		require.Error(t, err)
		assert.Equal(t, 0, resp.Total)
	})
}
