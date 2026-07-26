package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/learning-service/internal/application/dto"
	"tcm-history-ai/backend/learning-service/internal/application/usecase"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/response"
)

// StudyPlanController exposes HTTP handlers for study plans.
type StudyPlanController struct {
	uc *usecase.StudyPlanUseCase
}

// NewStudyPlanController constructs a StudyPlanController.
func NewStudyPlanController(uc *usecase.StudyPlanUseCase) *StudyPlanController {
	return &StudyPlanController{uc: uc}
}

// List GET /api/v1/learning/study-plans?user_id=&active=true
// When active=true, returns only active plans (no pagination).
func (h *StudyPlanController) List(ctx context.Context, c *app.RequestContext) {
	userID := queryUserID(c)
	if userID <= 0 {
		response.FailWith(ctx, c, errno.InvalidParams, "user_id query param is required")
		return
	}
	if string(c.Query("active")) == "true" {
		resp, err := h.uc.ListActive(ctx, userID)
		okOrFail(ctx, c, resp, err)
		return
	}
	p := pageParams(c)
	resp, err := h.uc.ListByUser(ctx, userID, p)
	okOrFail(ctx, c, resp, err)
}

// Create POST /api/v1/learning/study-plans
func (h *StudyPlanController) Create(ctx context.Context, c *app.RequestContext) {
	var req dto.StudyPlanRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Create(ctx, &req)
	createdOrFail(ctx, c, resp, err)
}

// Get GET /api/v1/learning/study-plans/:id
func (h *StudyPlanController) Get(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.uc.Get(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// Update PUT /api/v1/learning/study-plans/:id
func (h *StudyPlanController) Update(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.StudyPlanRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Update(ctx, id, &req)
	okOrFail(ctx, c, resp, err)
}

// Generate POST /api/v1/learning/study-plans/generate
// Asks the AI service to generate a study plan based on user goal.
func (h *StudyPlanController) Generate(ctx context.Context, c *app.RequestContext) {
	var req dto.StudyPlanGenerateRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	if uid := userIDFromHeader(c); uid > 0 {
		req.UserID = uid
	}
	resp, err := h.uc.Generate(ctx, &req)
	okOrFail(ctx, c, resp, err)
}

// Delete DELETE /api/v1/learning/study-plans/:id
func (h *StudyPlanController) Delete(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	err := h.uc.Delete(ctx, id)
	noContentOrFail(ctx, c, err)
}
