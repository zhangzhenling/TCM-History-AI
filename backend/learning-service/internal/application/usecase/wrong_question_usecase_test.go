package usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/learning-service/internal/application/usecase"
	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

// newWrongQuestionUseCase wires up a WrongQuestionUseCase with in-memory mocks.
// A pre-existing wrong-question row is seeded so Get/MarkMastered can find it.
func newWrongQuestionUseCase() (*usecase.WrongQuestionUseCase, *mockWrongQuestionRepo, *entity.WrongQuestion) {
	repo := newMockWrongQuestionRepo()
	wq := &entity.WrongQuestion{
		UserID:         7,
		QuestionID:     11,
		ExamID:         21,
		AttemptID:      31,
		UserAnswerJSON: json.RawMessage(`[0]`),
		WrongCount:     1,
		LastWrongAt:    time.Now(),
		IsMastered:     false,
	}
	wq.ID = idgen.Next()
	_ = repo.Create(context.Background(), wq)
	uc := usecase.NewWrongQuestionUseCase(repo)
	return uc, repo, wq
}

// TestWrongQuestionUseCase_Get_HappyPath verifies a stored wrong question is
// returned.
func TestWrongQuestionUseCase_Get_HappyPath(t *testing.T) {
	uc, repo, wq := newWrongQuestionUseCase()
	resp, err := uc.Get(context.Background(), wq.ID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, wq.ID, resp.ID)
	assert.Equal(t, int64(7), resp.UserID)
	assert.Equal(t, int64(11), resp.QuestionID)
	assert.Equal(t, int64(21), resp.ExamID)
	assert.Equal(t, int64(31), resp.AttemptID)
	assert.Equal(t, json.RawMessage(`[0]`), resp.UserAnswerJSON)
	assert.Equal(t, 1, resp.WrongCount)
	assert.False(t, resp.IsMastered)
	assert.NotEmpty(t, resp.LastWrongAt)
	_ = repo
}

// TestWrongQuestionUseCase_Get_NotFound verifies missing record is rejected.
func TestWrongQuestionUseCase_Get_NotFound(t *testing.T) {
	uc, _, _ := newWrongQuestionUseCase()
	_, err := uc.Get(context.Background(), 9999)
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.NotFound, e.Code)
	}
}

// TestWrongQuestionUseCase_Get_FindError verifies repo errors propagate.
func TestWrongQuestionUseCase_Get_FindError(t *testing.T) {
	uc, repo, _ := newWrongQuestionUseCase()
	repo.find = func(int64) (*entity.WrongQuestion, error) {
		return nil, errors.New("find err")
	}
	_, err := uc.Get(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find err")
}

// TestWrongQuestionUseCase_ListByUser covers happy / error paths and
// pagination.
func TestWrongQuestionUseCase_ListByUser(t *testing.T) {
	uc, repo, _ := newWrongQuestionUseCase()
	// Seed 2 more for user 7.
	for i := 0; i < 2; i++ {
		w := &entity.WrongQuestion{UserID: 7, QuestionID: int64(100 + i), ExamID: 1, WrongCount: 1}
		w.ID = idgen.Next()
		require.NoError(t, repo.Create(context.Background(), w))
	}

	t.Run("happy paginated", func(t *testing.T) {
		resp, err := uc.ListByUser(context.Background(), 7, pagination.Params{Page: 1, PageSize: 2})
		require.NoError(t, err)
		assert.Equal(t, 3, resp.Total)
		assert.Equal(t, 2, resp.TotalPage)
		require.Len(t, resp.Items, 2)
	})
	t.Run("repo error", func(t *testing.T) {
		repo.listByUser = func(int64, pagination.Params) ([]entity.WrongQuestion, int, error) {
			return nil, 0, errors.New("list err")
		}
		_, err := uc.ListByUser(context.Background(), 7, pagination.Params{Page: 1, PageSize: 2})
		require.Error(t, err)
	})
}

// TestWrongQuestionUseCase_ListByExam covers happy / error paths.
func TestWrongQuestionUseCase_ListByExam(t *testing.T) {
	uc, repo, wq := newWrongQuestionUseCase()
	// Seed 1 more for (user 7, exam 21).
	w := &entity.WrongQuestion{UserID: 7, QuestionID: 99, ExamID: 21, WrongCount: 1}
	w.ID = idgen.Next()
	require.NoError(t, repo.Create(context.Background(), w))
	_ = wq

	resp, err := uc.ListByExam(context.Background(), 7, 21, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Total)
	require.Len(t, resp.Items, 2)

	t.Run("repo error", func(t *testing.T) {
		repo.listByExam = func(int64, int64, pagination.Params) ([]entity.WrongQuestion, int, error) {
			return nil, 0, errors.New("list err")
		}
		_, err := uc.ListByExam(context.Background(), 7, 21, pagination.Params{Page: 1, PageSize: 10})
		require.Error(t, err)
	})
}

// TestWrongQuestionUseCase_MarkMastered_HappyPath verifies that marking as
// mastered flips IsMastered and refreshes UpdatedAt.
func TestWrongQuestionUseCase_MarkMastered_HappyPath(t *testing.T) {
	uc, repo, wq := newWrongQuestionUseCase()
	resp, err := uc.MarkMastered(context.Background(), wq.ID)
	require.NoError(t, err)
	assert.True(t, resp.IsMastered)
	assert.NotEmpty(t, resp.UpdatedAt)
	// Repo state also reflects the change.
	got, _ := repo.FindByID(context.Background(), wq.ID)
	require.NotNil(t, got)
	assert.True(t, got.IsMastered)
}

// TestWrongQuestionUseCase_MarkMastered_NotFound verifies missing record is
// rejected.
func TestWrongQuestionUseCase_MarkMastered_NotFound(t *testing.T) {
	uc, _, _ := newWrongQuestionUseCase()
	_, err := uc.MarkMastered(context.Background(), 9999)
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.NotFound, e.Code)
	}
}

// TestWrongQuestionUseCase_MarkMastered_FindError verifies find errors
// propagate.
func TestWrongQuestionUseCase_MarkMastered_FindError(t *testing.T) {
	uc, repo, _ := newWrongQuestionUseCase()
	repo.find = func(int64) (*entity.WrongQuestion, error) {
		return nil, errors.New("find err")
	}
	_, err := uc.MarkMastered(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find err")
}

// TestWrongQuestionUseCase_MarkMastered_RepoMarkError verifies the
// MarkMastered repo error propagates.
func TestWrongQuestionUseCase_MarkMastered_RepoMarkError(t *testing.T) {
	uc, repo, wq := newWrongQuestionUseCase()
	repo.markMastered = func(int64) error { return errors.New("mark err") }
	_, err := uc.MarkMastered(context.Background(), wq.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mark err")
}

// TestToWrongQuestionResponse_Timestamps verifies timestamps are formatted
// when set on the entity.
func TestToWrongQuestionResponse_Timestamps(t *testing.T) {
	uc, repo, wq := newWrongQuestionUseCase()
	repo.mu.Lock()
	if w, ok := repo.items[wq.ID]; ok {
		w.CreatedAt = time.Now()
		w.UpdatedAt = time.Now()
	}
	repo.mu.Unlock()
	got, err := uc.Get(context.Background(), wq.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, got.CreatedAt)
	assert.NotEmpty(t, got.UpdatedAt)
}
