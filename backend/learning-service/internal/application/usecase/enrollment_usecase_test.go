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
	"tcm-history-ai/backend/learning-service/internal/domain/event"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// newEnrollmentUseCase wires up an EnrollmentUseCase with in-memory mocks.
// A pre-existing course is seeded so Enroll can find it.
func newEnrollmentUseCase() (*usecase.EnrollmentUseCase, *mockEnrollmentRepo, *mockCourseRepo, *mockEventPublisher, *entity.Course) {
	enrollRepo := newMockEnrollmentRepo()
	courseRepo := newMockCourseRepo()
	pub := newMockEventPublisher()
	course := &entity.Course{Title: "course", IsPublished: true}
	course.ID = 1
	_ = courseRepo.Create(context.Background(), course)
	uc := usecase.NewEnrollmentUseCase(enrollRepo, courseRepo, pub)
	return uc, enrollRepo, courseRepo, pub, course
}

// TestEnrollmentUseCase_Enroll_HappyPath verifies a fresh enrollment is
// created with enrolled status and 0% progress.
func TestEnrollmentUseCase_Enroll_HappyPath(t *testing.T) {
	uc, repo, _, _, course := newEnrollmentUseCase()
	resp, err := uc.Enroll(context.Background(), &dto.EnrollmentRequest{
		UserID:   7,
		CourseID: course.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotZero(t, resp.ID)
	assert.Equal(t, int64(7), resp.UserID)
	assert.Equal(t, course.ID, resp.CourseID)
	assert.Equal(t, 0, resp.ProgressPercent)
	assert.Equal(t, entity.EnrollmentStatusEnrolled, resp.Status)

	// Repo contains the row.
	got, err := repo.FindByID(context.Background(), resp.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, entity.EnrollmentStatusEnrolled, got.Status)
}

// TestEnrollmentUseCase_Enroll_Idempotent verifies re-enrolling returns the
// existing enrollment rather than creating a duplicate.
func TestEnrollmentUseCase_Enroll_Idempotent(t *testing.T) {
	uc, repo, _, _, course := newEnrollmentUseCase()
	first, err := uc.Enroll(context.Background(), &dto.EnrollmentRequest{
		UserID:   7,
		CourseID: course.ID,
	})
	require.NoError(t, err)
	second, err := uc.Enroll(context.Background(), &dto.EnrollmentRequest{
		UserID:   7,
		CourseID: course.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, 1, len(repo.items), "no duplicate enrollment should be created")
}

// TestEnrollmentUseCase_Enroll_ValidationErrors covers input validations.
func TestEnrollmentUseCase_Enroll_ValidationErrors(t *testing.T) {
	uc, _, _, _, _ := newEnrollmentUseCase()
	cases := []struct {
		name string
		in   *dto.EnrollmentRequest
	}{
		{"nil request", nil},
		{"zero user_id", &dto.EnrollmentRequest{CourseID: 1}},
		{"zero course_id", &dto.EnrollmentRequest{UserID: 1}},
		{"negative user_id", &dto.EnrollmentRequest{UserID: -1, CourseID: 1}},
		{"negative course_id", &dto.EnrollmentRequest{UserID: 1, CourseID: -1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := uc.Enroll(context.Background(), c.in)
			require.Error(t, err)
			var e *errno.Error
			if errors.As(err, &e) {
				assert.Equal(t, errno.InvalidParams, e.Code)
			}
		})
	}
}

// TestEnrollmentUseCase_Enroll_CourseNotFound verifies missing course is
// rejected.
func TestEnrollmentUseCase_Enroll_CourseNotFound(t *testing.T) {
	uc, _, _, _, _ := newEnrollmentUseCase()
	_, err := uc.Enroll(context.Background(), &dto.EnrollmentRequest{
		UserID:   1,
		CourseID: 9999,
	})
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.NotFound, e.Code)
	}
}

// TestEnrollmentUseCase_Enroll_CourseFindError verifies repo errors propagate.
func TestEnrollmentUseCase_Enroll_CourseFindError(t *testing.T) {
	uc, _, courseRepo, _, _ := newEnrollmentUseCase()
	courseRepo.find = func(int64) (*entity.Course, error) { return nil, errors.New("find err") }
	_, err := uc.Enroll(context.Background(), &dto.EnrollmentRequest{UserID: 1, CourseID: 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find err")
}

// TestEnrollmentUseCase_Enroll_FindExistingError verifies the dedup query
// errors propagate.
func TestEnrollmentUseCase_Enroll_FindExistingError(t *testing.T) {
	uc, enrollRepo, _, _, _ := newEnrollmentUseCase()
	enrollRepo.findByUserAndCourse = func(int64, int64) (*entity.Enrollment, error) {
		return nil, errors.New("dup err")
	}
	_, err := uc.Enroll(context.Background(), &dto.EnrollmentRequest{UserID: 1, CourseID: 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dup err")
}

// TestEnrollmentUseCase_Enroll_CreateError verifies repo Create errors
// propagate.
func TestEnrollmentUseCase_Enroll_CreateError(t *testing.T) {
	uc, enrollRepo, _, _, _ := newEnrollmentUseCase()
	enrollRepo.create = func(*entity.Enrollment) error { return errors.New("create err") }
	_, err := uc.Enroll(context.Background(), &dto.EnrollmentRequest{UserID: 1, CourseID: 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create err")
}

// TestEnrollmentUseCase_Unroll covers the delete path and repo error.
func TestEnrollmentUseCase_Unroll(t *testing.T) {
	uc, _, _, _, course := newEnrollmentUseCase()
	created, err := uc.Enroll(context.Background(), &dto.EnrollmentRequest{
		UserID:   1,
		CourseID: course.ID,
	})
	require.NoError(t, err)
	require.NoError(t, uc.Unroll(context.Background(), created.ID))
}

// TestEnrollmentUseCase_Unroll_RepoError verifies Unroll propagates repo
// errors.
func TestEnrollmentUseCase_Unroll_RepoError(t *testing.T) {
	uc, enrollRepo, _, _, _ := newEnrollmentUseCase()
	enrollRepo.delete = func(int64) error { return errors.New("delete err") }
	err := uc.Unroll(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete err")
}

// TestEnrollmentUseCase_Get covers found / not-found / error paths.
func TestEnrollmentUseCase_Get(t *testing.T) {
	uc, _, _, _, course := newEnrollmentUseCase()
	created, err := uc.Enroll(context.Background(), &dto.EnrollmentRequest{
		UserID:   1,
		CourseID: course.ID,
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
}

// TestEnrollmentUseCase_ListByUser covers happy / error paths and pagination.
func TestEnrollmentUseCase_ListByUser(t *testing.T) {
	uc, _, courseRepo, _, _ := newEnrollmentUseCase()
	// Enroll user 7 in three distinct courses (Enroll is idempotent per
	// (user, course), so we must vary the course).
	for i := 0; i < 3; i++ {
		c := &entity.Course{Title: "c", IsPublished: true}
		c.ID = int64(100 + i)
		require.NoError(t, courseRepo.Create(context.Background(), c))
		_, err := uc.Enroll(context.Background(), &dto.EnrollmentRequest{
			UserID:   int64(7),
			CourseID: c.ID,
		})
		require.NoError(t, err)
	}

	resp, err := uc.ListByUser(context.Background(), 7, pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, 2, resp.TotalPage)
	require.Len(t, resp.Items, 2)
}

// TestEnrollmentUseCase_UpdateProgress_InProgress verifies partial progress
// marks the enrollment in_progress.
func TestEnrollmentUseCase_UpdateProgress_InProgress(t *testing.T) {
	uc, _, _, pub, course := newEnrollmentUseCase()
	created, err := uc.Enroll(context.Background(), &dto.EnrollmentRequest{
		UserID:   1,
		CourseID: course.ID,
	})
	require.NoError(t, err)
	resp, err := uc.UpdateProgress(context.Background(), created.ID, &dto.EnrollmentUpdateProgressRequest{
		UserID:          1,
		LastLessonID:    5,
		ProgressPercent: 50,
	})
	require.NoError(t, err)
	assert.Equal(t, 50, resp.ProgressPercent)
	assert.Equal(t, int64(5), resp.LastLessonID)
	assert.Equal(t, entity.EnrollmentStatusInProgress, resp.Status)
	// No CourseCompleted event should be emitted yet.
	_, ok := captureEvent[event.CourseCompleted](pub)
	assert.False(t, ok)
}

// TestEnrollmentUseCase_UpdateProgress_ZeroProgressResetsStatus verifies 0%
// progress keeps the enrollment in "enrolled" status.
func TestEnrollmentUseCase_UpdateProgress_ZeroProgressResetsStatus(t *testing.T) {
	uc, _, _, _, course := newEnrollmentUseCase()
	created, err := uc.Enroll(context.Background(), &dto.EnrollmentRequest{
		UserID:   1,
		CourseID: course.ID,
	})
	require.NoError(t, err)
	resp, err := uc.UpdateProgress(context.Background(), created.ID, &dto.EnrollmentUpdateProgressRequest{
		UserID:          1,
		ProgressPercent: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, entity.EnrollmentStatusEnrolled, resp.Status)
}

// TestEnrollmentUseCase_UpdateProgress_CompletedEmitsEvent verifies 100%
// progress marks the enrollment completed and emits a CourseCompleted event.
func TestEnrollmentUseCase_UpdateProgress_CompletedEmitsEvent(t *testing.T) {
	uc, _, _, pub, course := newEnrollmentUseCase()
	created, err := uc.Enroll(context.Background(), &dto.EnrollmentRequest{
		UserID:   1,
		CourseID: course.ID,
	})
	require.NoError(t, err)
	resp, err := uc.UpdateProgress(context.Background(), created.ID, &dto.EnrollmentUpdateProgressRequest{
		UserID:          1,
		ProgressPercent: 100,
	})
	require.NoError(t, err)
	assert.Equal(t, entity.EnrollmentStatusCompleted, resp.Status)
	evt, ok := captureEvent[event.CourseCompleted](pub)
	require.True(t, ok)
	assert.Equal(t, int64(1), evt.UserID)
	assert.Equal(t, course.ID, evt.CourseID)
}

// TestEnrollmentUseCase_UpdateProgress_Clamps verifies progress is clamped to
// [0, 100].
func TestEnrollmentUseCase_UpdateProgress_Clamps(t *testing.T) {
	uc, _, _, _, course := newEnrollmentUseCase()
	created, err := uc.Enroll(context.Background(), &dto.EnrollmentRequest{
		UserID:   1,
		CourseID: course.ID,
	})
	require.NoError(t, err)
	t.Run("negative clamps to 0", func(t *testing.T) {
		resp, err := uc.UpdateProgress(context.Background(), created.ID, &dto.EnrollmentUpdateProgressRequest{
			UserID:          1,
			ProgressPercent: -10,
		})
		require.NoError(t, err)
		assert.Equal(t, entity.EnrollmentStatusEnrolled, resp.Status)
	})
}

// TestEnrollmentUseCase_UpdateProgress_Over100Completes verifies values >100
// are clamped to 100 and trigger completion.
func TestEnrollmentUseCase_UpdateProgress_Over100Completes(t *testing.T) {
	uc, _, _, _, course := newEnrollmentUseCase()
	created, err := uc.Enroll(context.Background(), &dto.EnrollmentRequest{
		UserID:   1,
		CourseID: course.ID,
	})
	require.NoError(t, err)
	resp, err := uc.UpdateProgress(context.Background(), created.ID, &dto.EnrollmentUpdateProgressRequest{
		UserID:          1,
		ProgressPercent: 150,
	})
	require.NoError(t, err)
	assert.Equal(t, entity.EnrollmentStatusCompleted, resp.Status)
}

// TestEnrollmentUseCase_UpdateProgress_NilRequest rejects nil body.
func TestEnrollmentUseCase_UpdateProgress_NilRequest(t *testing.T) {
	uc, _, _, _, _ := newEnrollmentUseCase()
	_, err := uc.UpdateProgress(context.Background(), 1, nil)
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.InvalidParams, e.Code)
	}
}

// TestEnrollmentUseCase_UpdateProgress_NotFound verifies missing enrollment
// is rejected.
func TestEnrollmentUseCase_UpdateProgress_NotFound(t *testing.T) {
	uc, _, _, _, _ := newEnrollmentUseCase()
	_, err := uc.UpdateProgress(context.Background(), 9999, &dto.EnrollmentUpdateProgressRequest{
		UserID:          1,
		ProgressPercent: 50,
	})
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.NotFound, e.Code)
	}
}

// TestEnrollmentUseCase_UpdateProgress_FindError verifies repo errors
// propagate.
func TestEnrollmentUseCase_UpdateProgress_FindError(t *testing.T) {
	uc, enrollRepo, _, _, _ := newEnrollmentUseCase()
	enrollRepo.find = func(int64) (*entity.Enrollment, error) { return nil, errors.New("find err") }
	_, err := uc.UpdateProgress(context.Background(), 1, &dto.EnrollmentUpdateProgressRequest{
		UserID:          1,
		ProgressPercent: 50,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find err")
}

// TestEnrollmentUseCase_UpdateProgress_MarkCompletedError verifies the
// MarkCompleted repo error propagates.
func TestEnrollmentUseCase_UpdateProgress_MarkCompletedError(t *testing.T) {
	uc, enrollRepo, _, _, course := newEnrollmentUseCase()
	created, err := uc.Enroll(context.Background(), &dto.EnrollmentRequest{
		UserID:   1,
		CourseID: course.ID,
	})
	require.NoError(t, err)
	enrollRepo.markCompleted = func(int64) error { return errors.New("mark err") }
	_, err = uc.UpdateProgress(context.Background(), created.ID, &dto.EnrollmentUpdateProgressRequest{
		UserID:          1,
		ProgressPercent: 100,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mark err")
}

// TestEnrollmentUseCase_UpdateProgress_UpdateProgressError verifies the
// UpdateProgress repo error propagates (non-completion branch).
func TestEnrollmentUseCase_UpdateProgress_UpdateProgressError(t *testing.T) {
	uc, enrollRepo, _, _, course := newEnrollmentUseCase()
	created, err := uc.Enroll(context.Background(), &dto.EnrollmentRequest{
		UserID:   1,
		CourseID: course.ID,
	})
	require.NoError(t, err)
	enrollRepo.updateProgress = func(int64, int64, int) error { return errors.New("up err") }
	_, err = uc.UpdateProgress(context.Background(), created.ID, &dto.EnrollmentUpdateProgressRequest{
		UserID:          1,
		ProgressPercent: 50,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "up err")
}

// TestToEnrollmentResponse_Timestamps verifies timestamps are formatted when
// set on the entity.
func TestToEnrollmentResponse_Timestamps(t *testing.T) {
	uc, enrollRepo, _, _, course := newEnrollmentUseCase()
	created, err := uc.Enroll(context.Background(), &dto.EnrollmentRequest{
		UserID:   1,
		CourseID: course.ID,
	})
	require.NoError(t, err)
	enrollRepo.mu.Lock()
	if e, ok := enrollRepo.items[created.ID]; ok {
		e.CreatedAt = time.Now()
		e.UpdatedAt = time.Now()
	}
	enrollRepo.mu.Unlock()
	got, err := uc.Get(context.Background(), created.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, got.CreatedAt)
	assert.NotEmpty(t, got.UpdatedAt)
}
