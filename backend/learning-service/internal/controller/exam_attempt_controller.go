package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/learning-service/internal/application/dto"
	"tcm-history-ai/backend/learning-service/internal/application/usecase"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/response"
)

// ExamAttemptController exposes HTTP handlers for exam attempts.
type ExamAttemptController struct {
	uc *usecase.ExamAttemptUseCase
}

// NewExamAttemptController constructs an ExamAttemptController.
func NewExamAttemptController(uc *usecase.ExamAttemptUseCase) *ExamAttemptController {
	return &ExamAttemptController{uc: uc}
}

// Start POST /api/v1/learning/attempts
func (h *ExamAttemptController) Start(ctx context.Context, c *app.RequestContext) {
	var req dto.ExamAttemptStartRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Start(ctx, &req)
	createdOrFail(ctx, c, resp, err)
}

// Get GET /api/v1/learning/attempts/:id
func (h *ExamAttemptController) Get(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	resp, err := h.uc.Get(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// List GET /api/v1/learning/attempts?user_id=&exam_id=
func (h *ExamAttemptController) List(ctx context.Context, c *app.RequestContext) {
	userID := queryUserID(c)
	examID := queryExamID(c)
	if userID <= 0 || examID <= 0 {
		response.FailWith(ctx, c, errno.InvalidParams, "user_id and exam_id query params are required")
		return
	}
	p := pageParams(c)
	resp, err := h.uc.ListByUserAndExam(ctx, userID, examID, p)
	okOrFail(ctx, c, resp, err)
}

// Save POST /api/v1/learning/attempts/:id/save
// Auto-save current answers without submitting the exam.
func (h *ExamAttemptController) Save(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	var req dto.ExamAttemptSaveRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.SaveAnswers(ctx, id, &req)
	okOrFail(ctx, c, resp, err)
}

// Submit POST /api/v1/learning/attempts/:id/submit
func (h *ExamAttemptController) Submit(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	var req dto.ExamAttemptSubmitRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Submit(ctx, id, &req)
	okOrFail(ctx, c, resp, err)
}
