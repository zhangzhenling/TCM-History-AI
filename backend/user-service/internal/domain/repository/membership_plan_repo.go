package repository

import (
	"context"

	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

type MembershipPlanRepository interface {
	ListAll(ctx context.Context) ([]entity.MembershipPlan, error)
	ListActive(ctx context.Context) ([]entity.MembershipPlan, error)
	FindByID(ctx context.Context, id int64) (*entity.MembershipPlan, error)
	Create(ctx context.Context, plan *entity.MembershipPlan) error
	Update(ctx context.Context, plan *entity.MembershipPlan) error
	Delete(ctx context.Context, id int64) error
}
