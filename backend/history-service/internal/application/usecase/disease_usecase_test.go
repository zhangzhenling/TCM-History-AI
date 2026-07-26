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

// mockDiseaseRepo is an in-memory fake DiseaseRepository for unit tests.
type mockDiseaseRepo struct {
	items  map[int64]*entity.Disease
	create func(*entity.Disease) error
	update func(*entity.Disease) error
	delete func(int64) error
	find   func(int64) (*entity.Disease, error)
	list   func(pagination.Params) ([]entity.Disease, int, error)
	search func(string, pagination.Params) ([]entity.Disease, int, error)
}

func newMockDiseaseRepo() *mockDiseaseRepo {
	return &mockDiseaseRepo{items: map[int64]*entity.Disease{}}
}

func (m *mockDiseaseRepo) Create(_ context.Context, d *entity.Disease) error {
	if m.create != nil {
		return m.create(d)
	}
	if _, ok := m.items[d.ID]; ok {
		return errno.New(errno.AlreadyExists, "disease exists")
	}
	m.items[d.ID] = d
	return nil
}

func (m *mockDiseaseRepo) Update(_ context.Context, d *entity.Disease) error {
	if m.update != nil {
		return m.update(d)
	}
	if _, ok := m.items[d.ID]; !ok {
		return errno.New(errno.NotFound, "disease not found")
	}
	m.items[d.ID] = d
	return nil
}

func (m *mockDiseaseRepo) Delete(_ context.Context, id int64) error {
	if m.delete != nil {
		return m.delete(id)
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "disease not found")
	}
	delete(m.items, id)
	return nil
}

func (m *mockDiseaseRepo) FindByID(_ context.Context, id int64) (*entity.Disease, error) {
	if m.find != nil {
		return m.find(id)
	}
	if d, ok := m.items[id]; ok {
		clone := *d
		return &clone, nil
	}
	return nil, nil
}

func (m *mockDiseaseRepo) List(_ context.Context, p pagination.Params) ([]entity.Disease, int, error) {
	if m.list != nil {
		return m.list(p)
	}
	all := make([]entity.Disease, 0, len(m.items))
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

func (m *mockDiseaseRepo) Search(_ context.Context, keyword string, p pagination.Params) ([]entity.Disease, int, error) {
	if m.search != nil {
		return m.search(keyword, p)
	}
	all := make([]entity.Disease, 0, len(m.items))
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

// TestDiseaseUseCase_Create covers happy path, validation, and nil input.
func TestDiseaseUseCase_Create(t *testing.T) {
	repo := newMockDiseaseRepo()
	uc := usecase.NewDiseaseUseCase(repo)

	t.Run("happy path", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.DiseaseRequest{
			Name:            "Cold Damage",
			Pinyin:          "shanghan",
			Category:        entity.DiseaseCategoryExternalContraction,
			Description:     "Exterior-releasing disorder",
			Symptoms:        "fever, chills",
			TCMPathogenesis: "wind-cold invasion",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "Cold Damage", resp.Name)
		assert.Equal(t, "shanghan", resp.Pinyin)
		assert.Equal(t, entity.DiseaseCategoryExternalContraction, resp.Category)
		assert.NotZero(t, resp.ID)
	})

	t.Run("empty name rejected", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.DiseaseRequest{Name: ""})
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
		repo := newMockDiseaseRepo()
		repo.create = func(*entity.Disease) error { return errors.New("db down") }
		uc := usecase.NewDiseaseUseCase(repo)
		resp, err := uc.Create(context.Background(), &dto.DiseaseRequest{Name: "X"})
		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

// TestDiseaseUseCase_Get covers the found and not-found paths.
func TestDiseaseUseCase_Get(t *testing.T) {
	repo := newMockDiseaseRepo()
	uc := usecase.NewDiseaseUseCase(repo)

	created, err := uc.Create(context.Background(), &dto.DiseaseRequest{Name: "Wind-Heat"})
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		got, err := uc.Get(context.Background(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "Wind-Heat", got.Name)
	})

	t.Run("not found", func(t *testing.T) {
		got, err := uc.Get(context.Background(), 999999)
		require.Error(t, err)
		assert.Nil(t, got)
	})
}

// TestDiseaseUseCase_Update covers happy path and not-found.
func TestDiseaseUseCase_Update(t *testing.T) {
	repo := newMockDiseaseRepo()
	uc := usecase.NewDiseaseUseCase(repo)

	created, err := uc.Create(context.Background(), &dto.DiseaseRequest{Name: "Old"})
	require.NoError(t, err)

	t.Run("happy path", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), created.ID, &dto.DiseaseRequest{
			Name:        "New",
			Description: "Updated",
		})
		require.NoError(t, err)
		assert.Equal(t, "New", resp.Name)
		assert.Equal(t, "Updated", resp.Description)

		got, err := uc.Get(context.Background(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, "New", got.Name)
	})

	t.Run("not found", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), 999999, &dto.DiseaseRequest{Name: "X"})
		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("nil body rejected", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), created.ID, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

// TestDiseaseUseCase_Delete covers delete and not-found paths.
func TestDiseaseUseCase_Delete(t *testing.T) {
	repo := newMockDiseaseRepo()
	uc := usecase.NewDiseaseUseCase(repo)

	created, err := uc.Create(context.Background(), &dto.DiseaseRequest{Name: "ToDelete"})
	require.NoError(t, err)

	require.NoError(t, uc.Delete(context.Background(), created.ID))
	_, err = uc.Get(context.Background(), created.ID)
	require.Error(t, err)

	err = uc.Delete(context.Background(), created.ID)
	require.Error(t, err)

	err = uc.Delete(context.Background(), 0)
	require.Error(t, err)
}

// TestDiseaseUseCase_List exercises pagination and the total count.
func TestDiseaseUseCase_List(t *testing.T) {
	repo := newMockDiseaseRepo()
	uc := usecase.NewDiseaseUseCase(repo)

	for _, name := range []string{"A", "B", "C"} {
		_, err := uc.Create(context.Background(), &dto.DiseaseRequest{Name: name})
		require.NoError(t, err)
	}

	resp, err := uc.List(context.Background(), pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, 2, resp.TotalPage)
	require.Len(t, resp.Items, 2)
}

// TestDiseaseUseCase_Search exercises the keyword filter.
func TestDiseaseUseCase_Search(t *testing.T) {
	repo := newMockDiseaseRepo()
	uc := usecase.NewDiseaseUseCase(repo)

	_, err := uc.Create(context.Background(), &dto.DiseaseRequest{
		Name:        "Cold Damage",
		Description: "wind-cold invasion",
	})
	require.NoError(t, err)
	_, err = uc.Create(context.Background(), &dto.DiseaseRequest{
		Name:        "Warm Disease",
		Description: "wenbing",
	})
	require.NoError(t, err)

	resp, err := uc.Search(context.Background(), "Cold", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, "Cold Damage", resp.Items[0].Name)

	// Empty keyword falls back to List.
	resp, err = uc.Search(context.Background(), "", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Total)
}
