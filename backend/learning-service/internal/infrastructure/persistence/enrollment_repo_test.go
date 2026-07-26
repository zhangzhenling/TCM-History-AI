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

// newEnrollment builds an Enrollment with a fresh snowflake id.
func newEnrollment(userID, courseID int64) *entity.Enrollment {
	e := &entity.Enrollment{
		UserID:   userID,
		CourseID: courseID,
		Status:   entity.EnrollmentStatusEnrolled,
	}
	e.ID = idgen.Next()
	return e
}

func TestEnrollmentRepo_Create(t *testing.T) {
	db := setupDB(t, &entity.Enrollment{})
	repo := NewEnrollmentRepo(db)
	ctx := context.Background()

	e := newEnrollment(1, 100)
	e.ProgressPercent = 10
	require.NoError(t, repo.Create(ctx, e))

	var got entity.Enrollment
	require.NoError(t, db.First(&got, "id = ?", e.ID).Error)
	assert.Equal(t, int64(1), got.UserID)
	assert.Equal(t, int64(100), got.CourseID)
	assert.Equal(t, 10, got.ProgressPercent)
	assert.Equal(t, entity.EnrollmentStatusEnrolled, got.Status)
	assert.False(t, got.CreatedAt.IsZero())
	assert.False(t, got.UpdatedAt.IsZero())
}

func TestEnrollmentRepo_Create_DBError(t *testing.T) {
	db := setupDB(t) // no models → no enrollment table
	repo := NewEnrollmentRepo(db)
	ctx := context.Background()

	err := repo.Create(ctx, newEnrollment(1, 1))
	require.Error(t, err)
	fromErr := errno.From(err)
	require.NotNil(t, fromErr)
	assert.Equal(t, errno.InternalError, fromErr.Code)
}

func TestEnrollmentRepo_Delete(t *testing.T) {
	db := setupDB(t, &entity.Enrollment{})
	repo := NewEnrollmentRepo(db)
	ctx := context.Background()

	e := newEnrollment(1, 100)
	require.NoError(t, repo.Create(ctx, e))
	require.NoError(t, repo.Delete(ctx, e.ID))

	// Soft-deleted: should not be returned by FindByID.
	got, err := repo.FindByID(ctx, e.ID)
	require.NoError(t, err)
	require.Nil(t, got)

	// But the row should still exist with deleted_at set.
	var raw entity.Enrollment
	require.NoError(t, db.Unscoped().First(&raw, "id = ?", e.ID).Error)
	assert.False(t, raw.DeletedAt.Time.IsZero())
}

func TestEnrollmentRepo_Delete_NotFound(t *testing.T) {
	db := setupDB(t, &entity.Enrollment{})
	repo := NewEnrollmentRepo(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	require.Error(t, err)
	fromErr := errno.From(err)
	require.NotNil(t, fromErr)
	assert.Equal(t, errno.NotFound, fromErr.Code)
}

func TestEnrollmentRepo_FindByID(t *testing.T) {
	db := setupDB(t, &entity.Enrollment{})
	repo := NewEnrollmentRepo(db)
	ctx := context.Background()

	e := newEnrollment(1, 100)
	require.NoError(t, repo.Create(ctx, e))

	got, err := repo.FindByID(ctx, e.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, e.ID, got.ID)
	assert.Equal(t, int64(1), got.UserID)
	assert.Equal(t, int64(100), got.CourseID)
}

func TestEnrollmentRepo_FindByID_NotFound(t *testing.T) {
	db := setupDB(t, &entity.Enrollment{})
	repo := NewEnrollmentRepo(db)
	ctx := context.Background()

	got, err := repo.FindByID(ctx, 99999)
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestEnrollmentRepo_FindByUserAndCourse(t *testing.T) {
	db := setupDB(t, &entity.Enrollment{})
	repo := NewEnrollmentRepo(db)
	ctx := context.Background()

	e := newEnrollment(7, 200)
	require.NoError(t, repo.Create(ctx, e))

	got, err := repo.FindByUserAndCourse(ctx, 7, 200)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, e.ID, got.ID)
}

func TestEnrollmentRepo_FindByUserAndCourse_NotFound(t *testing.T) {
	db := setupDB(t, &entity.Enrollment{})
	repo := NewEnrollmentRepo(db)
	ctx := context.Background()

	got, err := repo.FindByUserAndCourse(ctx, 7, 200)
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestEnrollmentRepo_ListByUser(t *testing.T) {
	db := setupDB(t, &entity.Enrollment{})
	repo := NewEnrollmentRepo(db)
	ctx := context.Background()

	// User 1 has 3 enrollments; user 2 has 1.
	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Create(ctx, newEnrollment(1, int64(100+i))))
	}
	require.NoError(t, repo.Create(ctx, newEnrollment(2, 100)))

	items, total, err := repo.ListByUser(ctx, 1, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, items, 3)

	items2, total2, err := repo.ListByUser(ctx, 2, pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total2)
	require.Len(t, items2, 1)
}

func TestEnrollmentRepo_ListByUser_Pagination(t *testing.T) {
	db := setupDB(t, &entity.Enrollment{})
	repo := NewEnrollmentRepo(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Create(ctx, newEnrollment(1, int64(100+i))))
	}

	items, total, err := repo.ListByUser(ctx, 1, pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	require.Len(t, items, 2)

	items2, _, err := repo.ListByUser(ctx, 1, pagination.Params{Page: 3, PageSize: 2})
	require.NoError(t, err)
	require.Len(t, items2, 1)
}

func TestEnrollmentRepo_ListByUser_Defaults(t *testing.T) {
	db := setupDB(t, &entity.Enrollment{})
	repo := NewEnrollmentRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newEnrollment(1, 100)))
	require.NoError(t, repo.Create(ctx, newEnrollment(1, 101)))

	items, total, err := repo.ListByUser(ctx, 1, pagination.Params{})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, items, 2)
}

func TestEnrollmentRepo_UpdateProgress(t *testing.T) {
	db := setupDB(t, &entity.Enrollment{})
	repo := NewEnrollmentRepo(db)
	ctx := context.Background()

	e := newEnrollment(1, 100)
	require.NoError(t, repo.Create(ctx, e))

	// UpdateProgress(ctx, id, lastLessonID, progressPercent).
	require.NoError(t, repo.UpdateProgress(ctx, e.ID, 42, 7))

	got, err := repo.FindByID(ctx, e.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 7, got.ProgressPercent)
	assert.Equal(t, int64(42), got.LastLessonID)
	assert.Equal(t, entity.EnrollmentStatusInProgress, got.Status)
}

func TestEnrollmentRepo_UpdateProgress_ZeroResetsStatus(t *testing.T) {
	db := setupDB(t, &entity.Enrollment{})
	repo := NewEnrollmentRepo(db)
	ctx := context.Background()

	e := newEnrollment(1, 100)
	e.Status = entity.EnrollmentStatusInProgress
	e.ProgressPercent = 50
	require.NoError(t, repo.Create(ctx, e))

	require.NoError(t, repo.UpdateProgress(ctx, e.ID, 0, 0))

	got, err := repo.FindByID(ctx, e.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 0, got.ProgressPercent)
	assert.Equal(t, entity.EnrollmentStatusEnrolled, got.Status)
}

func TestEnrollmentRepo_UpdateProgress_NotFound(t *testing.T) {
	db := setupDB(t, &entity.Enrollment{})
	repo := NewEnrollmentRepo(db)
	ctx := context.Background()

	err := repo.UpdateProgress(ctx, 99999, 10, 1)
	require.Error(t, err)
	fromErr := errno.From(err)
	require.NotNil(t, fromErr)
	assert.Equal(t, errno.NotFound, fromErr.Code)
}

func TestEnrollmentRepo_MarkCompleted(t *testing.T) {
	db := setupDB(t, &entity.Enrollment{})
	repo := NewEnrollmentRepo(db)
	ctx := context.Background()

	e := newEnrollment(1, 100)
	require.NoError(t, repo.Create(ctx, e))

	require.NoError(t, repo.MarkCompleted(ctx, e.ID))

	got, err := repo.FindByID(ctx, e.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 100, got.ProgressPercent)
	assert.Equal(t, entity.EnrollmentStatusCompleted, got.Status)
	require.NotNil(t, got.CompletedAt)
}

func TestEnrollmentRepo_MarkCompleted_NotFound(t *testing.T) {
	db := setupDB(t, &entity.Enrollment{})
	repo := NewEnrollmentRepo(db)
	ctx := context.Background()

	err := repo.MarkCompleted(ctx, 99999)
	require.Error(t, err)
	fromErr := errno.From(err)
	require.NotNil(t, fromErr)
	assert.Equal(t, errno.NotFound, fromErr.Code)
}
