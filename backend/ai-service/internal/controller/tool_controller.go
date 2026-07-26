package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/ai-service/internal/application/dto"
	"tcm-history-ai/backend/ai-service/internal/application/usecase"
)

// ToolController exposes HTTP handlers for MCP tools.
type ToolController struct {
	uc *usecase.ToolUseCase
}

// NewToolController constructs a ToolController.
func NewToolController(uc *usecase.ToolUseCase) *ToolController {
	return &ToolController{uc: uc}
}

// List GET /api/v1/ai/tools?enabled=true
func (h *ToolController) List(ctx context.Context, c *app.RequestContext) {
	enabled := string(c.Query("enabled")) == "true"
	p := pageParams(c)
	resp, err := h.uc.List(ctx, enabled, p)
	okOrFail(ctx, c, resp, err)
}

// Create POST /api/v1/ai/tools
func (h *ToolController) Create(ctx context.Context, c *app.RequestContext) {
	var req dto.ToolRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Create(ctx, &req)
	createdOrFail(ctx, c, resp, err)
}

// Update PUT /api/v1/ai/tools/:id
func (h *ToolController) Update(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.ToolRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Update(ctx, id, &req)
	okOrFail(ctx, c, resp, err)
}

// Delete DELETE /api/v1/ai/tools/:id
func (h *ToolController) Delete(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	err := h.uc.Delete(ctx, id)
	noContentOrFail(ctx, c, err)
}

// Execute POST /api/v1/ai/tools/:id/execute
func (h *ToolController) Execute(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.ToolExecuteRequest
	// params 可选，空 body 也允许
	_ = c.BindJSON(&req)
	resp, err := h.uc.Execute(ctx, id, req.Params)
	okOrFail(ctx, c, resp, err)
}
