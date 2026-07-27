package usecase

import (
	"context"
	"strconv"
	"time"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
	"tcm-history-ai/backend/user-service/internal/domain/repository"
)

type OrderUseCase struct {
	orderRepo repository.MembershipOrderRepository
	planRepo  repository.MembershipPlanRepository
}

func NewOrderUseCase(
	orderRepo repository.MembershipOrderRepository,
	planRepo repository.MembershipPlanRepository,
) *OrderUseCase {
	return &OrderUseCase{
		orderRepo: orderRepo,
		planRepo:  planRepo,
	}
}

func (uc *OrderUseCase) ListOrders(ctx context.Context, userID int64, p pagination.Params) (*pagination.Page[dto.MembershipOrderResponse], error) {
	page, pageSize, _ := p.Normalise()
	orders, total, err := uc.orderRepo.FindByUserID(ctx, userID, pagination.From(page, pageSize))
	if err != nil {
		return nil, err
	}
	items := make([]dto.MembershipOrderResponse, 0, len(orders))
	planCache := make(map[int64]string)
	for _, o := range orders {
		planName := ""
		if name, ok := planCache[o.PlanID]; ok {
			planName = name
		} else {
			plan, _ := uc.planRepo.FindByID(ctx, o.PlanID)
			if plan != nil {
				planName = plan.Name
				planCache[o.PlanID] = planName
			}
		}
		items = append(items, *toMembershipOrderResponseWithPlan(&o, planName))
	}
	result := pagination.NewPage(page, pageSize, int(total), items)
	return &result, nil
}

func (uc *OrderUseCase) GetOrder(ctx context.Context, userID int64, orderID int64) (*dto.MembershipOrderResponse, error) {
	order, err := uc.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, errno.New(errno.NotFound, "order not found: "+strconv.FormatInt(orderID, 10))
	}
	if order.UserID != userID {
		return nil, errno.New(errno.Forbidden, "order not belongs to user")
	}
	plan, _ := uc.planRepo.FindByID(ctx, order.PlanID)
	return toMembershipOrderResponse(order, plan), nil
}

func toMembershipOrderResponse(o *entity.MembershipOrder, plan *entity.MembershipPlan) *dto.MembershipOrderResponse {
	planName := ""
	if plan != nil {
		planName = plan.Name
	}
	return toMembershipOrderResponseWithPlan(o, planName)
}

func toMembershipOrderResponseWithPlan(o *entity.MembershipOrder, planName string) *dto.MembershipOrderResponse {
	resp := &dto.MembershipOrderResponse{
		ID:            o.ID,
		UserID:        o.UserID,
		PlanID:        o.PlanID,
		PlanName:      planName,
		OrderNo:       o.OrderNo,
		AmountCents:   o.AmountCents,
		Status:        o.Status,
		PaymentMethod: o.PaymentMethod,
		TransactionID: o.TransactionID,
		CreatedAt:     o.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     o.UpdatedAt.Format(time.RFC3339),
	}
	if o.PaidAt != nil {
		resp.PaidAt = o.PaidAt.Format(time.RFC3339)
	}
	return resp
}
