package usecase_test

import (
	"context"
	"encoding/json"
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

// newStudyPlanUseCase wires up a StudyPlanUseCase with in-memory mocks.
func newStudyPlanUseCase() (*usecase.StudyPlanUseCase, *mockStudyPlanRepo, *mockEnrollmentRepo) {
	planRepo := newMockStudyPlanRepo()
	enrollRepo := newMockEnrollmentRepo()
	uc := usecase.NewStudyPlanUseCase(planRepo, enrollRepo, nil)
	return uc, planRepo, enrollRepo
}

// TestStudyPlanUseCase_Create_HappyPath verifies a plan is created with an
// active status default, an empty courses JSON default, and 0% progress.
func TestStudyPlanUseCase_Create_HappyPath(t *testing.T) {
	uc, repo, _ := newStudyPlanUseCase()
	resp, err := uc.Create(context.Background(), &dto.StudyPlanRequest{
		UserID: 7,
		Title:  "2025 学习计划",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotZero(t, resp.ID)
	assert.Equal(t, int64(7), resp.UserID)
	assert.Equal(t, "2025 学习计划", resp.Title)
	assert.Equal(t, entity.StudyPlanStatusActive, resp.Status)
	assert.Equal(t, 0, resp.ProgressPercent)
	assert.Equal(t, json.RawMessage("[]"), resp.CoursesJSON)

	// Repo contains the row.
	got, err := repo.FindByID(context.Background(), resp.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "2025 学习计划", got.Title)
}

// TestStudyPlanUseCase_Create_KeepsExplicitStatusAndCourses verifies explicit
// Status and CoursesJSON are preserved.
func TestStudyPlanUseCase_Create_KeepsExplicitStatusAndCourses(t *testing.T) {
	uc, repo, _ := newStudyPlanUseCase()
	target := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	resp, err := uc.Create(context.Background(), &dto.StudyPlanRequest{
		UserID:      1,
		Title:       "p",
		TargetDate:  &target,
		CoursesJSON: json.RawMessage(`[1,2]`),
		Status:      entity.StudyPlanStatusArchived,
	})
	require.NoError(t, err)
	got, _ := repo.FindByID(context.Background(), resp.ID)
	require.NotNil(t, got)
	assert.Equal(t, entity.StudyPlanStatusArchived, got.Status)
	assert.Equal(t, json.RawMessage(`[1,2]`), got.CoursesJSON)
	require.NotNil(t, got.TargetDate)
	assert.Equal(t, target, *got.TargetDate)
}

// TestStudyPlanUseCase_Create_ValidationErrors covers input validations.
func TestStudyPlanUseCase_Create_ValidationErrors(t *testing.T) {
	uc, _, _ := newStudyPlanUseCase()
	cases := []struct {
		name string
		in   *dto.StudyPlanRequest
	}{
		{"nil request", nil},
		{"zero user_id", &dto.StudyPlanRequest{Title: "t"}},
		{"empty title", &dto.StudyPlanRequest{UserID: 1}},
		{"negative user_id", &dto.StudyPlanRequest{UserID: -1, Title: "t"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := uc.Create(context.Background(), c.in)
			require.Error(t, err)
			var e *errno.Error
			if errors.As(err, &e) {
				assert.Equal(t, errno.InvalidParams, e.Code)
			}
		})
	}
}

// TestStudyPlanUseCase_Create_RepoCreateError verifies Create errors
// propagate.
func TestStudyPlanUseCase_Create_RepoCreateError(t *testing.T) {
	uc, repo, _ := newStudyPlanUseCase()
	repo.create = func(*entity.StudyPlan) error { return errors.New("create err") }
	_, err := uc.Create(context.Background(), &dto.StudyPlanRequest{UserID: 1, Title: "t"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create err")
}

// TestStudyPlanUseCase_Create_RefreshesProgressFromEnrollments verifies that
// when the plan lists courses the user is enrolled in, the progress percent
// is the average of those enrollments' progress.
func TestStudyPlanUseCase_Create_RefreshesProgressFromEnrollments(t *testing.T) {
	uc, planRepo, enrollRepo := newStudyPlanUseCase()
	// Two enrollments: 80% and 60% → average 70%.
	e1 := &entity.Enrollment{UserID: 1, CourseID: 10, ProgressPercent: 80, Status: entity.EnrollmentStatusInProgress}
	e1.ID = 100
	require.NoError(t, enrollRepo.Create(context.Background(), e1))
	e2 := &entity.Enrollment{UserID: 1, CourseID: 20, ProgressPercent: 60, Status: entity.EnrollmentStatusInProgress}
	e2.ID = 101
	require.NoError(t, enrollRepo.Create(context.Background(), e2))

	resp, err := uc.Create(context.Background(), &dto.StudyPlanRequest{
		UserID:      1,
		Title:       "p",
		CoursesJSON: json.RawMessage(`[10,20]`),
	})
	require.NoError(t, err)
	assert.Equal(t, 70, resp.ProgressPercent)

	// Plan in repo should also reflect the computed progress.
	got, _ := planRepo.FindByID(context.Background(), resp.ID)
	require.NotNil(t, got)
	assert.Equal(t, 70, got.ProgressPercent)
}

// TestStudyPlanUseCase_Create_RefreshProgressCompletesPlanAt100 verifies that
// when the average enrollment progress is 100, the plan status is promoted
// from active to completed.
func TestStudyPlanUseCase_Create_RefreshProgressCompletesPlanAt100(t *testing.T) {
	uc, _, enrollRepo := newStudyPlanUseCase()
	e := &entity.Enrollment{UserID: 1, CourseID: 5, ProgressPercent: 100, Status: entity.EnrollmentStatusCompleted}
	e.ID = 1
	require.NoError(t, enrollRepo.Create(context.Background(), e))

	resp, err := uc.Create(context.Background(), &dto.StudyPlanRequest{
		UserID:      1,
		Title:       "p",
		CoursesJSON: json.RawMessage(`[5]`),
	})
	require.NoError(t, err)
	assert.Equal(t, 100, resp.ProgressPercent)
	assert.Equal(t, entity.StudyPlanStatusCompleted, resp.Status, "plan should auto-complete at 100%")
}

// TestStudyPlanUseCase_Create_RefreshProgressIgnoresMissingEnrollments
// verifies that courses without an enrollment are simply skipped (no error,
// no contribution to the average).
func TestStudyPlanUseCase_Create_RefreshProgressIgnoresMissingEnrollments(t *testing.T) {
	uc, _, enrollRepo := newStudyPlanUseCase()
	// Only one enrollment for course 1; course 2 has no enrollment.
	e := &entity.Enrollment{UserID: 1, CourseID: 1, ProgressPercent: 50, Status: entity.EnrollmentStatusInProgress}
	e.ID = 1
	require.NoError(t, enrollRepo.Create(context.Background(), e))

	resp, err := uc.Create(context.Background(), &dto.StudyPlanRequest{
		UserID:      1,
		Title:       "p",
		CoursesJSON: json.RawMessage(`[1,2]`),
	})
	require.NoError(t, err)
	assert.Equal(t, 50, resp.ProgressPercent, "average should only include enrolled courses")
}

// TestStudyPlanUseCase_Create_RefreshProgressWithMalformedCoursesJSON
// verifies that a malformed CoursesJSON is silently ignored (refresh is
// best-effort).
func TestStudyPlanUseCase_Create_RefreshProgressWithMalformedCoursesJSON(t *testing.T) {
	uc, _, _ := newStudyPlanUseCase()
	resp, err := uc.Create(context.Background(), &dto.StudyPlanRequest{
		UserID:      1,
		Title:       "p",
		CoursesJSON: json.RawMessage(`not-json`),
	})
	require.NoError(t, err)
	assert.Equal(t, 0, resp.ProgressPercent)
}

// TestStudyPlanUseCase_Update covers happy / not-found / error / nil-body
// paths and verifies progress is refreshed.
func TestStudyPlanUseCase_Update(t *testing.T) {
	t.Run("happy overwrites fields and refreshes progress", func(t *testing.T) {
		uc, _, enrollRepo := newStudyPlanUseCase()
		e := &entity.Enrollment{UserID: 1, CourseID: 1, ProgressPercent: 100, Status: entity.EnrollmentStatusCompleted}
		e.ID = 1
		require.NoError(t, enrollRepo.Create(context.Background(), e))

		created, err := uc.Create(context.Background(), &dto.StudyPlanRequest{UserID: 1, Title: "p"})
		require.NoError(t, err)

		target := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		resp, err := uc.Update(context.Background(), created.ID, &dto.StudyPlanRequest{
			Title:       "p2",
			TargetDate:  &target,
			CoursesJSON: json.RawMessage(`[1]`),
			Status:      entity.StudyPlanStatusActive,
		})
		require.NoError(t, err)
		assert.Equal(t, "p2", resp.Title)
		assert.Equal(t, 100, resp.ProgressPercent)
		assert.Equal(t, entity.StudyPlanStatusCompleted, resp.Status, "should auto-complete at 100%")
		require.NotNil(t, resp.TargetDate)
		assert.Equal(t, target, *resp.TargetDate)
	})
	t.Run("empty CoursesJSON keeps existing", func(t *testing.T) {
		uc, _, _ := newStudyPlanUseCase()
		created, err := uc.Create(context.Background(), &dto.StudyPlanRequest{
			UserID:      1,
			Title:       "p",
			CoursesJSON: json.RawMessage(`[1,2]`),
		})
		require.NoError(t, err)
		resp, err := uc.Update(context.Background(), created.ID, &dto.StudyPlanRequest{
			Title: "p2",
			// no CoursesJSON — keeps existing
		})
		require.NoError(t, err)
		assert.Equal(t, json.RawMessage(`[1,2]`), resp.CoursesJSON)
	})
	t.Run("empty status keeps existing", func(t *testing.T) {
		uc, _, _ := newStudyPlanUseCase()
		created, err := uc.Create(context.Background(), &dto.StudyPlanRequest{
			UserID: 1, Title: "p", Status: entity.StudyPlanStatusArchived,
		})
		require.NoError(t, err)
		resp, err := uc.Update(context.Background(), created.ID, &dto.StudyPlanRequest{
			Title:  "p2",
			Status: "",
		})
		require.NoError(t, err)
		assert.Equal(t, entity.StudyPlanStatusArchived, resp.Status)
	})
	t.Run("nil body rejected", func(t *testing.T) {
		uc, _, _ := newStudyPlanUseCase()
		_, err := uc.Update(context.Background(), 1, nil)
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.InvalidParams, e.Code)
		}
	})
	t.Run("not found", func(t *testing.T) {
		uc, _, _ := newStudyPlanUseCase()
		_, err := uc.Update(context.Background(), 9999, &dto.StudyPlanRequest{Title: "p"})
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.NotFound, e.Code)
		}
	})
	t.Run("find error", func(t *testing.T) {
		uc, repo, _ := newStudyPlanUseCase()
		repo.find = func(int64) (*entity.StudyPlan, error) { return nil, errors.New("find err") }
		_, err := uc.Update(context.Background(), 1, &dto.StudyPlanRequest{Title: "p"})
		require.Error(t, err)
	})
	t.Run("update error", func(t *testing.T) {
		uc, repo, _ := newStudyPlanUseCase()
		created, err := uc.Create(context.Background(), &dto.StudyPlanRequest{UserID: 1, Title: "p"})
		require.NoError(t, err)
		repo.update = func(*entity.StudyPlan) error { return errors.New("update err") }
		_, err = uc.Update(context.Background(), created.ID, &dto.StudyPlanRequest{Title: "p2"})
		require.Error(t, err)
	})
}

// TestStudyPlanUseCase_Delete covers happy / error paths.
func TestStudyPlanUseCase_Delete(t *testing.T) {
	uc, repo, _ := newStudyPlanUseCase()
	created, err := uc.Create(context.Background(), &dto.StudyPlanRequest{UserID: 1, Title: "p"})
	require.NoError(t, err)
	require.NoError(t, uc.Delete(context.Background(), created.ID))

	t.Run("repo error", func(t *testing.T) {
		repo.delete = func(int64) error { return errors.New("delete err") }
		err := uc.Delete(context.Background(), 1)
		require.Error(t, err)
	})
}

// TestStudyPlanUseCase_Get covers found / not-found / error paths.
func TestStudyPlanUseCase_Get(t *testing.T) {
	uc, repo, _ := newStudyPlanUseCase()
	created, err := uc.Create(context.Background(), &dto.StudyPlanRequest{UserID: 1, Title: "p"})
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
		repo.find = func(int64) (*entity.StudyPlan, error) { return nil, errors.New("find err") }
		_, err := uc.Get(context.Background(), 1)
		require.Error(t, err)
	})
}

// TestStudyPlanUseCase_ListByUser covers happy / error paths and pagination.
func TestStudyPlanUseCase_ListByUser(t *testing.T) {
	uc, repo, _ := newStudyPlanUseCase()
	for i := 0; i < 3; i++ {
		_, err := uc.Create(context.Background(), &dto.StudyPlanRequest{UserID: 7, Title: "p"})
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
		repo.listByUser = func(int64, pagination.Params) ([]entity.StudyPlan, int, error) {
			return nil, 0, errors.New("list err")
		}
		_, err := uc.ListByUser(context.Background(), 7, pagination.Params{Page: 1, PageSize: 2})
		require.Error(t, err)
	})
}

// TestStudyPlanUseCase_ListActive covers happy / error paths and verifies
// only active plans are returned.
func TestStudyPlanUseCase_ListActive(t *testing.T) {
	uc, repo, _ := newStudyPlanUseCase()
	// One active, one archived (set directly on the repo).
	active := &entity.StudyPlan{UserID: 7, Title: "active", Status: entity.StudyPlanStatusActive}
	active.ID = 1
	require.NoError(t, repo.Create(context.Background(), active))
	archived := &entity.StudyPlan{UserID: 7, Title: "archived", Status: entity.StudyPlanStatusArchived}
	archived.ID = 2
	require.NoError(t, repo.Create(context.Background(), archived))

	resp, err := uc.ListActive(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, resp, 1)
	assert.Equal(t, "active", resp[0].Title)

	t.Run("repo error", func(t *testing.T) {
		repo.findActive = func(int64) ([]entity.StudyPlan, error) {
			return nil, errors.New("active err")
		}
		_, err := uc.ListActive(context.Background(), 7)
		require.Error(t, err)
	})
}

// TestToStudyPlanResponse_Timestamps verifies timestamps are formatted when
// set on the entity.
func TestToStudyPlanResponse_Timestamps(t *testing.T) {
	uc, repo, _ := newStudyPlanUseCase()
	created, err := uc.Create(context.Background(), &dto.StudyPlanRequest{UserID: 1, Title: "p"})
	require.NoError(t, err)
	repo.mu.Lock()
	if s, ok := repo.items[created.ID]; ok {
		s.CreatedAt = time.Now()
		s.UpdatedAt = time.Now()
	}
	repo.mu.Unlock()
	got, err := uc.Get(context.Background(), created.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, got.CreatedAt)
	assert.NotEmpty(t, got.UpdatedAt)
}
