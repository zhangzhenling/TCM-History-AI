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

// mockDynastyRepo is an in-memory fake DynastyRepository for unit tests.
type mockDynastyRepo struct {
	items  map[int64]*entity.Dynasty
	create func(*entity.Dynasty) error
	update func(*entity.Dynasty) error
	delete func(int64) error
	find   func(int64) (*entity.Dynasty, error)
	list   func(pagination.Params) ([]entity.Dynasty, int, error)
	search func(string, pagination.Params) ([]entity.Dynasty, int, error)
}

func newMockDynastyRepo() *mockDynastyRepo {
	return &mockDynastyRepo{items: map[int64]*entity.Dynasty{}}
}

func (m *mockDynastyRepo) Create(_ context.Context, d *entity.Dynasty) error {
	if m.create != nil {
		return m.create(d)
	}
	if _, ok := m.items[d.ID]; ok {
		return errno.New(errno.AlreadyExists, "dynasty exists")
	}
	m.items[d.ID] = d
	return nil
}

func (m *mockDynastyRepo) Update(_ context.Context, d *entity.Dynasty) error {
	if m.update != nil {
		return m.update(d)
	}
	if _, ok := m.items[d.ID]; !ok {
		return errno.New(errno.NotFound, "dynasty not found")
	}
	m.items[d.ID] = d
	return nil
}

func (m *mockDynastyRepo) Delete(_ context.Context, id int64) error {
	if m.delete != nil {
		return m.delete(id)
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "dynasty not found")
	}
	delete(m.items, id)
	return nil
}

func (m *mockDynastyRepo) FindByID(_ context.Context, id int64) (*entity.Dynasty, error) {
	if m.find != nil {
		return m.find(id)
	}
	if d, ok := m.items[id]; ok {
		clone := *d
		return &clone, nil
	}
	return nil, nil
}

func (m *mockDynastyRepo) List(_ context.Context, p pagination.Params) ([]entity.Dynasty, int, error) {
	if m.list != nil {
		return m.list(p)
	}
	all := make([]entity.Dynasty, 0, len(m.items))
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

func (m *mockDynastyRepo) Search(_ context.Context, keyword string, p pagination.Params) ([]entity.Dynasty, int, error) {
	if m.search != nil {
		return m.search(keyword, p)
	}
	all := make([]entity.Dynasty, 0, len(m.items))
	for _, d := range m.items {
		if contains(d.Name, keyword) || contains(d.Description, keyword) {
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

func contains(haystack, needle string) bool {
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

// TestDynastyUseCase_Create exercises the happy path and the empty-name
// validation path.
func TestDynastyUseCase_Create(t *testing.T) {
	repo := newMockDynastyRepo()
	uc := usecase.NewDynastyUseCase(repo, nil)

	t.Run("happy path", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.DynastyRequest{
			Name:        "Han",
			StartYear:   -202,
			EndYear:     220,
			SortOrder:   5,
			Description: "Western & Eastern Han",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "Han", resp.Name)
		assert.Equal(t, int16(-202), resp.StartYear)
		assert.Equal(t, int16(220), resp.EndYear)
		assert.Equal(t, 5, resp.SortOrder)
		assert.NotZero(t, resp.ID)
	})

	t.Run("empty name rejected", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.DynastyRequest{Name: ""})
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
}

// TestDynastyUseCase_List exercises pagination and the total count.
func TestDynastyUseCase_List(t *testing.T) {
	repo := newMockDynastyRepo()
	uc := usecase.NewDynastyUseCase(repo, nil)

	// Seed 3 dynasties.
	for _, name := range []string{"Han", "Tang", "Song"} {
		_, err := uc.Create(context.Background(), &dto.DynastyRequest{Name: name})
		require.NoError(t, err)
	}

	resp, err := uc.List(context.Background(), pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, 2, resp.TotalPage)
	require.Len(t, resp.Items, 2)
}

// TestDynastyUseCase_Get covers the found and not-found paths.
func TestDynastyUseCase_Get(t *testing.T) {
	repo := newMockDynastyRepo()
	uc := usecase.NewDynastyUseCase(repo, nil)

	created, err := uc.Create(context.Background(), &dto.DynastyRequest{Name: "Ming"})
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		got, err := uc.Get(context.Background(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "Ming", got.Name)
	})

	t.Run("not found", func(t *testing.T) {
		got, err := uc.Get(context.Background(), 999999)
		require.Error(t, err)
		assert.Nil(t, got)
	})
}

// TestDynastyUseCase_Delete covers the delete and not-found paths.
func TestDynastyUseCase_Delete(t *testing.T) {
	repo := newMockDynastyRepo()
	uc := usecase.NewDynastyUseCase(repo, nil)

	created, err := uc.Create(context.Background(), &dto.DynastyRequest{Name: "Qing"})
	require.NoError(t, err)

	require.NoError(t, uc.Delete(context.Background(), created.ID))
	got, err := uc.Get(context.Background(), created.ID)
	require.Error(t, err)
	assert.Nil(t, got)

	// Deleting again should fail.
	err = uc.Delete(context.Background(), created.ID)
	require.Error(t, err)
}

// TestDynastyUseCase_Search exercises the keyword filter.
func TestDynastyUseCase_Search(t *testing.T) {
	repo := newMockDynastyRepo()
	uc := usecase.NewDynastyUseCase(repo, nil)

	_, err := uc.Create(context.Background(), &dto.DynastyRequest{Name: "Han", Description: "Western Han dynasty"})
	require.NoError(t, err)
	_, err = uc.Create(context.Background(), &dto.DynastyRequest{Name: "Tang", Description: "Tang dynasty"})
	require.NoError(t, err)

	resp, err := uc.Search(context.Background(), "Han", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, "Han", resp.Items[0].Name)
}
