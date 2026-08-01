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
// ApiKeyRepository mock
// ---------------------------------------------------------------------------

type mockApiKeyRepo struct {
	items           map[int64]*entity.ApiKey
	create          func(*entity.ApiKey) error
	findByID        func(int64) (*entity.ApiKey, error)
	findByUserID    func(int64) ([]entity.ApiKey, error)
	findByKeyHash   func(string) (*entity.ApiKey, error)
	update          func(*entity.ApiKey) error
	del             func(int64) error
	listByUserID    func(int64, int, int) ([]entity.ApiKey, int, error)
}

func newMockApiKeyRepo() *mockApiKeyRepo {
	return &mockApiKeyRepo{items: map[int64]*entity.ApiKey{}}
}

func (m *mockApiKeyRepo) Create(_ context.Context, k *entity.ApiKey) error {
	if m.create != nil {
		return m.create(k)
	}
	m.items[k.ID] = k
	return nil
}

func (m *mockApiKeyRepo) FindByID(_ context.Context, id int64) (*entity.ApiKey, error) {
	if m.findByID != nil {
		return m.findByID(id)
	}
	if k, ok := m.items[id]; ok {
		c := *k
		return &c, nil
	}
	return nil, nil
}

func (m *mockApiKeyRepo) FindByUserID(_ context.Context, userID int64) ([]entity.ApiKey, error) {
	if m.findByUserID != nil {
		return m.findByUserID(userID)
	}
	var out []entity.ApiKey
	for _, k := range m.items {
		if k.UserID == userID {
			out = append(out, *k)
		}
	}
	return out, nil
}

func (m *mockApiKeyRepo) FindByKeyHash(_ context.Context, keyHash string) (*entity.ApiKey, error) {
	if m.findByKeyHash != nil {
		return m.findByKeyHash(keyHash)
	}
	for _, k := range m.items {
		if k.KeyHash == keyHash {
			c := *k
			return &c, nil
		}
	}
	return nil, nil
}

func (m *mockApiKeyRepo) Update(_ context.Context, k *entity.ApiKey) error {
	if m.update != nil {
		return m.update(k)
	}
	m.items[k.ID] = k
	return nil
}

func (m *mockApiKeyRepo) Delete(_ context.Context, id int64) error {
	if m.del != nil {
		return m.del(id)
	}
	delete(m.items, id)
	return nil
}

func (m *mockApiKeyRepo) ListByUserID(_ context.Context, userID int64, page, pageSize int) ([]entity.ApiKey, int, error) {
	if m.listByUserID != nil {
		return m.listByUserID(userID, page, pageSize)
	}
	var out []entity.ApiKey
	for _, k := range m.items {
		if k.UserID == userID {
			out = append(out, *k)
		}
	}
	return out, len(out), nil
}

func (m *mockApiKeyRepo) IncrementUsage(_ context.Context, id int64) error {
	return nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

type apiKeyHarness struct {
	uc      *usecase.ApiKeyUseCase
	apiRepo *mockApiKeyRepo
}

func newApiKeyHarness() *apiKeyHarness {
	apiRepo := newMockApiKeyRepo()
	uc := usecase.NewApiKeyUseCase(apiRepo)
	return &apiKeyHarness{uc: uc, apiRepo: apiRepo}
}

func seedApiKey(h *apiKeyHarness, id, userID int64, name, status string) *entity.ApiKey {
	k := &entity.ApiKey{
		ID:       id,
		UserID:   userID,
		Name:     name,
		KeyHash:  "hash_" + name,
		KeyPrefix: "sk_prefix",
		Status:   status,
		Scopes:   []byte("[]"),
	}
	h.apiRepo.items[id] = k
	return k
}

// ---------------------------------------------------------------------------
// ApiKeyUseCase.Create
// ---------------------------------------------------------------------------

func TestApiKeyUseCase_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newApiKeyHarness()
		resp, err := h.uc.Create(context.Background(), 100, &dto.CreateApiKeyRequest{
			Name:               "test-key",
			Scopes:             []string{"read", "write"},
			QuotaDaily:         1000,
			QuotaMonthly:       100000,
			RateLimitPerMinute: 60,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.PlainKey)
		assert.Equal(t, "test-key", resp.Name)
		assert.Equal(t, entity.ApiKeyStatusActive, resp.Status)
		assert.Equal(t, int64(1000), resp.QuotaDaily)

		assert.NotEmpty(t, resp.KeyPrefix)
		assert.Contains(t, resp.PlainKey, "sk_")

		var stored string
		for _, k := range h.apiRepo.items {
			assert.Equal(t, "test-key", k.Name)
			assert.Equal(t, int64(100), k.UserID)
			stored = k.KeyHash
		}
		assert.NotEmpty(t, stored)
	})

	t.Run("nil request rejected", func(t *testing.T) {
		h := newApiKeyHarness()
		resp, err := h.uc.Create(context.Background(), 1, nil)
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("empty name rejected", func(t *testing.T) {
		h := newApiKeyHarness()
		resp, err := h.uc.Create(context.Background(), 1, &dto.CreateApiKeyRequest{})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("invalid userID rejected", func(t *testing.T) {
		h := newApiKeyHarness()
		resp, err := h.uc.Create(context.Background(), 0, &dto.CreateApiKeyRequest{Name: "key"})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("invalid expires_at format", func(t *testing.T) {
		h := newApiKeyHarness()
		resp, err := h.uc.Create(context.Background(), 1, &dto.CreateApiKeyRequest{
			Name:      "key",
			ExpiresAt: "not-a-date",
		})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("valid expires_at parsed", func(t *testing.T) {
		h := newApiKeyHarness()
		resp, err := h.uc.Create(context.Background(), 1, &dto.CreateApiKeyRequest{
			Name:      "key",
			ExpiresAt: "2099-12-31T23:59:59Z",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.ExpiresAt)
	})

	t.Run("empty scopes defaults to empty array", func(t *testing.T) {
		h := newApiKeyHarness()
		resp, err := h.uc.Create(context.Background(), 1, &dto.CreateApiKeyRequest{
			Name: "key",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Empty(t, resp.Scopes)
	})

	t.Run("repo error", func(t *testing.T) {
		h := newApiKeyHarness()
		sentinel := errors.New("db error")
		h.apiRepo.create = func(*entity.ApiKey) error { return sentinel }

		resp, err := h.uc.Create(context.Background(), 1, &dto.CreateApiKeyRequest{Name: "key"})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})
}

// ---------------------------------------------------------------------------
// ApiKeyUseCase.List
// ---------------------------------------------------------------------------

func TestApiKeyUseCase_List(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newApiKeyHarness()
		seedApiKey(h, 1, 100, "key1", entity.ApiKeyStatusActive)
		seedApiKey(h, 2, 100, "key2", entity.ApiKeyStatusActive)

		resp, err := h.uc.List(context.Background(), 100, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Total)
		assert.Len(t, resp.Items, 2)
	})

	t.Run("empty list", func(t *testing.T) {
		h := newApiKeyHarness()
		resp, err := h.uc.List(context.Background(), 100, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, 0, resp.Total)
		assert.Empty(t, resp.Items)
	})

	t.Run("invalid userID rejected", func(t *testing.T) {
		h := newApiKeyHarness()
		resp, err := h.uc.List(context.Background(), 0, 1, 10)
		requireError(t, err, errno.InvalidParams)
		assert.Equal(t, 0, resp.Total)
	})

	t.Run("repo error", func(t *testing.T) {
		h := newApiKeyHarness()
		sentinel := errors.New("db error")
		h.apiRepo.listByUserID = func(int64, int, int) ([]entity.ApiKey, int, error) {
			return nil, 0, sentinel
		}

		_, err := h.uc.List(context.Background(), 1, 1, 10)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
	})
}

// ---------------------------------------------------------------------------
// ApiKeyUseCase.Get
// ---------------------------------------------------------------------------

func TestApiKeyUseCase_Get(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newApiKeyHarness()
		seedApiKey(h, 10, 100, "my-key", entity.ApiKeyStatusActive)

		resp, err := h.uc.Get(context.Background(), 100, 10)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, int64(10), resp.ID)
		assert.Equal(t, "my-key", resp.Name)
	})

	t.Run("not found", func(t *testing.T) {
		h := newApiKeyHarness()
		resp, err := h.uc.Get(context.Background(), 100, 999)
		requireError(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("wrong user", func(t *testing.T) {
		h := newApiKeyHarness()
		seedApiKey(h, 10, 100, "my-key", entity.ApiKeyStatusActive)

		resp, err := h.uc.Get(context.Background(), 999, 10)
		requireError(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("repo error", func(t *testing.T) {
		h := newApiKeyHarness()
		sentinel := errors.New("db error")
		h.apiRepo.findByID = func(int64) (*entity.ApiKey, error) { return nil, sentinel }

		resp, err := h.uc.Get(context.Background(), 1, 1)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})
}

// ---------------------------------------------------------------------------
// ApiKeyUseCase.Update
// ---------------------------------------------------------------------------

func TestApiKeyUseCase_Update(t *testing.T) {
	t.Run("success update name", func(t *testing.T) {
		h := newApiKeyHarness()
		seedApiKey(h, 10, 100, "old-name", entity.ApiKeyStatusActive)

		name := "new-name"
		resp, err := h.uc.Update(context.Background(), 100, 10, &dto.UpdateApiKeyRequest{
			Name: &name,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "new-name", resp.Name)
	})

	t.Run("success update all fields", func(t *testing.T) {
		h := newApiKeyHarness()
		seedApiKey(h, 10, 100, "key", entity.ApiKeyStatusActive)

		name := "updated"
		scopes := []string{"admin"}
		daily := int64(5000)
		monthly := int64(500000)
		rpm := 120
		status := entity.ApiKeyStatusDisabled
		resp, err := h.uc.Update(context.Background(), 100, 10, &dto.UpdateApiKeyRequest{
			Name:               &name,
			Scopes:             scopes,
			QuotaDaily:         &daily,
			QuotaMonthly:       &monthly,
			RateLimitPerMinute: &rpm,
			Status:             &status,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "updated", resp.Name)
		assert.Equal(t, entity.ApiKeyStatusDisabled, resp.Status)
		assert.Equal(t, int64(5000), resp.QuotaDaily)
		assert.Equal(t, int64(500000), resp.QuotaMonthly)
		assert.Equal(t, 120, resp.RateLimitPerMinute)
		assert.Equal(t, []string{"admin"}, resp.Scopes)
	})

	t.Run("not found", func(t *testing.T) {
		h := newApiKeyHarness()
		resp, err := h.uc.Update(context.Background(), 100, 999, &dto.UpdateApiKeyRequest{})
		requireError(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("wrong user", func(t *testing.T) {
		h := newApiKeyHarness()
		seedApiKey(h, 10, 100, "key", entity.ApiKeyStatusActive)

		resp, err := h.uc.Update(context.Background(), 999, 10, &dto.UpdateApiKeyRequest{})
		requireError(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("invalid status", func(t *testing.T) {
		h := newApiKeyHarness()
		seedApiKey(h, 10, 100, "key", entity.ApiKeyStatusActive)

		status := "invalid-status"
		resp, err := h.uc.Update(context.Background(), 100, 10, &dto.UpdateApiKeyRequest{
			Status: &status,
		})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("repo error on FindByID", func(t *testing.T) {
		h := newApiKeyHarness()
		sentinel := errors.New("db error")
		h.apiRepo.findByID = func(int64) (*entity.ApiKey, error) { return nil, sentinel }

		resp, err := h.uc.Update(context.Background(), 100, 1, &dto.UpdateApiKeyRequest{})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("repo error on Update", func(t *testing.T) {
		h := newApiKeyHarness()
		seedApiKey(h, 10, 100, "key", entity.ApiKeyStatusActive)
		sentinel := errors.New("update failed")
		h.apiRepo.update = func(*entity.ApiKey) error { return sentinel }

		resp, err := h.uc.Update(context.Background(), 100, 10, &dto.UpdateApiKeyRequest{})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("valid status transitions", func(t *testing.T) {
		cases := []string{
			entity.ApiKeyStatusActive,
			entity.ApiKeyStatusDisabled,
			entity.ApiKeyStatusRevoked,
		}
		for _, s := range cases {
			h := newApiKeyHarness()
			seedApiKey(h, 10, 100, "key", entity.ApiKeyStatusActive)

			status := s
			resp, err := h.uc.Update(context.Background(), 100, 10, &dto.UpdateApiKeyRequest{
				Status: &status,
			})
			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, s, resp.Status)
		}
	})
}

// ---------------------------------------------------------------------------
// ApiKeyUseCase.Delete
// ---------------------------------------------------------------------------

func TestApiKeyUseCase_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newApiKeyHarness()
		seedApiKey(h, 10, 100, "key", entity.ApiKeyStatusActive)

		err := h.uc.Delete(context.Background(), 100, 10)
		require.NoError(t, err)

		_, ok := h.apiRepo.items[10]
		assert.False(t, ok)
	})

	t.Run("not found", func(t *testing.T) {
		h := newApiKeyHarness()
		err := h.uc.Delete(context.Background(), 100, 999)
		requireError(t, err, errno.NotFound)
	})

	t.Run("wrong user", func(t *testing.T) {
		h := newApiKeyHarness()
		seedApiKey(h, 10, 100, "key", entity.ApiKeyStatusActive)

		err := h.uc.Delete(context.Background(), 999, 10)
		requireError(t, err, errno.NotFound)
	})

	t.Run("repo error on FindByID", func(t *testing.T) {
		h := newApiKeyHarness()
		sentinel := errors.New("db error")
		h.apiRepo.findByID = func(int64) (*entity.ApiKey, error) { return nil, sentinel }

		err := h.uc.Delete(context.Background(), 100, 1)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
	})

	t.Run("repo error on Delete", func(t *testing.T) {
		h := newApiKeyHarness()
		seedApiKey(h, 10, 100, "key", entity.ApiKeyStatusActive)
		sentinel := errors.New("delete failed")
		h.apiRepo.del = func(int64) error { return sentinel }

		err := h.uc.Delete(context.Background(), 100, 10)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
	})
}

// ---------------------------------------------------------------------------
// ApiKeyUseCase.Regenerate
// ---------------------------------------------------------------------------

func TestApiKeyUseCase_Regenerate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newApiKeyHarness()
		seedApiKey(h, 10, 100, "key", entity.ApiKeyStatusDisabled)

		resp, err := h.uc.Regenerate(context.Background(), 100, 10)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.PlainKey)
		assert.Contains(t, resp.PlainKey, "sk_")
		assert.Equal(t, entity.ApiKeyStatusActive, resp.Status)

		var stored *entity.ApiKey
		for _, k := range h.apiRepo.items {
			if k.ID == 10 {
				stored = k
			}
		}
		require.NotNil(t, stored)
		assert.Equal(t, entity.ApiKeyStatusActive, stored.Status)
	})

	t.Run("not found", func(t *testing.T) {
		h := newApiKeyHarness()
		resp, err := h.uc.Regenerate(context.Background(), 100, 999)
		requireError(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("wrong user", func(t *testing.T) {
		h := newApiKeyHarness()
		seedApiKey(h, 10, 100, "key", entity.ApiKeyStatusActive)

		resp, err := h.uc.Regenerate(context.Background(), 999, 10)
		requireError(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("repo error on FindByID", func(t *testing.T) {
		h := newApiKeyHarness()
		sentinel := errors.New("db error")
		h.apiRepo.findByID = func(int64) (*entity.ApiKey, error) { return nil, sentinel }

		resp, err := h.uc.Regenerate(context.Background(), 100, 1)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("repo error on Update", func(t *testing.T) {
		h := newApiKeyHarness()
		seedApiKey(h, 10, 100, "key", entity.ApiKeyStatusActive)
		sentinel := errors.New("update failed")
		h.apiRepo.update = func(*entity.ApiKey) error { return sentinel }

		resp, err := h.uc.Regenerate(context.Background(), 100, 10)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})
}

// ---------------------------------------------------------------------------
// ApiKeyUseCase.Validate
// ---------------------------------------------------------------------------

func TestApiKeyUseCase_Validate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newApiKeyHarness()
		k := &entity.ApiKey{
			ID:      10,
			UserID:  100,
			Name:    "valid-key",
			KeyHash: "hash_valid",
			Status:  entity.ApiKeyStatusActive,
			Scopes:  []byte("[]"),
		}
		h.apiRepo.items[10] = k

		result, err := h.uc.Validate(context.Background(), "hash_valid")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, int64(10), result.ID)
		assert.Equal(t, int64(100), result.UserID)
	})

	t.Run("invalid key hash", func(t *testing.T) {
		h := newApiKeyHarness()
		result, err := h.uc.Validate(context.Background(), "nonexistent_hash")
		requireError(t, err, errno.Unauthorized)
		assert.Nil(t, result)
	})

	t.Run("disabled key rejected", func(t *testing.T) {
		h := newApiKeyHarness()
		k := &entity.ApiKey{
			ID:      10,
			UserID:  100,
			Name:    "disabled-key",
			KeyHash: "hash_disabled",
			Status:  entity.ApiKeyStatusDisabled,
			Scopes:  []byte("[]"),
		}
		h.apiRepo.items[10] = k

		result, err := h.uc.Validate(context.Background(), "hash_disabled")
		requireError(t, err, errno.Unauthorized)
		assert.Nil(t, result)
	})

	t.Run("revoked key rejected", func(t *testing.T) {
		h := newApiKeyHarness()
		k := &entity.ApiKey{
			ID:      10,
			UserID:  100,
			Name:    "revoked-key",
			KeyHash: "hash_revoked",
			Status:  entity.ApiKeyStatusRevoked,
			Scopes:  []byte("[]"),
		}
		h.apiRepo.items[10] = k

		result, err := h.uc.Validate(context.Background(), "hash_revoked")
		requireError(t, err, errno.Unauthorized)
		assert.Nil(t, result)
	})

	t.Run("expired key rejected", func(t *testing.T) {
		h := newApiKeyHarness()
		past := time.Now().Add(-1 * time.Hour)
		k := &entity.ApiKey{
			ID:        10,
			UserID:    100,
			Name:      "expired-key",
			KeyHash:  "hash_expired",
			Status:    entity.ApiKeyStatusActive,
			ExpiresAt: &past,
			Scopes:    []byte("[]"),
		}
		h.apiRepo.items[10] = k

		result, err := h.uc.Validate(context.Background(), "hash_expired")
		requireError(t, err, errno.Unauthorized)
		assert.Nil(t, result)
	})

	t.Run("non-expiring key passes", func(t *testing.T) {
		h := newApiKeyHarness()
		k := &entity.ApiKey{
			ID:      10,
			UserID:  100,
			Name:    "no-expiry",
			KeyHash: "hash_no_expiry",
			Status:  entity.ApiKeyStatusActive,
			Scopes:  []byte("[]"),
		}
		h.apiRepo.items[10] = k

		result, err := h.uc.Validate(context.Background(), "hash_no_expiry")
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("repo error", func(t *testing.T) {
		h := newApiKeyHarness()
		sentinel := errors.New("db error")
		h.apiRepo.findByKeyHash = func(string) (*entity.ApiKey, error) { return nil, sentinel }

		result, err := h.uc.Validate(context.Background(), "hash")
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, result)
	})
}