package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/application/usecase"
)

// AuthController exposes HTTP handlers for /api/v1/auth/* endpoints.
type AuthController struct {
	uc *usecase.AuthUseCase
}

// NewAuthController constructs an AuthController.
func NewAuthController(uc *usecase.AuthUseCase) *AuthController {
	return &AuthController{uc: uc}
}

// Register POST /api/v1/auth/register
//
// Body: dto.RegisterRequest. On success returns 201 with a fresh token pair.
func (h *AuthController) Register(ctx context.Context, c *app.RequestContext) {
	var req dto.RegisterRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Register(ctx, &req)
	createdOrFail(ctx, c, resp, err)
}

// Login POST /api/v1/auth/login
//
// Body: dto.LoginRequest. On success returns 200 with a fresh token pair.
// The caller's IP (from X-Forwarded-For or RemoteAddr) is recorded against
// the user's last_login_at / last_login_ip for audit.
func (h *AuthController) Login(ctx context.Context, c *app.RequestContext) {
	var req dto.LoginRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	ip := clientIP(c)
	resp, err := h.uc.Login(ctx, &req, ip)
	okOrFail(ctx, c, resp, err)
}

// Refresh POST /api/v1/auth/refresh
//
// Body: dto.RefreshRequest. On success returns 200 with a rotated token pair.
func (h *AuthController) Refresh(ctx context.Context, c *app.RequestContext) {
	var req dto.RefreshRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Refresh(ctx, &req)
	okOrFail(ctx, c, resp, err)
}
