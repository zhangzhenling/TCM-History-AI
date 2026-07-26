package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/learning-service/internal/application/dto"
	"tcm-history-ai/backend/learning-service/internal/application/usecase"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/response"
)

// LearningRecordController exposes HTTP handlers for learning records.
type LearningRecordController struct {
	uc *usecase.LearningRecordUseCase
}

// NewLearningRecordController constructs a LearningRecordController.
func NewLearningRecordController(uc *usecase.LearningRecordUseCase) *LearningRecordController {
	return &LearningRecordController{uc: uc}
}

// Create POST /api/v1/learning/records
func (h *LearningRecordController) Create(ctx context.Context, c *app.RequestContext) {
	var req dto.LearningRecordRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Record(ctx, &req)
	createdOrFail(ctx, c, resp, err)
}

// List GET /api/v1/learning/records?user_id=&lesson_id=
// When lesson_id is provided, returns the single latest record for that pair.
// Otherwise returns paginated records for the user.
func (h *LearningRecordController) List(ctx context.Context, c *app.RequestContext) {
	userID := queryUserID(c)
	if userID <= 0 {
		response.FailWith(ctx, c, errno.InvalidParams, "user_id query param is required")
		return
	}
	lessonID := queryLessonID(c)
	if lessonID > 0 {
		resp, err := h.uc.ListByUserAndLesson(ctx, userID, lessonID)
		okOrFail(ctx, c, resp, err)
		return
	}
	p := pageParams(c)
	resp, err := h.uc.ListByUser(ctx, userID, p)
	okOrFail(ctx, c, resp, err)
}
