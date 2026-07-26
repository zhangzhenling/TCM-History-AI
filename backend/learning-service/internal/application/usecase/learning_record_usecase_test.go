package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/learning-service/internal/application/dto"
	"tcm-history-ai/backend/learning-service/internal/application/usecase"
	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// newLearningRecordUseCase wires up a LearningRecordUseCase with in-memory mocks.
func newLearningRecordUseCase() (*usecase.LearningRecordUseCase, *mockLearningRecordRepo) {
	repo := newMockLearningRecordRepo()
	uc := usecase.NewLearningRecordUseCase(repo)
	return uc, repo
}

// TestLearningRecordUseCase_Record_NewRecordCreates verifies a fresh record is
// created when no existing record is found for the (user, lesson) pair.
func TestLearningRecordUseCase_Record_NewRecordCreates(t *testing.T) {
	uc, repo := newLearningRecordUseCase()
	resp, err := uc.Record(context.Background(), &dto.LearningRecordRequest{
		UserID:          7,
		LessonID:        3,
		CourseID:        1,
		DurationSeconds: 60,
		PositionPercent: 30,
		LastPosition:    300,
		IsCompleted:     false,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotZero(t, resp.ID)
	assert.Equal(t, int64(7), resp.UserID)
	assert.Equal(t, int64(3), resp.LessonID)
	assert.Equal(t, int64(1), resp.CourseID)
	assert.Equal(t, 60, resp.DurationSeconds)
	assert.Equal(t, 30, resp.PositionPercent)
	assert.Equal(t, 300, resp.LastPosition)
	assert.False(t, resp.IsCompleted)
	assert.False(t, resp.LearnedAt.IsZero())

	// Repo contains the row.
	got, err := repo.FindByID(context.Background(), resp.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 60, got.DurationSeconds)
}

// TestLearningRecordUseCase_Record_ExistingRecordAccumulates verifies a
// subsequent Record call accumulates duration, keeps the larger position, and
// promotes to completed when IsCompleted becomes true.
func TestLearningRecordUseCase_Record_ExistingRecordAccumulates(t *testing.T) {
	uc, repo := newLearningRecordUseCase()
	_, err := uc.Record(context.Background(), &dto.LearningRecordRequest{
		UserID:          1,
		LessonID:        2,
		CourseID:        3,
		DurationSeconds: 60,
		PositionPercent: 30,
		LastPosition:    100,
	})
	require.NoError(t, err)

	resp, err := uc.Record(context.Background(), &dto.LearningRecordRequest{
		UserID:          1,
		LessonID:        2,
		CourseID:        3,
		DurationSeconds: 40, // adds 40 → 100
		PositionPercent: 50, // larger → 50
		LastPosition:    200, // larger → 200
		IsCompleted:     true, // promotes to completed
	})
	require.NoError(t, err)
	assert.Equal(t, 100, resp.DurationSeconds)
	assert.Equal(t, 50, resp.PositionPercent)
	assert.Equal(t, 200, resp.LastPosition)
	assert.True(t, resp.IsCompleted)
	// Repo should hold the same single record (upserted, not duplicated).
	assert.Len(t, repo.items, 1)
}

// TestLearningRecordUseCase_Record_KeepsExistingPositionWhenInputSmaller
// verifies that a smaller incoming PositionPercent does not overwrite the
// existing value.
func TestLearningRecordUseCase_Record_KeepsExistingPositionWhenInputSmaller(t *testing.T) {
	uc, repo := newLearningRecordUseCase()
	_, err := uc.Record(context.Background(), &dto.LearningRecordRequest{
		UserID: 1, LessonID: 2, CourseID: 3,
		DurationSeconds: 60,
		PositionPercent: 80,
		LastPosition:    800,
	})
	require.NoError(t, err)
	resp, err := uc.Record(context.Background(), &dto.LearningRecordRequest{
		UserID: 1, LessonID: 2, CourseID: 3,
		DurationSeconds: 10,
		PositionPercent: 20, // smaller — should be ignored
		LastPosition:    100, // smaller — should be ignored
	})
	require.NoError(t, err)
	assert.Equal(t, 80, resp.PositionPercent)
	assert.Equal(t, 800, resp.LastPosition)
	// Duration still accumulates.
	assert.Equal(t, 70, resp.DurationSeconds)
	_ = repo
}

// TestLearningRecordUseCase_Record_DoesNotUnsetCompletion verifies that
// IsCompleted, once true, stays true even if a later Record call sets
// IsCompleted=false.
func TestLearningRecordUseCase_Record_DoesNotUnsetCompletion(t *testing.T) {
	uc, _ := newLearningRecordUseCase()
	_, err := uc.Record(context.Background(), &dto.LearningRecordRequest{
		UserID: 1, LessonID: 2, CourseID: 3,
		IsCompleted: true,
	})
	require.NoError(t, err)
	resp, err := uc.Record(context.Background(), &dto.LearningRecordRequest{
		UserID: 1, LessonID: 2, CourseID: 3,
		IsCompleted: false, // should not unset
	})
	require.NoError(t, err)
	assert.True(t, resp.IsCompleted, "completion should be sticky")
}

// TestLearningRecordUseCase_Record_ValidationErrors covers input validations.
func TestLearningRecordUseCase_Record_ValidationErrors(t *testing.T) {
	uc, _ := newLearningRecordUseCase()
	cases := []struct {
		name string
		in   *dto.LearningRecordRequest
	}{
		{"nil request", nil},
		{"zero user_id", &dto.LearningRecordRequest{LessonID: 1, CourseID: 1}},
		{"zero lesson_id", &dto.LearningRecordRequest{UserID: 1, CourseID: 1}},
		{"zero course_id", &dto.LearningRecordRequest{UserID: 1, LessonID: 1}},
		{"negative user_id", &dto.LearningRecordRequest{UserID: -1, LessonID: 1, CourseID: 1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := uc.Record(context.Background(), c.in)
			require.Error(t, err)
			var e *errno.Error
			if errors.As(err, &e) {
				assert.Equal(t, errno.InvalidParams, e.Code)
			}
		})
	}
}

// TestLearningRecordUseCase_Record_FindError verifies repo errors propagate.
func TestLearningRecordUseCase_Record_FindError(t *testing.T) {
	uc, repo := newLearningRecordUseCase()
	repo.findByUserAndLesson = func(int64, int64) (*entity.LearningRecord, error) {
		return nil, errors.New("find err")
	}
	_, err := uc.Record(context.Background(), &dto.LearningRecordRequest{
		UserID: 1, LessonID: 1, CourseID: 1,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find err")
}

// TestLearningRecordUseCase_Record_NewRecordUpsertError verifies that Upsert
// errors propagate on the new-record path.
func TestLearningRecordUseCase_Record_NewRecordUpsertError(t *testing.T) {
	uc, repo := newLearningRecordUseCase()
	repo.upsert = func(*entity.LearningRecord) error { return errors.New("upsert err") }
	_, err := uc.Record(context.Background(), &dto.LearningRecordRequest{
		UserID: 1, LessonID: 1, CourseID: 1,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upsert err")
}

// TestLearningRecordUseCase_Record_ExistingRecordUpsertError verifies that
// Upsert errors propagate on the existing-record path.
func TestLearningRecordUseCase_Record_ExistingRecordUpsertError(t *testing.T) {
	uc, repo := newLearningRecordUseCase()
	_, err := uc.Record(context.Background(), &dto.LearningRecordRequest{
		UserID: 1, LessonID: 1, CourseID: 1,
	})
	require.NoError(t, err)
	repo.upsert = func(*entity.LearningRecord) error { return errors.New("upsert err") }
	_, err = uc.Record(context.Background(), &dto.LearningRecordRequest{
		UserID: 1, LessonID: 1, CourseID: 1,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upsert err")
}

// TestLearningRecordUseCase_Get covers found / not-found / error paths.
func TestLearningRecordUseCase_Get(t *testing.T) {
	uc, repo := newLearningRecordUseCase()
	created, err := uc.Record(context.Background(), &dto.LearningRecordRequest{
		UserID: 1, LessonID: 1, CourseID: 1,
	})
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		got, err := uc.Get(context.Background(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
	})
	t.Run("not found", func(t *testing.T) {
		_, err := uc.Get(context.Background(), 9999)
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.NotFound, e.Code)
		}
	})
	t.Run("find error", func(t *testing.T) {
		repo.find = func(int64) (*entity.LearningRecord, error) {
			return nil, errors.New("find err")
		}
		_, err := uc.Get(context.Background(), 1)
		require.Error(t, err)
	})
}

// TestLearningRecordUseCase_ListByUser covers happy / error paths and
// pagination.
func TestLearningRecordUseCase_ListByUser(t *testing.T) {
	uc, repo := newLearningRecordUseCase()
	for i := 0; i < 3; i++ {
		_, err := uc.Record(context.Background(), &dto.LearningRecordRequest{
			UserID: 7, LessonID: int64(i + 1), CourseID: 1,
		})
		require.NoError(t, err)
	}

	t.Run("happy paginated", func(t *testing.T) {
		resp, err := uc.ListByUser(context.Background(), 7, pagination.Params{Page: 1, PageSize: 2})
		require.NoError(t, err)
		assert.Equal(t, 3, resp.Total)
		assert.Equal(t, 2, resp.TotalPage)
		require.Len(t, resp.Items, 2)
	})
	t.Run("repo error", func(t *testing.T) {
		repo.listByUser = func(int64, pagination.Params) ([]entity.LearningRecord, int, error) {
			return nil, 0, errors.New("list err")
		}
		_, err := uc.ListByUser(context.Background(), 7, pagination.Params{Page: 1, PageSize: 2})
		require.Error(t, err)
	})
}

// TestLearningRecordUseCase_ListByUserAndLesson covers found / not-found /
// error paths.
func TestLearningRecordUseCase_ListByUserAndLesson(t *testing.T) {
	uc, repo := newLearningRecordUseCase()
	created, err := uc.Record(context.Background(), &dto.LearningRecordRequest{
		UserID: 7, LessonID: 3, CourseID: 1,
	})
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		got, err := uc.ListByUserAndLesson(context.Background(), 7, 3)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
	})
	t.Run("not found", func(t *testing.T) {
		_, err := uc.ListByUserAndLesson(context.Background(), 7, 9999)
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.NotFound, e.Code)
		}
	})
	t.Run("find error", func(t *testing.T) {
		repo.findByUserAndLesson = func(int64, int64) (*entity.LearningRecord, error) {
			return nil, errors.New("find err")
		}
		_, err := uc.ListByUserAndLesson(context.Background(), 7, 3)
		require.Error(t, err)
	})
}

// TestLearningRecordUseCase_MarkCompleted covers happy / error paths.
func TestLearningRecordUseCase_MarkCompleted(t *testing.T) {
	uc, repo := newLearningRecordUseCase()
	created, err := uc.Record(context.Background(), &dto.LearningRecordRequest{
		UserID: 1, LessonID: 1, CourseID: 1,
	})
	require.NoError(t, err)

	t.Run("happy", func(t *testing.T) {
		require.NoError(t, uc.MarkCompleted(context.Background(), created.ID))
		got, _ := repo.FindByID(context.Background(), created.ID)
		require.NotNil(t, got)
		assert.True(t, got.IsCompleted)
	})
	t.Run("repo error", func(t *testing.T) {
		repo.markCompleted = func(int64) error { return errors.New("mark err") }
		err := uc.MarkCompleted(context.Background(), created.ID)
		require.Error(t, err)
	})
}

// TestToLearningRecordResponse_Timestamps verifies timestamps are formatted
// when set on the entity.
func TestToLearningRecordResponse_Timestamps(t *testing.T) {
	uc, repo := newLearningRecordUseCase()
	created, err := uc.Record(context.Background(), &dto.LearningRecordRequest{
		UserID: 1, LessonID: 1, CourseID: 1,
	})
	require.NoError(t, err)
	repo.mu.Lock()
	if r, ok := repo.items[created.ID]; ok {
		r.CreatedAt = time.Now()
		r.UpdatedAt = time.Now()
	}
	repo.mu.Unlock()
	got, err := uc.Get(context.Background(), created.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, got.CreatedAt)
	assert.NotEmpty(t, got.UpdatedAt)
}
