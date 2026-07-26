package usecase

import (
	"context"
	"strconv"
	"time"

	"tcm-history-ai/backend/learning-service/internal/application/dto"
	"tcm-history-ai/backend/learning-service/internal/domain/entity"
	"tcm-history-ai/backend/learning-service/internal/domain/repository"
	"tcm-history-ai/backend/learning-service/internal/infrastructure/cache"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/logger"
	"tcm-history-ai/backend/pkg/pagination"
	"go.uber.org/zap"
)

// LearningRecordUseCase implements learning record operations.
type LearningRecordUseCase struct {
	recordRepo repository.LearningRecordRepository
	cache      *cache.RedisClient
}

// NewLearningRecordUseCase constructs a LearningRecordUseCase.
func NewLearningRecordUseCase(recordRepo repository.LearningRecordRepository, cache *cache.RedisClient) *LearningRecordUseCase {
	return &LearningRecordUseCase{recordRepo: recordRepo, cache: cache}
}

// Record upserts a learning record for the (user, lesson) pair. On existing
// rows it accumulates duration and updates position; on completion it marks
// the record as completed. The progress is also cached in Redis for fast
// "last learned position" lookups.
func (uc *LearningRecordUseCase) Record(ctx context.Context, in *dto.LearningRecordRequest) (*dto.LearningRecordResponse, error) {
	if in == nil || in.UserID <= 0 || in.LessonID <= 0 || in.CourseID <= 0 {
		return nil, errno.New(errno.InvalidParams, "user_id, lesson_id and course_id are required")
	}
	now := time.Now()
	existing, err := uc.recordRepo.FindByUserAndLesson(ctx, in.UserID, in.LessonID)
	if err != nil {
		return nil, err
	}
	var resp *dto.LearningRecordResponse
	if existing == nil {
		r := &entity.LearningRecord{
			UserID:          in.UserID,
			LessonID:        in.LessonID,
			CourseID:        in.CourseID,
			DurationSeconds: in.DurationSeconds,
			PositionPercent: in.PositionPercent,
			IsCompleted:     in.IsCompleted,
			LastPosition:    in.LastPosition,
			LearnedAt:       now,
		}
		r.ID = idgen.Next()
		if err := uc.recordRepo.Upsert(ctx, r); err != nil {
			return nil, err
		}
		resp = toLearningRecordResponse(r)
	} else {
		existing.DurationSeconds += in.DurationSeconds
		if in.PositionPercent > existing.PositionPercent {
			existing.PositionPercent = in.PositionPercent
		}
		if in.LastPosition > existing.LastPosition {
			existing.LastPosition = in.LastPosition
		}
		if in.IsCompleted && !existing.IsCompleted {
			existing.IsCompleted = true
		}
		existing.LearnedAt = now
		if err := uc.recordRepo.Upsert(ctx, existing); err != nil {
			return nil, err
		}
		resp = toLearningRecordResponse(existing)
	}
	if uc.cache != nil && resp != nil {
		if err := uc.cache.SetProgress(ctx, in.UserID, in.LessonID, resp.PositionPercent); err != nil {
			logger.Default().Warn("cache set progress failed",
				zap.Int64("user_id", in.UserID),
				zap.Int64("lesson_id", in.LessonID),
				zap.Error(err))
		}
	}
	return resp, nil
}

// Get retrieves a single learning record by id.
func (uc *LearningRecordUseCase) Get(ctx context.Context, id int64) (*dto.LearningRecordResponse, error) {
	r, err := uc.recordRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, errno.New(errno.NotFound, "record not found")
	}
	return toLearningRecordResponse(r), nil
}

// ListByUser returns paginated learning records for a user.
func (uc *LearningRecordUseCase) ListByUser(ctx context.Context, userID int64, p pagination.Params) (dto.ListResponse[dto.LearningRecordResponse], error) {
	items, total, err := uc.recordRepo.ListByUser(ctx, userID, p)
	if err != nil {
		return dto.ListResponse[dto.LearningRecordResponse]{}, err
	}
	resp := make([]dto.LearningRecordResponse, 0, len(items))
	for i := range items {
		resp = append(resp, *toLearningRecordResponse(&items[i]))
	}
	return dto.NewListResponse(p, total, resp), nil
}

// ListByUserAndLesson returns the latest record for a (user, lesson) pair,
// or 404 when absent. The PositionPercent is first read from Redis cache;
// on cache miss it falls back to the database value.
func (uc *LearningRecordUseCase) ListByUserAndLesson(ctx context.Context, userID, lessonID int64) (*dto.LearningRecordResponse, error) {
	r, err := uc.recordRepo.FindByUserAndLesson(ctx, userID, lessonID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, errno.New(errno.NotFound, "record not found for user "+strconv.FormatInt(userID, 10)+" lesson "+strconv.FormatInt(lessonID, 10))
	}
	resp := toLearningRecordResponse(r)
	if uc.cache != nil {
		if cached, err := uc.cache.GetProgress(ctx, userID, lessonID); err == nil && cached > 0 && cached > resp.PositionPercent {
			resp.PositionPercent = cached
		}
	}
	return resp, nil
}

// MarkCompleted marks a learning record as completed by id.
func (uc *LearningRecordUseCase) MarkCompleted(ctx context.Context, id int64) error {
	return uc.recordRepo.MarkCompleted(ctx, id)
}

// toLearningRecordResponse maps the entity to its wire DTO.
func toLearningRecordResponse(r *entity.LearningRecord) *dto.LearningRecordResponse {
	if r == nil {
		return nil
	}
	resp := &dto.LearningRecordResponse{
		ID:              r.ID,
		UserID:          r.UserID,
		LessonID:        r.LessonID,
		CourseID:        r.CourseID,
		DurationSeconds: r.DurationSeconds,
		PositionPercent: r.PositionPercent,
		IsCompleted:     r.IsCompleted,
		LastPosition:    r.LastPosition,
		LearnedAt:       r.LearnedAt,
	}
	if !r.CreatedAt.IsZero() {
		resp.CreatedAt = r.CreatedAt.Format(time.RFC3339)
	}
	if !r.UpdatedAt.IsZero() {
		resp.UpdatedAt = r.UpdatedAt.Format(time.RFC3339)
	}
	return resp
}
