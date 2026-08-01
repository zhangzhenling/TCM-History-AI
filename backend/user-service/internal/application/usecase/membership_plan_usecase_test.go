package usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/application/usecase"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// ---------------------------------------------------------------------------
// MembershipPlanRepository mock
// ---------------------------------------------------------------------------

type mockMembershipPlanRepo struct {
	items    map[int64]*entity.MembershipPlan
	listAll  func() ([]entity.MembershipPlan, error)
	listActive func() ([]entity.MembershipPlan, error)
	findByID func(int64) (*entity.MembershipPlan, error)
	create   func(*entity.MembershipPlan) error
	update   func(*entity.MembershipPlan) error
	del      func(int64) error
}

func newMockMembershipPlanRepo() *mockMembershipPlanRepo {
	return &mockMembershipPlanRepo{items: map[int64]*entity.MembershipPlan{}}
}

func (m *mockMembershipPlanRepo) ListAll(_ context.Context) ([]entity.MembershipPlan, error) {
	if m.listAll != nil {
		return m.listAll()
	}
	out := make([]entity.MembershipPlan, 0, len(m.items))
	for _, p := range m.items {
		out = append(out, *p)
	}
	return out, nil
}

func (m *mockMembershipPlanRepo) ListActive(_ context.Context) ([]entity.MembershipPlan, error) {
	if m.listActive != nil {
		return m.listActive()
	}
	var out []entity.MembershipPlan
	for _, p := range m.items {
		if p.IsActive {
			out = append(out, *p)
		}
	}
	return out, nil
}

func (m *mockMembershipPlanRepo) FindByID(_ context.Context, id int64) (*entity.MembershipPlan, error) {
	if m.findByID != nil {
		return m.findByID(id)
	}
	if p, ok := m.items[id]; ok {
		c := *p
		return &c, nil
	}
	return nil, nil
}

func (m *mockMembershipPlanRepo) Create(_ context.Context, p *entity.MembershipPlan) error {
	if m.create != nil {
		return m.create(p)
	}
	m.items[p.ID] = p
	return nil
}

func (m *mockMembershipPlanRepo) Update(_ context.Context, p *entity.MembershipPlan) error {
	if m.update != nil {
		return m.update(p)
	}
	m.items[p.ID] = p
	return nil
}

func (m *mockMembershipPlanRepo) Delete(_ context.Context, id int64) error {
	if m.del != nil {
		return m.del(id)
	}
	delete(m.items, id)
	return nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

type planHarness struct {
	uc      *usecase.MembershipPlanUseCase
	planRepo *mockMembershipPlanRepo
}

func newPlanHarness() *planHarness {
	planRepo := newMockMembershipPlanRepo()
	uc := usecase.NewMembershipPlanUseCase(planRepo)
	return &planHarness{uc: uc, planRepo: planRepo}
}

func seedPlan(h *planHarness, id int64, name string, isActive bool) *entity.MembershipPlan {
	p := &entity.MembershipPlan{
		ID:           id,
		Name:         name,
		IsActive:     isActive,
		PriceCents:   999,
		DurationDays: 30,
	}
	h.planRepo.items[id] = p
	return p
}

// ---------------------------------------------------------------------------
// MembershipPlanUseCase.ListPlans
// ---------------------------------------------------------------------------

func TestMembershipPlanUseCase_ListPlans(t *testing.T) {
	t.Run("success list active only", func(t *testing.T) {
		h := newPlanHarness()
		seedPlan(h, 1, "Basic", true)
		seedPlan(h, 2, "Premium", true)
		seedPlan(h, 3, "Deprecated", false)

		plans, err := h.uc.ListPlans(context.Background(), false)
		require.NoError(t, err)
		assert.Len(t, plans, 2)
	})

	t.Run("success list all including inactive", func(t *testing.T) {
		h := newPlanHarness()
		seedPlan(h, 1, "Basic", true)
		seedPlan(h, 2, "Deprecated", false)

		plans, err := h.uc.ListPlans(context.Background(), true)
		require.NoError(t, err)
		assert.Len(t, plans, 2)
	})

	t.Run("empty list", func(t *testing.T) {
		h := newPlanHarness()
		plans, err := h.uc.ListPlans(context.Background(), false)
		require.NoError(t, err)
		assert.Empty(t, plans)
	})

	t.Run("repo error on ListActive", func(t *testing.T) {
		h := newPlanHarness()
		sentinel := errors.New("db error")
		h.planRepo.listActive = func() ([]entity.MembershipPlan, error) { return nil, sentinel }

		plans, err := h.uc.ListPlans(context.Background(), false)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, plans)
	})

	t.Run("repo error on ListAll", func(t *testing.T) {
		h := newPlanHarness()
		sentinel := errors.New("db error")
		h.planRepo.listAll = func() ([]entity.MembershipPlan, error) { return nil, sentinel }

		plans, err := h.uc.ListPlans(context.Background(), true)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, plans)
	})
}

// ---------------------------------------------------------------------------
// MembershipPlanUseCase.GetPlan
// ---------------------------------------------------------------------------

func TestMembershipPlanUseCase_GetPlan(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newPlanHarness()
		seedPlan(h, 10, "Premium", true)

		resp, err := h.uc.GetPlan(context.Background(), 10)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, int64(10), resp.ID)
		assert.Equal(t, "Premium", resp.Name)
	})

	t.Run("not found", func(t *testing.T) {
		h := newPlanHarness()
		resp, err := h.uc.GetPlan(context.Background(), 999)
		requireError(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("repo error", func(t *testing.T) {
		h := newPlanHarness()
		sentinel := errors.New("db error")
		h.planRepo.findByID = func(int64) (*entity.MembershipPlan, error) { return nil, sentinel }

		resp, err := h.uc.GetPlan(context.Background(), 1)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})
}

// ---------------------------------------------------------------------------
// MembershipPlanUseCase.CreatePlan
// ---------------------------------------------------------------------------

func TestMembershipPlanUseCase_CreatePlan(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newPlanHarness()
		resp, err := h.uc.CreatePlan(context.Background(), &dto.CreateMembershipPlanRequest{
			Name:               "New Plan",
			PriceCents:         1999,
			DurationDays:       30,
			MaxDailyAIRequests: 500,
			MaxTokenPerMonth:   100000,
			IsActive:           true,
			SortOrder:          1,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "New Plan", resp.Name)
		assert.Equal(t, int64(1999), resp.PriceCents)
		assert.Equal(t, 30, resp.DurationDays)
		assert.True(t, resp.IsActive)
		assert.NotZero(t, resp.ID)

		stored, ok := h.planRepo.items[resp.ID]
		require.True(t, ok)
		assert.Equal(t, "New Plan", stored.Name)
	})

	t.Run("nil request rejected", func(t *testing.T) {
		h := newPlanHarness()
		resp, err := h.uc.CreatePlan(context.Background(), nil)
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("empty name rejected", func(t *testing.T) {
		h := newPlanHarness()
		resp, err := h.uc.CreatePlan(context.Background(), &dto.CreateMembershipPlanRequest{
			PriceCents:   100,
			DurationDays: 30,
		})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("negative price rejected", func(t *testing.T) {
		h := newPlanHarness()
		resp, err := h.uc.CreatePlan(context.Background(), &dto.CreateMembershipPlanRequest{
			Name:         "Bad Plan",
			PriceCents:   -1,
			DurationDays: 30,
		})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("zero duration rejected", func(t *testing.T) {
		h := newPlanHarness()
		resp, err := h.uc.CreatePlan(context.Background(), &dto.CreateMembershipPlanRequest{
			Name:         "Bad Plan",
			PriceCents:   100,
			DurationDays: 0,
		})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("negative duration rejected", func(t *testing.T) {
		h := newPlanHarness()
		resp, err := h.uc.CreatePlan(context.Background(), &dto.CreateMembershipPlanRequest{
			Name:         "Bad Plan",
			PriceCents:   100,
			DurationDays: -5,
		})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("zero price is allowed", func(t *testing.T) {
		h := newPlanHarness()
		resp, err := h.uc.CreatePlan(context.Background(), &dto.CreateMembershipPlanRequest{
			Name:         "Free Plan",
			PriceCents:   0,
			DurationDays: 30,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, int64(0), resp.PriceCents)
	})

	t.Run("with features", func(t *testing.T) {
		h := newPlanHarness()
		features := json.RawMessage(`{"ai":true,"export":false}`)
		resp, err := h.uc.CreatePlan(context.Background(), &dto.CreateMembershipPlanRequest{
			Name:     "Feature Plan",
			PriceCents: 100,
			DurationDays: 30,
			Features: features,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, features, resp.Features)
	})

	t.Run("repo error", func(t *testing.T) {
		h := newPlanHarness()
		sentinel := errors.New("db error")
		h.planRepo.create = func(*entity.MembershipPlan) error { return sentinel }

		resp, err := h.uc.CreatePlan(context.Background(), &dto.CreateMembershipPlanRequest{
			Name:         "Plan",
			PriceCents:   100,
			DurationDays: 30,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})
}

// ---------------------------------------------------------------------------
// MembershipPlanUseCase.UpdatePlan
// ---------------------------------------------------------------------------

func TestMembershipPlanUseCase_UpdatePlan(t *testing.T) {
	t.Run("success update name and price", func(t *testing.T) {
		h := newPlanHarness()
		seedPlan(h, 10, "Old Name", true)

		name := "New Name"
		price := int64(2999)
		resp, err := h.uc.UpdatePlan(context.Background(), 10, &dto.UpdateMembershipPlanRequest{
			Name:       &name,
			PriceCents: &price,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "New Name", resp.Name)
		assert.Equal(t, int64(2999), resp.PriceCents)
	})

	t.Run("success toggle active", func(t *testing.T) {
		h := newPlanHarness()
		seedPlan(h, 10, "Plan", true)

		active := false
		resp, err := h.uc.UpdatePlan(context.Background(), 10, &dto.UpdateMembershipPlanRequest{
			IsActive: &active,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.False(t, resp.IsActive)
	})

	t.Run("success update duration", func(t *testing.T) {
		h := newPlanHarness()
		seedPlan(h, 10, "Plan", true)

		days := 90
		resp, err := h.uc.UpdatePlan(context.Background(), 10, &dto.UpdateMembershipPlanRequest{
			DurationDays: &days,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, 90, resp.DurationDays)
	})

	t.Run("not found", func(t *testing.T) {
		h := newPlanHarness()
		resp, err := h.uc.UpdatePlan(context.Background(), 999, &dto.UpdateMembershipPlanRequest{})
		requireError(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("negative price rejected", func(t *testing.T) {
		h := newPlanHarness()
		seedPlan(h, 10, "Plan", true)

		price := int64(-5)
		resp, err := h.uc.UpdatePlan(context.Background(), 10, &dto.UpdateMembershipPlanRequest{
			PriceCents: &price,
		})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("zero duration rejected", func(t *testing.T) {
		h := newPlanHarness()
		seedPlan(h, 10, "Plan", true)

		days := 0
		resp, err := h.uc.UpdatePlan(context.Background(), 10, &dto.UpdateMembershipPlanRequest{
			DurationDays: &days,
		})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("negative duration rejected", func(t *testing.T) {
		h := newPlanHarness()
		seedPlan(h, 10, "Plan", true)

		days := -1
		resp, err := h.uc.UpdatePlan(context.Background(), 10, &dto.UpdateMembershipPlanRequest{
			DurationDays: &days,
		})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("repo error on FindByID", func(t *testing.T) {
		h := newPlanHarness()
		sentinel := errors.New("db error")
		h.planRepo.findByID = func(int64) (*entity.MembershipPlan, error) { return nil, sentinel }

		resp, err := h.uc.UpdatePlan(context.Background(), 1, &dto.UpdateMembershipPlanRequest{})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("repo error on Update", func(t *testing.T) {
		h := newPlanHarness()
		seedPlan(h, 10, "Plan", true)
		sentinel := errors.New("update failed")
		h.planRepo.update = func(*entity.MembershipPlan) error { return sentinel }

		resp, err := h.uc.UpdatePlan(context.Background(), 10, &dto.UpdateMembershipPlanRequest{})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})
}

// ---------------------------------------------------------------------------
// MembershipPlanUseCase.DeletePlan
// ---------------------------------------------------------------------------

func TestMembershipPlanUseCase_DeletePlan(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newPlanHarness()
		seedPlan(h, 10, "Plan", true)

		err := h.uc.DeletePlan(context.Background(), 10)
		require.NoError(t, err)

		_, ok := h.planRepo.items[10]
		assert.False(t, ok)
	})

	t.Run("not found", func(t *testing.T) {
		h := newPlanHarness()
		err := h.uc.DeletePlan(context.Background(), 999)
		requireError(t, err, errno.NotFound)
	})

	t.Run("repo error on FindByID", func(t *testing.T) {
		h := newPlanHarness()
		sentinel := errors.New("db error")
		h.planRepo.findByID = func(int64) (*entity.MembershipPlan, error) { return nil, sentinel }

		err := h.uc.DeletePlan(context.Background(), 1)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
	})

	t.Run("repo error on Delete", func(t *testing.T) {
		h := newPlanHarness()
		seedPlan(h, 10, "Plan", true)
		sentinel := errors.New("delete failed")
		h.planRepo.del = func(int64) error { return sentinel }

		err := h.uc.DeletePlan(context.Background(), 10)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
	})
}