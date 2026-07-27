package controller

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/application/usecase"
)

type ApiKeyController struct {
	uc *usecase.ApiKeyUseCase
}

func NewApiKeyController(uc *usecase.ApiKeyUseCase) *ApiKeyController {
	return &ApiKeyController{uc: uc}
}

func (h *ApiKeyController) List(ctx context.Context, c *app.RequestContext) {
	userID, ok := requireUserID(ctx, c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(string(c.Query("page")))
	pageSize, _ := strconv.Atoi(string(c.Query("page_size")))
	resp, err := h.uc.List(ctx, userID, page, pageSize)
	okOrFail(ctx, c, resp, err)
}

func (h *ApiKeyController) Create(ctx context.Context, c *app.RequestContext) {
	userID, ok := requireUserID(ctx, c)
	if !ok {
		return
	}
	var req dto.CreateApiKeyRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Create(ctx, userID, &req)
	createdOrFail(ctx, c, resp, err)
}

func (h *ApiKeyController) Get(ctx context.Context, c *app.RequestContext) {
	userID, ok := requireUserID(ctx, c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.uc.Get(ctx, userID, id)
	okOrFail(ctx, c, resp, err)
}

func (h *ApiKeyController) Update(ctx context.Context, c *app.RequestContext) {
	userID, ok := requireUserID(ctx, c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.UpdateApiKeyRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.uc.Update(ctx, userID, id, &req)
	okOrFail(ctx, c, resp, err)
}

func (h *ApiKeyController) Delete(ctx context.Context, c *app.RequestContext) {
	userID, ok := requireUserID(ctx, c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.uc.Delete(ctx, userID, id); err != nil {
		noContentOrFail(ctx, c, err)
		return
	}
	noContentOrFail(ctx, c, nil)
}

func (h *ApiKeyController) Regenerate(ctx context.Context, c *app.RequestContext) {
	userID, ok := requireUserID(ctx, c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.uc.Regenerate(ctx, userID, id)
	okOrFail(ctx, c, resp, err)
}
