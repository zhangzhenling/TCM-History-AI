package persistence

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

// newLearningRecord builds a LearningRecord with a fresh snowflake id.
func newLearningRecord(userID, lessonID, courseID int64) *entity.LearningRecord {
	r := &entity.LearningRecord{
		UserID:          userID,
		LessonID:        lessonID,
		CourseID:        courseID,
		DurationSeconds: 30,
		PositionPercent: 10,
		LastPosition:    100,
	}
	r.ID = idgen.Next()
	return r
}

func TestLearningRecordRepo_Upsert_Insert(t *testing.T) {
	db := setupDB(t, &entity.LearningRecord{})
	repo := NewLearningRecordRepo(db)
	ctx := context.Background()

	r := newLearningRecord(1, 10, 100)
	require.NoError(t, repo.Upsert(ctx, r))

	var got entity.LearningRecord
	require.NoError(t, db.First(&got, "id = ?", r.ID).Error)
	assert.Equal(t, int64(1), got.UserID)
	assert.Equal(t, int64(10), got.LessonID)
	assert.Equal(t, int64(100), got.CourseID)
	assert.Equal(t, 30, got.DurationSeconds)
	assert.Equal(t, 10, got.PositionPercent)
	assert.False(t, got.CreatedAt.IsZero())
}

func TestLearningRecordRepo_Upsert_Update(t *testing.T) {
	db := setupDB(t, &entity.LearningRecord{})
	repo := NewLearningRecordRepo(db)
	ctx := context.Background()

	r := newLearningRecord(1, 10, 100)
	require.NoError(t, repo.Upsert(ctx, r))

	// Re-upsert with the same id and updated progress fields.
	r.DurationSeconds = 90
	r.PositionPercent = 75
	r.LastPosition = 750
	r.IsCompleted = true
	require.NoError(t, repo.Upsert(ctx, r))

	var got entity.LearningRecord
	require.NoError(t, db.First(&got, "id = ?", r.ID).Error)
	assert.Equal(t, 90, got.DurationSeconds)
	assert.Equal(t, 75, got.PositionPercent)
	assert.Equal(t, 750, got.LastPosition)
	assert.True(t, got.IsCompleted)
}

func TestLearningRecordRepo_Upsert_DBError(t *testing.T) {
	db := setupDB(t) // no models → no learning_records table
	repo := NewLearningRecordRepo(db)
	ctx := context.Background()

	err := repo.Upsert(ctx, newLearningRecord(1, 1, 1))
	require.Error(t, err)
	fromErr := errno.From(err)
	require.NotNil(t, fromErr)
	assert.Equal(t, errno.InternalError, fromErr.Code)
}

func TestLearningRecordRepo_FindByID(t *testing.T) {
	db := setupDB(t, &entity.LearningRecord{})
	repo := NewLearningRecordRepo(db)
	ctx := context.Background()

	r := newLearningRecord(1, 10, 100)
	require.NoError(t, repo.Upsert(ctx, r))

	got, err := repo.FindByID(ctx, r.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, r.ID, got.ID)
	assert.Equal(t, int64(1), got.UserID)
	assert.Equal(t, int64(10), got.LessonID)
}

func TestLearningRecordRepo_FindByID_NotFound(t *testing.T) {
	db := setupDB(t, &entity.LearningRecord{})
	repo := NewLearningRecordRepo(db)
	ctx := context.Background()

	got, err := repo.FindByID(ctx, 99999)
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestLearningRecordRepo_FindByUserAndLesson(t *testing.T) {
	db := setupDB(t, &entity.LearningRecord{})
	repo := NewLearningRecordRepo(db)
	ctx := context.Background()

	r := newLearningRecord(7, 42, 100)
	require.NoError(t, repo.Upsert(ctx, r))

	got, err := repo.FindByUserAndLesson(ctx, 7, 42)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, r.ID, got.ID)
}

func TestLearningRecordRepo_FindByUserAndLesson_NotFound(t *testing.T) {
	db := setupDB(t, &entity.LearningRecord{})
	repo := NewLearningRecordRepo(db)
	ctx := context.Background()

	got, err := repo.FindByUserAndLesson(ctx, 7, 42)
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestLearningRecordRepo_ListByUser(t *testing.T) {
	db := setupDB(t, &entity.LearningRecord{})
	repo := NewLearningRecordRepo(db)
	ctx := context.Background()

	// User 1 has 3 records (across different lessons); user 2 has 1.
	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Upsert(ctx, newLearningRecord(1, int64(10+i), 100)))
	}
	require.NoError(t, repo.Upsert(ctx, newLearningRecord(2, 10, 100)))

	items, total, err := repo.ListByUser(ctx, 1, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, items, 3)

	items2, total2, err := repo.ListByUser(ctx, 2, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total2)
	require.Len(t, items2, 1)
}

func TestLearningRecordRepo_ListByUser_Pagination(t *testing.T) {
	db := setupDB(t, &entity.LearningRecord{})
	repo := NewLearningRecordRepo(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Upsert(ctx, newLearningRecord(1, int64(10+i), 100)))
	}

	items, total, err := repo.ListByUser(ctx, 1, pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	require.Len(t, items, 2)

	items2, _, err := repo.ListByUser(ctx, 1, pagination.Params{Page: 3, PageSize: 2})
	require.NoError(t, err)
	require.Len(t, items2, 1)
}

func TestLearningRecordRepo_ListByUserAndCourse(t *testing.T) {
	db := setupDB(t, &entity.LearningRecord{})
	repo := NewLearningRecordRepo(db)
	ctx := context.Background()

	// (user 1, course 100) has 3 records; (user 1, course 200) has 1.
	// Use distinct lesson_ids per row because learning_records has a
	// UNIQUE(user_id, lesson_id) constraint.
	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Upsert(ctx, newLearningRecord(1, int64(10+i), 100)))
	}
	require.NoError(t, repo.Upsert(ctx, newLearningRecord(1, 99, 200)))

	items, total, err := repo.ListByUserAndCourse(ctx, 1, 100, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, items, 3)

	items2, total2, err := repo.ListByUserAndCourse(ctx, 1, 200, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total2)
	require.Len(t, items2, 1)
}

func TestLearningRecordRepo_ListByUserAndCourse_Pagination(t *testing.T) {
	db := setupDB(t, &entity.LearningRecord{})
	repo := NewLearningRecordRepo(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Upsert(ctx, newLearningRecord(1, int64(10+i), 100)))
	}

	items, total, err := repo.ListByUserAndCourse(ctx, 1, 100, pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	require.Len(t, items, 2)

	items2, _, err := repo.ListByUserAndCourse(ctx, 1, 100, pagination.Params{Page: 3, PageSize: 2})
	require.NoError(t, err)
	require.Len(t, items2, 1)
}

func TestLearningRecordRepo_MarkCompleted(t *testing.T) {
	db := setupDB(t, &entity.LearningRecord{})
	repo := NewLearningRecordRepo(db)
	ctx := context.Background()

	r := newLearningRecord(1, 10, 100)
	require.NoError(t, repo.Upsert(ctx, r))

	require.NoError(t, repo.MarkCompleted(ctx, r.ID))

	got, err := repo.FindByID(ctx, r.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, got.IsCompleted)
	assert.False(t, got.LearnedAt.IsZero())
}

func TestLearningRecordRepo_MarkCompleted_NotFound(t *testing.T) {
	db := setupDB(t, &entity.LearningRecord{})
	repo := NewLearningRecordRepo(db)
	ctx := context.Background()

	err := repo.MarkCompleted(ctx, 99999)
	require.Error(t, err)
	fromErr := errno.From(err)
	require.NotNil(t, fromErr)
	assert.Equal(t, errno.NotFound, fromErr.Code)
}
