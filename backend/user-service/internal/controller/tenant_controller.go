package controller

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
	"tcm-history-ai/backend/pkg/response"
	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/application/usecase"
)

// TenantController exposes HTTP handlers for /api/v1/admin/tenants and the
// per-tenant member sub-resources. Every endpoint is admin-only; the admin
// role is enforced by the gateway RBAC middleware.
type TenantController struct {
	uc *usecase.TenantUseCase
}

// NewTenantController constructs a TenantController.
func NewTenantController(uc *usecase.TenantUseCase) *TenantController {
	return &TenantController{uc: uc}
}

// CreateTenant POST /api/v1/admin/tenants
func (h *TenantController) CreateTenant(ctx context.Context, c *app.RequestContext) {
	var req dto.CreateTenantRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.CreateTenant(ctx, &req)
	createdOrFail(ctx, c, resp, err)
}

// ListTenants GET /api/v1/admin/tenants
func (h *TenantController) ListTenants(ctx context.Context, c *app.RequestContext) {
	page, _ := strconv.Atoi(string(c.Query("page")))
	pageSize, _ := strconv.Atoi(string(c.Query("page_size")))
	status := string(c.Query("status"))
	resp, err := h.uc.ListTenants(ctx, pagination.From(page, pageSize), status)
	okOrFail(ctx, c, resp, err)
}

// GetTenant GET /api/v1/admin/tenants/:id
func (h *TenantController) GetTenant(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.uc.GetTenant(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// UpdateTenant PUT /api/v1/admin/tenants/:id
func (h *TenantController) UpdateTenant(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.UpdateTenantRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.UpdateTenant(ctx, id, &req)
	okOrFail(ctx, c, resp, err)
}

// DeleteTenant DELETE /api/v1/admin/tenants/:id
func (h *TenantController) DeleteTenant(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.uc.DeleteTenant(ctx, id); err != nil {
		noContentOrFail(ctx, c, err)
		return
	}
	noContentOrFail(ctx, c, nil)
}

// AddMember POST /api/v1/admin/tenants/:id/members
func (h *TenantController) AddMember(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.AddMemberRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.AddMember(ctx, id, &req)
	createdOrFail(ctx, c, resp, err)
}

// ListMembers GET /api/v1/admin/tenants/:id/members
func (h *TenantController) ListMembers(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.uc.ListMembers(ctx, id)
	okOrFail(ctx, c, resp, err)
}

// RemoveMember DELETE /api/v1/admin/tenants/:id/members/:user_id
func (h *TenantController) RemoveMember(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	userID, ok := pathMemberUserID(c)
	if !ok {
		return
	}
	if err := h.uc.RemoveMember(ctx, id, userID); err != nil {
		noContentOrFail(ctx, c, err)
		return
	}
	noContentOrFail(ctx, c, nil)
}

// ListUserTenants GET /api/v1/admin/tenants-for-user/:user_id
//
// This is a convenience lookup for admins ("which tenants does user X belong
// to?"). It is mounted outside the per-tenant sub-resource because it is
// keyed by user, not tenant.
func (h *TenantController) ListUserTenants(ctx context.Context, c *app.RequestContext) {
	userID, ok := pathMemberUserID(c)
	if !ok {
		return
	}
	resp, err := h.uc.ListUserTenants(ctx, userID)
	okOrFail(ctx, c, resp, err)
}

// pathMemberUserID extracts and validates the :user_id path parameter.
// Mirrors pathID but reads a different route param name.
func pathMemberUserID(c *app.RequestContext) (int64, bool) {
	raw := c.Param("user_id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		response.FailWith(context.Background(), c, errno.InvalidParams, "invalid user_id: "+raw)
		return 0, false
	}
	return id, true
}
