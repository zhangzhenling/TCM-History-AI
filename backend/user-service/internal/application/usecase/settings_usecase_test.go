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

// settingsHarness bundles a SettingsUseCase with its mock dependency.
type settingsHarness struct {
	uc          *usecase.SettingsUseCase
	settingsRepo *mockSettingsRepo
}

func newSettingsHarness() *settingsHarness {
	repo := newMockSettingsRepo()
	uc := usecase.NewSettingsUseCase(repo)
	return &settingsHarness{uc: uc, settingsRepo: repo}
}

func boolPtr(b bool) *bool { return &b }

// ---------------------------------------------------------------------------
// SettingsUseCase.Get
// ---------------------------------------------------------------------------

func TestSettingsUseCase_Get(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newSettingsHarness()
		s := &entity.UserSettings{
			UserID:          300,
			Locale:          "en-US",
			Theme:           "dark",
			NotifyEmail:     false,
			NotifyPush:      true,
			PreferencesJSON: []byte(`{"key":"value"}`),
		}
		h.settingsRepo.items[300] = s

		resp, err := h.uc.Get(context.Background(), 300)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(t, int64(300), resp.UserID)
		assert.Equal(t, "en-US", resp.Locale)
		assert.Equal(t, "dark", resp.Theme)
		assert.False(t, resp.NotifyEmail)
		assert.True(t, resp.NotifyPush)
		assert.Equal(t, json.RawMessage(`{"key":"value"}`), resp.Preferences)
	})

	t.Run("settings nil returns defaults", func(t *testing.T) {
		h := newSettingsHarness()
		resp, err := h.uc.Get(context.Background(), 301)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, int64(301), resp.UserID)
		assert.Equal(t, "zh-CN", resp.Locale)
		assert.Equal(t, "light", resp.Theme)
		assert.True(t, resp.NotifyEmail)
		assert.True(t, resp.NotifyPush)
		assert.Equal(t, json.RawMessage("{}"), resp.Preferences)
	})

	t.Run("dependency error on FindByUserID", func(t *testing.T) {
		h := newSettingsHarness()
		sentinel := errors.New("db down")
		h.settingsRepo.findByUserID = func(int64) (*entity.UserSettings, error) { return nil, sentinel }

		resp, err := h.uc.Get(context.Background(), 1)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("empty preferences normalised to {}", func(t *testing.T) {
		h := newSettingsHarness()
		s := &entity.UserSettings{UserID: 302, Locale: "zh-CN", Theme: "light"}
		h.settingsRepo.items[302] = s

		resp, err := h.uc.Get(context.Background(), 302)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, json.RawMessage("{}"), resp.Preferences)
	})
}

// ---------------------------------------------------------------------------
// SettingsUseCase.Update
// ---------------------------------------------------------------------------

func TestSettingsUseCase_Update(t *testing.T) {
	t.Run("success patches existing settings", func(t *testing.T) {
		h := newSettingsHarness()
		s := &entity.UserSettings{
			UserID:          400,
			Locale:          "zh-CN",
			Theme:           "light",
			NotifyEmail:     true,
			NotifyPush:      true,
			PreferencesJSON: []byte(`{}`),
		}
		h.settingsRepo.items[400] = s

		locale := "en-US"
		theme := "dark"
		resp, err := h.uc.Update(context.Background(), 400, &dto.UpdateSettingsRequest{
			Locale:      &locale,
			Theme:       &theme,
			NotifyEmail: boolPtr(false),
			Preferences: json.RawMessage(`{"k":"v"}`),
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "en-US", resp.Locale)
		assert.Equal(t, "dark", resp.Theme)
		assert.False(t, resp.NotifyEmail)
		assert.Equal(t, json.RawMessage(`{"k":"v"}`), resp.Preferences)

		// Persisted.
		stored, err := h.settingsRepo.FindByUserID(context.Background(), 400)
		require.NoError(t, err)
		require.NotNil(t, stored)
		assert.Equal(t, "en-US", stored.Locale)
		assert.Equal(t, "dark", stored.Theme)
		assert.False(t, stored.NotifyEmail)
		assert.Equal(t, json.RawMessage(`{"k":"v"}`), stored.PreferencesJSON)
	})

	t.Run("nil request rejected", func(t *testing.T) {
		h := newSettingsHarness()
		h.settingsRepo.items[401] = &entity.UserSettings{UserID: 401}
		resp, err := h.uc.Update(context.Background(), 401, nil)
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("settings nil creates a new row with defaults then patches", func(t *testing.T) {
		h := newSettingsHarness()
		// No settings row in the repo.

		locale := "ja-JP"
		resp, err := h.uc.Update(context.Background(), 402, &dto.UpdateSettingsRequest{
			Locale: &locale,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "ja-JP", resp.Locale)
		// Defaults carried over.
		assert.Equal(t, "light", resp.Theme)
		assert.True(t, resp.NotifyEmail)

		// Row was created and persisted.
		stored, err := h.settingsRepo.FindByUserID(context.Background(), 402)
		require.NoError(t, err)
		require.NotNil(t, stored)
		assert.Equal(t, "ja-JP", stored.Locale)
	})

	t.Run("notify_push patched independently", func(t *testing.T) {
		h := newSettingsHarness()
		h.settingsRepo.items[403] = &entity.UserSettings{
			UserID:      403,
			Locale:      "zh-CN",
			Theme:       "light",
			NotifyEmail: true,
			NotifyPush:  true,
		}

		resp, err := h.uc.Update(context.Background(), 403, &dto.UpdateSettingsRequest{
			NotifyPush: boolPtr(false),
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.False(t, resp.NotifyPush)
		assert.True(t, resp.NotifyEmail) // unchanged
	})

	t.Run("dependency error on FindByUserID", func(t *testing.T) {
		h := newSettingsHarness()
		sentinel := errors.New("db down")
		h.settingsRepo.findByUserID = func(int64) (*entity.UserSettings, error) { return nil, sentinel }

		locale := "en-US"
		resp, err := h.uc.Update(context.Background(), 404, &dto.UpdateSettingsRequest{Locale: &locale})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("dependency error on Update", func(t *testing.T) {
		h := newSettingsHarness()
		h.settingsRepo.items[405] = &entity.UserSettings{UserID: 405}
		sentinel := errors.New("update failed")
		h.settingsRepo.update = func(*entity.UserSettings) error { return sentinel }

		locale := "en-US"
		resp, err := h.uc.Update(context.Background(), 405, &dto.UpdateSettingsRequest{Locale: &locale})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("dependency error on Create when settings nil", func(t *testing.T) {
		h := newSettingsHarness()
		sentinel := errors.New("create failed")
		h.settingsRepo.create = func(*entity.UserSettings) error { return sentinel }

		locale := "en-US"
		resp, err := h.uc.Update(context.Background(), 406, &dto.UpdateSettingsRequest{Locale: &locale})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})
}
