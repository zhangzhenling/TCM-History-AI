package controller

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

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

// List GET /api/v1/graph/nodes?label=&page=&page_size=
func (h *NodeController) List(ctx context.Context, c *app.RequestContext) {
	label := queryString(c, "label")
	p := pageParams(c)
	resp, err := h.uc.List(ctx, label, p)
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
	uid, ok := pathUID(c)
	if !ok {
		return
	}
	resp, err := h.uc.Get(ctx, uid)
	okOrFail(ctx, c, resp, err)
}

// Delete DELETE /api/v1/graph/nodes/:uid
func (h *NodeController) Delete(ctx context.Context, c *app.RequestContext) {
	uid, ok := pathUID(c)
	if !ok {
		return
	}
	err := h.uc.Delete(ctx, uid)
	noContentOrFail(ctx, c, err)
}

// Search GET /api/v1/graph/nodes/search?keyword=&label=&limit=
func (h *NodeController) Search(ctx context.Context, c *app.RequestContext) {
	keyword := queryString(c, "keyword")
	label := queryString(c, "label")
	limit := queryInt(c, "limit", 20)
	req := &dto.SearchNodesRequest{Keyword: keyword, Label: label, Limit: limit}
	resp, err := h.uc.Search(ctx, req)
	okOrFail(ctx, c, resp, err)
}

// Suppress unused-import warning for strconv when only consts is used elsewhere.
var _ = strconv.Itoa
var _ = consts.StatusOK
