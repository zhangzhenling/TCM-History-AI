package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/graph-service/internal/application/dto"
	"tcm-history-ai/backend/graph-service/internal/application/usecase"
)

// RelationshipController exposes HTTP handlers for graph relationships.
type RelationshipController struct {
	uc *usecase.RelationshipUseCase
}

// NewRelationshipController constructs a RelationshipController.
func NewRelationshipController(uc *usecase.RelationshipUseCase) *RelationshipController {
	return &RelationshipController{uc: uc}
}

// Create POST /api/v1/graph/relationships
func (h *RelationshipController) Create(ctx context.Context, c *app.RequestContext) {
	var req dto.RelationshipRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Create(ctx, &req)
	createdOrFail(ctx, c, resp, err)
}

// Get GET /api/v1/graph/relationships/:uid
func (h *RelationshipController) Get(ctx context.Context, c *app.RequestContext) {
	uid, ok := pathUID(c)
	if !ok {
		return
	}
	resp, err := h.uc.Get(ctx, uid)
	okOrFail(ctx, c, resp, err)
}

// Delete DELETE /api/v1/graph/relationships/:uid
func (h *RelationshipController) Delete(ctx context.Context, c *app.RequestContext) {
	uid, ok := pathUID(c)
	if !ok {
		return
	}
	err := h.uc.Delete(ctx, uid)
	noContentOrFail(ctx, c, err)
}
