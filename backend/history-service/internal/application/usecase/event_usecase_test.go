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

// mockEventRepo is an in-memory fake EventRepository for unit tests.
type mockEventRepo struct {
	items  map[int64]*entity.Event
	create func(*entity.Event) error
	update func(*entity.Event) error
	delete func(int64) error
	find   func(int64) (*entity.Event, error)
	list   func(pagination.Params) ([]entity.Event, int, error)
	search func(string, pagination.Params) ([]entity.Event, int, error)
}

func newMockEventRepo() *mockEventRepo {
	return &mockEventRepo{items: map[int64]*entity.Event{}}
}

func (m *mockEventRepo) Create(_ context.Context, e *entity.Event) error {
	if m.create != nil {
		return m.create(e)
	}
	if _, ok := m.items[e.ID]; ok {
		return errno.New(errno.AlreadyExists, "event exists")
	}
	m.items[e.ID] = e
	return nil
}

func (m *mockEventRepo) Update(_ context.Context, e *entity.Event) error {
	if m.update != nil {
		return m.update(e)
	}
	if _, ok := m.items[e.ID]; !ok {
		return errno.New(errno.NotFound, "event not found")
	}
	m.items[e.ID] = e
	return nil
}

func (m *mockEventRepo) Delete(_ context.Context, id int64) error {
	if m.delete != nil {
		return m.delete(id)
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "event not found")
	}
	delete(m.items, id)
	return nil
}

func (m *mockEventRepo) FindByID(_ context.Context, id int64) (*entity.Event, error) {
	if m.find != nil {
		return m.find(id)
	}
	if e, ok := m.items[id]; ok {
		clone := *e
		return &clone, nil
	}
	return nil, nil
}

func (m *mockEventRepo) List(_ context.Context, p pagination.Params) ([]entity.Event, int, error) {
	if m.list != nil {
		return m.list(p)
	}
	all := make([]entity.Event, 0, len(m.items))
	for _, e := range m.items {
		all = append(all, *e)
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

func (m *mockEventRepo) Search(_ context.Context, keyword string, p pagination.Params) ([]entity.Event, int, error) {
	if m.search != nil {
		return m.search(keyword, p)
	}
	all := make([]entity.Event, 0, len(m.items))
	for _, e := range m.items {
		if contains(e.Title, keyword) || contains(e.Description, keyword) {
			all = append(all, *e)
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

// TestEventUseCase_Create covers happy path, validation, and nil input.
func TestEventUseCase_Create(t *testing.T) {
	repo := newMockEventRepo()
	uc := usecase.NewEventUseCase(repo)

	t.Run("happy path with valid event type", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.EventRequest{
			Title:        "Publication of Shanghan Lun",
			DynastyID:    1,
			OccurredYear: 210,
			EventType:    entity.EventTypePublish,
			Description:  "Zhang Zhongjing's work circulated",
			Impact:       "Foundational text",
			Location:     "Chang'an",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "Publication of Shanghan Lun", resp.Title)
		assert.Equal(t, entity.EventTypePublish, resp.EventType)
		assert.NotZero(t, resp.ID)
	})

	t.Run("happy path with empty event type", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.EventRequest{Title: "Untitled"})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "", resp.EventType)
	})

	t.Run("empty title rejected", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.EventRequest{Title: ""})
		require.Error(t, err)
		assert.Nil(t, resp)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.InvalidParams, e.Code)
		}
	})

	t.Run("invalid event type rejected", func(t *testing.T) {
		resp, err := uc.Create(context.Background(), &dto.EventRequest{
			Title:     "X",
			EventType: "coup",
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
		repo := newMockEventRepo()
		repo.create = func(*entity.Event) error { return errors.New("db down") }
		uc := usecase.NewEventUseCase(repo)
		resp, err := uc.Create(context.Background(), &dto.EventRequest{Title: "X"})
		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

// TestEventUseCase_Get covers the found and not-found paths.
func TestEventUseCase_Get(t *testing.T) {
	repo := newMockEventRepo()
	uc := usecase.NewEventUseCase(repo)

	created, err := uc.Create(context.Background(), &dto.EventRequest{Title: "War of Red Cliffs"})
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		got, err := uc.Get(context.Background(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "War of Red Cliffs", got.Title)
	})

	t.Run("not found", func(t *testing.T) {
		got, err := uc.Get(context.Background(), 999999)
		require.Error(t, err)
		assert.Nil(t, got)
	})
}

// TestEventUseCase_Update covers happy path, not-found, and validation.
func TestEventUseCase_Update(t *testing.T) {
	repo := newMockEventRepo()
	uc := usecase.NewEventUseCase(repo)

	created, err := uc.Create(context.Background(), &dto.EventRequest{Title: "Old"})
	require.NoError(t, err)

	t.Run("happy path", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), created.ID, &dto.EventRequest{
			Title:     "New",
			EventType: entity.EventTypeAcademic,
		})
		require.NoError(t, err)
		assert.Equal(t, "New", resp.Title)
		assert.Equal(t, entity.EventTypeAcademic, resp.EventType)
	})

	t.Run("not found", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), 999999, &dto.EventRequest{Title: "X"})
		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("invalid event type rejected", func(t *testing.T) {
		resp, err := uc.Update(context.Background(), created.ID, &dto.EventRequest{
			Title:     "X",
			EventType: "bogus",
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

// TestEventUseCase_Delete covers delete and not-found paths.
func TestEventUseCase_Delete(t *testing.T) {
	repo := newMockEventRepo()
	uc := usecase.NewEventUseCase(repo)

	created, err := uc.Create(context.Background(), &dto.EventRequest{Title: "ToDelete"})
	require.NoError(t, err)

	require.NoError(t, uc.Delete(context.Background(), created.ID))
	_, err = uc.Get(context.Background(), created.ID)
	require.Error(t, err)

	err = uc.Delete(context.Background(), created.ID)
	require.Error(t, err)

	err = uc.Delete(context.Background(), 0)
	require.Error(t, err)
}

// TestEventUseCase_List exercises pagination and the total count.
func TestEventUseCase_List(t *testing.T) {
	repo := newMockEventRepo()
	uc := usecase.NewEventUseCase(repo)

	for _, title := range []string{"A", "B", "C"} {
		_, err := uc.Create(context.Background(), &dto.EventRequest{Title: title})
		require.NoError(t, err)
	}

	resp, err := uc.List(context.Background(), pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, 2, resp.TotalPage)
	require.Len(t, resp.Items, 2)
}

// TestEventUseCase_Search exercises the keyword filter.
func TestEventUseCase_Search(t *testing.T) {
	repo := newMockEventRepo()
	uc := usecase.NewEventUseCase(repo)

	_, err := uc.Create(context.Background(), &dto.EventRequest{
		Title:       "Publishing",
		Description: "Shanghan Lun published",
	})
	require.NoError(t, err)
	_, err = uc.Create(context.Background(), &dto.EventRequest{
		Title:       "War",
		Description: "Red Cliffs",
	})
	require.NoError(t, err)

	resp, err := uc.Search(context.Background(), "Publishing", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, "Publishing", resp.Items[0].Title)

	// Empty keyword falls back to List.
	resp, err = uc.Search(context.Background(), "", pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Total)
}
