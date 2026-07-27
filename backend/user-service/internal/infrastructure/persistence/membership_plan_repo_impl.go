package persistence

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
	"tcm-history-ai/backend/user-service/internal/domain/repository"
)

type MembershipPlanRepo struct {
	baseRepo
}

func NewMembershipPlanRepo(db *gorm.DB) *MembershipPlanRepo {
	return &MembershipPlanRepo{baseRepo{db: db}}
}

var _ repository.MembershipPlanRepository = (*MembershipPlanRepo)(nil)

func (r *MembershipPlanRepo) ListAll(ctx context.Context) ([]entity.MembershipPlan, error) {
	var plans []entity.MembershipPlan
	if err := txFrom(ctx, r.db).Order("sort_order ASC, id ASC").Find(&plans).Error; err != nil {
		return nil, errno.Wrap(errno.InternalError, "list membership plans", err)
	}
	return plans, nil
}

func (r *MembershipPlanRepo) ListActive(ctx context.Context) ([]entity.MembershipPlan, error) {
	var plans []entity.MembershipPlan
	if err := txFrom(ctx, r.db).Where("is_active = ?", true).Order("sort_order ASC, id ASC").Find(&plans).Error; err != nil {
		return nil, errno.Wrap(errno.InternalError, "list active membership plans", err)
	}
	return plans, nil
}

func (r *MembershipPlanRepo) FindByID(ctx context.Context, id int64) (*entity.MembershipPlan, error) {
	var plan entity.MembershipPlan
	err := txFrom(ctx, r.db).First(&plan, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find membership plan by id", err)
	}
	return &plan, nil
}

func (r *MembershipPlanRepo) Create(ctx context.Context, plan *entity.MembershipPlan) error {
	if err := txFrom(ctx, r.db).Create(plan).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create membership plan", err)
	}
	return nil
}

func (r *MembershipPlanRepo) Update(ctx context.Context, plan *entity.MembershipPlan) error {
	if err := txFrom(ctx, r.db).Save(plan).Error; err != nil {
		return errno.Wrap(errno.InternalError, "update membership plan", err)
	}
	return nil
}

func (r *MembershipPlanRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Where("id = ?", id).Delete(&entity.MembershipPlan{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete membership plan", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "membership plan not found")
	}
	return nil
}
