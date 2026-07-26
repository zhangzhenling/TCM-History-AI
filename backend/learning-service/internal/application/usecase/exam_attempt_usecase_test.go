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
	"tcm-history-ai/backend/learning-service/internal/domain/event"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

// newExamAttemptUseCase wires up an ExamAttemptUseCase with in-memory mocks.
// A pre-existing published exam is seeded so Start can find it.
func newExamAttemptUseCase() (*usecase.ExamAttemptUseCase, *mockExamAttemptRepo, *mockExamRepo, *mockQuestionRepo, *mockWrongQuestionRepo, *mockEventPublisher, *entity.Exam) {
	attemptRepo := newMockExamAttemptRepo()
	examRepo := newMockExamRepo()
	questionRepo := newMockQuestionRepo()
	wrongQRepo := newMockWrongQuestionRepo()
	pub := newMockEventPublisher()
	exam := &entity.Exam{Title: "exam", IsPublished: true, PassScore: 60}
	exam.ID = 1
	_ = examRepo.Create(context.Background(), exam)
	uc := usecase.NewExamAttemptUseCase(attemptRepo, examRepo, questionRepo, wrongQRepo, pub)
	return uc, attemptRepo, examRepo, questionRepo, wrongQRepo, pub, exam
}

// ============================================================================
// Start
// ============================================================================

// TestExamAttemptUseCase_Start_HappyPath verifies a fresh attempt is created
// with 0 score and an empty AnswersJSON.
func TestExamAttemptUseCase_Start_HappyPath(t *testing.T) {
	uc, attemptRepo, _, _, _, _, exam := newExamAttemptUseCase()
	resp, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: exam.ID,
		UserID: 7,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotZero(t, resp.ID)
	assert.Equal(t, exam.ID, resp.ExamID)
	assert.Equal(t, int64(7), resp.UserID)
	assert.Equal(t, 0, resp.Score)
	assert.Equal(t, 0, resp.TotalScore)
	assert.False(t, resp.IsPassed)
	assert.NotEmpty(t, resp.StartedAt)
	assert.Equal(t, json.RawMessage("[]"), resp.AnswersJSON)

	got, err := attemptRepo.FindByID(context.Background(), resp.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
}

// TestExamAttemptUseCase_Start_ValidationErrors covers input validations.
func TestExamAttemptUseCase_Start_ValidationErrors(t *testing.T) {
	uc, _, _, _, _, _, _ := newExamAttemptUseCase()
	cases := []struct {
		name string
		in   *dto.ExamAttemptStartRequest
	}{
		{"nil request", nil},
		{"zero exam_id", &dto.ExamAttemptStartRequest{UserID: 1}},
		{"zero user_id", &dto.ExamAttemptStartRequest{ExamID: 1}},
		{"negative exam_id", &dto.ExamAttemptStartRequest{ExamID: -1, UserID: 1}},
		{"negative user_id", &dto.ExamAttemptStartRequest{ExamID: 1, UserID: -1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := uc.Start(context.Background(), c.in)
			require.Error(t, err)
			var e *errno.Error
			if errors.As(err, &e) {
				assert.Equal(t, errno.InvalidParams, e.Code)
			}
		})
	}
}

// TestExamAttemptUseCase_Start_ExamNotFound verifies missing exam is rejected.
func TestExamAttemptUseCase_Start_ExamNotFound(t *testing.T) {
	uc, _, _, _, _, _, _ := newExamAttemptUseCase()
	_, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: 9999, UserID: 1,
	})
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.NotFound, e.Code)
	}
}

// TestExamAttemptUseCase_Start_ExamFindError verifies exam repo errors
// propagate.
func TestExamAttemptUseCase_Start_ExamFindError(t *testing.T) {
	uc, _, examRepo, _, _, _, _ := newExamAttemptUseCase()
	examRepo.find = func(int64) (*entity.Exam, error) { return nil, errors.New("find err") }
	_, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{ExamID: 1, UserID: 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find err")
}

// TestExamAttemptUseCase_Start_UnpublishedExamForbidden verifies a non-published
// exam cannot be started.
func TestExamAttemptUseCase_Start_UnpublishedExamForbidden(t *testing.T) {
	uc, _, examRepo, _, _, _, _ := newExamAttemptUseCase()
	unpublished := &entity.Exam{Title: "draft", IsPublished: false}
	unpublished.ID = idgen.Next()
	require.NoError(t, examRepo.Create(context.Background(), unpublished))
	_, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: unpublished.ID, UserID: 1,
	})
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.Forbidden, e.Code)
	}
}

// TestExamAttemptUseCase_Start_RepoCreateError verifies attempt repo errors
// propagate.
func TestExamAttemptUseCase_Start_RepoCreateError(t *testing.T) {
	uc, attemptRepo, _, _, _, _, exam := newExamAttemptUseCase()
	attemptRepo.create = func(*entity.ExamAttempt) error { return errors.New("create err") }
	_, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: exam.ID, UserID: 1,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create err")
}

// ============================================================================
// Get / List
// ============================================================================

// TestExamAttemptUseCase_Get covers found / not-found / error paths.
func TestExamAttemptUseCase_Get(t *testing.T) {
	uc, attemptRepo, _, _, _, _, exam := newExamAttemptUseCase()
	created, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: exam.ID, UserID: 1,
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
		attemptRepo.find = func(int64) (*entity.ExamAttempt, error) {
			return nil, errors.New("find err")
		}
		_, err := uc.Get(context.Background(), 1)
		require.Error(t, err)
	})
}

// TestExamAttemptUseCase_ListByUserAndExam covers happy / error paths.
func TestExamAttemptUseCase_ListByUserAndExam(t *testing.T) {
	uc, _, _, _, _, _, exam := newExamAttemptUseCase()
	for i := 0; i < 3; i++ {
		_, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
			ExamID: exam.ID, UserID: 7,
		})
		require.NoError(t, err)
	}
	resp, err := uc.ListByUserAndExam(context.Background(), 7, exam.ID, pagination.Params{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, resp.Total)
	assert.Equal(t, 2, resp.TotalPage)
	require.Len(t, resp.Items, 2)
}

// ============================================================================
// Submit
// ============================================================================

// seedQuestionsForExam inserts the given questions into the question repo and
// returns them.
func seedQuestionsForExam(t *testing.T, repo *mockQuestionRepo, examID int64, qs []*entity.Question) {
	t.Helper()
	for _, q := range qs {
		if q.ID == 0 {
			q.ID = idgen.Next()
		}
		q.ExamID = examID
		require.NoError(t, repo.Create(context.Background(), q))
	}
}

// TestExamAttemptUseCase_Submit_HappyPath_SingleChoice verifies a correct
// single-choice answer is scored and the attempt is marked passed.
func TestExamAttemptUseCase_Submit_HappyPath_SingleChoice(t *testing.T) {
	uc, attemptRepo, examRepo, questionRepo, _, pub, exam := newExamAttemptUseCase()
	// Seed one question worth 1 point; correct answer is index 1.
	q := &entity.Question{
		Type:       entity.QuestionTypeSingleChoice,
		Content:    "1+1=?",
		OptionsJSON: json.RawMessage(`["0","2","3"]`),
		AnswerJSON:  json.RawMessage(`1`),
		Score:      1,
	}
	seedQuestionsForExam(t, questionRepo, exam.ID, []*entity.Question{q})

	attempt, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: exam.ID, UserID: 7,
	})
	require.NoError(t, err)
	// Backdate StartedAt so the integer-second duration is guaranteed > 0.
	attemptRepo.mu.Lock()
	if a, ok := attemptRepo.items[attempt.ID]; ok {
		a.StartedAt = time.Now().Add(-90 * time.Second)
	}
	attemptRepo.mu.Unlock()

	resp, err := uc.Submit(context.Background(), attempt.ID, &dto.ExamAttemptSubmitRequest{
		UserID: 7,
		Answers: []dto.ExamAttemptAnswerItem{
			{QuestionID: q.ID, Answer: json.RawMessage(`1`)},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Score)
	assert.Equal(t, 1, resp.TotalScore)
	assert.True(t, resp.IsPassed, "100% score >= 60 pass_score should pass")
	assert.NotEmpty(t, resp.SubmittedAt)
	assert.NotZero(t, resp.DurationSeconds)

	// Repo persisted the graded attempt.
	got, err := attemptRepo.FindByID(context.Background(), attempt.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, got.Score)
	require.NotNil(t, got.SubmittedAt)

	// ExamSubmitted event emitted.
	evt, ok := captureEvent[event.ExamSubmitted](pub)
	require.True(t, ok)
	assert.Equal(t, attempt.ID, evt.AttemptID)
	assert.Equal(t, exam.ID, evt.ExamID)
	assert.Equal(t, int64(7), evt.UserID)
	assert.True(t, evt.IsPassed)
	_ = examRepo
}

// TestExamAttemptUseCase_Submit_WrongAnswerRecordsWrongQuestion verifies that
// an incorrect objective answer creates a wrong-question entry.
func TestExamAttemptUseCase_Submit_WrongAnswerRecordsWrongQuestion(t *testing.T) {
	uc, _, _, questionRepo, wrongQRepo, _, exam := newExamAttemptUseCase()
	q := &entity.Question{
		Type:       entity.QuestionTypeSingleChoice,
		Content:    "1+1=?",
		OptionsJSON: json.RawMessage(`["0","2","3"]`),
		AnswerJSON:  json.RawMessage(`1`), // correct = 1
		Score:      1,
	}
	seedQuestionsForExam(t, questionRepo, exam.ID, []*entity.Question{q})

	attempt, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: exam.ID, UserID: 7,
	})
	require.NoError(t, err)

	_, err = uc.Submit(context.Background(), attempt.ID, &dto.ExamAttemptSubmitRequest{
		UserID: 7,
		Answers: []dto.ExamAttemptAnswerItem{
			{QuestionID: q.ID, Answer: json.RawMessage(`0`)}, // wrong
		},
	})
	require.NoError(t, err)

	// Wrong question should be persisted.
	require.Len(t, wrongQRepo.items, 1)
	for _, w := range wrongQRepo.items {
		assert.Equal(t, int64(7), w.UserID)
		assert.Equal(t, q.ID, w.QuestionID)
		assert.Equal(t, exam.ID, w.ExamID)
		assert.Equal(t, attempt.ID, w.AttemptID)
		assert.Equal(t, 1, w.WrongCount)
		assert.False(t, w.IsMastered)
	}
}

// TestExamAttemptUseCase_Submit_RepeatedWrongAnswerIncrementsCount verifies
// that re-submitting a wrong answer for the same question increments the
// existing wrong-question's count rather than creating a duplicate.
func TestExamAttemptUseCase_Submit_RepeatedWrongAnswerIncrementsCount(t *testing.T) {
	uc, _, _, questionRepo, wrongQRepo, _, exam := newExamAttemptUseCase()
	q := &entity.Question{
		Type:       entity.QuestionTypeSingleChoice,
		Content:    "1+1=?",
		AnswerJSON: json.RawMessage(`1`),
		Score:      1,
	}
	seedQuestionsForExam(t, questionRepo, exam.ID, []*entity.Question{q})

	// First wrong submission.
	a1, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: exam.ID, UserID: 7,
	})
	require.NoError(t, err)
	_, err = uc.Submit(context.Background(), a1.ID, &dto.ExamAttemptSubmitRequest{
		UserID: 7,
		Answers: []dto.ExamAttemptAnswerItem{
			{QuestionID: q.ID, Answer: json.RawMessage(`0`)},
		},
	})
	require.NoError(t, err)
	require.Len(t, wrongQRepo.items, 1)
	var firstID int64
	for _, w := range wrongQRepo.items {
		firstID = w.ID
		assert.Equal(t, 1, w.WrongCount)
	}

	// Second attempt: same wrong answer.
	a2, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: exam.ID, UserID: 7,
	})
	require.NoError(t, err)
	_, err = uc.Submit(context.Background(), a2.ID, &dto.ExamAttemptSubmitRequest{
		UserID: 7,
		Answers: []dto.ExamAttemptAnswerItem{
			{QuestionID: q.ID, Answer: json.RawMessage(`0`)},
		},
	})
	require.NoError(t, err)
	require.Len(t, wrongQRepo.items, 1, "no duplicate wrong-question should be created")
	got, _ := wrongQRepo.FindByID(context.Background(), firstID)
	require.NotNil(t, got)
	assert.Equal(t, 2, got.WrongCount, "wrong_count should be incremented")
}

// TestExamAttemptUseCase_Submit_TrueFalseCorrect verifies true/false scoring.
func TestExamAttemptUseCase_Submit_TrueFalseCorrect(t *testing.T) {
	uc, _, _, questionRepo, _, _, exam := newExamAttemptUseCase()
	q := &entity.Question{
		Type:       entity.QuestionTypeTrueFalse,
		Content:    "太阳病属表",
		AnswerJSON: json.RawMessage(`true`),
		Score:      2,
	}
	seedQuestionsForExam(t, questionRepo, exam.ID, []*entity.Question{q})

	a, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: exam.ID, UserID: 7,
	})
	require.NoError(t, err)
	resp, err := uc.Submit(context.Background(), a.ID, &dto.ExamAttemptSubmitRequest{
		UserID: 7,
		Answers: []dto.ExamAttemptAnswerItem{
			{QuestionID: q.ID, Answer: json.RawMessage(`true`)},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Score)
	assert.True(t, resp.IsPassed)
}

// TestExamAttemptUseCase_Submit_MultipleChoiceCorrect verifies unordered
// multiple_choice scoring.
func TestExamAttemptUseCase_Submit_MultipleChoiceCorrect(t *testing.T) {
	uc, _, _, questionRepo, _, _, exam := newExamAttemptUseCase()
	q := &entity.Question{
		Type:       entity.QuestionTypeMultipleChoice,
		Content:    "下列哪些是伤寒论方剂？",
		AnswerJSON: json.RawMessage(`[0,1]`),
		Score:      2,
	}
	seedQuestionsForExam(t, questionRepo, exam.ID, []*entity.Question{q})

	a, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: exam.ID, UserID: 7,
	})
	require.NoError(t, err)
	// Submit answers in opposite order; should still be marked correct.
	resp, err := uc.Submit(context.Background(), a.ID, &dto.ExamAttemptSubmitRequest{
		UserID: 7,
		Answers: []dto.ExamAttemptAnswerItem{
			{QuestionID: q.ID, Answer: json.RawMessage(`[1,0]`)},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Score)
	assert.True(t, resp.IsPassed)
}

// TestExamAttemptUseCase_Submit_MultipleChoiceWrongLength verifies that a
// different-length answer set is marked wrong.
func TestExamAttemptUseCase_Submit_MultipleChoiceWrongLength(t *testing.T) {
	uc, _, _, questionRepo, _, _, exam := newExamAttemptUseCase()
	q := &entity.Question{
		Type:       entity.QuestionTypeMultipleChoice,
		Content:    "q",
		AnswerJSON: json.RawMessage(`[0,1]`),
		Score:      2,
	}
	seedQuestionsForExam(t, questionRepo, exam.ID, []*entity.Question{q})

	a, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: exam.ID, UserID: 7,
	})
	require.NoError(t, err)
	resp, err := uc.Submit(context.Background(), a.ID, &dto.ExamAttemptSubmitRequest{
		UserID: 7,
		Answers: []dto.ExamAttemptAnswerItem{
			{QuestionID: q.ID, Answer: json.RawMessage(`[0]`)}, // wrong length
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Score)
	assert.False(t, resp.IsPassed)
}

// TestExamAttemptUseCase_Submit_EssayNotScored verifies that fill_blank/essay
// questions are not auto-scored and do not generate wrong-question entries.
func TestExamAttemptUseCase_Submit_EssayNotScored(t *testing.T) {
	uc, _, _, questionRepo, wrongQRepo, _, exam := newExamAttemptUseCase()
	q := &entity.Question{
		Type:       entity.QuestionTypeEssay,
		Content:    "论述伤寒论",
		Score:      5,
	}
	seedQuestionsForExam(t, questionRepo, exam.ID, []*entity.Question{q})

	a, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: exam.ID, UserID: 7,
	})
	require.NoError(t, err)
	resp, err := uc.Submit(context.Background(), a.ID, &dto.ExamAttemptSubmitRequest{
		UserID: 7,
		Answers: []dto.ExamAttemptAnswerItem{
			{QuestionID: q.ID, Answer: json.RawMessage(`"伤寒论是张仲景所著..."`)},
		},
	})
	require.NoError(t, err)
	// Essay counts toward total but score is 0.
	assert.Equal(t, 0, resp.Score)
	assert.Equal(t, 5, resp.TotalScore)
	assert.False(t, resp.IsPassed, "0/5 = 0% should be below 60 pass_score")
	// No wrong-question entry should be created for essay.
	assert.Empty(t, wrongQRepo.items)
}

// TestExamAttemptUseCase_Submit_FillBlankNotScored verifies that fill_blank
// questions are also not auto-scored.
func TestExamAttemptUseCase_Submit_FillBlankNotScored(t *testing.T) {
	uc, _, _, questionRepo, wrongQRepo, _, exam := newExamAttemptUseCase()
	q := &entity.Question{
		Type:    entity.QuestionTypeFillBlank,
		Content: "伤寒论的作者是____",
		Score:   1,
	}
	seedQuestionsForExam(t, questionRepo, exam.ID, []*entity.Question{q})

	a, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: exam.ID, UserID: 7,
	})
	require.NoError(t, err)
	_, err = uc.Submit(context.Background(), a.ID, &dto.ExamAttemptSubmitRequest{
		UserID: 7,
		Answers: []dto.ExamAttemptAnswerItem{
			{QuestionID: q.ID, Answer: json.RawMessage(`"张仲景"`)},
		},
	})
	require.NoError(t, err)
	assert.Empty(t, wrongQRepo.items)
}

// TestExamAttemptUseCase_Submit_PassByAbsoluteScore verifies that when
// totalScore is 0, the absolute score is compared against pass_score.
func TestExamAttemptUseCase_Submit_PassByAbsoluteScore(t *testing.T) {
	_, _, examRepo, _, _, _, _ := newExamAttemptUseCase()
	// Add an exam with pass_score = 0 and no auto-scored questions.
	exam2 := &entity.Exam{Title: "exam2", IsPublished: true, PassScore: 0}
	exam2.ID = idgen.Next()
	require.NoError(t, examRepo.Create(context.Background(), exam2))

	// Insert an essay question (no scoring contribution).
	qRepo := newMockQuestionRepo()
	// We can't easily swap the question repo of the existing usecase, so we
	// construct a fresh usecase wired to the seeded exam2 and a populated
	// question repo.
	attemptRepo := newMockExamAttemptRepo()
	pub := newMockEventPublisher()
	wrongQRepo := newMockWrongQuestionRepo()
	uc2 := usecase.NewExamAttemptUseCase(attemptRepo, examRepo, qRepo, wrongQRepo, pub)

	a, err := uc2.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: exam2.ID, UserID: 7,
	})
	require.NoError(t, err)
	resp, err := uc2.Submit(context.Background(), a.ID, &dto.ExamAttemptSubmitRequest{
		UserID: 7,
		Answers: nil,
	})
	require.NoError(t, err)
	// score=0, totalScore=0, pass_score=0; 0 >= 0 → pass.
	assert.True(t, resp.IsPassed, "0 >= 0 should pass")
}

// TestExamAttemptUseCase_Submit_NilRequest rejects nil body.
func TestExamAttemptUseCase_Submit_NilRequest(t *testing.T) {
	uc, _, _, _, _, _, _ := newExamAttemptUseCase()
	_, err := uc.Submit(context.Background(), 1, nil)
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.InvalidParams, e.Code)
	}
}

// TestExamAttemptUseCase_Submit_NotFound verifies missing attempt is rejected.
func TestExamAttemptUseCase_Submit_NotFound(t *testing.T) {
	uc, _, _, _, _, _, _ := newExamAttemptUseCase()
	_, err := uc.Submit(context.Background(), 9999, &dto.ExamAttemptSubmitRequest{UserID: 1})
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.NotFound, e.Code)
	}
}

// TestExamAttemptUseCase_Submit_FindError verifies attempt repo errors
// propagate.
func TestExamAttemptUseCase_Submit_FindError(t *testing.T) {
	uc, attemptRepo, _, _, _, _, _ := newExamAttemptUseCase()
	attemptRepo.find = func(int64) (*entity.ExamAttempt, error) {
		return nil, errors.New("find err")
	}
	_, err := uc.Submit(context.Background(), 1, &dto.ExamAttemptSubmitRequest{UserID: 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "find err")
}

// TestExamAttemptUseCase_Submit_AlreadySubmitted verifies re-submitting a
// finalized attempt is rejected with AlreadyExists.
func TestExamAttemptUseCase_Submit_AlreadySubmitted(t *testing.T) {
	uc, attemptRepo, _, _, _, _, exam := newExamAttemptUseCase()
	a, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: exam.ID, UserID: 7,
	})
	require.NoError(t, err)
	// Manually mark as submitted to simulate prior submission.
	now := time.Now()
	got, _ := attemptRepo.FindByID(context.Background(), a.ID)
	require.NotNil(t, got)
	got.SubmittedAt = &now
	attemptRepo.items[a.ID] = got

	_, err = uc.Submit(context.Background(), a.ID, &dto.ExamAttemptSubmitRequest{
		UserID: 7,
		Answers: nil,
	})
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.AlreadyExists, e.Code)
	}
}

// TestExamAttemptUseCase_Submit_UserMismatch verifies that a submit call from
// a different user than the attempt owner is rejected with Forbidden.
func TestExamAttemptUseCase_Submit_UserMismatch(t *testing.T) {
	uc, _, _, _, _, _, exam := newExamAttemptUseCase()
	a, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: exam.ID, UserID: 7,
	})
	require.NoError(t, err)
	_, err = uc.Submit(context.Background(), a.ID, &dto.ExamAttemptSubmitRequest{
		UserID: 99, // not the owner
	})
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.Forbidden, e.Code)
	}
}

// TestExamAttemptUseCase_Submit_QuestionListError verifies question repo errors
// propagate.
func TestExamAttemptUseCase_Submit_QuestionListError(t *testing.T) {
	uc, _, _, questionRepo, _, _, exam := newExamAttemptUseCase()
	questionRepo.listByExam = func(int64) ([]entity.Question, error) {
		return nil, errors.New("list err")
	}
	a, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: exam.ID, UserID: 7,
	})
	require.NoError(t, err)
	_, err = uc.Submit(context.Background(), a.ID, &dto.ExamAttemptSubmitRequest{
		UserID: 7,
		Answers: []dto.ExamAttemptAnswerItem{
			{QuestionID: 1, Answer: json.RawMessage(`1`)},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list err")
}

// TestExamAttemptUseCase_Submit_AttemptUpdateError verifies attempt repo
// Update errors propagate.
func TestExamAttemptUseCase_Submit_AttemptUpdateError(t *testing.T) {
	uc, attemptRepo, _, questionRepo, _, _, exam := newExamAttemptUseCase()
	q := &entity.Question{
		Type:       entity.QuestionTypeSingleChoice,
		Content:    "q",
		AnswerJSON: json.RawMessage(`1`),
		Score:      1,
	}
	seedQuestionsForExam(t, questionRepo, exam.ID, []*entity.Question{q})

	a, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: exam.ID, UserID: 7,
	})
	require.NoError(t, err)
	attemptRepo.update = func(*entity.ExamAttempt) error { return errors.New("update err") }
	_, err = uc.Submit(context.Background(), a.ID, &dto.ExamAttemptSubmitRequest{
		UserID: 7,
		Answers: []dto.ExamAttemptAnswerItem{
			{QuestionID: q.ID, Answer: json.RawMessage(`1`)},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update err")
}

// TestExamAttemptUseCase_Submit_AnswerForUnknownQuestionIsIgnored verifies
// answers whose question_id does not match any question in the exam are
// silently skipped (no score, no wrong-question entry, no error).
func TestExamAttemptUseCase_Submit_AnswerForUnknownQuestionIsIgnored(t *testing.T) {
	uc, _, _, questionRepo, wrongQRepo, _, exam := newExamAttemptUseCase()
	// No questions seeded; the answer points at a non-existent question.
	a, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: exam.ID, UserID: 7,
	})
	require.NoError(t, err)
	resp, err := uc.Submit(context.Background(), a.ID, &dto.ExamAttemptSubmitRequest{
		UserID: 7,
		Answers: []dto.ExamAttemptAnswerItem{
			{QuestionID: 9999, Answer: json.RawMessage(`1`)},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Score)
	assert.Equal(t, 0, resp.TotalScore)
	assert.Empty(t, wrongQRepo.items)
	_ = questionRepo
}

// TestExamAttemptUseCase_Submit_MalformedAnswerTreatedAsWrong verifies that a
// malformed answer for an objective question is treated as wrong (score 0)
// and recorded as a wrong-question entry.
func TestExamAttemptUseCase_Submit_MalformedAnswerTreatedAsWrong(t *testing.T) {
	uc, _, _, questionRepo, wrongQRepo, _, exam := newExamAttemptUseCase()
	q := &entity.Question{
		Type:       entity.QuestionTypeSingleChoice,
		Content:    "q",
		AnswerJSON: json.RawMessage(`1`),
		Score:      1,
	}
	seedQuestionsForExam(t, questionRepo, exam.ID, []*entity.Question{q})

	a, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: exam.ID, UserID: 7,
	})
	require.NoError(t, err)
	resp, err := uc.Submit(context.Background(), a.ID, &dto.ExamAttemptSubmitRequest{
		UserID: 7,
		Answers: []dto.ExamAttemptAnswerItem{
			{QuestionID: q.ID, Answer: json.RawMessage(`"not-an-int"`)},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Score)
	assert.Len(t, wrongQRepo.items, 1)
}

// TestExamAttemptUseCase_Submit_MalformedQuestionAnswerTreatedAsWrong verifies
// that a malformed question's answer JSON is treated as wrong.
func TestExamAttemptUseCase_Submit_MalformedQuestionAnswerTreatedAsWrong(t *testing.T) {
	uc, _, _, questionRepo, _, _, exam := newExamAttemptUseCase()
	q := &entity.Question{
		Type:       entity.QuestionTypeSingleChoice,
		Content:    "q",
		AnswerJSON: json.RawMessage(`"not-an-int"`), // malformed
		Score:      1,
	}
	seedQuestionsForExam(t, questionRepo, exam.ID, []*entity.Question{q})

	a, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: exam.ID, UserID: 7,
	})
	require.NoError(t, err)
	resp, err := uc.Submit(context.Background(), a.ID, &dto.ExamAttemptSubmitRequest{
		UserID: 7,
		Answers: []dto.ExamAttemptAnswerItem{
			{QuestionID: q.ID, Answer: json.RawMessage(`1`)},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Score)
}

// TestExamAttemptUseCase_Submit_PublishesExamSubmittedEvent verifies the event
// payload carries the graded score and pass/fail flag.
func TestExamAttemptUseCase_Submit_PublishesExamSubmittedEvent(t *testing.T) {
	uc, _, _, questionRepo, _, pub, exam := newExamAttemptUseCase()
	q := &entity.Question{
		Type:       entity.QuestionTypeSingleChoice,
		Content:    "q",
		AnswerJSON: json.RawMessage(`1`),
		Score:      1,
	}
	seedQuestionsForExam(t, questionRepo, exam.ID, []*entity.Question{q})

	a, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: exam.ID, UserID: 7,
	})
	require.NoError(t, err)
	_, err = uc.Submit(context.Background(), a.ID, &dto.ExamAttemptSubmitRequest{
		UserID: 7,
		Answers: []dto.ExamAttemptAnswerItem{
			{QuestionID: q.ID, Answer: json.RawMessage(`1`)},
		},
	})
	require.NoError(t, err)
	evt, ok := captureEvent[event.ExamSubmitted](pub)
	require.True(t, ok)
	assert.Equal(t, 1, evt.Score)
	assert.True(t, evt.IsPassed)
}

// ============================================================================
// Response mapper
// ============================================================================

// TestToExamAttemptResponse_Timestamps verifies timestamps are formatted when
// set on the entity.
func TestToExamAttemptResponse_Timestamps(t *testing.T) {
	uc, attemptRepo, _, _, _, _, exam := newExamAttemptUseCase()
	created, err := uc.Start(context.Background(), &dto.ExamAttemptStartRequest{
		ExamID: exam.ID, UserID: 1,
	})
	require.NoError(t, err)
	attemptRepo.mu.Lock()
	if a, ok := attemptRepo.items[created.ID]; ok {
		a.CreatedAt = time.Now()
		a.UpdatedAt = time.Now()
	}
	attemptRepo.mu.Unlock()
	got, err := uc.Get(context.Background(), created.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, got.CreatedAt)
	assert.NotEmpty(t, got.UpdatedAt)
}

// TestExamAttemptUseCase_Get_NotFoundError verifies the repo's find error path
// is mapped to a plain error (the usecase's Get wraps with errno only on
// nil-not-found).
func TestExamAttemptUseCase_Get_FindError(t *testing.T) {
	uc, attemptRepo, _, _, _, _, _ := newExamAttemptUseCase()
	attemptRepo.find = func(int64) (*entity.ExamAttempt, error) {
		return nil, errors.New("find err")
	}
	_, err := uc.Get(context.Background(), idgen.Next())
	require.Error(t, err)
}
