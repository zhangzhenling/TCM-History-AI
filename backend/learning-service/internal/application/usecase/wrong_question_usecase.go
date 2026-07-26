package usecase

import (
	"context"
	"time"

	"tcm-history-ai/backend/learning-service/internal/application/dto"
	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/learning-service/internal/domain/repository"
	"tcm-history-ai/backend/learning-service/internal/infrastructure/cache"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// WrongQuestionUseCase implements read-only access to the wrong-question
// book plus marking-as-mastered. Wrong questions are persisted during exam
// submission (see ExamAttemptUseCase.Submit).
type WrongQuestionUseCase struct {
	repo  repository.WrongQuestionRepository
	cache *cache.RedisClient
}

// NewWrongQuestionUseCase constructs a WrongQuestionUseCase.
func NewWrongQuestionUseCase(repo repository.WrongQuestionRepository, cache *cache.RedisClient) *WrongQuestionUseCase {
	return &WrongQuestionUseCase{repo: repo, cache: cache}
}

// ListRecentWrongIDs returns up to N recent wrong-question IDs from the
// Redis cache. Returns an empty slice when Redis is unavailable.
func (uc *WrongQuestionUseCase) ListRecentWrongIDs(ctx context.Context, userID int64, n int) ([]int64, error) {
	if uc.cache == nil {
		return []int64{}, nil
	}
	return uc.cache.ListRecentWrong(ctx, userID, n)
}

// Get retrieves a single wrong question by id.
func (uc *WrongQuestionUseCase) Get(ctx context.Context, id int64) (*dto.WrongQuestionResponse, error) {
	w, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, errno.New(errno.NotFound, "wrong question not found")
	}
	return toWrongQuestionResponse(w), nil
}

// ListByUser returns paginated wrong questions for a user.
func (uc *WrongQuestionUseCase) ListByUser(ctx context.Context, userID int64, p pagination.Params) (dto.ListResponse[dto.WrongQuestionResponse], error) {
	items, total, err := uc.repo.ListByUser(ctx, userID, p)
	if err != nil {
		return dto.ListResponse[dto.WrongQuestionResponse]{}, err
	}
	resp := make([]dto.WrongQuestionResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toWrongQuestionResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// ListByExam returns paginated wrong questions for a (user, exam) pair.
func (uc *WrongQuestionUseCase) ListByExam(ctx context.Context, userID, examID int64, p pagination.Params) (dto.ListResponse[dto.WrongQuestionResponse], error) {
	items, total, err := uc.repo.ListByExam(ctx, userID, examID, p)
	if err != nil {
		return dto.ListResponse[dto.WrongQuestionResponse]{}, err
	}
	resp := make([]dto.WrongQuestionResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toWrongQuestionResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// MarkMastered marks a wrong question as mastered.
func (uc *WrongQuestionUseCase) MarkMastered(ctx context.Context, id int64) (*dto.WrongQuestionResponse, error) {
	w, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, errno.New(errno.NotFound, "wrong question not found")
	}
	if err := uc.repo.MarkMastered(ctx, id); err != nil {
		return nil, err
	}
	w.IsMastered = true
	w.UpdatedAt = time.Now()
	return toWrongQuestionResponse(w), nil
}

// toWrongQuestionResponse maps the entity to its wire DTO.
func toWrongQuestionResponse(w *entity.WrongQuestion) *dto.WrongQuestionResponse {
	if w == nil {
		return nil
	}
	resp := &dto.WrongQuestionResponse{
		ID:             w.ID,
		UserID:         w.UserID,
		QuestionID:     w.QuestionID,
		ExamID:         w.ExamID,
		AttemptID:      w.AttemptID,
		UserAnswerJSON: w.UserAnswerJSON,
		WrongCount:     w.WrongCount,
		LastWrongAt:    w.LastWrongAt.Format(time.RFC3339),
		IsMastered:     w.IsMastered,
	}
	if !w.CreatedAt.IsZero() {
		resp.CreatedAt = w.CreatedAt.Format(time.RFC3339)
	}
	if !w.UpdatedAt.IsZero() {
		resp.UpdatedAt = w.UpdatedAt.Format(time.RFC3339)
	}
	return resp
}
