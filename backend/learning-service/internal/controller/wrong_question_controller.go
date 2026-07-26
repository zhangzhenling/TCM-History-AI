package controller

import (
	"context"
	"strconv"

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

// RecentIDs GET /api/v1/learning/wrong-questions/recent?user_id=&limit=
// Returns up to N recent wrong-question IDs from the cache.
func (h *WrongQuestionController) RecentIDs(ctx context.Context, c *app.RequestContext) {
	userID := queryUserID(c)
	if userID <= 0 {
		response.FailWith(ctx, c, errno.InvalidParams, "user_id query param is required")
		return
	}
	limit := 10
	if v := string(c.Query("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}
	ids, err := h.uc.ListRecentWrongIDs(ctx, userID, limit)
	if err != nil {
		response.FailWith(ctx, c, errno.InternalError, err.Error())
		return
	}
	response.OKWith(ctx, c, "ok", ids)
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
