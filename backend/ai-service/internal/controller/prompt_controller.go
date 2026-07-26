package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/ai-service/internal/application/dto"
	"tcm-history-ai/backend/ai-service/internal/application/usecase"
)

// PromptController exposes HTTP handlers for prompt templates.
type PromptController struct {
	uc *usecase.PromptUseCase
}

// NewPromptController constructs a PromptController.
func NewPromptController(uc *usecase.PromptUseCase) *PromptController {
	return &PromptController{uc: uc}
}

// List GET /api/v1/ai/prompts
func (h *PromptController) List(ctx context.Context, c *app.RequestContext) {
	scene := string(c.Query("scene"))
	p := pageParams(c)
	resp, err := h.uc.List(ctx, scene, p)
	okOrFail(ctx, c, resp, err)
}

// Create POST /api/v1/ai/prompts
func (h *PromptController) Create(ctx context.Context, c *app.RequestContext) {
	var req dto.PromptTemplateRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Create(ctx, &req)
	createdOrFail(ctx, c, resp, err)
}

// Get GET /api/v1/ai/prompts/:id
func (h *PromptController) Get(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.uc.Get(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// Update PUT /api/v1/ai/prompts/:id
func (h *PromptController) Update(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.PromptTemplateRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Update(ctx, id, &req)
	okOrFail(ctx, c, resp, err)
}

// Delete DELETE /api/v1/ai/prompts/:id
func (h *PromptController) Delete(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	err := h.uc.Delete(ctx, id)
	noContentOrFail(ctx, c, err)
}

// Activate PATCH /api/v1/ai/prompts/:id/activate
// 将指定模板置为同 scene 下唯一激活态。
func (h *PromptController) Activate(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.uc.Activate(ctx, id)
	okOrFail(ctx, c, resp, err)
}
