package persistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
	"tcm-history-ai/backend/user-service/internal/domain/repository"
)

type MembershipOrderRepo struct {
	baseRepo
}

func NewMembershipOrderRepo(db *gorm.DB) *MembershipOrderRepo {
	return &MembershipOrderRepo{baseRepo{db: db}}
}

var _ repository.MembershipOrderRepository = (*MembershipOrderRepo)(nil)

func (r *MembershipOrderRepo) Create(ctx context.Context, order *entity.MembershipOrder) error {
	if err := txFrom(ctx, r.db).Create(order).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create membership order", err)
	}
	return nil
}

func (r *MembershipOrderRepo) FindByID(ctx context.Context, id int64) (*entity.MembershipOrder, error) {
	var order entity.MembershipOrder
	err := txFrom(ctx, r.db).First(&order, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find membership order by id", err)
	}
	return &order, nil
}

func (r *MembershipOrderRepo) FindByUserID(ctx context.Context, userID int64, p pagination.Params) ([]entity.MembershipOrder, int64, error) {
	_, pageSize, offset := p.Normalise()
	var orders []entity.MembershipOrder
	var total int64

	db := txFrom(ctx, r.db).Model(&entity.MembershipOrder{}).Where("user_id = ?", userID)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count membership orders", err)
	}

	if err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&orders).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "find membership orders by user id", err)
	}
	return orders, total, nil
}

func (r *MembershipOrderRepo) FindByOrderNo(ctx context.Context, orderNo string) (*entity.MembershipOrder, error) {
	var order entity.MembershipOrder
	err := txFrom(ctx, r.db).First(&order, "order_no = ?", orderNo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find membership order by order no", err)
	}
	return &order, nil
}

func (r *MembershipOrderRepo) UpdateStatus(ctx context.Context, id int64, status string, paidAt *time.Time, paymentMethod, transactionID string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if paidAt != nil {
		updates["paid_at"] = paidAt
	}
	if paymentMethod != "" {
		updates["payment_method"] = paymentMethod
	}
	if transactionID != "" {
		updates["transaction_id"] = transactionID
	}
	res := txFrom(ctx, r.db).Model(&entity.MembershipOrder{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "update membership order status", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "membership order not found")
	}
	return nil
}
