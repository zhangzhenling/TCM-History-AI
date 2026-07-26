package controller

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/pkg/pagination"
	"tcm-history-ai/backend/pkg/response"
	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/application/usecase"
)

// AdminController exposes HTTP handlers for /api/v1/admin/users endpoints.
// These endpoints require admin-level access, enforced by the gateway RBAC
// middleware.
type AdminController struct {
	uc *usecase.AdminUseCase
}

// NewAdminController constructs an AdminController.
func NewAdminController(uc *usecase.AdminUseCase) *AdminController {
	return &AdminController{uc: uc}
}

// ListUsers GET /api/v1/admin/users
func (h *AdminController) ListUsers(ctx context.Context, c *app.RequestContext) {
	page, _ := strconv.Atoi(string(c.Query("page")))
	pageSize, _ := strconv.Atoi(string(c.Query("page_size")))
	status := string(c.Query("status"))
	p := pagination.From(page, pageSize)
	result, err := h.uc.ListUsers(ctx, p, status)
	okOrFail(ctx, c, result, err)
}

// GetUser GET /api/v1/admin/users/:id
func (h *AdminController) GetUser(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.uc.GetUser(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// UpdateStatus PATCH /api/v1/admin/users/:id/status
func (h *AdminController) UpdateStatus(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.UpdateUserStatusRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	if err := h.uc.UpdateUserStatus(ctx, id, req.Status); err != nil {
		response.Fail(ctx, c, err)
		return
	}
	response.OK(ctx, c, map[string]any{"id": id, "status": req.Status})
}

// AssignRoles PUT /api/v1/admin/users/:id/roles
func (h *AdminController) AssignRoles(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.AssignRolesRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.AssignUserRoles(ctx, id, req.RoleIDs)
	okOrFail(ctx, c, resp, err)
}
