package usecase

import (
	"context"
	"strconv"
	"time"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
	"tcm-history-ai/backend/user-service/internal/domain/repository"
)

type MembershipPlanUseCase struct {
	planRepo repository.MembershipPlanRepository
}

func NewMembershipPlanUseCase(planRepo repository.MembershipPlanRepository) *MembershipPlanUseCase {
	return &MembershipPlanUseCase{
		planRepo: planRepo,
	}
}

func (uc *MembershipPlanUseCase) ListPlans(ctx context.Context, includeInactive bool) ([]dto.MembershipPlanResponse, error) {
	var plans []entity.MembershipPlan
	var err error
	if includeInactive {
		plans, err = uc.planRepo.ListAll(ctx)
	} else {
		plans, err = uc.planRepo.ListActive(ctx)
	}
	if err != nil {
		return nil, err
	}
	out := make([]dto.MembershipPlanResponse, 0, len(plans))
	for _, p := range plans {
		out = append(out, toMembershipPlanResponse(&p))
	}
	return out, nil
}

func (uc *MembershipPlanUseCase) GetPlan(ctx context.Context, id int64) (*dto.MembershipPlanResponse, error) {
	plan, err := uc.planRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, errno.New(errno.NotFound, "plan not found: "+strconv.FormatInt(id, 10))
	}
	resp := toMembershipPlanResponse(plan)
	return &resp, nil
}

func (uc *MembershipPlanUseCase) CreatePlan(ctx context.Context, in *dto.CreateMembershipPlanRequest) (*dto.MembershipPlanResponse, error) {
	if in == nil || in.Name == "" {
		return nil, errno.New(errno.InvalidParams, "name is required")
	}
	if in.PriceCents < 0 {
		return nil, errno.New(errno.InvalidParams, "price_cents must be >= 0")
	}
	if in.DurationDays <= 0 {
		return nil, errno.New(errno.InvalidParams, "duration_days must be > 0")
	}
	plan := &entity.MembershipPlan{
		ID:                 idgen.Next(),
		Name:               in.Name,
		PriceCents:         in.PriceCents,
		DurationDays:       in.DurationDays,
		MaxDailyAIRequests: in.MaxDailyAIRequests,
		MaxTokenPerMonth:   in.MaxTokenPerMonth,
		Features:           in.Features,
		IsActive:           in.IsActive,
		SortOrder:          in.SortOrder,
	}
	if err := uc.planRepo.Create(ctx, plan); err != nil {
		return nil, err
	}
	resp := toMembershipPlanResponse(plan)
	return &resp, nil
}

func (uc *MembershipPlanUseCase) UpdatePlan(ctx context.Context, id int64, in *dto.UpdateMembershipPlanRequest) (*dto.MembershipPlanResponse, error) {
	plan, err := uc.planRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, errno.New(errno.NotFound, "plan not found")
	}
	if in.Name != nil {
		plan.Name = *in.Name
	}
	if in.PriceCents != nil {
		if *in.PriceCents < 0 {
			return nil, errno.New(errno.InvalidParams, "price_cents must be >= 0")
		}
		plan.PriceCents = *in.PriceCents
	}
	if in.DurationDays != nil {
		if *in.DurationDays <= 0 {
			return nil, errno.New(errno.InvalidParams, "duration_days must be > 0")
		}
		plan.DurationDays = *in.DurationDays
	}
	if in.MaxDailyAIRequests != nil {
		plan.MaxDailyAIRequests = *in.MaxDailyAIRequests
	}
	if in.MaxTokenPerMonth != nil {
		plan.MaxTokenPerMonth = *in.MaxTokenPerMonth
	}
	if in.Features != nil {
		plan.Features = *in.Features
	}
	if in.IsActive != nil {
		plan.IsActive = *in.IsActive
	}
	if in.SortOrder != nil {
		plan.SortOrder = *in.SortOrder
	}
	if err := uc.planRepo.Update(ctx, plan); err != nil {
		return nil, err
	}
	resp := toMembershipPlanResponse(plan)
	return &resp, nil
}

func (uc *MembershipPlanUseCase) DeletePlan(ctx context.Context, id int64) error {
	plan, err := uc.planRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if plan == nil {
		return errno.New(errno.NotFound, "plan not found")
	}
	return uc.planRepo.Delete(ctx, id)
}

func toMembershipPlanResponse(p *entity.MembershipPlan) dto.MembershipPlanResponse {
	return dto.MembershipPlanResponse{
		ID:                 p.ID,
		Name:               p.Name,
		PriceCents:         p.PriceCents,
		DurationDays:       p.DurationDays,
		MaxDailyAIRequests: p.MaxDailyAIRequests,
		MaxTokenPerMonth:   p.MaxTokenPerMonth,
		Features:           p.Features,
		IsActive:           p.IsActive,
		SortOrder:          p.SortOrder,
		CreatedAt:          p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          p.UpdatedAt.Format(time.RFC3339),
	}
}
