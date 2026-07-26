package usecase

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"tcm-history-ai/backend/learning-service/internal/application/dto"
	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/learning-service/internal/domain/event"
	"tcm-history-ai/backend/learning-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

// ExamAttemptUseCase implements start / submit / list operations on exam
// attempts, including auto-scoring of objective questions and integration
// with the wrong-question collector.
type ExamAttemptUseCase struct {
	attemptRepo    repository.ExamAttemptRepository
	examRepo       repository.ExamRepository
	questionRepo   repository.QuestionRepository
	wrongQRepo     repository.WrongQuestionRepository
	pub            event.EventPublisher
}

// NewExamAttemptUseCase constructs an ExamAttemptUseCase.
func NewExamAttemptUseCase(
	attemptRepo repository.ExamAttemptRepository,
	examRepo repository.ExamRepository,
	questionRepo repository.QuestionRepository,
	wrongQRepo repository.WrongQuestionRepository,
	pub event.EventPublisher,
) *ExamAttemptUseCase {
	return &ExamAttemptUseCase{
		attemptRepo:  attemptRepo,
		examRepo:     examRepo,
		questionRepo: questionRepo,
		wrongQRepo:   wrongQRepo,
		pub:          pub,
	}
}

// Start creates a new in-progress attempt for a (user, exam) pair.
func (uc *ExamAttemptUseCase) Start(ctx context.Context, in *dto.ExamAttemptStartRequest) (*dto.ExamAttemptResponse, error) {
	if in == nil || in.ExamID <= 0 || in.UserID <= 0 {
		return nil, errno.New(errno.InvalidParams, "exam_id and user_id are required")
	}
	e, err := uc.examRepo.FindByID(ctx, in.ExamID)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errno.New(errno.NotFound, "exam not found")
	}
	if !e.IsPublished {
		return nil, errno.New(errno.Forbidden, "exam is not published")
	}
	a := &entity.ExamAttempt{
		ExamID:      in.ExamID,
		UserID:      in.UserID,
		Score:       0,
		TotalScore:  0,
		IsPassed:    false,
		StartedAt:   time.Now(),
		AnswersJSON: []byte("[]"),
	}
	a.ID = idgen.Next()
	if err := uc.attemptRepo.Create(ctx, a); err != nil {
		return nil, err
	}
	return toExamAttemptResponse(a), nil
}

// Get retrieves a single attempt by id.
func (uc *ExamAttemptUseCase) Get(ctx context.Context, id int64) (*dto.ExamAttemptResponse, error) {
	a, err := uc.attemptRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, errno.New(errno.NotFound, "attempt not found")
	}
	return toExamAttemptResponse(a), nil
}

// ListByUserAndExam returns paginated attempts for a (user, exam) pair.
func (uc *ExamAttemptUseCase) ListByUserAndExam(ctx context.Context, userID, examID int64, p pagination.Params) (dto.ListResponse[dto.ExamAttemptResponse], error) {
	items, total, err := uc.attemptRepo.ListByUserAndExam(ctx, userID, examID, p)
	if err != nil {
		return dto.ListResponse[dto.ExamAttemptResponse]{}, err
	}
	resp := make([]dto.ExamAttemptResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toExamAttemptResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// Submit grades a submitted attempt, persists score and wrong questions,
// and emits an ExamSubmitted event.
//
// Auto-scoring covers single_choice / multiple_choice / true_false:
//   - single_choice: user answer equals the correct option index
//   - multiple_choice: user answer set equals the correct option set
//   - true_false: user answer (bool) equals the correct answer (bool)
// fill_blank / essay are not auto-scored (counted as 0 in this version).
func (uc *ExamAttemptUseCase) Submit(ctx context.Context, id int64, in *dto.ExamAttemptSubmitRequest) (*dto.ExamAttemptResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "body is required")
	}
	a, err := uc.attemptRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, errno.New(errno.NotFound, "attempt not found: "+strconv.FormatInt(id, 10))
	}
	if a.SubmittedAt != nil {
		return nil, errno.New(errno.AlreadyExists, "attempt already submitted")
	}
	if in.UserID != a.UserID {
		return nil, errno.New(errno.Forbidden, "attempt does not belong to user")
	}
	questions, err := uc.questionRepo.ListByExam(ctx, a.ExamID)
	if err != nil {
		return nil, err
	}
	questionByID := make(map[int64]*entity.Question, len(questions))
	for i := range questions {
		questionByID[questions[i].ID] = &questions[i]
	}
	// Build answers payload (question_id -> user_answer) for storage.
	type storedAnswer struct {
		QuestionID int64           `json:"question_id"`
		UserAnswer json.RawMessage `json:"user_answer"`
		Correct    bool            `json:"correct"`
		Score      int             `json:"score"`
	}
	stored := make([]storedAnswer, 0, len(in.Answers))
	score := 0
	totalScore := 0
	wrongQuestions := make([]entity.WrongQuestion, 0)
	for _, ans := range in.Answers {
		q, ok := questionByID[ans.QuestionID]
		if !ok {
			continue
		}
		totalScore += q.Score
		correct, scored := autoScore(q, ans.Answer)
		gained := 0
		if correct {
			gained = q.Score
			score += gained
		} else if scored {
			// Collect as wrong question (objective questions only).
			wq := entity.WrongQuestion{
				UserID:         a.UserID,
				QuestionID:     q.ID,
				ExamID:         a.ExamID,
				AttemptID:      a.ID,
				UserAnswerJSON: ans.Answer,
				WrongCount:     1,
				LastWrongAt:    time.Now(),
				IsMastered:     false,
			}
			wq.ID = idgen.Next()
			wrongQuestions = append(wrongQuestions, wq)
		}
		stored = append(stored, storedAnswer{
			QuestionID: ans.QuestionID,
			UserAnswer: ans.Answer,
			Correct:    correct,
			Score:      gained,
		})
	}
	answersJSON, _ := json.Marshal(stored)
	a.AnswersJSON = answersJSON
	a.Score = score
	a.TotalScore = totalScore
	now := time.Now()
	a.SubmittedAt = &now
	a.DurationSeconds = int(now.Sub(a.StartedAt).Seconds())
	// Determine pass/fail. Use percentage against total when total > 0;
	// otherwise compare absolute score to exam pass_score.
	if a.ExamID > 0 {
		if e, err := uc.examRepo.FindByID(ctx, a.ExamID); err == nil && e != nil {
			if totalScore > 0 {
				a.IsPassed = score*100/totalScore >= e.PassScore
			} else {
				a.IsPassed = score >= e.PassScore
			}
		}
	}
	if err := uc.attemptRepo.Update(ctx, a); err != nil {
		return nil, err
	}
	// Persist wrong questions (best-effort, dedup by user+question).
	for i := range wrongQuestions {
		wq := wrongQuestions[i]
		existing, err := uc.wrongQRepo.FindByUserAndQuestion(ctx, wq.UserID, wq.QuestionID)
		if err != nil {
			continue
		}
		if existing == nil {
			_ = uc.wrongQRepo.Create(ctx, &wq)
			continue
		}
		existing.WrongCount++
		existing.LastWrongAt = now
		existing.IsMastered = false
		existing.UserAnswerJSON = wq.UserAnswerJSON
		existing.AttemptID = wq.AttemptID
		_ = uc.wrongQRepo.Update(ctx, existing)
	}
	if uc.pub != nil {
		_ = uc.pub.Publish(ctx, event.ExamSubmitted{
			AttemptID: a.ID,
			ExamID:    a.ExamID,
			UserID:    a.UserID,
			Score:     a.Score,
			IsPassed:  a.IsPassed,
		})
	}
	return toExamAttemptResponse(a), nil
}

// autoScore grades an objective question. Returns (correct, scoreable).
// scoreable=false for fill_blank / essay questions.
func autoScore(q *entity.Question, userAnswer json.RawMessage) (bool, bool) {
	switch q.Type {
	case entity.QuestionTypeSingleChoice:
		var got int
		if err := json.Unmarshal(userAnswer, &got); err != nil {
			return false, true
		}
		var want int
		if err := json.Unmarshal(q.AnswerJSON, &want); err != nil {
			return false, true
		}
		return got == want, true
	case entity.QuestionTypeTrueFalse:
		var got bool
		if err := json.Unmarshal(userAnswer, &got); err != nil {
			return false, true
		}
		var want bool
		if err := json.Unmarshal(q.AnswerJSON, &want); err != nil {
			return false, true
		}
		return got == want, true
	case entity.QuestionTypeMultipleChoice:
		var got []int
		if err := json.Unmarshal(userAnswer, &got); err != nil {
			return false, true
		}
		var want []int
		if err := json.Unmarshal(q.AnswerJSON, &want); err != nil {
			return false, true
		}
		return intSliceEqualUnordered(got, want), true
	default:
		// fill_blank / essay: not auto-scored.
		return false, false
	}
}

// intSliceEqualUnordered reports whether two int slices contain the same
// elements regardless of order.
func intSliceEqualUnordered(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[int]int, len(a))
	for _, v := range a {
		counts[v]++
	}
	for _, v := range b {
		counts[v]--
		if counts[v] < 0 {
			return false
		}
	}
	return true
}

// toExamAttemptResponse maps the entity to its wire DTO.
func toExamAttemptResponse(a *entity.ExamAttempt) *dto.ExamAttemptResponse {
	if a == nil {
		return nil
	}
	resp := &dto.ExamAttemptResponse{
		ID:              a.ID,
		ExamID:          a.ExamID,
		UserID:          a.UserID,
		Score:           a.Score,
		TotalScore:      a.TotalScore,
		IsPassed:        a.IsPassed,
		StartedAt:       a.StartedAt.Format(time.RFC3339),
		DurationSeconds: a.DurationSeconds,
		AnswersJSON:     a.AnswersJSON,
	}
	if a.SubmittedAt != nil {
		resp.SubmittedAt = a.SubmittedAt.Format(time.RFC3339)
	}
	if !a.CreatedAt.IsZero() {
		resp.CreatedAt = a.CreatedAt.Format(time.RFC3339)
	}
	if !a.UpdatedAt.IsZero() {
		resp.UpdatedAt = a.UpdatedAt.Format(time.RFC3339)
	}
	return resp
}
