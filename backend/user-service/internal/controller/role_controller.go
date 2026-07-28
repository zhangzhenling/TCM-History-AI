package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/application/usecase"
)

// RoleController exposes HTTP handlers for /api/v1/admin/roles endpoints.
type RoleController struct {
	uc *usecase.RoleUseCase
}

// NewRoleController constructs a RoleController.
func NewRoleController(uc *usecase.RoleUseCase) *RoleController {
	return &RoleController{uc: uc}
}

// List GET /api/v1/admin/roles
func (h *RoleController) List(ctx context.Context, c *app.RequestContext) {
	resp, err := h.uc.ListRoles(ctx)
	okOrFail(ctx, c, resp, err)
}

// Get GET /api/v1/admin/roles/:id
func (h *RoleController) Get(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	resp, err := h.uc.GetRole(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// Create POST /api/v1/admin/roles
func (h *RoleController) Create(ctx context.Context, c *app.RequestContext) {
	var req dto.CreateRoleRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.CreateRole(ctx, &req)
	createdOrFail(ctx, c, resp, err)
}

// Update PUT /api/v1/admin/roles/:id
func (h *RoleController) Update(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	var req dto.UpdateRoleRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.UpdateRole(ctx, id, &req)
	okOrFail(ctx, c, resp, err)
}

// Delete DELETE /api/v1/admin/roles/:id
func (h *RoleController) Delete(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	if err := h.uc.DeleteRole(ctx, id); err != nil {
		noContentOrFail(ctx, c, err)
		return
	}
	noContentOrFail(ctx, c, nil)
}

// AssignPermissions PUT /api/v1/admin/roles/:id/permissions
func (h *RoleController) AssignPermissions(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(ctx, c)
	if !ok {
		return
	}
	var req dto.AssignPermissionsRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.AssignPermissions(ctx, id, req.PermissionIDs)
	okOrFail(ctx, c, resp, err)
}

// ListPermissions GET /api/v1/admin/permissions
func (h *RoleController) ListPermissions(ctx context.Context, c *app.RequestContext) {
	resp, err := h.uc.ListPermissions(ctx)
	okOrFail(ctx, c, resp, err)
}
