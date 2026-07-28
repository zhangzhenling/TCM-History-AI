package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/graph-service/internal/application/dto"
	"tcm-history-ai/backend/graph-service/internal/application/usecase"
)

// EdgeController exposes HTTP handlers for graph edges.
type EdgeController struct {
	uc *usecase.EdgeUseCase
}

// NewEdgeController constructs an EdgeController.
func NewEdgeController(uc *usecase.EdgeUseCase) *EdgeController {
	return &EdgeController{uc: uc}
}

// List GET /api/v1/graph/edges?source_uid=&target_uid=&type=&page=&page_size=
// 至少应提供一种过滤条件以约束结果集；当全部为空时按 created_at 倒序分页。
func (h *EdgeController) List(ctx context.Context, c *app.RequestContext) {
	sourceUID := queryString(c, "source_uid")
	targetUID := queryString(c, "target_uid")
	edgeType := queryString(c, "type")
	p := pageParams(c)
	resp, err := h.uc.List(ctx, sourceUID, targetUID, edgeType, p)
	okOrFail(ctx, c, resp, err)
}

// Create POST /api/v1/graph/edges (MERGE semantics by uid)
func (h *EdgeController) Create(ctx context.Context, c *app.RequestContext) {
	var req dto.EdgeRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Create(ctx, &req)
	createdOrFail(ctx, c, resp, err)
}

// Get GET /api/v1/graph/edges/:uid
func (h *EdgeController) Get(ctx context.Context, c *app.RequestContext) {
	uid, ok := pathUID(ctx, c)
	if !ok {
		return
	}
	resp, err := h.uc.Get(ctx, uid)
	okOrFail(ctx, c, resp, err)
}

// Update PUT /api/v1/graph/edges/:uid
func (h *EdgeController) Update(ctx context.Context, c *app.RequestContext) {
	uid, ok := pathUID(ctx, c)
	if !ok {
		return
	}
	var req dto.EdgeRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Update(ctx, uid, &req)
	okOrFail(ctx, c, resp, err)
}

// Delete DELETE /api/v1/graph/edges/:uid
func (h *EdgeController) Delete(ctx context.Context, c *app.RequestContext) {
	uid, ok := pathUID(ctx, c)
	if !ok {
		return
	}
	err := h.uc.Delete(ctx, uid)
	noContentOrFail(ctx, c, err)
}
