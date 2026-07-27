package controller

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/pkg/pagination"
	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/application/usecase"
)

type MembershipController struct {
	planUC         *usecase.MembershipPlanUseCase
	subscriptionUC *usecase.SubscriptionUseCase
	orderUC        *usecase.OrderUseCase
}

func NewMembershipController(
	planUC *usecase.MembershipPlanUseCase,
	subscriptionUC *usecase.SubscriptionUseCase,
	orderUC *usecase.OrderUseCase,
) *MembershipController {
	return &MembershipController{
		planUC:         planUC,
		subscriptionUC: subscriptionUC,
		orderUC:        orderUC,
	}
}

func (h *MembershipController) ListPlans(ctx context.Context, c *app.RequestContext) {
	resp, err := h.planUC.ListPlans(ctx, false)
	okOrFail(ctx, c, resp, err)
}

func (h *MembershipController) GetSubscription(ctx context.Context, c *app.RequestContext) {
	uid, ok := requireUserID(ctx, c)
	if !ok {
		return
	}
	resp, err := h.subscriptionUC.GetCurrentSubscription(ctx, uid)
	okOrFail(ctx, c, resp, err)
}

func (h *MembershipController) Subscribe(ctx context.Context, c *app.RequestContext) {
	uid, ok := requireUserID(ctx, c)
	if !ok {
		return
	}
	var req dto.SubscribeRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.subscriptionUC.Subscribe(ctx, uid, req.PlanID)
	createdOrFail(ctx, c, resp, err)
}

func (h *MembershipController) CancelAutoRenew(ctx context.Context, c *app.RequestContext) {
	uid, ok := requireUserID(ctx, c)
	if !ok {
		return
	}
	if err := h.subscriptionUC.CancelAutoRenew(ctx, uid); err != nil {
		noContentOrFail(ctx, c, err)
		return
	}
	noContentOrFail(ctx, c, nil)
}

func (h *MembershipController) ListOrders(ctx context.Context, c *app.RequestContext) {
	uid, ok := requireUserID(ctx, c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	resp, err := h.orderUC.ListOrders(ctx, uid, pagination.From(page, pageSize))
	okOrFail(ctx, c, resp, err)
}

func (h *MembershipController) AdminListPlans(ctx context.Context, c *app.RequestContext) {
	resp, err := h.planUC.ListPlans(ctx, true)
	okOrFail(ctx, c, resp, err)
}

func (h *MembershipController) AdminGetPlan(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	resp, err := h.planUC.GetPlan(ctx, id)
	okOrFail(ctx, c, resp, err)
}

func (h *MembershipController) AdminCreatePlan(ctx context.Context, c *app.RequestContext) {
	var req dto.CreateMembershipPlanRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.planUC.CreatePlan(ctx, &req)
	createdOrFail(ctx, c, resp, err)
}

func (h *MembershipController) AdminUpdatePlan(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.UpdateMembershipPlanRequest
	if !bindAndValidate(ctx, c, &req) {
		return
	}
	resp, err := h.planUC.UpdatePlan(ctx, id, &req)
	okOrFail(ctx, c, resp, err)
}

func (h *MembershipController) AdminDeletePlan(ctx context.Context, c *app.RequestContext) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.planUC.DeletePlan(ctx, id); err != nil {
		noContentOrFail(ctx, c, err)
		return
	}
	noContentOrFail(ctx, c, nil)
}
