package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/history-service/internal/application/dto"
	"tcm-history-ai/backend/history-service/internal/application/usecase"
	"tcm-history-ai/backend/history-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// mockBookRepo is an in-memory fake BookRepository for unit tests.
type mockBookRepo struct {
	items  map[int64]*entity.Book
	create func(*entity.Book) error
	update func(*entity.Book) error
	delete func(int64) error
	find   func(int64) (*entity.Book, error)
	list   func(pagination.Params) ([]entity.Book, int, error)
	search func(string, pagination.Params) ([]entity.Book, int, error)
}

func newMockBookRepo() *mockBookRepo {
	return &mockBookRepo{items: map[int64]*entity.Book{}}
}

func (m *mockBookRepo) Create(_ context.Context, b *entity.Book) error {
	if m.create != nil {
		return m.create(b)
	}
	if _, ok := m.items[b.ID]; ok {
		return errno.New(errno.AlreadyExists, "book exists")
	}
	m.items[b.ID] = b
	return nil
}

func (m *mockBookRepo) Update(_ context.Context, b *entity.Book) error {
	if m.update != nil {
		return m.update(b)
	}
	if _, ok := m.items[b.ID]; !ok {
		return errno.New(errno.NotFound, "book not found")
	}
	m.items[b.ID] = b
	return nil
}

func (m *mockBookRepo) Delete(_ context.Context, id int64) error {
	if m.delete != nil {
		return m.delete(id)
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "book not found")
	}
	delete(m.items, id)
	return nil
}

func (m *mockBookRepo) FindByID(_ context.Context, id int64) (*entity.Book, error) {
	if m.find != nil {
		return m.find(id)
	}
	if b, ok := m.items[id]; ok {
		clone := *b
		return &clone, nil
	}
	return nil, nil
}

func (m *mockBookRepo) List(_ context.Context, p pagination.Params) ([]entity.Book, int, error) {
	if m.list != nil {
		return m.list(p)
	}
	all := make([]entity.Book, 0, len(m.items))
	for _, b := range m.items {
		all = append(all, *b)
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

func (m *mockBookRepo) Search(_ context.Context, keyword string, p pagination.Params) ([]entity.Book, int, error) {
	if m.search != nil {
		return m.search(keyword, p)
	}
	all := make([]entity.Book, 0, len(m.items))
	for _, b := range m.items {
		if contains(b.Title, keyword) || contains(b.Summary, keyword) {
			all = append(all, *b)
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

// TestBookUseCase_Create covers happy path, validation, and nil input.
func TestBookUseCase_Create(t *testing.T) {
	repo := newMockBookRepo()
	uc := usecase.NewBookUseCase(repo, nil)

	t.Run("happy path with explicit IsExtant", func(t *testing.T) {
		extant := false
		resp, err := uc.Create(context.Background(), &dto.BookRequest{
			Title:         "Shanghan Lun",
			DynastyID:     1,
			PublishedYear: 210,
			Category:      entity.BookCategoryClassic,
			Summary:       "Cold damage classic",
			VolumeCount:   22,
			IsExtant:      &extant,
			FileURL:       "s3://books/shanghan.pdf",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "Shanghan Lun", resp.Title)
		assert.Equal(t, int16(210), resp.PublishedYear)
		assert.Equal(t, 22, resp.VolumeCount)
		assert.False(t, resp.IsExtant)
		assert.NotZero(t, resp.ID)
	})

	t.Run("happy path default IsExtant true", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.BookRequest{Title: "Jin Gui Yao Lue"})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.IsExtant)
	})

	t.Run("empty title rejected", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.BookRequest{Title: ""})
		require.Error(t, err)
		assert.Nil(t, resp)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.InvalidParams, e.Code)
		}
	})

	t.Run("nil request rejected", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), nil)
		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("repo error propagated", func(t *testing.T) {
		repo := newMockBookRepo()
		repo.create = func(*entity.Book) error { return errors.New("db down") }
		uc := usecase.NewBookUseCase(repo, nil)
		resp, err := uc.Create(context.Background(), &dto.BookRequest{Title: "X"})
		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

// TestBookUseCase_Get covers the found and not-found paths.
func TestBookUseCase_Get(t *testing.T) {
	repo := newMockBookRepo()
	uc := usecase.NewBookUseCase(repo, nil)

	created, err := uc.Create(context.Background(), &dto.BookRequest{Title: "Ben Cao Gang Mu"})
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		got, err := uc.Get(context.Background(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "Ben Cao Gang Mu", got.Title)
	})

	t.Run("not found", func(t *testing.T) {
		got, err := uc.Get(context.Background(), 999999)
		require.Error(t, err)
		assert.Nil(t, got)
	})
}

// TestBookUseCase_Update covers happy path and not-found.
func TestBookUseCase_Update(t *testing.T) {
	repo := newMockBookRepo()
	uc := usecase.NewBookUseCase(repo, nil)

	created, err := uc.Create(context.Background(), &dto.BookRequest{Title: "Old Title"})
	require.NoError(t, err)

	t.Run("happy path", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), created.ID, &dto.BookRequest{
			Title:   "New Title",
			Summary: "Updated summary",
		})
		require.NoError(t, err)
		assert.Equal(t, "New Title", resp.Title)
		assert.Equal(t, "Updated summary", resp.Summary)

		got, err := uc.Get(context.Background(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, "New Title", got.Title)
	})

	t.Run("not found", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), 999999, &dto.BookRequest{Title: "X"})
		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("nil body rejected", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), created.ID, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

// TestBookUseCase_Delete covers delete and not-found paths.
func TestBookUseCase_Delete(t *testing.T) {
	repo := newMockBookRepo()
	uc := usecase.NewBookUseCase(repo, nil)

	created, err := uc.Create(context.Background(), &dto.BookRequest{Title: "To Delete"})
	require.NoError(t, err)

	require.NoError(t, uc.Delete(context.Background(), created.ID))
	_, err = uc.Get(context.Background(), created.ID)
	require.Error(t, err)

	// Deleting again should fail.
	err = uc.Delete(context.Background(), created.ID)
	require.Error(t, err)

	// Invalid id should be rejected.
	err = uc.Delete(context.Background(), 0)
	require.Error(t, err)
}

// TestBookUseCase_List exercises pagination and the total count.
func TestBookUseCase_List(t *testing.T) {
	repo := newMockBookRepo()
	uc := usecase.NewBookUseCase(repo, nil)

	for _, title := range []string{"A", "B", "C"} {
		_, err := uc.Create(context.Background(), &dto.BookRequest{Title: title})
		require.NoError(t, err)
	}

	resp, err := uc.List(context.Background(), pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, 2, resp.TotalPage)
	require.Len(t, resp.Items, 2)
}

// TestBookUseCase_Search exercises the keyword filter.
func TestBookUseCase_Search(t *testing.T) {
	repo := newMockBookRepo()
	uc := usecase.NewBookUseCase(repo, nil)

	_, err := uc.Create(context.Background(), &dto.BookRequest{
		Title:   "Shanghan Lun",
		Summary: "Cold damage treatise",
	})
	require.NoError(t, err)
	_, err = uc.Create(context.Background(), &dto.BookRequest{
		Title:   "Ben Cao",
		Summary: "Materia medica",
	})
	require.NoError(t, err)

	resp, err := uc.Search(context.Background(), "Shanghan", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, "Shanghan Lun", resp.Items[0].Title)

	// Empty keyword falls back to List.
	resp, err = uc.Search(context.Background(), "", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Total)
}
