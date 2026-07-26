package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/ai-service/internal/application/dto"
	"tcm-history-ai/backend/ai-service/internal/application/usecase"
)

// AgentController exposes HTTP handlers for Agent runs.
type AgentController struct {
	uc *usecase.AgentUseCase
}

// NewAgentController constructs an AgentController.
func NewAgentController(uc *usecase.AgentUseCase) *AgentController {
	return &AgentController{uc: uc}
}

// Run POST /api/v1/ai/agents/run
func (h *AgentController) Run(ctx context.Context, c *app.RequestContext) {
	var req dto.AgentRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	if uid := userIDFromHeader(c); uid > 0 {
		req.UserID = uid
	}
	resp, err := h.uc.Run(ctx, &req)
	okOrFail(ctx, c, resp, err)
}

// ListAgentRuns GET /api/v1/ai/agent-runs
func (h *AgentController) ListAgentRuns(ctx context.Context, c *app.RequestContext) {
	p := pageParams(c)
	resp, err := h.uc.ListAgentRuns(ctx, p)
	okOrFail(ctx, c, resp, err)
}

// GetAgentRun GET /api/v1/ai/agent-runs/:id
func (h *AgentController) GetAgentRun(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.uc.GetAgentRun(ctx, id)
	okOrFail(ctx, c, resp, err)
}
