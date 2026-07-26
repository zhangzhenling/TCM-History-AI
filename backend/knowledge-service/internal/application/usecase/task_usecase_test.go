package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/knowledge-service/internal/application/usecase"
	"tcm-history-ai/backend/knowledge-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// --- mock EmbeddingTaskRepository ---

type mockTaskRepo struct {
	items          map[int64]*entity.EmbeddingTask
	createErr      error
	updateErr      error
	findErr        error
	findByDocErr   error
	listErr        error
	listByStatusErr error
}

func newMockTaskRepo() *mockTaskRepo {
	return &mockTaskRepo{items: map[int64]*entity.EmbeddingTask{}}
}

func (m *mockTaskRepo) Create(_ context.Context, t *entity.EmbeddingTask) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.items[t.ID] = t
	return nil
}

func (m *mockTaskRepo) Update(_ context.Context, t *entity.EmbeddingTask) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.items[t.ID] = t
	return nil
}

func (m *mockTaskRepo) FindByID(_ context.Context, id int64) (*entity.EmbeddingTask, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	if t, ok := m.items[id]; ok {
		clone := *t
		return &clone, nil
	}
	return nil, nil
}

func (m *mockTaskRepo) FindByDocumentID(_ context.Context, documentID int64) ([]entity.EmbeddingTask, error) {
	if m.findByDocErr != nil {
		return nil, m.findByDocErr
	}
	out := make([]entity.EmbeddingTask, 0)
	for _, t := range m.items {
		if t.DocumentID == documentID {
			out = append(out, *t)
		}
	}
	return out, nil
}

func (m *mockTaskRepo) List(_ context.Context, p pagination.Params) ([]entity.EmbeddingTask, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	all := make([]entity.EmbeddingTask, 0, len(m.items))
	for _, t := range m.items {
		all = append(all, *t)
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

func (m *mockTaskRepo) ListByStatus(_ context.Context, status string, p pagination.Params) ([]entity.EmbeddingTask, int, error) {
	if m.listByStatusErr != nil {
		return nil, 0, m.listByStatusErr
	}
	all := make([]entity.EmbeddingTask, 0, len(m.items))
	for _, t := range m.items {
		if status == "" || t.Status == status {
			all = append(all, *t)
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

// seedTask inserts a task into the mock repo and returns it.
func seedTask(repo *mockTaskRepo, id int64, status string, docID int64) *entity.EmbeddingTask {
	t := &entity.EmbeddingTask{
		DocumentID: docID,
		TaskType:   entity.TaskTypeDocument,
		Status:     status,
		Stage:      entity.StageEmbed,
		Progress:   50,
	}
	t.ID = id // ID lives on the embedded BaseModel; assign via field access.
	repo.items[id] = t
	return t
}

// --- tests ---

func TestTaskUseCase_Get(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		repo := newMockTaskRepo()
		seedTask(repo, 1, entity.TaskStatusRunning, 10)
		uc := usecase.NewTaskUseCase(repo)
		got, err := uc.Get(context.Background(), 1)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, int64(1), got.ID)
		assert.Equal(t, int64(10), got.DocumentID)
		assert.Equal(t, entity.TaskStatusRunning, got.Status)
		assert.Equal(t, entity.StageEmbed, got.Stage)
	})

	t.Run("not found", func(t *testing.T) {
		uc := usecase.NewTaskUseCase(newMockTaskRepo())
		got, err := uc.Get(context.Background(), 999)
		requireError(t, err, errno.NotFound)
		assert.Nil(t, got)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := newMockTaskRepo()
		repo.findErr = errors.New("db down")
		uc := usecase.NewTaskUseCase(repo)
		got, err := uc.Get(context.Background(), 1)
		require.Error(t, err)
		assert.Nil(t, got)
	})
}

func TestTaskUseCase_ListByDocument(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		repo := newMockTaskRepo()
		seedTask(repo, 1, entity.TaskStatusDone, 10)
		seedTask(repo, 2, entity.TaskStatusFailed, 10)
		seedTask(repo, 3, entity.TaskStatusDone, 11)
		uc := usecase.NewTaskUseCase(repo)
		got, err := uc.ListByDocument(context.Background(), 10)
		require.NoError(t, err)
		require.Len(t, got, 2)
	})

	t.Run("empty result", func(t *testing.T) {
		uc := usecase.NewTaskUseCase(newMockTaskRepo())
		got, err := uc.ListByDocument(context.Background(), 999)
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := newMockTaskRepo()
		repo.findByDocErr = errors.New("boom")
		uc := usecase.NewTaskUseCase(repo)
		got, err := uc.ListByDocument(context.Background(), 10)
		require.Error(t, err)
		assert.Nil(t, got)
	})
}

func TestTaskUseCase_List(t *testing.T) {
	repo := newMockTaskRepo()
	seedTask(repo, 1, entity.TaskStatusDone, 10)
	seedTask(repo, 2, entity.TaskStatusQueued, 10)
	seedTask(repo, 3, entity.TaskStatusFailed, 11)

	uc := usecase.NewTaskUseCase(repo)
	resp, err := uc.List(context.Background(), pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, 2, resp.TotalPage)
	require.Len(t, resp.Items, 2)
}

func TestTaskUseCase_List_RepoError(t *testing.T) {
	repo := newMockTaskRepo()
	repo.listErr = errors.New("boom")
	uc := usecase.NewTaskUseCase(repo)
	resp, err := uc.List(context.Background(), pagination.Params{Page: 1, PageSize: 10})
	require.Error(t, err)
	assert.Equal(t, 0, resp.Total)
}

func TestTaskUseCase_ListByStatus(t *testing.T) {
	repo := newMockTaskRepo()
	seedTask(repo, 1, entity.TaskStatusDone, 10)
	seedTask(repo, 2, entity.TaskStatusQueued, 10)
	seedTask(repo, 3, entity.TaskStatusQueued, 11)

	uc := usecase.NewTaskUseCase(repo)

	t.Run("filter by status", func(t *testing.T) {
		resp, err := uc.ListByStatus(context.Background(), entity.TaskStatusQueued, pagination.Params{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Total)
		require.Len(t, resp.Items, 2)
	})

	t.Run("empty status falls back to List", func(t *testing.T) {
		resp, err := uc.ListByStatus(context.Background(), "", pagination.Params{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, 3, resp.Total)
	})

	t.Run("repo error", func(t *testing.T) {
		repo := newMockTaskRepo()
		repo.listByStatusErr = errors.New("boom")
		uc := usecase.NewTaskUseCase(repo)
		resp, err := uc.ListByStatus(context.Background(), entity.TaskStatusDone, pagination.Params{Page: 1, PageSize: 10})
		require.Error(t, err)
		assert.Equal(t, 0, resp.Total)
	})
}

func TestTaskUseCase_ListByStatus_RepoErrorOnListFallback(t *testing.T) {
	// When status is empty, the usecase delegates to List (not ListByStatus),
	// so List's error must surface.
	repo := newMockTaskRepo()
	repo.listErr = errors.New("list fail")
	uc := usecase.NewTaskUseCase(repo)
	resp, err := uc.ListByStatus(context.Background(), "", pagination.Params{Page: 1, PageSize: 10})
	require.Error(t, err)
	assert.Equal(t, 0, resp.Total)
}
