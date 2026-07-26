package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/knowledge-service/internal/application/dto"
	"tcm-history-ai/backend/knowledge-service/internal/application/usecase"
	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
	"tcm-history-ai/backend/knowledge-service/internal/domain/event"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// --- mock DocumentRepository ---

type mockDocumentRepo struct {
	items  map[int64]*entity.Document
	byHash map[string]*entity.Document

	createErr error
	updateErr error
	deleteErr error
	listErr   error
}

func newMockDocumentRepo() *mockDocumentRepo {
	return &mockDocumentRepo{items: map[int64]*entity.Document{}, byHash: map[string]*entity.Document{}}
}

func (m *mockDocumentRepo) Create(_ context.Context, d *entity.Document) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.items[d.ID] = d
	if d.ContentHash != "" {
		m.byHash[d.ContentHash] = d
	}
	return nil
}

func (m *mockDocumentRepo) Update(_ context.Context, d *entity.Document) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if _, ok := m.items[d.ID]; !ok {
		return errno.New(errno.NotFound, "document not found")
	}
	m.items[d.ID] = d
	if d.ContentHash != "" {
		m.byHash[d.ContentHash] = d
	}
	return nil
}

func (m *mockDocumentRepo) Delete(_ context.Context, id int64) error {
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

func (m *mockDocumentRepo) FindByID(_ context.Context, id int64) (*entity.Document, error) {
	if d, ok := m.items[id]; ok {
		clone := *d
		return &clone, nil
	}
	return nil, nil
}

func (m *mockDocumentRepo) FindByContentHash(_ context.Context, hash string) (*entity.Document, error) {
	if d, ok := m.byHash[hash]; ok {
		clone := *d
		return &clone, nil
	}
	return nil, nil
}

func (m *mockDocumentRepo) List(_ context.Context, p pagination.Params) ([]entity.Document, int, error) {
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

func (m *mockDocumentRepo) ListByClassic(_ context.Context, classicCode string, p pagination.Params) ([]entity.Document, int, error) {
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

// --- tests ---

func TestDocumentUseCase_Create_HappyPath(t *testing.T) {
	repo := newMockDocumentRepo()
	pub := &mockEventPublisher{}
	uc := usecase.NewDocumentUseCase(repo, pub, nil)

	resp, err := uc.Create(context.Background(), &dto.DocumentRequest{
		ClassicCode: "HuangDiNeiJing",
		Title:       "黄帝内经",
		Version:     "v1",
		Dynasty:     "Han",
		Author:      "佚名",
		SourceType:  entity.SourceBook,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "HuangDiNeiJing", resp.ClassicCode)
	assert.Equal(t, "黄帝内经", resp.Title)
	assert.Equal(t, entity.DocumentStatusPending, resp.Status)
	assert.Equal(t, entity.SourceBook, resp.SourceType)
	assert.NotZero(t, resp.ID)
	// Default metadata should be {} when caller passes nil.
	assert.JSONEq(t, "{}", string(resp.MetadataJSON))
	// No PDF object key → no event published.
	assert.Empty(t, pub.published)
}

func TestDocumentUseCase_Create_DefaultSourceType(t *testing.T) {
	repo := newMockDocumentRepo()
	uc := usecase.NewDocumentUseCase(repo, &mockEventPublisher{}, nil)

	resp, err := uc.Create(context.Background(), &dto.DocumentRequest{
		ClassicCode: "X",
		Title:       "T",
		// SourceType intentionally empty.
	})
	require.NoError(t, err)
	assert.Equal(t, entity.SourceBook, resp.SourceType)
}

func TestDocumentUseCase_Create_PublishesEventWhenPDFUploaded(t *testing.T) {
	repo := newMockDocumentRepo()
	pub := &mockEventPublisher{}
	uc := usecase.NewDocumentUseCase(repo, pub, nil)

	resp, err := uc.Create(context.Background(), &dto.DocumentRequest{
		ClassicCode:  "ShangHanLun",
		Title:        "傷寒論",
		PDFObjectKey: "uploads/shanghanlun.pdf",
	})
	require.NoError(t, err)
	require.Len(t, pub.published, 1)

	evt, ok := pub.published[0].(event.DocumentUploaded)
	require.True(t, ok)
	assert.Equal(t, resp.ID, evt.DocumentID)
	assert.Equal(t, "ShangHanLun", evt.ClassicCode)
	assert.Equal(t, "uploads/shanghanlun.pdf", evt.ObjectKey)
}

func TestDocumentUseCase_Create_DedupByContentHash(t *testing.T) {
	repo := newMockDocumentRepo()
	uc := usecase.NewDocumentUseCase(repo, &mockEventPublisher{}, nil)

	first, err := uc.Create(context.Background(), &dto.DocumentRequest{
		ClassicCode: "X",
		Title:       "First",
		ContentHash: "abc123",
	})
	require.NoError(t, err)

	// Same hash → returns the existing document.
	second, err := uc.Create(context.Background(), &dto.DocumentRequest{
		ClassicCode: "Y",
		Title:       "Second",
		ContentHash: "abc123",
	})
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, "First", second.Title) // unchanged
}

func TestDocumentUseCase_Create_ValidationErrors(t *testing.T) {
	uc := usecase.NewDocumentUseCase(newMockDocumentRepo(), &mockEventPublisher{}, nil)

	t.Run("nil request", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), nil)
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})
	t.Run("empty title", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.DocumentRequest{
			ClassicCode: "X",
		})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})
	t.Run("empty classic_code", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.DocumentRequest{
			Title: "T",
		})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})
}

func TestDocumentUseCase_Create_RepoError(t *testing.T) {
	repo := newMockDocumentRepo()
	repo.createErr = errors.New("db down")
	uc := usecase.NewDocumentUseCase(repo, &mockEventPublisher{}, nil)

	resp, err := uc.Create(context.Background(), &dto.DocumentRequest{
		ClassicCode: "X", Title: "T",
	})
	require.Error(t, err)
	assert.Nil(t, resp)
}

func TestDocumentUseCase_Update(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		repo := newMockDocumentRepo()
		uc := usecase.NewDocumentUseCase(repo, &mockEventPublisher{}, nil)
		created, err := uc.Create(context.Background(), &dto.DocumentRequest{
			ClassicCode: "X", Title: "Old",
		})
		require.NoError(t, err)

		resp, err := uc.Update(context.Background(), created.ID, &dto.DocumentRequest{
			ClassicCode: "Y", Title: "New", VolumeCount: 10,
		})
		require.NoError(t, err)
		assert.Equal(t, "New", resp.Title)
		assert.Equal(t, "Y", resp.ClassicCode)
		assert.Equal(t, 10, resp.VolumeCount)

		// Verify persistence.
		got, err := uc.Get(context.Background(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, "New", got.Title)
	})

	t.Run("nil body", func(t *testing.T) {
		uc := usecase.NewDocumentUseCase(newMockDocumentRepo(), &mockEventPublisher{}, nil)
		resp, err := uc.Update(context.Background(), 1, nil)
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("not found", func(t *testing.T) {
		uc := usecase.NewDocumentUseCase(newMockDocumentRepo(), &mockEventPublisher{}, nil)
		resp, err := uc.Update(context.Background(), 999, &dto.DocumentRequest{Title: "x"})
		requireError(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("repo error on find", func(t *testing.T) {
		repo := newMockDocumentRepo()
		repo.updateErr = errors.New("boom")
		uc := usecase.NewDocumentUseCase(repo, &mockEventPublisher{}, nil)
		created, err := uc.Create(context.Background(), &dto.DocumentRequest{
			ClassicCode: "X", Title: "T",
		})
		require.NoError(t, err)
		_, err = uc.Update(context.Background(), created.ID, &dto.DocumentRequest{Title: "New"})
		require.Error(t, err)
	})

	t.Run("metadata preserved when not provided", func(t *testing.T) {
		repo := newMockDocumentRepo()
		uc := usecase.NewDocumentUseCase(repo, &mockEventPublisher{}, nil)
		created, err := uc.Create(context.Background(), &dto.DocumentRequest{
			ClassicCode: "X", Title: "T",
			MetadataJSON: []byte(`{"k":"v"}`),
		})
		require.NoError(t, err)
		// Update without MetadataJSON should preserve existing metadata.
		resp, err := uc.Update(context.Background(), created.ID, &dto.DocumentRequest{Title: "T2"})
		require.NoError(t, err)
		assert.JSONEq(t, `{"k":"v"}`, string(resp.MetadataJSON))
	})
}

func TestDocumentUseCase_Delete(t *testing.T) {
	repo := newMockDocumentRepo()
	uc := usecase.NewDocumentUseCase(repo, &mockEventPublisher{}, nil)
	created, err := uc.Create(context.Background(), &dto.DocumentRequest{
		ClassicCode: "X", Title: "T",
	})
	require.NoError(t, err)

	require.NoError(t, uc.Delete(context.Background(), created.ID))
	// Subsequent Get should fail with NotFound.
	_, err = uc.Get(context.Background(), created.ID)
	requireError(t, err, errno.NotFound)
}

func TestDocumentUseCase_Get(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		uc := usecase.NewDocumentUseCase(newMockDocumentRepo(), &mockEventPublisher{}, nil)
		created, err := uc.Create(context.Background(), &dto.DocumentRequest{
			ClassicCode: "X", Title: "T",
		})
		require.NoError(t, err)
		got, err := uc.Get(context.Background(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "T", got.Title)
	})
	t.Run("not found", func(t *testing.T) {
		uc := usecase.NewDocumentUseCase(newMockDocumentRepo(), &mockEventPublisher{}, nil)
		resp, err := uc.Get(context.Background(), 999)
		requireError(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})
}

func TestDocumentUseCase_List(t *testing.T) {
	repo := newMockDocumentRepo()
	uc := usecase.NewDocumentUseCase(repo, &mockEventPublisher{}, nil)
	for _, title := range []string{"A", "B", "C"} {
		_, err := uc.Create(context.Background(), &dto.DocumentRequest{
			ClassicCode: "X", Title: title,
		})
		require.NoError(t, err)
	}

	resp, err := uc.List(context.Background(), pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, 2, resp.TotalPage)
	require.Len(t, resp.Items, 2)
}

func TestDocumentUseCase_List_RepoError(t *testing.T) {
	repo := newMockDocumentRepo()
	repo.listErr = errors.New("fail")
	uc := usecase.NewDocumentUseCase(repo, &mockEventPublisher{}, nil)
	resp, err := uc.List(context.Background(), pagination.Params{Page: 1, PageSize: 10})
	require.Error(t, err)
	assert.Equal(t, 0, resp.Total)
}

func TestDocumentUseCase_ListByClassic(t *testing.T) {
	repo := newMockDocumentRepo()
	uc := usecase.NewDocumentUseCase(repo, &mockEventPublisher{}, nil)
	_, err := uc.Create(context.Background(), &dto.DocumentRequest{ClassicCode: "A", Title: "A1"})
	require.NoError(t, err)
	_, err = uc.Create(context.Background(), &dto.DocumentRequest{ClassicCode: "A", Title: "A2"})
	require.NoError(t, err)
	_, err = uc.Create(context.Background(), &dto.DocumentRequest{ClassicCode: "B", Title: "B1"})
	require.NoError(t, err)

	t.Run("filter by classic", func(t *testing.T) {
		resp, err := uc.ListByClassic(context.Background(), "A", pagination.Params{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Total)
		require.Len(t, resp.Items, 2)
	})

	t.Run("empty classic falls back to List", func(t *testing.T) {
		resp, err := uc.ListByClassic(context.Background(), "", pagination.Params{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, 3, resp.Total)
	})
}

func TestToDocumentResponse_TimeFormatting(t *testing.T) {
	// Test the unexported toDocumentResponse indirectly by ensuring a
	// document with non-zero CreatedAt/UpdatedAt surfaces them as RFC3339.
	repo := newMockDocumentRepo()
	uc := usecase.NewDocumentUseCase(repo, &mockEventPublisher{}, nil)
	created, err := uc.Create(context.Background(), &dto.DocumentRequest{
		ClassicCode: "X", Title: "T",
	})
	require.NoError(t, err)
	// Inject timestamps directly into the stored entity (simulating DB load).
	// toDocumentResponse formats these as RFC3339 strings in the DTO.
	stored := repo.items[created.ID]
	require.NotNil(t, stored)
	now := time.Now()
	stored.CreatedAt = now
	stored.UpdatedAt = now.Add(time.Second)

	resp, err := uc.Get(context.Background(), created.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.CreatedAt)
	assert.NotEmpty(t, resp.UpdatedAt)
	// Verify the formatted timestamp round-trips through RFC3339 parsing.
	_, err = time.Parse(time.RFC3339, resp.CreatedAt)
	assert.NoError(t, err)
}

// requireError asserts that err is non-nil and (when possible) carries the
// expected errno code.
func requireError(t *testing.T, err error, code errno.Errno) {
	t.Helper()
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, code, e.Code)
	}
}
