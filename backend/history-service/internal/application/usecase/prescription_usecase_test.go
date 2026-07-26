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

// mockPrescriptionRepo is an in-memory fake PrescriptionRepository for unit tests.
type mockPrescriptionRepo struct {
	items  map[int64]*entity.Prescription
	create func(*entity.Prescription) error
	update func(*entity.Prescription) error
	delete func(int64) error
	find   func(int64) (*entity.Prescription, error)
	list   func(pagination.Params) ([]entity.Prescription, int, error)
	search func(string, pagination.Params) ([]entity.Prescription, int, error)
}

func newMockPrescriptionRepo() *mockPrescriptionRepo {
	return &mockPrescriptionRepo{items: map[int64]*entity.Prescription{}}
}

func (m *mockPrescriptionRepo) Create(_ context.Context, p *entity.Prescription) error {
	if m.create != nil {
		return m.create(p)
	}
	if _, ok := m.items[p.ID]; ok {
		return errno.New(errno.AlreadyExists, "prescription exists")
	}
	m.items[p.ID] = p
	return nil
}

func (m *mockPrescriptionRepo) Update(_ context.Context, p *entity.Prescription) error {
	if m.update != nil {
		return m.update(p)
	}
	if _, ok := m.items[p.ID]; !ok {
		return errno.New(errno.NotFound, "prescription not found")
	}
	m.items[p.ID] = p
	return nil
}

func (m *mockPrescriptionRepo) Delete(_ context.Context, id int64) error {
	if m.delete != nil {
		return m.delete(id)
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "prescription not found")
	}
	delete(m.items, id)
	return nil
}

func (m *mockPrescriptionRepo) FindByID(_ context.Context, id int64) (*entity.Prescription, error) {
	if m.find != nil {
		return m.find(id)
	}
	if p, ok := m.items[id]; ok {
		clone := *p
		return &clone, nil
	}
	return nil, nil
}

func (m *mockPrescriptionRepo) List(_ context.Context, p pagination.Params) ([]entity.Prescription, int, error) {
	if m.list != nil {
		return m.list(p)
	}
	all := make([]entity.Prescription, 0, len(m.items))
	for _, p := range m.items {
		all = append(all, *p)
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

func (m *mockPrescriptionRepo) Search(_ context.Context, keyword string, p pagination.Params) ([]entity.Prescription, int, error) {
	if m.search != nil {
		return m.search(keyword, p)
	}
	all := make([]entity.Prescription, 0, len(m.items))
	for _, p := range m.items {
		if contains(p.Name, keyword) || contains(p.Composition, keyword) {
			all = append(all, *p)
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

// TestPrescriptionUseCase_Create covers happy path, validation, and nil input.
func TestPrescriptionUseCase_Create(t *testing.T) {
	repo := newMockPrescriptionRepo()
	uc := usecase.NewPrescriptionUseCase(repo)

	t.Run("happy path with category", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.PrescriptionRequest{
			Name:           "Gui Zhi Tang",
			Pinyin:         "guizhitang",
			SourceBookID:   1,
			SourcePersonID: 2,
			DynastyID:      3,
			Composition:    "Gui Zhi, Bai Shao, Sheng Jiang",
			Usage:          "decoction",
			Indications:    "exterior wind-cold",
			Category:       entity.PrescriptionCategoryHarmonizing,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "Gui Zhi Tang", resp.Name)
		assert.Equal(t, entity.PrescriptionCategoryHarmonizing, resp.Category)
		assert.Equal(t, int64(1), resp.SourceBookID)
		assert.NotZero(t, resp.ID)
	})

	t.Run("happy path with empty category", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.PrescriptionRequest{Name: "Ma Huang Tang"})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "", resp.Category)
	})

	t.Run("empty name rejected", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.PrescriptionRequest{Name: ""})
		require.Error(t, err)
		assert.Nil(t, resp)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.InvalidParams, e.Code)
		}
	})

	t.Run("invalid category rejected", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.PrescriptionRequest{
			Name:     "X",
			Category: "bogus",
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
		repo := newMockPrescriptionRepo()
		repo.create = func(*entity.Prescription) error { return errors.New("db down") }
		uc := usecase.NewPrescriptionUseCase(repo)
		resp, err := uc.Create(context.Background(), &dto.PrescriptionRequest{Name: "X"})
		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

// TestPrescriptionUseCase_Get covers the found and not-found paths.
func TestPrescriptionUseCase_Get(t *testing.T) {
	repo := newMockPrescriptionRepo()
	uc := usecase.NewPrescriptionUseCase(repo)

	created, err := uc.Create(context.Background(), &dto.PrescriptionRequest{Name: "Yin Qiao San"})
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		got, err := uc.Get(context.Background(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "Yin Qiao San", got.Name)
	})

	t.Run("not found", func(t *testing.T) {
		got, err := uc.Get(context.Background(), 999999)
		require.Error(t, err)
		assert.Nil(t, got)
	})
}

// TestPrescriptionUseCase_Update covers happy path, not-found, and validation.
func TestPrescriptionUseCase_Update(t *testing.T) {
	repo := newMockPrescriptionRepo()
	uc := usecase.NewPrescriptionUseCase(repo)

	created, err := uc.Create(context.Background(), &dto.PrescriptionRequest{Name: "Old"})
	require.NoError(t, err)

	t.Run("happy path", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), created.ID, &dto.PrescriptionRequest{
			Name:     "New",
			Category: entity.PrescriptionCategoryExteriorReleasing,
		})
		require.NoError(t, err)
		assert.Equal(t, "New", resp.Name)
		assert.Equal(t, entity.PrescriptionCategoryExteriorReleasing, resp.Category)
	})

	t.Run("not found", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), 999999, &dto.PrescriptionRequest{Name: "X"})
		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("invalid category rejected", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), created.ID, &dto.PrescriptionRequest{
			Name:     "X",
			Category: "bogus",
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

// TestPrescriptionUseCase_Delete covers delete and not-found paths.
func TestPrescriptionUseCase_Delete(t *testing.T) {
	repo := newMockPrescriptionRepo()
	uc := usecase.NewPrescriptionUseCase(repo)

	created, err := uc.Create(context.Background(), &dto.PrescriptionRequest{Name: "ToDelete"})
	require.NoError(t, err)

	require.NoError(t, uc.Delete(context.Background(), created.ID))
	_, err = uc.Get(context.Background(), created.ID)
	require.Error(t, err)

	err = uc.Delete(context.Background(), created.ID)
	require.Error(t, err)

	err = uc.Delete(context.Background(), 0)
	require.Error(t, err)
}

// TestPrescriptionUseCase_List exercises pagination and the total count.
func TestPrescriptionUseCase_List(t *testing.T) {
	repo := newMockPrescriptionRepo()
	uc := usecase.NewPrescriptionUseCase(repo)

	for _, name := range []string{"A", "B", "C"} {
		_, err := uc.Create(context.Background(), &dto.PrescriptionRequest{Name: name})
		require.NoError(t, err)
	}

	resp, err := uc.List(context.Background(), pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, 2, resp.TotalPage)
	require.Len(t, resp.Items, 2)
}

// TestPrescriptionUseCase_Search exercises the keyword filter.
func TestPrescriptionUseCase_Search(t *testing.T) {
	repo := newMockPrescriptionRepo()
	uc := usecase.NewPrescriptionUseCase(repo)

	_, err := uc.Create(context.Background(), &dto.PrescriptionRequest{
		Name:        "Gui Zhi Tang",
		Composition: "Gui Zhi, Bai Shao",
	})
	require.NoError(t, err)
	_, err = uc.Create(context.Background(), &dto.PrescriptionRequest{
		Name:        "Ma Huang Tang",
		Composition: "Ma Huang, Gui Zhi",
	})
	require.NoError(t, err)

	resp, err := uc.Search(context.Background(), "Ma Huang", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, "Ma Huang Tang", resp.Items[0].Name)

	// Empty keyword falls back to List.
	resp, err = uc.Search(context.Background(), "", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Total)
}
