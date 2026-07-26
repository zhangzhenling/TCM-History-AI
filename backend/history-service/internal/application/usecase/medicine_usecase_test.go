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

// mockMedicineRepo is an in-memory fake MedicineRepository for unit tests.
type mockMedicineRepo struct {
	items  map[int64]*entity.Medicine
	create func(*entity.Medicine) error
	update func(*entity.Medicine) error
	delete func(int64) error
	find   func(int64) (*entity.Medicine, error)
	list   func(pagination.Params) ([]entity.Medicine, int, error)
	search func(string, pagination.Params) ([]entity.Medicine, int, error)
}

func newMockMedicineRepo() *mockMedicineRepo {
	return &mockMedicineRepo{items: map[int64]*entity.Medicine{}}
}

func (m *mockMedicineRepo) Create(_ context.Context, med *entity.Medicine) error {
	if m.create != nil {
		return m.create(med)
	}
	if _, ok := m.items[med.ID]; ok {
		return errno.New(errno.AlreadyExists, "medicine exists")
	}
	m.items[med.ID] = med
	return nil
}

func (m *mockMedicineRepo) Update(_ context.Context, med *entity.Medicine) error {
	if m.update != nil {
		return m.update(med)
	}
	if _, ok := m.items[med.ID]; !ok {
		return errno.New(errno.NotFound, "medicine not found")
	}
	m.items[med.ID] = med
	return nil
}

func (m *mockMedicineRepo) Delete(_ context.Context, id int64) error {
	if m.delete != nil {
		return m.delete(id)
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "medicine not found")
	}
	delete(m.items, id)
	return nil
}

func (m *mockMedicineRepo) FindByID(_ context.Context, id int64) (*entity.Medicine, error) {
	if m.find != nil {
		return m.find(id)
	}
	if med, ok := m.items[id]; ok {
		clone := *med
		return &clone, nil
	}
	return nil, nil
}

func (m *mockMedicineRepo) List(_ context.Context, p pagination.Params) ([]entity.Medicine, int, error) {
	if m.list != nil {
		return m.list(p)
	}
	all := make([]entity.Medicine, 0, len(m.items))
	for _, med := range m.items {
		all = append(all, *med)
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

func (m *mockMedicineRepo) Search(_ context.Context, keyword string, p pagination.Params) ([]entity.Medicine, int, error) {
	if m.search != nil {
		return m.search(keyword, p)
	}
	all := make([]entity.Medicine, 0, len(m.items))
	for _, med := range m.items {
		if contains(med.Name, keyword) || contains(med.Efficacy, keyword) {
			all = append(all, *med)
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

// TestMedicineUseCase_Create covers happy path, validation, and nil input.
func TestMedicineUseCase_Create(t *testing.T) {
	repo := newMockMedicineRepo()
	uc := usecase.NewMedicineUseCase(repo)

	t.Run("happy path with nature and alias", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.MedicineRequest{
			Name:      "Gui Zhi",
			Pinyin:    "guizhi",
			AliasJSON: []string{"Cinnamon Twig"},
			Nature:    entity.MedicineNatureWarm,
			Flavor:    "sweet, pungent",
			Meridian:  "heart, lung, bladder",
			Efficacy:  "releases exterior",
			Dosage:    "3-9g",
			Toxicity:  entity.MedicineToxicityNone,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "Gui Zhi", resp.Name)
		assert.Equal(t, entity.MedicineNatureWarm, resp.Nature)
		require.Len(t, resp.AliasJSON, 1)
		assert.NotZero(t, resp.ID)
	})

	t.Run("happy path with nil alias defaults to empty slice", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.MedicineRequest{
			Name:   "Ma Huang",
			Nature: entity.MedicineNatureHot,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, []string{}, resp.AliasJSON)
	})

	t.Run("empty name rejected", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.MedicineRequest{Name: ""})
		require.Error(t, err)
		assert.Nil(t, resp)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.InvalidParams, e.Code)
		}
	})

	t.Run("invalid nature rejected", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.MedicineRequest{
			Name:   "X",
			Nature: "frozen",
		})
		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("nil request rejected", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), nil)
		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("repo error propagated", func(t *testing.T) {
		repo := newMockMedicineRepo()
		repo.create = func(*entity.Medicine) error { return errors.New("db down") }
		uc := usecase.NewMedicineUseCase(repo)
		resp, err := uc.Create(context.Background(), &dto.MedicineRequest{Name: "X"})
		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

// TestMedicineUseCase_Get covers the found and not-found paths.
func TestMedicineUseCase_Get(t *testing.T) {
	repo := newMockMedicineRepo()
	uc := usecase.NewMedicineUseCase(repo)

	created, err := uc.Create(context.Background(), &dto.MedicineRequest{Name: "Ren Shen"})
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		got, err := uc.Get(context.Background(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "Ren Shen", got.Name)
	})

	t.Run("not found", func(t *testing.T) {
		got, err := uc.Get(context.Background(), 999999)
		require.Error(t, err)
		assert.Nil(t, got)
	})
}

// TestMedicineUseCase_Update covers happy path, not-found, and validation.
func TestMedicineUseCase_Update(t *testing.T) {
	repo := newMockMedicineRepo()
	uc := usecase.NewMedicineUseCase(repo)

	created, err := uc.Create(context.Background(), &dto.MedicineRequest{Name: "Old"})
	require.NoError(t, err)

	t.Run("happy path", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), created.ID, &dto.MedicineRequest{
			Name:   "New",
			Nature: entity.MedicineNatureCold,
		})
		require.NoError(t, err)
		assert.Equal(t, "New", resp.Name)
		assert.Equal(t, entity.MedicineNatureCold, resp.Nature)
	})

	t.Run("happy path with alias update", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), created.ID, &dto.MedicineRequest{
			Name:      "New2",
			AliasJSON: []string{"alias1"},
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"alias1"}, resp.AliasJSON)
	})

	t.Run("not found", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), 999999, &dto.MedicineRequest{Name: "X"})
		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("invalid nature rejected", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), created.ID, &dto.MedicineRequest{
			Name:   "X",
			Nature: "bogus",
		})
		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("nil body rejected", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), created.ID, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

// TestMedicineUseCase_Delete covers delete and not-found paths.
func TestMedicineUseCase_Delete(t *testing.T) {
	repo := newMockMedicineRepo()
	uc := usecase.NewMedicineUseCase(repo)

	created, err := uc.Create(context.Background(), &dto.MedicineRequest{Name: "ToDelete"})
	require.NoError(t, err)

	require.NoError(t, uc.Delete(context.Background(), created.ID))
	_, err = uc.Get(context.Background(), created.ID)
	require.Error(t, err)

	err = uc.Delete(context.Background(), created.ID)
	require.Error(t, err)

	err = uc.Delete(context.Background(), 0)
	require.Error(t, err)
}

// TestMedicineUseCase_List exercises pagination and the total count.
func TestMedicineUseCase_List(t *testing.T) {
	repo := newMockMedicineRepo()
	uc := usecase.NewMedicineUseCase(repo)

	for _, name := range []string{"A", "B", "C"} {
		_, err := uc.Create(context.Background(), &dto.MedicineRequest{Name: name})
		require.NoError(t, err)
	}

	resp, err := uc.List(context.Background(), pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, 2, resp.TotalPage)
	require.Len(t, resp.Items, 2)
}

// TestMedicineUseCase_Search exercises the keyword filter.
func TestMedicineUseCase_Search(t *testing.T) {
	repo := newMockMedicineRepo()
	uc := usecase.NewMedicineUseCase(repo)

	_, err := uc.Create(context.Background(), &dto.MedicineRequest{
		Name:     "Gui Zhi",
		Efficacy: "releases exterior",
	})
	require.NoError(t, err)
	_, err = uc.Create(context.Background(), &dto.MedicineRequest{
		Name:     "Ma Huang",
		Efficacy: "induces sweating",
	})
	require.NoError(t, err)

	resp, err := uc.Search(context.Background(), "Gui", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, "Gui Zhi", resp.Items[0].Name)

	// Empty keyword falls back to List.
	resp, err = uc.Search(context.Background(), "", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Total)
}
