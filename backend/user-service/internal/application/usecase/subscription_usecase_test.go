package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/application/usecase"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// ---------------------------------------------------------------------------
// UserSubscriptionRepository mock
// ---------------------------------------------------------------------------

type mockSubRepo struct {
	items        map[int64]*entity.UserSubscription
	findActive   func(int64) (*entity.UserSubscription, error)
	findByUser   func(int64) ([]entity.UserSubscription, error)
	findByID     func(int64) (*entity.UserSubscription, error)
	create       func(*entity.UserSubscription) error
	update       func(*entity.UserSubscription) error
	extend       func(int64, int) error
}

func newMockSubRepo() *mockSubRepo {
	return &mockSubRepo{items: map[int64]*entity.UserSubscription{}}
}

func (m *mockSubRepo) FindByUserID(_ context.Context, userID int64) ([]entity.UserSubscription, error) {
	if m.findByUser != nil {
		return m.findByUser(userID)
	}
	var out []entity.UserSubscription
	for _, s := range m.items {
		if s.UserID == userID {
			out = append(out, *s)
		}
	}
	return out, nil
}

func (m *mockSubRepo) FindActiveByUserID(_ context.Context, userID int64) (*entity.UserSubscription, error) {
	if m.findActive != nil {
		return m.findActive(userID)
	}
	for _, s := range m.items {
		if s.UserID == userID && s.Status == entity.SubscriptionStatusActive {
			c := *s
			return &c, nil
		}
	}
	return nil, nil
}

func (m *mockSubRepo) FindByID(_ context.Context, id int64) (*entity.UserSubscription, error) {
	if m.findByID != nil {
		return m.findByID(id)
	}
	if s, ok := m.items[id]; ok {
		c := *s
		return &c, nil
	}
	return nil, nil
}

func (m *mockSubRepo) Create(_ context.Context, s *entity.UserSubscription) error {
	if m.create != nil {
		return m.create(s)
	}
	m.items[s.ID] = s
	return nil
}

func (m *mockSubRepo) Update(_ context.Context, s *entity.UserSubscription) error {
	if m.update != nil {
		return m.update(s)
	}
	m.items[s.ID] = s
	return nil
}

func (m *mockSubRepo) Extend(_ context.Context, id int64, days int) error {
	if m.extend != nil {
		return m.extend(id, days)
	}
	if s, ok := m.items[id]; ok {
		s.ExpiresAt = s.ExpiresAt.AddDate(0, 0, days)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

type subHarness struct {
	uc       *usecase.SubscriptionUseCase
	planRepo *mockMembershipPlanRepo
	subRepo  *mockSubRepo
	orderRepo *mockOrderRepo
	apiRepo  *mockApiKeyRepo
}

func newSubHarness() *subHarness {
	planRepo := newMockMembershipPlanRepo()
	subRepo := newMockSubRepo()
	orderRepo := newMockOrderRepo()
	apiRepo := newMockApiKeyRepo()
	uc := usecase.NewSubscriptionUseCase(planRepo, subRepo, orderRepo, apiRepo)
	return &subHarness{
		uc:        uc,
		planRepo:  planRepo,
		subRepo:   subRepo,
		orderRepo: orderRepo,
		apiRepo:   apiRepo,
	}
}

func seedSubPlan(h *subHarness, id int64, name string, isActive bool, days int) *entity.MembershipPlan {
	p := &entity.MembershipPlan{
		ID:           id,
		Name:         name,
		IsActive:     isActive,
		PriceCents:   999,
		DurationDays: days,
	}
	h.planRepo.items[id] = p
	return p
}

func seedSubscription(h *subHarness, id, userID, planID int64, status string) *entity.UserSubscription {
	now := time.Now()
	s := &entity.UserSubscription{
		ID:        id,
		UserID:    userID,
		PlanID:    planID,
		Status:    status,
		StartedAt: now,
		ExpiresAt: now.AddDate(0, 0, 30),
		AutoRenew: true,
	}
	h.subRepo.items[id] = s
	return s
}

// ---------------------------------------------------------------------------
// SubscriptionUseCase.GetCurrentSubscription
// ---------------------------------------------------------------------------

func TestSubscriptionUseCase_GetCurrentSubscription(t *testing.T) {
	t.Run("success with active subscription", func(t *testing.T) {
		h := newSubHarness()
		seedSubPlan(h, 10, "Premium", true, 30)
		seedSubscription(h, 1, 100, 10, entity.SubscriptionStatusActive)

		resp, err := h.uc.GetCurrentSubscription(context.Background(), 100)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, int64(1), resp.ID)
		assert.Equal(t, int64(100), resp.UserID)
		assert.Equal(t, entity.SubscriptionStatusActive, resp.Status)
		assert.Equal(t, "Premium", resp.PlanName)
	})

	t.Run("no active subscription returns nil", func(t *testing.T) {
		h := newSubHarness()
		resp, err := h.uc.GetCurrentSubscription(context.Background(), 100)
		require.NoError(t, err)
		assert.Nil(t, resp)
	})

	t.Run("repo error on FindActiveByUserID", func(t *testing.T) {
		h := newSubHarness()
		sentinel := errors.New("db error")
		h.subRepo.findActive = func(int64) (*entity.UserSubscription, error) { return nil, sentinel }

		resp, err := h.uc.GetCurrentSubscription(context.Background(), 100)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("repo error on plan FindByID", func(t *testing.T) {
		h := newSubHarness()
		seedSubscription(h, 1, 100, 10, entity.SubscriptionStatusActive)
		sentinel := errors.New("plan db error")
		h.planRepo.findByID = func(int64) (*entity.MembershipPlan, error) { return nil, sentinel }

		resp, err := h.uc.GetCurrentSubscription(context.Background(), 100)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})
}

// ---------------------------------------------------------------------------
// SubscriptionUseCase.Subscribe
// ---------------------------------------------------------------------------

func TestSubscriptionUseCase_Subscribe(t *testing.T) {
	t.Run("success subscribe", func(t *testing.T) {
		h := newSubHarness()
		seedSubPlan(h, 10, "Premium", true, 30)

		resp, err := h.uc.Subscribe(context.Background(), 100, 10)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, int64(100), resp.UserID)
		assert.Equal(t, int64(10), resp.PlanID)
		assert.Equal(t, entity.OrderStatusPaid, resp.Status)
		assert.NotEmpty(t, resp.OrderNo)

		assert.NotEmpty(t, resp.TransactionID)
		assert.Equal(t, "simulated", resp.PaymentMethod)
	})

	t.Run("plan not found", func(t *testing.T) {
		h := newSubHarness()
		resp, err := h.uc.Subscribe(context.Background(), 100, 999)
		requireError(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("plan inactive rejected", func(t *testing.T) {
		h := newSubHarness()
		seedSubPlan(h, 10, "Deprecated", false, 30)

		resp, err := h.uc.Subscribe(context.Background(), 100, 10)
		requireError(t, err, errno.ValidationFailed)
		assert.Nil(t, resp)
	})

	t.Run("repo error on plan FindByID", func(t *testing.T) {
		h := newSubHarness()
		sentinel := errors.New("db error")
		h.planRepo.findByID = func(int64) (*entity.MembershipPlan, error) { return nil, sentinel }

		resp, err := h.uc.Subscribe(context.Background(), 100, 10)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("repo error on order Create", func(t *testing.T) {
		h := newSubHarness()
		seedSubPlan(h, 10, "Premium", true, 30)
		sentinel := errors.New("create failed")
		h.orderRepo.create = func(*entity.MembershipOrder) error { return sentinel }

		resp, err := h.uc.Subscribe(context.Background(), 100, 10)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("new subscription created when none exists", func(t *testing.T) {
		h := newSubHarness()
		seedSubPlan(h, 10, "Premium", true, 30)

		_, err := h.uc.Subscribe(context.Background(), 100, 10)
		require.NoError(t, err)

		subs, err := h.subRepo.FindByUserID(context.Background(), 100)
		require.NoError(t, err)
		assert.NotEmpty(t, subs)
	})

	t.Run("existing subscription extends when same plan", func(t *testing.T) {
		h := newSubHarness()
		seedSubPlan(h, 10, "Premium", true, 30)
		seedSubscription(h, 1, 100, 10, entity.SubscriptionStatusActive)

		resp, err := h.uc.Subscribe(context.Background(), 100, 10)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

// ---------------------------------------------------------------------------
// SubscriptionUseCase.CancelAutoRenew
// ---------------------------------------------------------------------------

func TestSubscriptionUseCase_CancelAutoRenew(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newSubHarness()
		seedSubscription(h, 1, 100, 10, entity.SubscriptionStatusActive)

		err := h.uc.CancelAutoRenew(context.Background(), 100)
		require.NoError(t, err)

		stored, _ := h.subRepo.FindActiveByUserID(context.Background(), 100)
		require.NotNil(t, stored)
		assert.False(t, stored.AutoRenew)
	})

	t.Run("no active subscription", func(t *testing.T) {
		h := newSubHarness()
		err := h.uc.CancelAutoRenew(context.Background(), 100)
		requireError(t, err, errno.NotFound)
	})

	t.Run("repo error on FindActiveByUserID", func(t *testing.T) {
		h := newSubHarness()
		sentinel := errors.New("db error")
		h.subRepo.findActive = func(int64) (*entity.UserSubscription, error) { return nil, sentinel }

		err := h.uc.CancelAutoRenew(context.Background(), 100)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
	})

	t.Run("repo error on Update", func(t *testing.T) {
		h := newSubHarness()
		seedSubscription(h, 1, 100, 10, entity.SubscriptionStatusActive)
		sentinel := errors.New("update failed")
		h.subRepo.update = func(*entity.UserSubscription) error { return sentinel }

		err := h.uc.CancelAutoRenew(context.Background(), 100)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
	})
}

// ---------------------------------------------------------------------------
// SubscriptionUseCase.HandlePaymentCallback
// ---------------------------------------------------------------------------

func TestSubscriptionUseCase_HandlePaymentCallback(t *testing.T) {
	t.Run("success - paid callback", func(t *testing.T) {
		h := newSubHarness()
		seedSubPlan(h, 10, "Premium", true, 30)
		order := &entity.MembershipOrder{
			ID:          1,
			UserID:      100,
			PlanID:      10,
			OrderNo:     "MB100_test",
			AmountCents: 999,
			Status:      entity.OrderStatusPending,
		}
		h.orderRepo.items[1] = order

		resp, err := h.uc.HandlePaymentCallback(context.Background(), &dto.PaymentCallbackRequest{
			OrderNo:       "MB100_test",
			Status:        "paid",
			PaymentMethod: "alipay",
			TransactionID: "tx_001",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, entity.OrderStatusPaid, resp.Status)
		assert.Equal(t, "alipay", resp.PaymentMethod)
		assert.Equal(t, "tx_001", resp.TransactionID)
	})

	t.Run("nil request rejected", func(t *testing.T) {
		h := newSubHarness()
		resp, err := h.uc.HandlePaymentCallback(context.Background(), nil)
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("empty order_no rejected", func(t *testing.T) {
		h := newSubHarness()
		resp, err := h.uc.HandlePaymentCallback(context.Background(), &dto.PaymentCallbackRequest{})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("order not found", func(t *testing.T) {
		h := newSubHarness()
		resp, err := h.uc.HandlePaymentCallback(context.Background(), &dto.PaymentCallbackRequest{
			OrderNo: "unknown",
			Status:  "paid",
		})
		requireError(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("already paid returns early", func(t *testing.T) {
		h := newSubHarness()
		seedSubPlan(h, 10, "Premium", true, 30)
		order := &entity.MembershipOrder{
			ID:          1,
			UserID:      100,
			PlanID:      10,
			OrderNo:     "MB100_paid",
			AmountCents: 999,
			Status:      entity.OrderStatusPaid,
		}
		h.orderRepo.items[1] = order

		resp, err := h.uc.HandlePaymentCallback(context.Background(), &dto.PaymentCallbackRequest{
			OrderNo: "MB100_paid",
			Status:  "paid",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, entity.OrderStatusPaid, resp.Status)
	})

	t.Run("non-paid status keeps pending", func(t *testing.T) {
		h := newSubHarness()
		order := &entity.MembershipOrder{
			ID:          1,
			UserID:      100,
			PlanID:      10,
			OrderNo:     "MB100_pending",
			AmountCents: 999,
			Status:      entity.OrderStatusPending,
		}
		h.orderRepo.items[1] = order

		resp, err := h.uc.HandlePaymentCallback(context.Background(), &dto.PaymentCallbackRequest{
			OrderNo: "MB100_pending",
			Status:  "failed",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, entity.OrderStatusPending, resp.Status)
	})

	t.Run("repo error on FindByOrderNo", func(t *testing.T) {
		h := newSubHarness()
		sentinel := errors.New("db error")
		h.orderRepo.findByNo = func(string) (*entity.MembershipOrder, error) { return nil, sentinel }

		resp, err := h.uc.HandlePaymentCallback(context.Background(), &dto.PaymentCallbackRequest{
			OrderNo: "MB100_test",
			Status:  "paid",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})
}