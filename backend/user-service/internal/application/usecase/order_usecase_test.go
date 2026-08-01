package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
	"tcm-history-ai/backend/user-service/internal/application/usecase"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// ---------------------------------------------------------------------------
// MembershipOrderRepository mock
// ---------------------------------------------------------------------------

type mockOrderRepo struct {
	items      map[int64]*entity.MembershipOrder
	create     func(*entity.MembershipOrder) error
	findByID   func(int64) (*entity.MembershipOrder, error)
	findByUser func(int64, pagination.Params) ([]entity.MembershipOrder, int64, error)
	findByNo   func(string) (*entity.MembershipOrder, error)
	updateStatus func(int64, string, *time.Time, string, string) error
}

func newMockOrderRepo() *mockOrderRepo {
	return &mockOrderRepo{items: map[int64]*entity.MembershipOrder{}}
}

func (m *mockOrderRepo) Create(_ context.Context, o *entity.MembershipOrder) error {
	if m.create != nil {
		return m.create(o)
	}
	m.items[o.ID] = o
	return nil
}

func (m *mockOrderRepo) FindByID(_ context.Context, id int64) (*entity.MembershipOrder, error) {
	if m.findByID != nil {
		return m.findByID(id)
	}
	if o, ok := m.items[id]; ok {
		c := *o
		return &c, nil
	}
	return nil, nil
}

func (m *mockOrderRepo) FindByUserID(_ context.Context, userID int64, p pagination.Params) ([]entity.MembershipOrder, int64, error) {
	if m.findByUser != nil {
		return m.findByUser(userID, p)
	}
	var out []entity.MembershipOrder
	for _, o := range m.items {
		if o.UserID == userID {
			out = append(out, *o)
		}
	}
	return out, int64(len(out)), nil
}

func (m *mockOrderRepo) FindByOrderNo(_ context.Context, orderNo string) (*entity.MembershipOrder, error) {
	if m.findByNo != nil {
		return m.findByNo(orderNo)
	}
	for _, o := range m.items {
		if o.OrderNo == orderNo {
			c := *o
			return &c, nil
		}
	}
	return nil, nil
}

func (m *mockOrderRepo) UpdateStatus(_ context.Context, id int64, status string, paidAt *time.Time, paymentMethod, transactionID string) error {
	if m.updateStatus != nil {
		return m.updateStatus(id, status, paidAt, paymentMethod, transactionID)
	}
	if o, ok := m.items[id]; ok {
		o.Status = status
		o.PaidAt = paidAt
		o.PaymentMethod = paymentMethod
		o.TransactionID = transactionID
	}
	return nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

type orderHarness struct {
	uc       *usecase.OrderUseCase
	orderRepo *mockOrderRepo
	planRepo  *mockMembershipPlanRepo
}

func newOrderHarness() *orderHarness {
	orderRepo := newMockOrderRepo()
	planRepo := newMockMembershipPlanRepo()
	uc := usecase.NewOrderUseCase(orderRepo, planRepo)
	return &orderHarness{uc: uc, orderRepo: orderRepo, planRepo: planRepo}
}

func seedOrder(h *orderHarness, id, userID, planID int64, orderNo, status string) *entity.MembershipOrder {
	o := &entity.MembershipOrder{
		ID:          id,
		UserID:      userID,
		PlanID:      planID,
		OrderNo:     orderNo,
		AmountCents: 999,
		Status:      status,
	}
	h.orderRepo.items[id] = o
	return o
}

func addPlan(repo *mockMembershipPlanRepo, id int64, name string, isActive bool) *entity.MembershipPlan {
	p := &entity.MembershipPlan{
		ID:           id,
		Name:         name,
		IsActive:     isActive,
		PriceCents:   999,
		DurationDays: 30,
	}
	repo.items[id] = p
	return p
}

// ---------------------------------------------------------------------------
// OrderUseCase.ListOrders
// ---------------------------------------------------------------------------

func TestOrderUseCase_ListOrders(t *testing.T) {
	t.Run("success with plan names resolved", func(t *testing.T) {
		h := newOrderHarness()
		addPlan(h.planRepo, 10, "Premium", true)
		addPlan(h.planRepo, 20, "Basic", true)
		seedOrder(h, 1, 100, 10, "MB100_001", entity.OrderStatusPaid)
		seedOrder(h, 2, 100, 20, "MB100_002", entity.OrderStatusPending)

		p := pagination.From(1, 10)
		page, err := h.uc.ListOrders(context.Background(), 100, p)
		require.NoError(t, err)
		require.NotNil(t, page)
		assert.Equal(t, 2, page.Total)
		assert.Len(t, page.Items, 2)
	})

	t.Run("plan name resolved correctly", func(t *testing.T) {
		h := newOrderHarness()
		addPlan(h.planRepo, 10, "Premium", true)
		seedOrder(h, 1, 100, 10, "MB100_001", entity.OrderStatusPaid)

		p := pagination.From(1, 10)
		page, err := h.uc.ListOrders(context.Background(), 100, p)
		require.NoError(t, err)
		require.NotNil(t, page)
		assert.Equal(t, "Premium", page.Items[0].PlanName)
	})

	t.Run("plan not found returns empty plan name", func(t *testing.T) {
		h := newOrderHarness()
		seedOrder(h, 1, 100, 99, "MB100_001", entity.OrderStatusPaid)

		p := pagination.From(1, 10)
		page, err := h.uc.ListOrders(context.Background(), 100, p)
		require.NoError(t, err)
		require.NotNil(t, page)
		assert.Equal(t, "", page.Items[0].PlanName)
	})

	t.Run("empty list", func(t *testing.T) {
		h := newOrderHarness()
		p := pagination.From(1, 10)
		page, err := h.uc.ListOrders(context.Background(), 100, p)
		require.NoError(t, err)
		assert.Equal(t, 0, page.Total)
		assert.Empty(t, page.Items)
	})

	t.Run("repo error", func(t *testing.T) {
		h := newOrderHarness()
		sentinel := errors.New("db error")
		h.orderRepo.findByUser = func(int64, pagination.Params) ([]entity.MembershipOrder, int64, error) {
			return nil, 0, sentinel
		}

		p := pagination.From(1, 10)
		page, err := h.uc.ListOrders(context.Background(), 100, p)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, page)
	})
}

// ---------------------------------------------------------------------------
// OrderUseCase.GetOrder
// ---------------------------------------------------------------------------

func TestOrderUseCase_GetOrder(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newOrderHarness()
		addPlan(h.planRepo, 10, "Premium", true)
		seedOrder(h, 10, 100, 10, "MB100_001", entity.OrderStatusPaid)

		resp, err := h.uc.GetOrder(context.Background(), 100, 10)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, int64(10), resp.ID)
		assert.Equal(t, "MB100_001", resp.OrderNo)
		assert.Equal(t, entity.OrderStatusPaid, resp.Status)
		assert.Equal(t, "Premium", resp.PlanName)
	})

	t.Run("order not found", func(t *testing.T) {
		h := newOrderHarness()
		resp, err := h.uc.GetOrder(context.Background(), 100, 999)
		requireError(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("forbidden - wrong user", func(t *testing.T) {
		h := newOrderHarness()
		seedOrder(h, 10, 100, 10, "MB100_001", entity.OrderStatusPaid)

		resp, err := h.uc.GetOrder(context.Background(), 999, 10)
		requireError(t, err, errno.Forbidden)
		assert.Nil(t, resp)
	})

	t.Run("repo error on FindByID", func(t *testing.T) {
		h := newOrderHarness()
		sentinel := errors.New("db error")
		h.orderRepo.findByID = func(int64) (*entity.MembershipOrder, error) { return nil, sentinel }

		resp, err := h.uc.GetOrder(context.Background(), 100, 1)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("plan not found returns empty plan name", func(t *testing.T) {
		h := newOrderHarness()
		seedOrder(h, 10, 100, 99, "MB100_001", entity.OrderStatusPaid)

		resp, err := h.uc.GetOrder(context.Background(), 100, 10)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "", resp.PlanName)
	})
}