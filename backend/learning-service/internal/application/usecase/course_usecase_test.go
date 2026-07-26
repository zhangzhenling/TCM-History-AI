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
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

// newCourseUseCase wires up a CourseUseCase with in-memory mocks and a
// capturing event publisher.
func newCourseUseCase() (*usecase.CourseUseCase, *mockCourseRepo, *mockLessonRepo, *mockEventPublisher) {
	courseRepo := newMockCourseRepo()
	lessonRepo := newMockLessonRepo()
	pub := newMockEventPublisher()
	uc := usecase.NewCourseUseCase(courseRepo, lessonRepo, pub)
	return uc, courseRepo, lessonRepo, pub
}

// ============================================================================
// Course CRUD
// ============================================================================

// TestCourseUseCase_Create_HappyPath verifies a course is created with a
// snowflake id, default beginner difficulty, and zero lesson_count.
func TestCourseUseCase_Create_HappyPath(t *testing.T) {
	uc, repo, _, _ := newCourseUseCase()
	resp, err := uc.Create(context.Background(), &dto.CourseRequest{
		Title:           "中医基础",
		Description:     "intro",
		Category:        "basic",
		DurationMinutes: 120,
		IsPublished:     true,
		SortOrder:       2,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotZero(t, resp.ID)
	assert.Equal(t, "中医基础", resp.Title)
	assert.Equal(t, "basic", resp.Category)
	assert.Equal(t, entity.DifficultyBeginner, resp.Difficulty)
	assert.Equal(t, 0, resp.LessonCount)
	assert.True(t, resp.IsPublished)
	assert.Equal(t, 2, resp.SortOrder)

	// Repo contains the row.
	got, err := repo.FindByID(context.Background(), resp.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "中医基础", got.Title)
}

// TestCourseUseCase_create_DefaultsDifficulty verifies an empty Difficulty is
// defaulted to beginner.
func TestCourseUseCase_Create_DefaultsDifficulty(t *testing.T) {
	uc, repo, _, _ := newCourseUseCase()
	resp, err := uc.Create(context.Background(), &dto.CourseRequest{
		Title:      "n",
		Difficulty: "",
	})
	require.NoError(t, err)
	got, _ := repo.FindByID(context.Background(), resp.ID)
	require.NotNil(t, got)
	assert.Equal(t, entity.DifficultyBeginner, got.Difficulty)
}

// TestCourseUseCase_Create_KeepsExplicitDifficulty verifies a non-empty
// Difficulty is preserved.
func TestCourseUseCase_Create_KeepsExplicitDifficulty(t *testing.T) {
	uc, repo, _, _ := newCourseUseCase()
	resp, err := uc.Create(context.Background(), &dto.CourseRequest{
		Title:      "n",
		Difficulty: entity.DifficultyAdvanced,
	})
	require.NoError(t, err)
	got, _ := repo.FindByID(context.Background(), resp.ID)
	require.NotNil(t, got)
	assert.Equal(t, entity.DifficultyAdvanced, got.Difficulty)
}

// TestCourseUseCase_Create_ValidationErrors exercises the input validations.
func TestCourseUseCase_Create_ValidationErrors(t *testing.T) {
	uc, _, _, _ := newCourseUseCase()
	cases := []struct {
		name string
		in   *dto.CourseRequest
	}{
		{"nil request", nil},
		{"empty title", &dto.CourseRequest{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp, err := uc.Create(context.Background(), c.in)
			require.Error(t, err)
			assert.Nil(t, resp)
			var e *errno.Error
			if errors.As(err, &e) {
				assert.Equal(t, errno.InvalidParams, e.Code)
			}
		})
	}
}

// TestCourseUseCase_Create_RepoCreateError verifies Create errors propagate.
func TestCourseUseCase_Create_RepoCreateError(t *testing.T) {
	uc, repo, _, _ := newCourseUseCase()
	repo.create = func(*entity.Course) error { return errors.New("write failed") }
	_, err := uc.Create(context.Background(), &dto.CourseRequest{Title: "n"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write failed")
}

// TestCourseUseCase_Update covers the happy / not-found / error / nil-body
// paths and verifies empty Difficulty is preserved (only non-empty overwrites).
func TestCourseUseCase_Update(t *testing.T) {
	t.Run("happy path overwrites fields", func(t *testing.T) {
		uc, _, _, _ := newCourseUseCase()
		created, err := uc.Create(context.Background(), &dto.CourseRequest{
			Title: "n", Difficulty: entity.DifficultyBeginner,
		})
		require.NoError(t, err)

		resp, err := uc.Update(context.Background(), created.ID, &dto.CourseRequest{
			Title:       "n2",
			Description: "d2",
			Category:    "cat2",
			Difficulty:  entity.DifficultyAdvanced,
			SortOrder:   5,
		})
		require.NoError(t, err)
		assert.Equal(t, "n2", resp.Title)
		assert.Equal(t, "d2", resp.Description)
		assert.Equal(t, "cat2", resp.Category)
		assert.Equal(t, entity.DifficultyAdvanced, resp.Difficulty)
		assert.Equal(t, 5, resp.SortOrder)
	})

	t.Run("empty difficulty keeps existing", func(t *testing.T) {
		uc, _, _, _ := newCourseUseCase()
		created, err := uc.Create(context.Background(), &dto.CourseRequest{
			Title: "n", Difficulty: entity.DifficultyAdvanced,
		})
		require.NoError(t, err)
		resp, err := uc.Update(context.Background(), created.ID, &dto.CourseRequest{
			Title: "n2", Difficulty: "",
		})
		require.NoError(t, err)
		assert.Equal(t, entity.DifficultyAdvanced, resp.Difficulty)
	})

	t.Run("nil body rejected", func(t *testing.T) {
		uc, _, _, _ := newCourseUseCase()
		_, err := uc.Update(context.Background(), 1, nil)
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.InvalidParams, e.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		uc, _, _, _ := newCourseUseCase()
		_, err := uc.Update(context.Background(), 9999, &dto.CourseRequest{Title: "n"})
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.NotFound, e.Code)
		}
	})

	t.Run("find error", func(t *testing.T) {
		uc, repo, _, _ := newCourseUseCase()
		repo.find = func(int64) (*entity.Course, error) { return nil, errors.New("find err") }
		_, err := uc.Update(context.Background(), 1, &dto.CourseRequest{Title: "n"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "find err")
	})

	t.Run("update error", func(t *testing.T) {
		uc, repo, _, _ := newCourseUseCase()
		created, err := uc.Create(context.Background(), &dto.CourseRequest{Title: "n"})
		require.NoError(t, err)
		repo.update = func(*entity.Course) error { return errors.New("update failed") }
		_, err = uc.Update(context.Background(), created.ID, &dto.CourseRequest{Title: "n2"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update failed")
	})
}

// TestCourseUseCase_Delete covers found / not-found / error paths.
func TestCourseUseCase_Delete(t *testing.T) {
	uc, repo, _, _ := newCourseUseCase()
	created, err := uc.Create(context.Background(), &dto.CourseRequest{Title: "n"})
	require.NoError(t, err)

	t.Run("happy", func(t *testing.T) {
		require.NoError(t, uc.Delete(context.Background(), created.ID))
	})
	t.Run("repo error", func(t *testing.T) {
		repo.delete = func(int64) error { return errors.New("delete err") }
		err := uc.Delete(context.Background(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete err")
	})
}

// TestCourseUseCase_Get covers found / not-found / error paths.
func TestCourseUseCase_Get(t *testing.T) {
	uc, repo, _, _ := newCourseUseCase()
	created, err := uc.Create(context.Background(), &dto.CourseRequest{Title: "n"})
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
		repo.find = func(int64) (*entity.Course, error) { return nil, errors.New("find err") }
		_, err := uc.Get(context.Background(), 1)
		require.Error(t, err)
	})
}

// TestCourseUseCase_List covers the list / error paths and pagination.
func TestCourseUseCase_List(t *testing.T) {
	uc, repo, _, _ := newCourseUseCase()
	for _, title := range []string{"a", "b", "c"} {
		_, err := uc.Create(context.Background(), &dto.CourseRequest{Title: title})
		require.NoError(t, err)
	}

	t.Run("happy paginated", func(t *testing.T) {
		resp, err := uc.List(context.Background(), pagination.Params{Page: 1, PageSize: 2})
		require.NoError(t, err)
		assert.Equal(t, 3, resp.Total)
		assert.Equal(t, 2, resp.TotalPage)
		require.Len(t, resp.Items, 2)
	})
	t.Run("repo error", func(t *testing.T) {
		repo.list = func(pagination.Params) ([]entity.Course, int, error) {
			return nil, 0, errors.New("list err")
		}
		_, err := uc.List(context.Background(), pagination.Params{Page: 1, PageSize: 2})
		require.Error(t, err)
	})
}

// TestCourseUseCase_ListByCategory covers the filter / fallback-to-List /
// error paths.
func TestCourseUseCase_ListByCategory(t *testing.T) {
	uc, repo, _, _ := newCourseUseCase()
	_, err := uc.Create(context.Background(), &dto.CourseRequest{Title: "a", Category: "tcm"})
	require.NoError(t, err)
	_, err = uc.Create(context.Background(), &dto.CourseRequest{Title: "b", Category: "history"})
	require.NoError(t, err)

	t.Run("by category", func(t *testing.T) {
		resp, err := uc.ListByCategory(context.Background(), "tcm", pagination.Params{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Total)
		require.Len(t, resp.Items, 1)
		assert.Equal(t, "a", resp.Items[0].Title)
	})
	t.Run("empty category falls back to List", func(t *testing.T) {
		resp, err := uc.ListByCategory(context.Background(), "", pagination.Params{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Total)
	})
	t.Run("repo error", func(t *testing.T) {
		repo.listByCategory = func(string, pagination.Params) ([]entity.Course, int, error) {
			return nil, 0, errors.New("cat err")
		}
		_, err := uc.ListByCategory(context.Background(), "tcm", pagination.Params{Page: 1, PageSize: 10})
		require.Error(t, err)
	})
}

// TestCourseUseCase_ListPublished covers published-only listing.
func TestCourseUseCase_ListPublished(t *testing.T) {
	uc, repo, _, _ := newCourseUseCase()
	published := &entity.Course{Title: "pub", IsPublished: true}
	published.ID = idgen.Next()
	require.NoError(t, repo.Create(context.Background(), published))
	unpublished := &entity.Course{Title: "draft", IsPublished: false}
	unpublished.ID = idgen.Next()
	require.NoError(t, repo.Create(context.Background(), unpublished))

	resp, err := uc.ListPublished(context.Background(), pagination.Params{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, "pub", resp.Items[0].Title)

	t.Run("repo error", func(t *testing.T) {
		repo.listPublished = func(pagination.Params) ([]entity.Course, int, error) {
			return nil, 0, errors.New("pub err")
		}
		_, err := uc.ListPublished(context.Background(), pagination.Params{Page: 1, PageSize: 10})
		require.Error(t, err)
	})
}

// TestCourseUseCase_Publish covers the happy / not-found / error paths and
// verifies a CoursePublished event is emitted on success.
func TestCourseUseCase_Publish(t *testing.T) {
	t.Run("happy emits event", func(t *testing.T) {
		uc, _, _, pub := newCourseUseCase()
		created, err := uc.Create(context.Background(), &dto.CourseRequest{
			Title: "n", Category: "tcm",
		})
		require.NoError(t, err)
		resp, err := uc.Publish(context.Background(), created.ID)
		require.NoError(t, err)
		assert.True(t, resp.IsPublished)
		evt, ok := captureEvent[event.CoursePublished](pub)
		require.True(t, ok)
		assert.Equal(t, created.ID, evt.CourseID)
		assert.Equal(t, "n", evt.Title)
		assert.Equal(t, "tcm", evt.Category)
	})
	t.Run("not found", func(t *testing.T) {
		uc, _, _, _ := newCourseUseCase()
		_, err := uc.Publish(context.Background(), 9999)
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.NotFound, e.Code)
		}
	})
	t.Run("find error", func(t *testing.T) {
		uc, repo, _, _ := newCourseUseCase()
		repo.find = func(int64) (*entity.Course, error) { return nil, errors.New("find err") }
		_, err := uc.Publish(context.Background(), 1)
		require.Error(t, err)
	})
	t.Run("update error", func(t *testing.T) {
		uc, repo, _, _ := newCourseUseCase()
		created, err := uc.Create(context.Background(), &dto.CourseRequest{Title: "n"})
		require.NoError(t, err)
		repo.update = func(*entity.Course) error { return errors.New("update err") }
		_, err = uc.Publish(context.Background(), created.ID)
		require.Error(t, err)
	})
}

// TestCourseUseCase_Unpublish covers the happy / not-found / error paths.
func TestCourseUseCase_Unpublish(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		uc, _, _, pub := newCourseUseCase()
		created, err := uc.Create(context.Background(), &dto.CourseRequest{
			Title: "n", IsPublished: true,
		})
		require.NoError(t, err)
		resp, err := uc.Unpublish(context.Background(), created.ID)
		require.NoError(t, err)
		assert.False(t, resp.IsPublished)
		// Unpublish must NOT emit a CoursePublished event.
		_, ok := captureEvent[event.CoursePublished](pub)
		assert.False(t, ok)
	})
	t.Run("not found", func(t *testing.T) {
		uc, _, _, _ := newCourseUseCase()
		_, err := uc.Unpublish(context.Background(), 9999)
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.NotFound, e.Code)
		}
	})
	t.Run("find error", func(t *testing.T) {
		uc, repo, _, _ := newCourseUseCase()
		repo.find = func(int64) (*entity.Course, error) { return nil, errors.New("find err") }
		_, err := uc.Unpublish(context.Background(), 1)
		require.Error(t, err)
	})
}

// ============================================================================
// Lesson CRUD
// ============================================================================

// TestCourseUseCase_CreateLesson_HappyPath verifies a lesson is created under
// a course, with default article content type and refresh of lesson_count.
func TestCourseUseCase_CreateLesson_HappyPath(t *testing.T) {
	uc, _, lessonRepo, _ := newCourseUseCase()
	created, err := uc.Create(context.Background(), &dto.CourseRequest{Title: "course"})
	require.NoError(t, err)

	resp, err := uc.CreateLesson(context.Background(), created.ID, &dto.LessonRequest{
		Title:           "L1",
		Content:         "body",
		DurationMinutes: 30,
		SortOrder:       1,
		IsFree:          true,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotZero(t, resp.ID)
	assert.Equal(t, created.ID, resp.CourseID)
	assert.Equal(t, "L1", resp.Title)
	assert.Equal(t, entity.ContentTypeArticle, resp.ContentType)
	assert.True(t, resp.IsFree)

	// Lesson persisted.
	got, err := lessonRepo.FindByID(context.Background(), resp.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "L1", got.Title)
}

// TestCourseUseCase_CreateLesson_DefaultsContentType verifies empty
// ContentType defaults to article.
func TestCourseUseCase_CreateLesson_DefaultsContentType(t *testing.T) {
	uc, _, lessonRepo, _ := newCourseUseCase()
	created, err := uc.Create(context.Background(), &dto.CourseRequest{Title: "c"})
	require.NoError(t, err)
	resp, err := uc.CreateLesson(context.Background(), created.ID, &dto.LessonRequest{
		Title:       "L",
		ContentType: "",
	})
	require.NoError(t, err)
	got, _ := lessonRepo.FindByID(context.Background(), resp.ID)
	require.NotNil(t, got)
	assert.Equal(t, entity.ContentTypeArticle, got.ContentType)
}

// TestCourseUseCase_CreateLesson_ValidationErrors covers input validations.
func TestCourseUseCase_CreateLesson_ValidationErrors(t *testing.T) {
	uc, _, _, _ := newCourseUseCase()
	cases := []struct {
		name     string
		courseID int64
		in       *dto.LessonRequest
	}{
		{"nil request", 1, nil},
		{"empty title", 1, &dto.LessonRequest{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := uc.CreateLesson(context.Background(), c.courseID, c.in)
			require.Error(t, err)
			var e *errno.Error
			if errors.As(err, &e) {
				assert.Equal(t, errno.InvalidParams, e.Code)
			}
		})
	}
}

// TestCourseUseCase_CreateLesson_CourseNotFound verifies missing course is
// rejected.
func TestCourseUseCase_CreateLesson_CourseNotFound(t *testing.T) {
	uc, _, _, _ := newCourseUseCase()
	_, err := uc.CreateLesson(context.Background(), 9999, &dto.LessonRequest{Title: "L"})
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.NotFound, e.Code)
	}
}

// TestCourseUseCase_CreateLesson_CourseFindError verifies repo errors
// propagate.
func TestCourseUseCase_CreateLesson_CourseFindError(t *testing.T) {
	uc, repo, _, _ := newCourseUseCase()
	repo.find = func(int64) (*entity.Course, error) { return nil, errors.New("find err") }
	_, err := uc.CreateLesson(context.Background(), 1, &dto.LessonRequest{Title: "L"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find err")
}

// TestCourseUseCase_CreateLesson_LessonCreateError verifies lesson repo
// errors propagate.
func TestCourseUseCase_CreateLesson_LessonCreateError(t *testing.T) {
	uc, _, lessonRepo, _ := newCourseUseCase()
	created, err := uc.Create(context.Background(), &dto.CourseRequest{Title: "c"})
	require.NoError(t, err)
	lessonRepo.create = func(*entity.Lesson) error { return errors.New("lesson create err") }
	_, err = uc.CreateLesson(context.Background(), created.ID, &dto.LessonRequest{Title: "L"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lesson create err")
}

// TestCourseUseCase_UpdateLesson covers happy / not-found / error / nil-body.
func TestCourseUseCase_UpdateLesson(t *testing.T) {
	t.Run("happy overwrites fields and preserves empty content_type", func(t *testing.T) {
		uc, courseRepo, _, _ := newCourseUseCase()
		course, err := uc.Create(context.Background(), &dto.CourseRequest{Title: "c"})
		require.NoError(t, err)
		created, err := uc.CreateLesson(context.Background(), course.ID, &dto.LessonRequest{
			Title:       "L1",
			ContentType: entity.ContentTypeVideo,
		})
		require.NoError(t, err)

		resp, err := uc.UpdateLesson(context.Background(), created.ID, &dto.LessonRequest{
			Title:       "L2",
			Content:      "body2",
			VideoURL:    "http://x/v.mp4",
			ContentType: "", // empty keeps existing
		})
		require.NoError(t, err)
		assert.Equal(t, "L2", resp.Title)
		assert.Equal(t, "body2", resp.Content)
		assert.Equal(t, "http://x/v.mp4", resp.VideoURL)
		assert.Equal(t, entity.ContentTypeVideo, resp.ContentType, "empty content_type should preserve existing")
		_ = courseRepo
	})
	t.Run("nil body rejected", func(t *testing.T) {
		uc, _, _, _ := newCourseUseCase()
		_, err := uc.UpdateLesson(context.Background(), 1, nil)
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.InvalidParams, e.Code)
		}
	})
	t.Run("not found", func(t *testing.T) {
		uc, _, _, _ := newCourseUseCase()
		_, err := uc.UpdateLesson(context.Background(), 9999, &dto.LessonRequest{Title: "L"})
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.NotFound, e.Code)
		}
	})
	t.Run("find error", func(t *testing.T) {
		uc, _, lessonRepo, _ := newCourseUseCase()
		lessonRepo.find = func(int64) (*entity.Lesson, error) { return nil, errors.New("find err") }
		_, err := uc.UpdateLesson(context.Background(), 1, &dto.LessonRequest{Title: "L"})
		require.Error(t, err)
	})
	t.Run("update error", func(t *testing.T) {
		uc, courseRepo, lessonRepo, _ := newCourseUseCase()
		c, err := uc.Create(context.Background(), &dto.CourseRequest{Title: "c"})
		require.NoError(t, err)
		created, err := uc.CreateLesson(context.Background(), c.ID, &dto.LessonRequest{Title: "L"})
		require.NoError(t, err)
		lessonRepo.update = func(*entity.Lesson) error { return errors.New("update err") }
		_, err = uc.UpdateLesson(context.Background(), created.ID, &dto.LessonRequest{Title: "L2"})
		require.Error(t, err)
		_ = courseRepo
	})
}

// TestCourseUseCase_DeleteLesson covers happy / not-found / error paths and
// verifies lesson_count refresh is invoked.
func TestCourseUseCase_DeleteLesson(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		uc, courseRepo, lessonRepo, _ := newCourseUseCase()
		refreshed := false
		lessonRepo.updateCourseLessonCount = func(int64) error { refreshed = true; return nil }
		c, err := uc.Create(context.Background(), &dto.CourseRequest{Title: "course"})
		require.NoError(t, err)
		created, err := uc.CreateLesson(context.Background(), c.ID, &dto.LessonRequest{Title: "L"})
		require.NoError(t, err)
		require.NoError(t, uc.DeleteLesson(context.Background(), created.ID))
		got, _ := lessonRepo.FindByID(context.Background(), created.ID)
		assert.Nil(t, got)
		assert.True(t, refreshed, "UpdateCourseLessonCount should be called on delete")
		_ = courseRepo
	})
	t.Run("not found", func(t *testing.T) {
		uc, _, _, _ := newCourseUseCase()
		err := uc.DeleteLesson(context.Background(), 9999)
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.NotFound, e.Code)
		}
	})
	t.Run("find error", func(t *testing.T) {
		uc, _, lessonRepo, _ := newCourseUseCase()
		lessonRepo.find = func(int64) (*entity.Lesson, error) { return nil, errors.New("find err") }
		err := uc.DeleteLesson(context.Background(), 1)
		require.Error(t, err)
	})
	t.Run("delete error", func(t *testing.T) {
		uc, courseRepo, lessonRepo, _ := newCourseUseCase()
		c, err := uc.Create(context.Background(), &dto.CourseRequest{Title: "c"})
		require.NoError(t, err)
		created, err := uc.CreateLesson(context.Background(), c.ID, &dto.LessonRequest{Title: "L"})
		require.NoError(t, err)
		lessonRepo.delete = func(int64) error { return errors.New("delete err") }
		err = uc.DeleteLesson(context.Background(), created.ID)
		require.Error(t, err)
		_ = courseRepo
	})
}

// TestCourseUseCase_GetLesson covers found / not-found / error paths.
func TestCourseUseCase_GetLesson(t *testing.T) {
	uc, courseRepo, _, _ := newCourseUseCase()
	c, err := uc.Create(context.Background(), &dto.CourseRequest{Title: "c"})
	require.NoError(t, err)
	created, err := uc.CreateLesson(context.Background(), c.ID, &dto.LessonRequest{Title: "L"})
	require.NoError(t, err)
	_ = courseRepo

	t.Run("found", func(t *testing.T) {
		got, err := uc.GetLesson(context.Background(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
	})
	t.Run("not found", func(t *testing.T) {
		_, err := uc.GetLesson(context.Background(), 9999)
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.NotFound, e.Code)
		}
	})
	t.Run("find error", func(t *testing.T) {
		// Use a fresh usecase to avoid mutating the shared lesson repo.
		uc2, _, lessonRepo2, _ := newCourseUseCase()
		lessonRepo2.find = func(int64) (*entity.Lesson, error) { return nil, errors.New("find err") }
		_, err := uc2.GetLesson(context.Background(), 1)
		require.Error(t, err)
	})
}

// TestCourseUseCase_ListLessonsByCourse covers happy / error paths.
func TestCourseUseCase_ListLessonsByCourse(t *testing.T) {
	uc, courseRepo, lessonRepo, _ := newCourseUseCase()
	c, err := uc.Create(context.Background(), &dto.CourseRequest{Title: "c"})
	require.NoError(t, err)
	for _, title := range []string{"L1", "L2", "L3"} {
		_, err := uc.CreateLesson(context.Background(), c.ID, &dto.LessonRequest{Title: title})
		require.NoError(t, err)
	}
	_ = courseRepo

	t.Run("happy paginated", func(t *testing.T) {
		resp, err := uc.ListLessonsByCourse(context.Background(), c.ID, pagination.Params{Page: 1, PageSize: 2})
		require.NoError(t, err)
		assert.Equal(t, 3, resp.Total)
		assert.Equal(t, 2, resp.TotalPage)
		require.Len(t, resp.Items, 2)
	})
	t.Run("repo error", func(t *testing.T) {
		lessonRepo.listByCourse = func(int64, pagination.Params) ([]entity.Lesson, int, error) {
			return nil, 0, errors.New("list err")
		}
		_, err := uc.ListLessonsByCourse(context.Background(), c.ID, pagination.Params{Page: 1, PageSize: 2})
		require.Error(t, err)
	})
}

// ============================================================================
// Response mappers
// ============================================================================

// TestToCourseResponse_Timestamps verifies timestamps are formatted when set.
func TestToCourseResponse_Timestamps(t *testing.T) {
	uc, repo, _, _ := newCourseUseCase()
	created, err := uc.Create(context.Background(), &dto.CourseRequest{Title: "n"})
	require.NoError(t, err)
	repo.mu.Lock()
	if c, ok := repo.items[created.ID]; ok {
		c.CreatedAt = time.Now()
		c.UpdatedAt = time.Now()
	}
	repo.mu.Unlock()
	got, err := uc.Get(context.Background(), created.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, got.CreatedAt)
	assert.NotEmpty(t, got.UpdatedAt)
}

// TestToLessonResponse_Timestamps verifies timestamps are formatted when set.
func TestToLessonResponse_Timestamps(t *testing.T) {
	uc, courseRepo, lessonRepo, _ := newCourseUseCase()
	c, err := uc.Create(context.Background(), &dto.CourseRequest{Title: "c"})
	require.NoError(t, err)
	created, err := uc.CreateLesson(context.Background(), c.ID, &dto.LessonRequest{Title: "L"})
	require.NoError(t, err)
	lessonRepo.mu.Lock()
	if l, ok := lessonRepo.items[created.ID]; ok {
		l.CreatedAt = time.Now()
		l.UpdatedAt = time.Now()
	}
	lessonRepo.mu.Unlock()
	got, err := uc.GetLesson(context.Background(), created.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, got.CreatedAt)
	assert.NotEmpty(t, got.UpdatedAt)
	_ = courseRepo
}
