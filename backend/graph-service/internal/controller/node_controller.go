package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/graph-service/internal/application/dto"
	"tcm-history-ai/backend/graph-service/internal/application/usecase"
)

// NodeController exposes HTTP handlers for graph nodes.
type NodeController struct {
	uc *usecase.NodeUseCase
}

// NewNodeController constructs a NodeController.
func NewNodeController(uc *usecase.NodeUseCase) *NodeController {
	return &NodeController{uc: uc}
}

// List GET /api/v1/graph/nodes?label=&keyword=&page=&page_size=
// 当 keyword 非空时按名称模糊检索，否则按 label 列出。
func (h *NodeController) List(ctx context.Context, c *app.RequestContext) {
	label := queryString(c, "label")
	keyword := queryString(c, "keyword")
	p := pageParams(c)
	resp, err := h.uc.List(ctx, label, keyword, p)
	okOrFail(ctx, c, resp, err)
}

// Create POST /api/v1/graph/nodes (MERGE semantics by uid)
func (h *NodeController) Create(ctx context.Context, c *app.RequestContext) {
	var req dto.NodeRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Create(ctx, &req)
	createdOrFail(ctx, c, resp, err)
}

// Get GET /api/v1/graph/nodes/:uid
func (h *NodeController) Get(ctx context.Context, c *app.RequestContext) {
	uid, ok := pathUID(ctx, c)
	if !ok {
		return
	}
	resp, err := h.uc.Get(ctx, uid)
	okOrFail(ctx, c, resp, err)
}

// Update PUT /api/v1/graph/nodes/:uid
func (h *NodeController) Update(ctx context.Context, c *app.RequestContext) {
	uid, ok := pathUID(ctx, c)
	if !ok {
		return
	}
	var req dto.NodeRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Update(ctx, uid, &req)
	okOrFail(ctx, c, resp, err)
}

// Delete DELETE /api/v1/graph/nodes/:uid
func (h *NodeController) Delete(ctx context.Context, c *app.RequestContext) {
	uid, ok := pathUID(ctx, c)
	if !ok {
		return
	}
	err := h.uc.Delete(ctx, uid)
	noContentOrFail(ctx, c, err)
}
