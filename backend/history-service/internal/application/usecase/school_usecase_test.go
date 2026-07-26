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

// mockSchoolRepo is an in-memory fake SchoolRepository for unit tests.
type mockSchoolRepo struct {
	items  map[int64]*entity.School
	create func(*entity.School) error
	update func(*entity.School) error
	delete func(int64) error
	find   func(int64) (*entity.School, error)
	list   func(pagination.Params) ([]entity.School, int, error)
	search func(string, pagination.Params) ([]entity.School, int, error)
}

func newMockSchoolRepo() *mockSchoolRepo {
	return &mockSchoolRepo{items: map[int64]*entity.School{}}
}

func (m *mockSchoolRepo) Create(_ context.Context, s *entity.School) error {
	if m.create != nil {
		return m.create(s)
	}
	if _, ok := m.items[s.ID]; ok {
		return errno.New(errno.AlreadyExists, "school exists")
	}
	m.items[s.ID] = s
	return nil
}

func (m *mockSchoolRepo) Update(_ context.Context, s *entity.School) error {
	if m.update != nil {
		return m.update(s)
	}
	if _, ok := m.items[s.ID]; !ok {
		return errno.New(errno.NotFound, "school not found")
	}
	m.items[s.ID] = s
	return nil
}

func (m *mockSchoolRepo) Delete(_ context.Context, id int64) error {
	if m.delete != nil {
		return m.delete(id)
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "school not found")
	}
	delete(m.items, id)
	return nil
}

func (m *mockSchoolRepo) FindByID(_ context.Context, id int64) (*entity.School, error) {
	if m.find != nil {
		return m.find(id)
	}
	if s, ok := m.items[id]; ok {
		clone := *s
		return &clone, nil
	}
	return nil, nil
}

func (m *mockSchoolRepo) List(_ context.Context, p pagination.Params) ([]entity.School, int, error) {
	if m.list != nil {
		return m.list(p)
	}
	all := make([]entity.School, 0, len(m.items))
	for _, s := range m.items {
		all = append(all, *s)
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

func (m *mockSchoolRepo) Search(_ context.Context, keyword string, p pagination.Params) ([]entity.School, int, error) {
	if m.search != nil {
		return m.search(keyword, p)
	}
	all := make([]entity.School, 0, len(m.items))
	for _, s := range m.items {
		if contains(s.Name, keyword) || contains(s.Summary, keyword) {
			all = append(all, *s)
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

// TestSchoolUseCase_Create covers happy path, validation, and nil input.
func TestSchoolUseCase_Create(t *testing.T) {
	repo := newMockSchoolRepo()
	uc := usecase.NewSchoolUseCase(repo)

	t.Run("happy path", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.SchoolRequest{
			Name:            "Yishui School",
			DynastyID:       1,
			FounderPersonID: 2,
			Summary:         "Cold and cooling school",
			EstablishedYear: 1200,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "Yishui School", resp.Name)
		assert.Equal(t, int64(2), resp.FounderPersonID)
		assert.Equal(t, int16(1200), resp.EstablishedYear)
		assert.NotZero(t, resp.ID)
	})

	t.Run("empty name rejected", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.SchoolRequest{Name: ""})
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
		repo := newMockSchoolRepo()
		repo.create = func(*entity.School) error { return errors.New("db down") }
		uc := usecase.NewSchoolUseCase(repo)
		resp, err := uc.Create(context.Background(), &dto.SchoolRequest{Name: "X"})
		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

// TestSchoolUseCase_Get covers the found and not-found paths.
func TestSchoolUseCase_Get(t *testing.T) {
	repo := newMockSchoolRepo()
	uc := usecase.NewSchoolUseCase(repo)

	created, err := uc.Create(context.Background(), &dto.SchoolRequest{Name: "Wenbing School"})
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		got, err := uc.Get(context.Background(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "Wenbing School", got.Name)
	})

	t.Run("not found", func(t *testing.T) {
		got, err := uc.Get(context.Background(), 999999)
		require.Error(t, err)
		assert.Nil(t, got)
	})
}

// TestSchoolUseCase_Update covers happy path, not-found, and nil body.
func TestSchoolUseCase_Update(t *testing.T) {
	repo := newMockSchoolRepo()
	uc := usecase.NewSchoolUseCase(repo)

	created, err := uc.Create(context.Background(), &dto.SchoolRequest{Name: "Old"})
	require.NoError(t, err)

	t.Run("happy path", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), created.ID, &dto.SchoolRequest{
			Name:    "New",
			Summary: "Updated summary",
		})
		require.NoError(t, err)
		assert.Equal(t, "New", resp.Name)
		assert.Equal(t, "Updated summary", resp.Summary)
	})

	t.Run("not found", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), 999999, &dto.SchoolRequest{Name: "X"})
		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("nil body rejected", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), created.ID, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

// TestSchoolUseCase_Delete covers delete and not-found paths.
func TestSchoolUseCase_Delete(t *testing.T) {
	repo := newMockSchoolRepo()
	uc := usecase.NewSchoolUseCase(repo)

	created, err := uc.Create(context.Background(), &dto.SchoolRequest{Name: "ToDelete"})
	require.NoError(t, err)

	require.NoError(t, uc.Delete(context.Background(), created.ID))
	_, err = uc.Get(context.Background(), created.ID)
	require.Error(t, err)

	err = uc.Delete(context.Background(), created.ID)
	require.Error(t, err)

	err = uc.Delete(context.Background(), 0)
	require.Error(t, err)
}

// TestSchoolUseCase_List exercises pagination and the total count.
func TestSchoolUseCase_List(t *testing.T) {
	repo := newMockSchoolRepo()
	uc := usecase.NewSchoolUseCase(repo)

	for _, name := range []string{"A", "B", "C"} {
		_, err := uc.Create(context.Background(), &dto.SchoolRequest{Name: name})
		require.NoError(t, err)
	}

	resp, err := uc.List(context.Background(), pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, 2, resp.TotalPage)
	require.Len(t, resp.Items, 2)
}

// TestSchoolUseCase_Search exercises the keyword filter.
func TestSchoolUseCase_Search(t *testing.T) {
	repo := newMockSchoolRepo()
	uc := usecase.NewSchoolUseCase(repo)

	_, err := uc.Create(context.Background(), &dto.SchoolRequest{
		Name:    "Yishui School",
		Summary: "Cold and cooling school",
	})
	require.NoError(t, err)
	_, err = uc.Create(context.Background(), &dto.SchoolRequest{
		Name:    "Wenbing School",
		Summary: "Warm disease school",
	})
	require.NoError(t, err)

	resp, err := uc.Search(context.Background(), "Yishui", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, "Yishui School", resp.Items[0].Name)

	// Empty keyword falls back to List.
	resp, err = uc.Search(context.Background(), "", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Total)
}
