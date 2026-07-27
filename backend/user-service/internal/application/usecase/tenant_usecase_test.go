package usecase_test

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/application/usecase"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// ---------------------------------------------------------------------------
// TenantRepository mock
// ---------------------------------------------------------------------------

// mockTenantRepo is an in-memory fake repository.TenantRepository. Every
// method has an optional hook field so individual tests can inject errors
// without touching the shared state.
type mockTenantRepo struct {
	mu        sync.Mutex
	items     map[int64]*entity.Tenant
	createErr func(*entity.Tenant) error
	updateErr func(*entity.Tenant) error
	deleteErr func(int64) error
	listErr   func() error
}

func newMockTenantRepo() *mockTenantRepo {
	return &mockTenantRepo{items: map[int64]*entity.Tenant{}}
}

func (m *mockTenantRepo) Create(_ context.Context, t *entity.Tenant) error {
	if m.createErr != nil {
		return m.createErr(t)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[t.ID]; ok {
		return errno.New(errno.AlreadyExists, "tenant exists")
	}
	stored := *t
	m.items[t.ID] = &stored
	return nil
}

func (m *mockTenantRepo) Update(_ context.Context, t *entity.Tenant) error {
	if m.updateErr != nil {
		return m.updateErr(t)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[t.ID]; !ok {
		return errno.New(errno.NotFound, "tenant not found")
	}
	stored := *t
	m.items[t.ID] = &stored
	return nil
}

func (m *mockTenantRepo) Delete(_ context.Context, id int64) error {
	if m.deleteErr != nil {
		return m.deleteErr(id)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "tenant not found")
	}
	delete(m.items, id)
	return nil
}

func (m *mockTenantRepo) FindByID(_ context.Context, id int64) (*entity.Tenant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.items[id]; ok {
		c := *t
		return &c, nil
	}
	return nil, nil
}

func (m *mockTenantRepo) FindByCode(_ context.Context, code string) (*entity.Tenant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.items {
		if t.Code == code {
			c := *t
			return &c, nil
		}
	}
	return nil, nil
}

func (m *mockTenantRepo) List(_ context.Context, p pagination.Params, status string) ([]entity.Tenant, int64, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]entity.Tenant, 0, len(m.items))
	for _, t := range m.items {
		if status == "" || t.Status == status {
			out = append(out, *t)
		}
	}
	return out, int64(len(out)), nil
}

// ---------------------------------------------------------------------------
// TenantMemberRepository mock
// ---------------------------------------------------------------------------

type mockTenantMemberRepo struct {
	mu          sync.Mutex
	items       map[int64]*entity.TenantMember // keyed by member.ID
	addErr      func(*entity.TenantMember) error
	removeErr   func(int64, int64) error
	findMembers func(int64) ([]entity.TenantMember, error)
	isMember    func(int64, int64) (*entity.TenantMember, bool, error)
	countErr    func(int64) error
}

func newMockTenantMemberRepo() *mockTenantMemberRepo {
	return &mockTenantMemberRepo{items: map[int64]*entity.TenantMember{}}
}

func (m *mockTenantMemberRepo) AddMember(_ context.Context, mem *entity.TenantMember) error {
	if m.addErr != nil {
		return m.addErr(mem)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.items {
		if existing.TenantID == mem.TenantID && existing.UserID == mem.UserID {
			return errno.New(errno.AlreadyExists, "member exists")
		}
	}
	stored := *mem
	m.items[mem.ID] = &stored
	return nil
}

func (m *mockTenantMemberRepo) RemoveMember(_ context.Context, tenantID, userID int64) error {
	if m.removeErr != nil {
		return m.removeErr(tenantID, userID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, existing := range m.items {
		if existing.TenantID == tenantID && existing.UserID == userID {
			delete(m.items, id)
			return nil
		}
	}
	return errno.New(errno.NotFound, "tenant member not found")
}

func (m *mockTenantMemberRepo) FindMembers(_ context.Context, tenantID int64) ([]entity.TenantMember, error) {
	if m.findMembers != nil {
		return m.findMembers(tenantID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]entity.TenantMember, 0)
	for _, existing := range m.items {
		if existing.TenantID == tenantID {
			out = append(out, *existing)
		}
	}
	return out, nil
}

func (m *mockTenantMemberRepo) FindUserTenants(_ context.Context, userID int64) ([]entity.TenantMember, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]entity.TenantMember, 0)
	for _, existing := range m.items {
		if existing.UserID == userID {
			out = append(out, *existing)
		}
	}
	return out, nil
}

func (m *mockTenantMemberRepo) IsMember(_ context.Context, tenantID, userID int64) (*entity.TenantMember, bool, error) {
	if m.isMember != nil {
		return m.isMember(tenantID, userID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.items {
		if existing.TenantID == tenantID && existing.UserID == userID {
			c := *existing
			return &c, true, nil
		}
	}
	return nil, false, nil
}

func (m *mockTenantMemberRepo) CountMembers(_ context.Context, tenantID int64) (int64, error) {
	if m.countErr != nil {
		return 0, m.countErr(tenantID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	for _, existing := range m.items {
		if existing.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

// ---------------------------------------------------------------------------
// Harness
// ---------------------------------------------------------------------------

// tenantHarness bundles a freshly wired TenantUseCase with its mock deps so
// each test can configure only what it needs.
type tenantHarness struct {
	uc         *usecase.TenantUseCase
	tenantRepo *mockTenantRepo
	memberRepo *mockTenantMemberRepo
}

func newTenantHarness() *tenantHarness {
	tenantRepo := newMockTenantRepo()
	memberRepo := newMockTenantMemberRepo()
	uc := usecase.NewTenantUseCase(tenantRepo, memberRepo)
	return &tenantHarness{uc: uc, tenantRepo: tenantRepo, memberRepo: memberRepo}
}

// seedTenant inserts a tenant directly into the mock and returns it. Useful
// for AddMember / member-management tests.
func seedTenant(repo *mockTenantRepo, id int64, code string, maxUsers int) *entity.Tenant {
	t := &entity.Tenant{
		Name:     "School " + code,
		Code:     code,
		Plan:     entity.PlanStandard,
		Status:   entity.TenantStatusActive,
		MaxUsers: maxUsers,
	}
	t.ID = id
	repo.items[id] = t
	return t
}

// ---------------------------------------------------------------------------
// CreateTenant
// ---------------------------------------------------------------------------

func TestTenantUseCase_CreateTenant(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newTenantHarness()
		resp, err := h.uc.CreateTenant(context.Background(), &dto.CreateTenantRequest{
			Name:     "Beijing University of Chinese Medicine",
			Code:     "BUCM",
			Plan:     entity.PlanPremium,
			MaxUsers: 500,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotZero(t, resp.ID)
		assert.Equal(t, "Beijing University of Chinese Medicine", resp.Name)
		assert.Equal(t, "BUCM", resp.Code)
		assert.Equal(t, entity.PlanPremium, resp.Plan)
		assert.Equal(t, entity.TenantStatusActive, resp.Status)
		assert.Equal(t, 500, resp.MaxUsers)

		// Stored in repo.
		stored, err := h.tenantRepo.FindByCode(context.Background(), "BUCM")
		require.NoError(t, err)
		require.NotNil(t, stored)
		assert.Equal(t, resp.ID, stored.ID)
	})

	t.Run("default plan when omitted", func(t *testing.T) {
		h := newTenantHarness()
		resp, err := h.uc.CreateTenant(context.Background(), &dto.CreateTenantRequest{
			Name:     "Plain School",
			Code:     "PLAIN",
			MaxUsers: 10,
		})
		require.NoError(t, err)
		assert.Equal(t, entity.PlanStandard, resp.Plan)
	})

	t.Run("rejects empty name", func(t *testing.T) {
		h := newTenantHarness()
		_, err := h.uc.CreateTenant(context.Background(), &dto.CreateTenantRequest{
			Code: "NO_NAME",
		})
		require.Error(t, err)
		assert.Equal(t, errno.InvalidParams, errno.From(err).Code)
	})

	t.Run("rejects empty code", func(t *testing.T) {
		h := newTenantHarness()
		_, err := h.uc.CreateTenant(context.Background(), &dto.CreateTenantRequest{
			Name: "No Code School",
		})
		require.Error(t, err)
		assert.Equal(t, errno.InvalidParams, errno.From(err).Code)
	})

	t.Run("rejects invalid plan", func(t *testing.T) {
		h := newTenantHarness()
		_, err := h.uc.CreateTenant(context.Background(), &dto.CreateTenantRequest{
			Name: "Bad Plan",
			Code: "BADPLAN",
			Plan: "ultimate",
		})
		require.Error(t, err)
		assert.Equal(t, errno.InvalidParams, errno.From(err).Code)
	})

	t.Run("rejects negative max_users", func(t *testing.T) {
		h := newTenantHarness()
		_, err := h.uc.CreateTenant(context.Background(), &dto.CreateTenantRequest{
			Name:     "Negative",
			Code:     "NEG",
			MaxUsers: -1,
		})
		require.Error(t, err)
		assert.Equal(t, errno.InvalidParams, errno.From(err).Code)
	})

	t.Run("rejects duplicate code", func(t *testing.T) {
		h := newTenantHarness()
		_, err := h.uc.CreateTenant(context.Background(), &dto.CreateTenantRequest{
			Name:     "First",
			Code:     "DUP",
			MaxUsers: 10,
		})
		require.NoError(t, err)

		_, err = h.uc.CreateTenant(context.Background(), &dto.CreateTenantRequest{
			Name:     "Second",
			Code:     "DUP",
			MaxUsers: 10,
		})
		require.Error(t, err)
		assert.Equal(t, errno.AlreadyExists, errno.From(err).Code)
	})

	t.Run("rejects bad expires_at", func(t *testing.T) {
		h := newTenantHarness()
		_, err := h.uc.CreateTenant(context.Background(), &dto.CreateTenantRequest{
			Name:      "Bad Expiry",
			Code:      "BAD_EXP",
			ExpiresAt: "not-a-date",
		})
		require.Error(t, err)
		assert.Equal(t, errno.InvalidParams, errno.From(err).Code)
	})

	t.Run("honours RFC3339 expires_at", func(t *testing.T) {
		h := newTenantHarness()
		want := "2027-12-31T23:59:59Z"
		resp, err := h.uc.CreateTenant(context.Background(), &dto.CreateTenantRequest{
			Name:      "Dated",
			Code:      "DATED",
			ExpiresAt: want,
		})
		require.NoError(t, err)
		assert.Equal(t, want, resp.ExpiresAt)
	})
}

// ---------------------------------------------------------------------------
// UpdateTenant
// ---------------------------------------------------------------------------

func TestTenantUseCase_UpdateTenant(t *testing.T) {
	t.Run("success patches fields", func(t *testing.T) {
		h := newTenantHarness()
		t1 := seedTenant(h.tenantRepo, idgen.Next(), "UPD", 10)

		newName := "Updated Name"
		newPlan := entity.PlanEnterprise
		newStatus := entity.TenantStatusSuspended
		newMax := 999
		resp, err := h.uc.UpdateTenant(context.Background(), t1.ID, &dto.UpdateTenantRequest{
			Name:     &newName,
			Plan:     &newPlan,
			Status:   &newStatus,
			MaxUsers: &newMax,
		})
		require.NoError(t, err)
		assert.Equal(t, newName, resp.Name)
		assert.Equal(t, newPlan, resp.Plan)
		assert.Equal(t, newStatus, resp.Status)
		assert.Equal(t, newMax, resp.MaxUsers)
	})

	t.Run("rejects unknown id", func(t *testing.T) {
		h := newTenantHarness()
		_, err := h.uc.UpdateTenant(context.Background(), 99999, &dto.UpdateTenantRequest{})
		require.Error(t, err)
		assert.Equal(t, errno.NotFound, errno.From(err).Code)
	})

	t.Run("rejects invalid plan", func(t *testing.T) {
		h := newTenantHarness()
		t1 := seedTenant(h.tenantRepo, idgen.Next(), "BADPLAN2", 10)
		bad := "ultimate"
		_, err := h.uc.UpdateTenant(context.Background(), t1.ID, &dto.UpdateTenantRequest{Plan: &bad})
		require.Error(t, err)
		assert.Equal(t, errno.InvalidParams, errno.From(err).Code)
	})

	t.Run("rejects invalid status", func(t *testing.T) {
		h := newTenantHarness()
		t1 := seedTenant(h.tenantRepo, idgen.Next(), "BADSTATUS", 10)
		bad := "deleted"
		_, err := h.uc.UpdateTenant(context.Background(), t1.ID, &dto.UpdateTenantRequest{Status: &bad})
		require.Error(t, err)
		assert.Equal(t, errno.InvalidParams, errno.From(err).Code)
	})

	t.Run("clears expires_at on empty string", func(t *testing.T) {
		h := newTenantHarness()
		t1 := seedTenant(h.tenantRepo, idgen.Next(), "CLEAREXP", 10)
		past := time.Now().Add(-time.Hour)
		t1.ExpiresAt = &past
		h.tenantRepo.items[t1.ID] = t1

		empty := ""
		resp, err := h.uc.UpdateTenant(context.Background(), t1.ID, &dto.UpdateTenantRequest{ExpiresAt: &empty})
		require.NoError(t, err)
		assert.Empty(t, resp.ExpiresAt)
	})
}

// ---------------------------------------------------------------------------
// GetTenant / ListTenants / DeleteTenant
// ---------------------------------------------------------------------------

func TestTenantUseCase_GetTenant(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newTenantHarness()
		t1 := seedTenant(h.tenantRepo, idgen.Next(), "GET", 10)
		resp, err := h.uc.GetTenant(context.Background(), t1.ID)
		require.NoError(t, err)
		assert.Equal(t, t1.ID, resp.ID)
		assert.Equal(t, "GET", resp.Code)
	})

	t.Run("not found returns nil-aware error", func(t *testing.T) {
		h := newTenantHarness()
		_, err := h.uc.GetTenant(context.Background(), 404)
		require.Error(t, err)
		assert.Equal(t, errno.NotFound, errno.From(err).Code)
	})
}

func TestTenantUseCase_ListTenants(t *testing.T) {
	t.Run("returns all tenants unfiltered", func(t *testing.T) {
		h := newTenantHarness()
		seedTenant(h.tenantRepo, idgen.Next(), "L1", 10)
		seedTenant(h.tenantRepo, idgen.Next(), "L2", 20)

		resp, err := h.uc.ListTenants(context.Background(), pagination.From(1, 20), "")
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Total)
		assert.Len(t, resp.Items, 2)
	})

	t.Run("filters by status", func(t *testing.T) {
		h := newTenantHarness()
		active := seedTenant(h.tenantRepo, idgen.Next(), "ACTIVE", 10)
		_ = active
		suspended := seedTenant(h.tenantRepo, idgen.Next(), "SUSP", 10)
		suspended.Status = entity.TenantStatusSuspended
		h.tenantRepo.items[suspended.ID] = suspended

		resp, err := h.uc.ListTenants(context.Background(), pagination.From(1, 20), entity.TenantStatusSuspended)
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Total)
		require.Len(t, resp.Items, 1)
		assert.Equal(t, entity.TenantStatusSuspended, resp.Items[0].Status)
	})
}

func TestTenantUseCase_DeleteTenant(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newTenantHarness()
		t1 := seedTenant(h.tenantRepo, idgen.Next(), "DEL", 10)
		require.NoError(t, h.uc.DeleteTenant(context.Background(), t1.ID))
		stored, err := h.tenantRepo.FindByID(context.Background(), t1.ID)
		require.NoError(t, err)
		assert.Nil(t, stored)
	})

	t.Run("not found", func(t *testing.T) {
		h := newTenantHarness()
		err := h.uc.DeleteTenant(context.Background(), 404)
		require.Error(t, err)
		assert.Equal(t, errno.NotFound, errno.From(err).Code)
	})
}

// ---------------------------------------------------------------------------
// AddMember
// ---------------------------------------------------------------------------

func TestTenantUseCase_AddMember(t *testing.T) {
	t.Run("success default role student", func(t *testing.T) {
		h := newTenantHarness()
		t1 := seedTenant(h.tenantRepo, idgen.Next(), "MEM", 10)
		resp, err := h.uc.AddMember(context.Background(), t1.ID, &dto.AddMemberRequest{
			UserID: 1001,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, t1.ID, resp.TenantID)
		assert.Equal(t, int64(1001), resp.UserID)
		assert.Equal(t, entity.TenantRoleStudent, resp.Role)
	})

	t.Run("success honours requested role", func(t *testing.T) {
		h := newTenantHarness()
		t1 := seedTenant(h.tenantRepo, idgen.Next(), "MEM2", 10)
		resp, err := h.uc.AddMember(context.Background(), t1.ID, &dto.AddMemberRequest{
			UserID: 1002,
			Role:   entity.TenantRoleSchoolAdmin,
		})
		require.NoError(t, err)
		assert.Equal(t, entity.TenantRoleSchoolAdmin, resp.Role)
	})

	t.Run("rejects invalid user_id", func(t *testing.T) {
		h := newTenantHarness()
		t1 := seedTenant(h.tenantRepo, idgen.Next(), "MEM3", 10)
		_, err := h.uc.AddMember(context.Background(), t1.ID, &dto.AddMemberRequest{UserID: 0})
		require.Error(t, err)
		assert.Equal(t, errno.InvalidParams, errno.From(err).Code)
	})

	t.Run("rejects invalid role", func(t *testing.T) {
		h := newTenantHarness()
		t1 := seedTenant(h.tenantRepo, idgen.Next(), "MEM4", 10)
		_, err := h.uc.AddMember(context.Background(), t1.ID, &dto.AddMemberRequest{
			UserID: 1003,
			Role:   "principal",
		})
		require.Error(t, err)
		assert.Equal(t, errno.InvalidParams, errno.From(err).Code)
	})

	t.Run("rejects unknown tenant", func(t *testing.T) {
		h := newTenantHarness()
		_, err := h.uc.AddMember(context.Background(), 404, &dto.AddMemberRequest{UserID: 1004})
		require.Error(t, err)
		assert.Equal(t, errno.NotFound, errno.From(err).Code)
	})

	t.Run("rejects suspended tenant", func(t *testing.T) {
		h := newTenantHarness()
		t1 := seedTenant(h.tenantRepo, idgen.Next(), "SUSPENDED", 10)
		t1.Status = entity.TenantStatusSuspended
		h.tenantRepo.items[t1.ID] = t1
		_, err := h.uc.AddMember(context.Background(), t1.ID, &dto.AddMemberRequest{UserID: 1005})
		require.Error(t, err)
		assert.Equal(t, errno.InvalidParams, errno.From(err).Code)
	})

	t.Run("rejects duplicate member", func(t *testing.T) {
		h := newTenantHarness()
		t1 := seedTenant(h.tenantRepo, idgen.Next(), "DUPMEM", 10)
		_, err := h.uc.AddMember(context.Background(), t1.ID, &dto.AddMemberRequest{UserID: 1006})
		require.NoError(t, err)

		_, err = h.uc.AddMember(context.Background(), t1.ID, &dto.AddMemberRequest{UserID: 1006})
		require.Error(t, err)
		assert.Equal(t, errno.AlreadyExists, errno.From(err).Code)
	})

	t.Run("enforces max_users quota", func(t *testing.T) {
		h := newTenantHarness()
		// Quota of 2; fill it, then expect the third add to be rejected.
		t1 := seedTenant(h.tenantRepo, idgen.Next(), "QUOTA", 2)
		_, err := h.uc.AddMember(context.Background(), t1.ID, &dto.AddMemberRequest{UserID: 2001})
		require.NoError(t, err)
		_, err = h.uc.AddMember(context.Background(), t1.ID, &dto.AddMemberRequest{UserID: 2002})
		require.NoError(t, err)

		_, err = h.uc.AddMember(context.Background(), t1.ID, &dto.AddMemberRequest{UserID: 2003})
		require.Error(t, err)
		assert.Equal(t, errno.Forbidden, errno.From(err).Code)
	})

	t.Run("max_users=0 disables quota", func(t *testing.T) {
		h := newTenantHarness()
		t1 := seedTenant(h.tenantRepo, idgen.Next(), "NOQUOTA", 0)
		for i := int64(1); i <= 5; i++ {
			_, err := h.uc.AddMember(context.Background(), t1.ID, &dto.AddMemberRequest{UserID: 3000 + i})
			require.NoError(t, err, "user %d should be admitted", 3000+i)
		}
	})

	t.Run("honours RFC3339 expires_at", func(t *testing.T) {
		h := newTenantHarness()
		t1 := seedTenant(h.tenantRepo, idgen.Next(), "EXPMEM", 10)
		want := "2027-12-31T23:59:59Z"
		resp, err := h.uc.AddMember(context.Background(), t1.ID, &dto.AddMemberRequest{
			UserID:    4001,
			ExpiresAt: want,
		})
		require.NoError(t, err)
		assert.Equal(t, want, resp.ExpiredAt)
	})

	t.Run("repository error propagates", func(t *testing.T) {
		h := newTenantHarness()
		t1 := seedTenant(h.tenantRepo, idgen.Next(), "REPOERR", 10)
		sentinel := errors.New("db down")
		h.memberRepo.addErr = func(*entity.TenantMember) error { return sentinel }
		_, err := h.uc.AddMember(context.Background(), t1.ID, &dto.AddMemberRequest{UserID: 5001})
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// RemoveMember / ListMembers / ListUserTenants
// ---------------------------------------------------------------------------

func TestTenantUseCase_RemoveMember(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newTenantHarness()
		t1 := seedTenant(h.tenantRepo, idgen.Next(), "RM", 10)
		_, err := h.uc.AddMember(context.Background(), t1.ID, &dto.AddMemberRequest{UserID: 6001})
		require.NoError(t, err)

		require.NoError(t, h.uc.RemoveMember(context.Background(), t1.ID, 6001))

		// Subsequent removal returns NotFound.
		err = h.uc.RemoveMember(context.Background(), t1.ID, 6001)
		require.Error(t, err)
		assert.Equal(t, errno.NotFound, errno.From(err).Code)
	})

	t.Run("rejects zero ids", func(t *testing.T) {
		h := newTenantHarness()
		err := h.uc.RemoveMember(context.Background(), 0, 100)
		require.Error(t, err)
		assert.Equal(t, errno.InvalidParams, errno.From(err).Code)

		err = h.uc.RemoveMember(context.Background(), 1, 0)
		require.Error(t, err)
		assert.Equal(t, errno.InvalidParams, errno.From(err).Code)
	})
}

func TestTenantUseCase_ListMembers(t *testing.T) {
	t.Run("returns all members for tenant", func(t *testing.T) {
		h := newTenantHarness()
		t1 := seedTenant(h.tenantRepo, idgen.Next(), "LISTMEM", 50)
		_, err := h.uc.AddMember(context.Background(), t1.ID, &dto.AddMemberRequest{UserID: 7001})
		require.NoError(t, err)
		_, err = h.uc.AddMember(context.Background(), t1.ID, &dto.AddMemberRequest{UserID: 7002, Role: entity.TenantRoleTeacher})
		require.NoError(t, err)

		resp, err := h.uc.ListMembers(context.Background(), t1.ID)
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Total)
		assert.Len(t, resp.Items, 2)
	})

	t.Run("empty tenant returns empty list", func(t *testing.T) {
		h := newTenantHarness()
		t1 := seedTenant(h.tenantRepo, idgen.Next(), "EMPTYMEM", 50)
		resp, err := h.uc.ListMembers(context.Background(), t1.ID)
		require.NoError(t, err)
		assert.Empty(t, resp.Items)
		assert.Zero(t, resp.Total)
	})
}

func TestTenantUseCase_ListUserTenants(t *testing.T) {
	t.Run("returns all memberships for user", func(t *testing.T) {
		h := newTenantHarness()
		t1 := seedTenant(h.tenantRepo, idgen.Next(), "UT1", 50)
		t2 := seedTenant(h.tenantRepo, idgen.Next(), "UT2", 50)
		_, err := h.uc.AddMember(context.Background(), t1.ID, &dto.AddMemberRequest{UserID: 8001})
		require.NoError(t, err)
		_, err = h.uc.AddMember(context.Background(), t2.ID, &dto.AddMemberRequest{UserID: 8001, Role: entity.TenantRoleTeacher})
		require.NoError(t, err)

		resp, err := h.uc.ListUserTenants(context.Background(), 8001)
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Total)
		assert.Len(t, resp.Items, 2)
	})

	t.Run("rejects zero user_id", func(t *testing.T) {
		h := newTenantHarness()
		_, err := h.uc.ListUserTenants(context.Background(), 0)
		require.Error(t, err)
		assert.Equal(t, errno.InvalidParams, errno.From(err).Code)
	})
}

// ---------------------------------------------------------------------------
// Smoke test: ensures the harness wiring itself is sane.
// ---------------------------------------------------------------------------

func TestTenantUseCase_WiringSmoke(t *testing.T) {
	h := newTenantHarness()
	require.NotNil(t, h.uc)
	require.NotNil(t, h.tenantRepo)
	require.NotNil(t, h.memberRepo)
	// Sanity: idgen is seeded by TestMain so IDs are non-zero.
	assert.NotZero(t, idgen.Next())
	// And strconv is used to format IDs in error messages, so make sure
	// the import is exercised.
	assert.Equal(t, "42", strconv.FormatInt(42, 10))
}
