package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/knowledge-service/internal/application/usecase"
)

// TaskController exposes HTTP handlers for embedding tasks.
type TaskController struct {
	uc *usecase.TaskUseCase
}

// NewTaskController constructs a TaskController.
func NewTaskController(uc *usecase.TaskUseCase) *TaskController {
	return &TaskController{uc: uc}
}

// List GET /api/v1/knowledge/tasks?status=...
func (h *TaskController) List(ctx context.Context, c *app.RequestContext) {
	status := string(c.Query("status"))
	p := pageParams(c)
	resp, err := h.uc.ListByStatus(ctx, status, p)
	okOrFail(ctx, c, resp, err)
}

// Get GET /api/v1/knowledge/tasks/:id
func (h *TaskController) Get(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.uc.Get(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// ListByDocument GET /api/v1/knowledge/documents/:id/tasks
func (h *TaskController) ListByDocument(ctx context.Context, c *app.RequestContext) {
	docID, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.uc.ListByDocument(ctx, docID)
	okOrFail(ctx, c, resp, err)
}
