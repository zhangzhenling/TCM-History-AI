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
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

// newExamUseCase wires up an ExamUseCase with in-memory mocks.
func newExamUseCase() (*usecase.ExamUseCase, *mockExamRepo, *mockQuestionRepo) {
	examRepo := newMockExamRepo()
	questionRepo := newMockQuestionRepo()
	uc := usecase.NewExamUseCase(examRepo, questionRepo)
	return uc, examRepo, questionRepo
}

// ============================================================================
// Exam CRUD
// ============================================================================

// TestExamUseCase_Create_HappyPath verifies an exam is created with a
// snowflake id and a default pass_score of 60.
func TestExamUseCase_Create_HappyPath(t *testing.T) {
	uc, repo, _ := newExamUseCase()
	resp, err := uc.Create(context.Background(), &dto.ExamRequest{
		Title:           "测验",
		CourseID:        5,
		Description:     "desc",
		DurationMinutes: 60,
		IsPublished:     true,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotZero(t, resp.ID)
	assert.Equal(t, "测验", resp.Title)
	assert.Equal(t, int64(5), resp.CourseID)
	assert.Equal(t, 60, resp.PassScore)
	assert.Equal(t, 60, resp.DurationMinutes)
	assert.True(t, resp.IsPublished)

	got, err := repo.FindByID(context.Background(), resp.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "测验", got.Title)
}

// TestExamUseCase_Create_DefaultPassScore verifies an empty PassScore defaults
// to 60.
func TestExamUseCase_Create_DefaultPassScore(t *testing.T) {
	uc, repo, _ := newExamUseCase()
	resp, err := uc.Create(context.Background(), &dto.ExamRequest{Title: "n"})
	require.NoError(t, err)
	got, _ := repo.FindByID(context.Background(), resp.ID)
	require.NotNil(t, got)
	assert.Equal(t, 60, got.PassScore)
}

// TestExamUseCase_Create_KeepsExplicitPassScore verifies a non-zero PassScore
// is preserved.
func TestExamUseCase_Create_KeepsExplicitPassScore(t *testing.T) {
	uc, repo, _ := newExamUseCase()
	resp, err := uc.Create(context.Background(), &dto.ExamRequest{
		Title:     "n",
		PassScore: 80,
	})
	require.NoError(t, err)
	got, _ := repo.FindByID(context.Background(), resp.ID)
	require.NotNil(t, got)
	assert.Equal(t, 80, got.PassScore)
}

// TestExamUseCase_Create_ValidationErrors covers input validations.
func TestExamUseCase_Create_ValidationErrors(t *testing.T) {
	uc, _, _ := newExamUseCase()
	cases := []struct {
		name string
		in   *dto.ExamRequest
	}{
		{"nil request", nil},
		{"empty title", &dto.ExamRequest{}},
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

// TestExamUseCase_Create_RepoCreateError verifies Create errors propagate.
func TestExamUseCase_Create_RepoCreateError(t *testing.T) {
	uc, repo, _ := newExamUseCase()
	repo.create = func(*entity.Exam) error { return errors.New("create err") }
	_, err := uc.Create(context.Background(), &dto.ExamRequest{Title: "n"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create err")
}

// TestExamUseCase_Update covers happy / not-found / error / nil-body paths
// and verifies PassScore is only overwritten when >0.
func TestExamUseCase_Update(t *testing.T) {
	t.Run("happy overwrites fields, keeps pass_score when input is 0", func(t *testing.T) {
		uc, _, _ := newExamUseCase()
		created, err := uc.Create(context.Background(), &dto.ExamRequest{
			Title:     "n",
			PassScore: 70,
		})
		require.NoError(t, err)
		resp, err := uc.Update(context.Background(), created.ID, &dto.ExamRequest{
			Title:       "n2",
			Description: "d2",
			PassScore:   0, // should keep existing 70
			DurationMinutes: 90,
		})
		require.NoError(t, err)
		assert.Equal(t, "n2", resp.Title)
		assert.Equal(t, "d2", resp.Description)
		assert.Equal(t, 70, resp.PassScore)
		assert.Equal(t, 90, resp.DurationMinutes)
	})
	t.Run("explicit pass_score overwrites", func(t *testing.T) {
		uc, _, _ := newExamUseCase()
		created, err := uc.Create(context.Background(), &dto.ExamRequest{Title: "n", PassScore: 50})
		require.NoError(t, err)
		resp, err := uc.Update(context.Background(), created.ID, &dto.ExamRequest{
			Title:     "n2",
			PassScore: 90,
		})
		require.NoError(t, err)
		assert.Equal(t, 90, resp.PassScore)
	})
	t.Run("nil body rejected", func(t *testing.T) {
		uc, _, _ := newExamUseCase()
		_, err := uc.Update(context.Background(), 1, nil)
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.InvalidParams, e.Code)
		}
	})
	t.Run("not found", func(t *testing.T) {
		uc, _, _ := newExamUseCase()
		_, err := uc.Update(context.Background(), 9999, &dto.ExamRequest{Title: "n"})
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.NotFound, e.Code)
		}
	})
	t.Run("find error", func(t *testing.T) {
		uc, repo, _ := newExamUseCase()
		repo.find = func(int64) (*entity.Exam, error) { return nil, errors.New("find err") }
		_, err := uc.Update(context.Background(), 1, &dto.ExamRequest{Title: "n"})
		require.Error(t, err)
	})
	t.Run("update error", func(t *testing.T) {
		uc, repo, _ := newExamUseCase()
		created, err := uc.Create(context.Background(), &dto.ExamRequest{Title: "n"})
		require.NoError(t, err)
		repo.update = func(*entity.Exam) error { return errors.New("update err") }
		_, err = uc.Update(context.Background(), created.ID, &dto.ExamRequest{Title: "n2"})
		require.Error(t, err)
	})
}

// TestExamUseCase_Delete covers happy / error paths.
func TestExamUseCase_Delete(t *testing.T) {
	uc, repo, _ := newExamUseCase()
	created, err := uc.Create(context.Background(), &dto.ExamRequest{Title: "n"})
	require.NoError(t, err)
	require.NoError(t, uc.Delete(context.Background(), created.ID))

	t.Run("repo error", func(t *testing.T) {
		repo.delete = func(int64) error { return errors.New("delete err") }
		err := uc.Delete(context.Background(), 1)
		require.Error(t, err)
	})
}

// TestExamUseCase_Get covers found / not-found / error paths.
func TestExamUseCase_Get(t *testing.T) {
	uc, repo, _ := newExamUseCase()
	created, err := uc.Create(context.Background(), &dto.ExamRequest{Title: "n"})
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
		repo.find = func(int64) (*entity.Exam, error) { return nil, errors.New("find err") }
		_, err := uc.Get(context.Background(), 1)
		require.Error(t, err)
	})
}

// TestExamUseCase_List covers happy / error paths and pagination.
func TestExamUseCase_List(t *testing.T) {
	uc, repo, _ := newExamUseCase()
	for _, title := range []string{"a", "b", "c"} {
		_, err := uc.Create(context.Background(), &dto.ExamRequest{Title: title})
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
		repo.list = func(pagination.Params) ([]entity.Exam, int, error) {
			return nil, 0, errors.New("list err")
		}
		_, err := uc.List(context.Background(), pagination.Params{Page: 1, PageSize: 2})
		require.Error(t, err)
	})
}

// TestExamUseCase_ListByCourse covers the filter / fallback-to-List / error
// paths.
func TestExamUseCase_ListByCourse(t *testing.T) {
	uc, repo, _ := newExamUseCase()
	_, err := uc.Create(context.Background(), &dto.ExamRequest{Title: "a", CourseID: 1})
	require.NoError(t, err)
	_, err = uc.Create(context.Background(), &dto.ExamRequest{Title: "b", CourseID: 2})
	require.NoError(t, err)

	t.Run("by course", func(t *testing.T) {
		resp, err := uc.ListByCourse(context.Background(), 1, pagination.Params{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Total)
		require.Len(t, resp.Items, 1)
	})
	t.Run("zero course_id falls back to List", func(t *testing.T) {
		resp, err := uc.ListByCourse(context.Background(), 0, pagination.Params{Page: 1, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Total)
	})
	t.Run("repo error", func(t *testing.T) {
		repo.listByCourse = func(int64, pagination.Params) ([]entity.Exam, int, error) {
			return nil, 0, errors.New("by course err")
		}
		_, err := uc.ListByCourse(context.Background(), 1, pagination.Params{Page: 1, PageSize: 10})
		require.Error(t, err)
	})
}

// TestExamUseCase_Publish covers happy / not-found / error paths.
func TestExamUseCase_Publish(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		uc, _, _ := newExamUseCase()
		created, err := uc.Create(context.Background(), &dto.ExamRequest{Title: "n"})
		require.NoError(t, err)
		resp, err := uc.Publish(context.Background(), created.ID)
		require.NoError(t, err)
		assert.True(t, resp.IsPublished)
	})
	t.Run("not found", func(t *testing.T) {
		uc, _, _ := newExamUseCase()
		_, err := uc.Publish(context.Background(), 9999)
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.NotFound, e.Code)
		}
	})
	t.Run("find error", func(t *testing.T) {
		uc, repo, _ := newExamUseCase()
		repo.find = func(int64) (*entity.Exam, error) { return nil, errors.New("find err") }
		_, err := uc.Publish(context.Background(), 1)
		require.Error(t, err)
	})
	t.Run("update error", func(t *testing.T) {
		uc, repo, _ := newExamUseCase()
		created, err := uc.Create(context.Background(), &dto.ExamRequest{Title: "n"})
		require.NoError(t, err)
		repo.update = func(*entity.Exam) error { return errors.New("update err") }
		_, err = uc.Publish(context.Background(), created.ID)
		require.Error(t, err)
	})
}

// ============================================================================
// Question CRUD
// ============================================================================

// TestExamUseCase_CreateQuestion_HappyPath verifies a question is created
// under an exam with defaults for type/difficulty/score/options/answer.
func TestExamUseCase_CreateQuestion_HappyPath(t *testing.T) {
	uc, examRepo, questionRepo := newExamUseCase()
	exam, err := uc.Create(context.Background(), &dto.ExamRequest{Title: "exam"})
	require.NoError(t, err)
	resp, err := uc.CreateQuestion(context.Background(), exam.ID, &dto.QuestionRequest{
		Content:     "1+1=?",
		Explanation: "addition",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotZero(t, resp.ID)
	assert.Equal(t, exam.ID, resp.ExamID)
	assert.Equal(t, entity.QuestionTypeSingleChoice, resp.Type)
	assert.Equal(t, entity.DifficultyBeginner, resp.Difficulty)
	assert.Equal(t, 1, resp.Score)
	assert.Equal(t, json.RawMessage("[]"), resp.OptionsJSON)
	assert.Equal(t, json.RawMessage("[]"), resp.AnswerJSON)

	got, err := questionRepo.FindByID(context.Background(), resp.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "1+1=?", got.Content)
	_ = examRepo
}

// TestExamUseCase_CreateQuestion_KeepsExplicitFields verifies explicit
// type/difficulty/score/options/answer are preserved.
func TestExamUseCase_CreateQuestion_KeepsExplicitFields(t *testing.T) {
	uc, _, _ := newExamUseCase()
	exam, err := uc.Create(context.Background(), &dto.ExamRequest{Title: "exam"})
	require.NoError(t, err)
	resp, err := uc.CreateQuestion(context.Background(), exam.ID, &dto.QuestionRequest{
		Content:      "下列哪些是伤寒论方剂？",
		Type:         entity.QuestionTypeMultipleChoice,
		OptionsJSON:  json.RawMessage(`["麻黄汤","桂枝汤"]`),
		AnswerJSON:   json.RawMessage(`[0,1]`),
		Explanation:  "ex",
		Score:        3,
		Difficulty:   entity.DifficultyAdvanced,
	})
	require.NoError(t, err)
	assert.Equal(t, entity.QuestionTypeMultipleChoice, resp.Type)
	assert.Equal(t, entity.DifficultyAdvanced, resp.Difficulty)
	assert.Equal(t, 3, resp.Score)
	assert.Equal(t, json.RawMessage(`["麻黄汤","桂枝汤"]`), resp.OptionsJSON)
	assert.Equal(t, json.RawMessage(`[0,1]`), resp.AnswerJSON)
}

// TestExamUseCase_CreateQuestion_ValidationErrors covers input validations.
func TestExamUseCase_CreateQuestion_ValidationErrors(t *testing.T) {
	uc, _, _ := newExamUseCase()
	cases := []struct {
		name    string
		examID  int64
		in      *dto.QuestionRequest
	}{
		{"nil request", 1, nil},
		{"empty content", 1, &dto.QuestionRequest{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := uc.CreateQuestion(context.Background(), c.examID, c.in)
			require.Error(t, err)
			var e *errno.Error
			if errors.As(err, &e) {
				assert.Equal(t, errno.InvalidParams, e.Code)
			}
		})
	}
}

// TestExamUseCase_CreateQuestion_ExamNotFound verifies missing exam is
// rejected.
func TestExamUseCase_CreateQuestion_ExamNotFound(t *testing.T) {
	uc, _, _ := newExamUseCase()
	_, err := uc.CreateQuestion(context.Background(), 9999, &dto.QuestionRequest{Content: "c"})
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.NotFound, e.Code)
	}
}

// TestExamUseCase_CreateQuestion_ExamFindError verifies exam repo errors
// propagate.
func TestExamUseCase_CreateQuestion_ExamFindError(t *testing.T) {
	uc, examRepo, _ := newExamUseCase()
	examRepo.find = func(int64) (*entity.Exam, error) { return nil, errors.New("find err") }
	_, err := uc.CreateQuestion(context.Background(), 1, &dto.QuestionRequest{Content: "c"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find err")
}

// TestExamUseCase_CreateQuestion_QuestionCreateError verifies question repo
// errors propagate.
func TestExamUseCase_CreateQuestion_QuestionCreateError(t *testing.T) {
	uc, examRepo, questionRepo := newExamUseCase()
	exam, err := uc.Create(context.Background(), &dto.ExamRequest{Title: "e"})
	require.NoError(t, err)
	questionRepo.create = func(*entity.Question) error { return errors.New("q create err") }
	_, err = uc.CreateQuestion(context.Background(), exam.ID, &dto.QuestionRequest{Content: "c"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "q create err")
	_ = examRepo
}

// TestExamUseCase_UpdateQuestion covers happy / not-found / nil-body / error
// paths and verifies partial updates preserve existing fields.
func TestExamUseCase_UpdateQuestion(t *testing.T) {
	t.Run("happy partial update keeps untouched fields", func(t *testing.T) {
		uc, _, _ := newExamUseCase()
		exam, err := uc.Create(context.Background(), &dto.ExamRequest{Title: "e"})
		require.NoError(t, err)
		created, err := uc.CreateQuestion(context.Background(), exam.ID, &dto.QuestionRequest{
			Content:    "q",
			Type:       entity.QuestionTypeMultipleChoice,
			Score:      2,
			Difficulty: entity.DifficultyAdvanced,
			OptionsJSON: json.RawMessage(`["a"]`),
			AnswerJSON:  json.RawMessage(`[0]`),
		})
		require.NoError(t, err)

		// Update only Content + Explanation; type/score/difficulty/options/answer
		// should be preserved because they are zero/empty.
		resp, err := uc.UpdateQuestion(context.Background(), created.ID, &dto.QuestionRequest{
			Content:     "q2",
			Explanation: "ex2",
		})
		require.NoError(t, err)
		assert.Equal(t, "q2", resp.Content)
		assert.Equal(t, "ex2", resp.Explanation)
		assert.Equal(t, entity.QuestionTypeMultipleChoice, resp.Type, "type should be preserved")
		assert.Equal(t, 2, resp.Score, "score should be preserved")
		assert.Equal(t, entity.DifficultyAdvanced, resp.Difficulty, "difficulty should be preserved")
		assert.Equal(t, json.RawMessage(`["a"]`), resp.OptionsJSON, "options should be preserved")
		assert.Equal(t, json.RawMessage(`[0]`), resp.AnswerJSON, "answer should be preserved")
	})
	t.Run("nil body rejected", func(t *testing.T) {
		uc, _, _ := newExamUseCase()
		_, err := uc.UpdateQuestion(context.Background(), 1, nil)
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.InvalidParams, e.Code)
		}
	})
	t.Run("not found", func(t *testing.T) {
		uc, _, _ := newExamUseCase()
		_, err := uc.UpdateQuestion(context.Background(), 9999, &dto.QuestionRequest{Content: "c"})
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.NotFound, e.Code)
		}
	})
	t.Run("find error", func(t *testing.T) {
		uc, _, questionRepo := newExamUseCase()
		questionRepo.find = func(int64) (*entity.Question, error) { return nil, errors.New("find err") }
		_, err := uc.UpdateQuestion(context.Background(), 1, &dto.QuestionRequest{Content: "c"})
		require.Error(t, err)
	})
	t.Run("update error", func(t *testing.T) {
		uc, _, questionRepo := newExamUseCase()
		exam, err := uc.Create(context.Background(), &dto.ExamRequest{Title: "e"})
		require.NoError(t, err)
		created, err := uc.CreateQuestion(context.Background(), exam.ID, &dto.QuestionRequest{Content: "c"})
		require.NoError(t, err)
		questionRepo.update = func(*entity.Question) error { return errors.New("update err") }
		_, err = uc.UpdateQuestion(context.Background(), created.ID, &dto.QuestionRequest{Content: "c2"})
		require.Error(t, err)
	})
}

// TestExamUseCase_DeleteQuestion covers happy / not-found / error paths and
// verifies UpdateExamCount is invoked.
func TestExamUseCase_DeleteQuestion(t *testing.T) {
	t.Run("happy refreshes exam count", func(t *testing.T) {
		uc, _, questionRepo := newExamUseCase()
		exam, err := uc.Create(context.Background(), &dto.ExamRequest{Title: "e"})
		require.NoError(t, err)
		created, err := uc.CreateQuestion(context.Background(), exam.ID, &dto.QuestionRequest{Content: "c"})
		require.NoError(t, err)
		refreshed := false
		questionRepo.updateExamCount = func(int64) error { refreshed = true; return nil }
		require.NoError(t, uc.DeleteQuestion(context.Background(), created.ID))
		assert.True(t, refreshed)
		got, _ := questionRepo.FindByID(context.Background(), created.ID)
		assert.Nil(t, got)
	})
	t.Run("not found", func(t *testing.T) {
		uc, _, _ := newExamUseCase()
		err := uc.DeleteQuestion(context.Background(), 9999)
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.NotFound, e.Code)
		}
	})
	t.Run("find error", func(t *testing.T) {
		uc, _, questionRepo := newExamUseCase()
		questionRepo.find = func(int64) (*entity.Question, error) { return nil, errors.New("find err") }
		err := uc.DeleteQuestion(context.Background(), 1)
		require.Error(t, err)
	})
	t.Run("delete error", func(t *testing.T) {
		uc, _, questionRepo := newExamUseCase()
		exam, err := uc.Create(context.Background(), &dto.ExamRequest{Title: "e"})
		require.NoError(t, err)
		created, err := uc.CreateQuestion(context.Background(), exam.ID, &dto.QuestionRequest{Content: "c"})
		require.NoError(t, err)
		questionRepo.delete = func(int64) error { return errors.New("delete err") }
		err = uc.DeleteQuestion(context.Background(), created.ID)
		require.Error(t, err)
	})
}

// TestExamUseCase_GetQuestion covers found / not-found / error paths.
func TestExamUseCase_GetQuestion(t *testing.T) {
	uc, _, _ := newExamUseCase()
	exam, err := uc.Create(context.Background(), &dto.ExamRequest{Title: "e"})
	require.NoError(t, err)
	created, err := uc.CreateQuestion(context.Background(), exam.ID, &dto.QuestionRequest{Content: "c"})
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		got, err := uc.GetQuestion(context.Background(), created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
	})
	t.Run("not found", func(t *testing.T) {
		_, err := uc.GetQuestion(context.Background(), 9999)
		require.Error(t, err)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.NotFound, e.Code)
		}
	})
}

// TestExamUseCase_ListQuestionsByExam covers happy / error paths.
func TestExamUseCase_ListQuestionsByExam(t *testing.T) {
	uc, _, _ := newExamUseCase()
	exam, err := uc.Create(context.Background(), &dto.ExamRequest{Title: "e"})
	require.NoError(t, err)
	for _, c := range []string{"q1", "q2", "q3"} {
		_, err := uc.CreateQuestion(context.Background(), exam.ID, &dto.QuestionRequest{Content: c})
		require.NoError(t, err)
	}
	resp, err := uc.ListQuestionsByExam(context.Background(), exam.ID)
	require.NoError(t, err)
	assert.Len(t, resp, 3)
}

// TestExamUseCase_ListQuestionsByExam_Error verifies repo errors propagate.
func TestExamUseCase_ListQuestionsByExam_Error(t *testing.T) {
	uc, _, questionRepo := newExamUseCase()
	questionRepo.listByExam = func(int64) ([]entity.Question, error) {
		return nil, errors.New("list err")
	}
	_, err := uc.ListQuestionsByExam(context.Background(), 1)
	require.Error(t, err)
}

// ============================================================================
// Response mappers
// ============================================================================

// TestToExamResponse_Timestamps verifies timestamps are formatted when set.
func TestToExamResponse_Timestamps(t *testing.T) {
	uc, repo, _ := newExamUseCase()
	created, err := uc.Create(context.Background(), &dto.ExamRequest{Title: "n"})
	require.NoError(t, err)
	repo.mu.Lock()
	if e, ok := repo.items[created.ID]; ok {
		e.CreatedAt = time.Now()
		e.UpdatedAt = time.Now()
	}
	repo.mu.Unlock()
	got, err := uc.Get(context.Background(), created.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, got.CreatedAt)
	assert.NotEmpty(t, got.UpdatedAt)
}

// TestToQuestionResponse_Timestamps verifies timestamps are formatted when
// set on the entity.
func TestToQuestionResponse_Timestamps(t *testing.T) {
	uc, _, questionRepo := newExamUseCase()
	exam, err := uc.Create(context.Background(), &dto.ExamRequest{Title: "e"})
	require.NoError(t, err)
	created, err := uc.CreateQuestion(context.Background(), exam.ID, &dto.QuestionRequest{Content: "c"})
	require.NoError(t, err)
	questionRepo.mu.Lock()
	if q, ok := questionRepo.items[created.ID]; ok {
		q.CreatedAt = time.Now()
		q.UpdatedAt = time.Now()
	}
	questionRepo.mu.Unlock()
	got, err := uc.GetQuestion(context.Background(), created.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, got.CreatedAt)
	assert.NotEmpty(t, got.UpdatedAt)
}

// TestExamUseCase_RepoNotFoundReturnsNil verifies the exam repo's not-found
// path returns (nil, nil) and is mapped to errno.NotFound by the usecase.
func TestExamUseCase_RepoNotFoundReturnsNil(t *testing.T) {
	uc, _, _ := newExamUseCase()
	_, err := uc.Get(context.Background(), idgen.Next())
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.NotFound, e.Code)
	}
}
