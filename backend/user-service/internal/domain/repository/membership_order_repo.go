package repository

import (
	"context"
	"time"

	"tcm-history-ai/backend/pkg/pagination"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

type MembershipOrderRepository interface {
	Create(ctx context.Context, order *entity.MembershipOrder) error
	FindByID(ctx context.Context, id int64) (*entity.MembershipOrder, error)
	FindByUserID(ctx context.Context, userID int64, p pagination.Params) ([]entity.MembershipOrder, int64, error)
	FindByOrderNo(ctx context.Context, orderNo string) (*entity.MembershipOrder, error)
	UpdateStatus(ctx context.Context, id int64, status string, paidAt *time.Time, paymentMethod, transactionID string) error
}
