package usecase

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
	"tcm-history-ai/backend/user-service/internal/domain/repository"
)

type SubscriptionUseCase struct {
	planRepo   repository.MembershipPlanRepository
	subRepo    repository.UserSubscriptionRepository
	orderRepo  repository.MembershipOrderRepository
	apiKeyRepo repository.ApiKeyRepository
}

func NewSubscriptionUseCase(
	planRepo repository.MembershipPlanRepository,
	subRepo repository.UserSubscriptionRepository,
	orderRepo repository.MembershipOrderRepository,
	apiKeyRepo repository.ApiKeyRepository,
) *SubscriptionUseCase {
	return &SubscriptionUseCase{
		planRepo:   planRepo,
		subRepo:    subRepo,
		orderRepo:  orderRepo,
		apiKeyRepo: apiKeyRepo,
	}
}

func (uc *SubscriptionUseCase) GetCurrentSubscription(ctx context.Context, userID int64) (*dto.UserSubscriptionResponse, error) {
	sub, err := uc.subRepo.FindActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, nil
	}
	plan, err := uc.planRepo.FindByID(ctx, sub.PlanID)
	if err != nil {
		return nil, err
	}
	resp := toUserSubscriptionResponse(sub, plan)
	return &resp, nil
}

func (uc *SubscriptionUseCase) Subscribe(ctx context.Context, userID int64, planID int64) (*dto.MembershipOrderResponse, error) {
	plan, err := uc.planRepo.FindByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, errno.New(errno.NotFound, "plan not found: "+strconv.FormatInt(planID, 10))
	}
	if !plan.IsActive {
		return nil, errno.New(errno.ValidationFailed, "plan is not active")
	}

	orderNo := generateOrderNo(userID)
	order := &entity.MembershipOrder{
		ID:          idgen.Next(),
		UserID:      userID,
		PlanID:      planID,
		OrderNo:     orderNo,
		AmountCents: plan.PriceCents,
		Status:      entity.OrderStatusPending,
	}
	if err := uc.orderRepo.Create(ctx, order); err != nil {
		return nil, err
	}

	resp, err := uc.simulatePaymentCallback(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (uc *SubscriptionUseCase) simulatePaymentCallback(ctx context.Context, orderID int64) (*dto.MembershipOrderResponse, error) {
	order, err := uc.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, errno.New(errno.NotFound, "order not found")
	}
	if order.Status != entity.OrderStatusPending {
		return toMembershipOrderResponse(order, nil), nil
	}

	now := time.Now()
	err = uc.orderRepo.UpdateStatus(ctx, orderID, entity.OrderStatusPaid, &now, "simulated", "sim_tx_"+order.OrderNo)
	if err != nil {
		return nil, err
	}

	err = uc.activateOrExtendSubscription(ctx, order.UserID, order.PlanID)
	if err != nil {
		return nil, err
	}

	order, err = uc.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	plan, _ := uc.planRepo.FindByID(ctx, order.PlanID)
	return toMembershipOrderResponse(order, plan), nil
}

func (uc *SubscriptionUseCase) activateOrExtendSubscription(ctx context.Context, userID int64, planID int64) error {
	plan, err := uc.planRepo.FindByID(ctx, planID)
	if err != nil {
		return err
	}
	if plan == nil {
		return errno.New(errno.NotFound, "plan not found")
	}

	existing, err := uc.subRepo.FindActiveByUserID(ctx, userID)
	if err != nil {
		return err
	}

	now := time.Now()
	if existing != nil && existing.PlanID == planID {
		return uc.subRepo.Extend(ctx, existing.ID, plan.DurationDays)
	}

	if existing != nil {
		existing.Status = entity.SubscriptionStatusCancelled
		existing.CancelledAt = &now
		if err := uc.subRepo.Update(ctx, existing); err != nil {
			return err
		}
	}

	newSub := &entity.UserSubscription{
		ID:        idgen.Next(),
		UserID:    userID,
		PlanID:    planID,
		Status:    entity.SubscriptionStatusActive,
		StartedAt: now,
		ExpiresAt: now.AddDate(0, 0, plan.DurationDays),
		AutoRenew: true,
	}
	return uc.subRepo.Create(ctx, newSub)
}

func (uc *SubscriptionUseCase) CancelAutoRenew(ctx context.Context, userID int64) error {
	sub, err := uc.subRepo.FindActiveByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if sub == nil {
		return errno.New(errno.NotFound, "no active subscription")
	}
	sub.AutoRenew = false
	return uc.subRepo.Update(ctx, sub)
}

func (uc *SubscriptionUseCase) HandlePaymentCallback(ctx context.Context, req *dto.PaymentCallbackRequest) (*dto.MembershipOrderResponse, error) {
	if req == nil || req.OrderNo == "" {
		return nil, errno.New(errno.InvalidParams, "order_no is required")
	}

	order, err := uc.orderRepo.FindByOrderNo(ctx, req.OrderNo)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, errno.New(errno.NotFound, "order not found: "+req.OrderNo)
	}

	if order.Status == entity.OrderStatusPaid {
		plan, _ := uc.planRepo.FindByID(ctx, order.PlanID)
		return toMembershipOrderResponse(order, plan), nil
	}

	if req.Status != "paid" {
		return toMembershipOrderResponse(order, nil), nil
	}

	now := time.Now()
	err = uc.orderRepo.UpdateStatus(ctx, order.ID, entity.OrderStatusPaid, &now, req.PaymentMethod, req.TransactionID)
	if err != nil {
		return nil, err
	}

	err = uc.activateOrExtendSubscription(ctx, order.UserID, order.PlanID)
	if err != nil {
		return nil, err
	}

	plan, err := uc.planRepo.FindByID(ctx, order.PlanID)
	if err != nil {
		return nil, err
	}

	if plan != nil {
		err = uc.syncApiKeyQuota(ctx, order.UserID, plan)
		if err != nil {
			return nil, err
		}
	}

	order, err = uc.orderRepo.FindByID(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	return toMembershipOrderResponse(order, plan), nil
}

func (uc *SubscriptionUseCase) syncApiKeyQuota(ctx context.Context, userID int64, plan *entity.MembershipPlan) error {
	apiKeys, err := uc.apiKeyRepo.FindByUserID(ctx, userID)
	if err != nil {
		return err
	}

	for i := range apiKeys {
		key := &apiKeys[i]
		key.QuotaDaily = int64(plan.MaxDailyAIRequests)
		key.QuotaMonthly = plan.MaxTokenPerMonth
		key.RateLimitPerMinute = plan.MaxDailyAIRequests / 60
		if err := uc.apiKeyRepo.Update(ctx, key); err != nil {
			return err
		}
	}

	return nil
}

func generateOrderNo(userID int64) string {
	now := time.Now()
	return fmt.Sprintf("MB%d%s%06d", userID, now.Format("20060102150405"), idgen.Next()%1000000)
}

func toUserSubscriptionResponse(s *entity.UserSubscription, plan *entity.MembershipPlan) dto.UserSubscriptionResponse {
	resp := dto.UserSubscriptionResponse{
		ID:        s.ID,
		UserID:    s.UserID,
		PlanID:    s.PlanID,
		Status:    s.Status,
		StartedAt: s.StartedAt.Format(time.RFC3339),
		ExpiresAt: s.ExpiresAt.Format(time.RFC3339),
		AutoRenew: s.AutoRenew,
		CreatedAt: s.CreatedAt.Format(time.RFC3339),
		UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
	}
	if s.CancelledAt != nil {
		resp.CancelledAt = s.CancelledAt.Format(time.RFC3339)
	}
	if plan != nil {
		resp.PlanName = plan.Name
	}
	return resp
}
