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

// mockPersonRepo is an in-memory fake PersonRepository for unit tests.
type mockPersonRepo struct {
	items  map[int64]*entity.Person
	create func(*entity.Person) error
	update func(*entity.Person) error
	delete func(int64) error
	find   func(int64) (*entity.Person, error)
	list   func(pagination.Params) ([]entity.Person, int, error)
	search func(string, pagination.Params) ([]entity.Person, int, error)
}

func newMockPersonRepo() *mockPersonRepo {
	return &mockPersonRepo{items: map[int64]*entity.Person{}}
}

func (m *mockPersonRepo) Create(_ context.Context, p *entity.Person) error {
	if m.create != nil {
		return m.create(p)
	}
	if _, ok := m.items[p.ID]; ok {
		return errno.New(errno.AlreadyExists, "person exists")
	}
	m.items[p.ID] = p
	return nil
}

func (m *mockPersonRepo) Update(_ context.Context, p *entity.Person) error {
	if m.update != nil {
		return m.update(p)
	}
	if _, ok := m.items[p.ID]; !ok {
		return errno.New(errno.NotFound, "person not found")
	}
	m.items[p.ID] = p
	return nil
}

func (m *mockPersonRepo) Delete(_ context.Context, id int64) error {
	if m.delete != nil {
		return m.delete(id)
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "person not found")
	}
	delete(m.items, id)
	return nil
}

func (m *mockPersonRepo) FindByID(_ context.Context, id int64) (*entity.Person, error) {
	if m.find != nil {
		return m.find(id)
	}
	if p, ok := m.items[id]; ok {
		clone := *p
		return &clone, nil
	}
	return nil, nil
}

func (m *mockPersonRepo) List(_ context.Context, p pagination.Params) ([]entity.Person, int, error) {
	if m.list != nil {
		return m.list(p)
	}
	all := make([]entity.Person, 0, len(m.items))
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

func (m *mockPersonRepo) Search(_ context.Context, keyword string, p pagination.Params) ([]entity.Person, int, error) {
	if m.search != nil {
		return m.search(keyword, p)
	}
	all := make([]entity.Person, 0, len(m.items))
	for _, p := range m.items {
		if contains(p.Name, keyword) || contains(p.Biography, keyword) {
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

// TestPersonUseCase_Create covers happy path, validation, and nil input.
func TestPersonUseCase_Create(t *testing.T) {
	repo := newMockPersonRepo()
	uc := usecase.NewPersonUseCase(repo, nil)

	t.Run("happy path", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.PersonRequest{
			Name:         "Zhang Zhongjing",
			CourtesyName: "Ji",
			AliasName:    "Sage of Medicine",
			DynastyID:    1,
			BirthYear:    150,
			DeathYear:    219,
			Gender:       entity.GenderMale,
			Title:        "Prefect of Changsha",
			Biography:    "Author of Shanghan Lun",
			Achievements: "Founded cold damage school",
			PortraitURL:  "s3://portraits/zhang.jpg",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "Zhang Zhongjing", resp.Name)
		assert.Equal(t, entity.GenderMale, resp.Gender)
		assert.Equal(t, int16(150), resp.BirthYear)
		assert.Equal(t, int16(219), resp.DeathYear)
		assert.NotZero(t, resp.ID)
	})

	t.Run("empty name rejected", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.PersonRequest{Name: ""})
		require.Error(t, err)
		assert.Nil(t, resp)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.InvalidParams, e.Code)
		}
	})

	t.Run("invalid gender rejected", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.PersonRequest{
			Name:   "X",
			Gender: "robot",
		})
		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("birth_year > death_year rejected", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.PersonRequest{
			Name:      "X",
			BirthYear: 200,
			DeathYear: 100,
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
		repo := newMockPersonRepo()
		repo.create = func(*entity.Person) error { return errors.New("db down") }
		uc := usecase.NewPersonUseCase(repo, nil)
		resp, err := uc.Create(context.Background(), &dto.PersonRequest{Name: "X"})
		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

// TestPersonUseCase_Get covers the found and not-found paths.
func TestPersonUseCase_Get(t *testing.T) {
	repo := newMockPersonRepo()
	uc := usecase.NewPersonUseCase(repo, nil)

	created, err := uc.Create(context.Background(), &dto.PersonRequest{Name: "Hua Tuo"})
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		got, err := uc.Get(context.Background(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "Hua Tuo", got.Name)
	})

	t.Run("not found", func(t *testing.T) {
		got, err := uc.Get(context.Background(), 999999)
		require.Error(t, err)
		assert.Nil(t, got)
	})
}

// TestPersonUseCase_Update covers happy path, not-found, and validation.
func TestPersonUseCase_Update(t *testing.T) {
	repo := newMockPersonRepo()
	uc := usecase.NewPersonUseCase(repo, nil)

	created, err := uc.Create(context.Background(), &dto.PersonRequest{Name: "Old"})
	require.NoError(t, err)

	t.Run("happy path", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), created.ID, &dto.PersonRequest{
			Name:   "New",
			Gender: entity.GenderFemale,
		})
		require.NoError(t, err)
		assert.Equal(t, "New", resp.Name)
		assert.Equal(t, entity.GenderFemale, resp.Gender)
	})

	t.Run("not found", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), 999999, &dto.PersonRequest{Name: "X"})
		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("invalid gender rejected", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), created.ID, &dto.PersonRequest{
			Name:   "X",
			Gender: "bogus",
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

// TestPersonUseCase_Delete covers delete and not-found paths.
func TestPersonUseCase_Delete(t *testing.T) {
	repo := newMockPersonRepo()
	uc := usecase.NewPersonUseCase(repo, nil)

	created, err := uc.Create(context.Background(), &dto.PersonRequest{Name: "ToDelete"})
	require.NoError(t, err)

	require.NoError(t, uc.Delete(context.Background(), created.ID))
	_, err = uc.Get(context.Background(), created.ID)
	require.Error(t, err)

	err = uc.Delete(context.Background(), created.ID)
	require.Error(t, err)

	err = uc.Delete(context.Background(), 0)
	require.Error(t, err)
}

// TestPersonUseCase_List exercises pagination and the total count.
func TestPersonUseCase_List(t *testing.T) {
	repo := newMockPersonRepo()
	uc := usecase.NewPersonUseCase(repo, nil)

	for _, name := range []string{"A", "B", "C"} {
		_, err := uc.Create(context.Background(), &dto.PersonRequest{Name: name})
		require.NoError(t, err)
	}

	resp, err := uc.List(context.Background(), pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, 2, resp.TotalPage)
	require.Len(t, resp.Items, 2)
}

// TestPersonUseCase_Search exercises the keyword filter.
func TestPersonUseCase_Search(t *testing.T) {
	repo := newMockPersonRepo()
	uc := usecase.NewPersonUseCase(repo, nil)

	_, err := uc.Create(context.Background(), &dto.PersonRequest{
		Name:      "Zhang Zhongjing",
		Biography: "Author of Shanghan Lun",
	})
	require.NoError(t, err)
	_, err = uc.Create(context.Background(), &dto.PersonRequest{
		Name:      "Hua Tuo",
		Biography: "Surgeon and inventor",
	})
	require.NoError(t, err)

	resp, err := uc.Search(context.Background(), "Zhang", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, "Zhang Zhongjing", resp.Items[0].Name)

	// Empty keyword falls back to List.
	resp, err = uc.Search(context.Background(), "", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Total)
}
