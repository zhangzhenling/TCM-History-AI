package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/learning-service/internal/application/usecase"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/response"
)

// WrongQuestionController exposes HTTP handlers for the wrong-question book.
type WrongQuestionController struct {
	uc *usecase.WrongQuestionUseCase
}

// NewWrongQuestionController constructs a WrongQuestionController.
func NewWrongQuestionController(uc *usecase.WrongQuestionUseCase) *WrongQuestionController {
	return &WrongQuestionController{uc: uc}
}

// List GET /api/v1/learning/wrong-questions?user_id=&exam_id=&page=&page_size=
func (h *WrongQuestionController) List(ctx context.Context, c *app.RequestContext) {
	userID := queryUserID(c)
	if userID <= 0 {
		response.FailWith(ctx, c, errno.InvalidParams, "user_id query param is required")
		return
	}
	examID := queryExamID(c)
	p := pageParams(c)
	if examID > 0 {
		resp, err := h.uc.ListByExam(ctx, userID, examID, p)
		okOrFail(ctx, c, resp, err)
		return
	}
	resp, err := h.uc.ListByUser(ctx, userID, p)
	okOrFail(ctx, c, resp, err)
}

// MarkMastered PUT /api/v1/learning/wrong-questions/:id/master
func (h *WrongQuestionController) MarkMastered(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.uc.MarkMastered(ctx, id)
	okOrFail(ctx, c, resp, err)
}
