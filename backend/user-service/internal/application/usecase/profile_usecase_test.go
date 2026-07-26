package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/application/usecase"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// profileHarness bundles a ProfileUseCase with its mock dependencies.
type profileHarness struct {
	uc          *usecase.ProfileUseCase
	userRepo    *mockUserRepo
	profileRepo *mockProfileRepo
	pub         *mockEventPublisher
}

func newProfileHarness() *profileHarness {
	userRepo := newMockUserRepo()
	profileRepo := newMockProfileRepo()
	pub := newMockEventPublisher()
	uc := usecase.NewProfileUseCase(userRepo, profileRepo, pub)
	return &profileHarness{uc: uc, userRepo: userRepo, profileRepo: profileRepo, pub: pub}
}

// seedUserWithProfile inserts a user + profile pair into the mocks and returns
// both.
func seedUserWithProfile(h *profileHarness, userID int64, username string) (*entity.User, *entity.UserProfile) {
	u := &entity.User{Username: username, Status: entity.StatusActive}
	u.ID = userID
	h.userRepo.items[userID] = u
	p := &entity.UserProfile{UserID: userID, Nickname: "nick", Gender: entity.GenderMale}
	h.profileRepo.items[userID] = p
	return u, p
}

// ---------------------------------------------------------------------------
// ProfileUseCase.Get
// ---------------------------------------------------------------------------

func TestProfileUseCase_Get(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := newProfileHarness()
		u, p := seedUserWithProfile(h, 100, "alice")
		email := "alice@example.com"
		u.Email = &email
		p.AvatarURL = "https://cdn/avatar.png"
		p.Bio = "hello"

		resp, err := h.uc.Get(context.Background(), 100)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(t, u.ID, resp.UserID)
		assert.Equal(t, "alice", resp.Username)
		assert.Equal(t, "alice@example.com", resp.Email)
		assert.Equal(t, entity.StatusActive, resp.Status)
		assert.Equal(t, "nick", resp.Nickname)
		assert.Equal(t, "https://cdn/avatar.png", resp.AvatarURL)
		assert.Equal(t, entity.GenderMale, resp.Gender)
		assert.Equal(t, "hello", resp.Bio)
	})

	t.Run("user not found", func(t *testing.T) {
		h := newProfileHarness()
		resp, err := h.uc.Get(context.Background(), 404)
		requireError(t, err, errno.NotFound)
		assert.Nil(t, resp)
	})

	t.Run("profile nil returns defensive default", func(t *testing.T) {
		h := newProfileHarness()
		u := &entity.User{Username: "bob", Status: entity.StatusActive}
		u.ID = 101
		h.userRepo.items[101] = u
		// No profile row in the repo.

		resp, err := h.uc.Get(context.Background(), 101)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "bob", resp.Username)
		assert.Empty(t, resp.Nickname)
		assert.Empty(t, resp.Gender)
	})

	t.Run("dependency error on FindByID", func(t *testing.T) {
		h := newProfileHarness()
		sentinel := errors.New("db down")
		h.userRepo.findByID = func(int64) (*entity.User, error) { return nil, sentinel }

		resp, err := h.uc.Get(context.Background(), 1)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("dependency error on profile FindByUserID", func(t *testing.T) {
		h := newProfileHarness()
		u := &entity.User{Username: "carol", Status: entity.StatusActive}
		u.ID = 102
		h.userRepo.items[102] = u
		sentinel := errors.New("profile db down")
		h.profileRepo.findByUserID = func(int64) (*entity.UserProfile, error) { return nil, sentinel }

		resp, err := h.uc.Get(context.Background(), 102)
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})
}

// ---------------------------------------------------------------------------
// ProfileUseCase.Update
// ---------------------------------------------------------------------------

func TestProfileUseCase_Update(t *testing.T) {
	t.Run("success patches existing profile", func(t *testing.T) {
		h := newProfileHarness()
		seedUserWithProfile(h, 200, "alice")

		nickname := "newnick"
		gender := entity.GenderFemale
		bio := "updated bio"
		resp, err := h.uc.Update(context.Background(), 200, &dto.UpdateProfileRequest{
			Nickname: &nickname,
			Gender:   &gender,
			Bio:      &bio,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "newnick", resp.Nickname)
		assert.Equal(t, entity.GenderFemale, resp.Gender)
		assert.Equal(t, "updated bio", resp.Bio)

		// Persisted in the repo.
		stored, err := h.profileRepo.FindByUserID(context.Background(), 200)
		require.NoError(t, err)
		require.NotNil(t, stored)
		assert.Equal(t, "newnick", stored.Nickname)
		assert.Equal(t, entity.GenderFemale, stored.Gender)
		assert.Equal(t, "updated bio", stored.Bio)
	})

	t.Run("nil request rejected", func(t *testing.T) {
		h := newProfileHarness()
		seedUserWithProfile(h, 201, "alice")
		resp, err := h.uc.Update(context.Background(), 201, nil)
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("profile nil creates a new row", func(t *testing.T) {
		h := newProfileHarness()
		u := &entity.User{Username: "newbie", Status: entity.StatusActive}
		u.ID = 202
		h.userRepo.items[202] = u
		// No profile row.

		nickname := "firstnick"
		resp, err := h.uc.Update(context.Background(), 202, &dto.UpdateProfileRequest{
			Nickname: &nickname,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "firstnick", resp.Nickname)

		// Row was created.
		stored, err := h.profileRepo.FindByUserID(context.Background(), 202)
		require.NoError(t, err)
		require.NotNil(t, stored)
		assert.Equal(t, "firstnick", stored.Nickname)
	})

	t.Run("invalid birth_date", func(t *testing.T) {
		h := newProfileHarness()
		seedUserWithProfile(h, 203, "alice")

		bad := "not-a-date"
		resp, err := h.uc.Update(context.Background(), 203, &dto.UpdateProfileRequest{
			BirthDate: &bad,
		})
		requireError(t, err, errno.InvalidParams)
		assert.Nil(t, resp)
	})

	t.Run("valid birth_date parsed", func(t *testing.T) {
		h := newProfileHarness()
		seedUserWithProfile(h, 204, "alice")

		bd := "1990-01-02T00:00:00Z"
		resp, err := h.uc.Update(context.Background(), 204, &dto.UpdateProfileRequest{
			BirthDate: &bd,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Contains(t, resp.BirthDate, "1990-01-02")
	})

	t.Run("dependency error on FindByUserID", func(t *testing.T) {
		h := newProfileHarness()
		seedUserWithProfile(h, 205, "alice")
		sentinel := errors.New("profile db down")
		h.profileRepo.findByUserID = func(int64) (*entity.UserProfile, error) { return nil, sentinel }

		nickname := "x"
		resp, err := h.uc.Update(context.Background(), 205, &dto.UpdateProfileRequest{Nickname: &nickname})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("dependency error on Update", func(t *testing.T) {
		h := newProfileHarness()
		seedUserWithProfile(h, 206, "alice")
		sentinel := errors.New("update failed")
		h.profileRepo.update = func(*entity.UserProfile) error { return sentinel }

		nickname := "x"
		resp, err := h.uc.Update(context.Background(), 206, &dto.UpdateProfileRequest{Nickname: &nickname})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("dependency error on Create when profile nil", func(t *testing.T) {
		h := newProfileHarness()
		u := &entity.User{Username: "newbie", Status: entity.StatusActive}
		u.ID = 207
		h.userRepo.items[207] = u
		sentinel := errors.New("create failed")
		h.profileRepo.create = func(*entity.UserProfile) error { return sentinel }

		nickname := "x"
		resp, err := h.uc.Update(context.Background(), 207, &dto.UpdateProfileRequest{Nickname: &nickname})
		require.Error(t, err)
		assert.ErrorIs(t, err, sentinel)
		assert.Nil(t, resp)
	})

	t.Run("avatar_url and all fields patched together", func(t *testing.T) {
		h := newProfileHarness()
		seedUserWithProfile(h, 208, "alice")

		avatar := "https://cdn/new.png"
		gender := entity.GenderUnknown
		resp, err := h.uc.Update(context.Background(), 208, &dto.UpdateProfileRequest{
			AvatarURL: &avatar,
			Gender:    &gender,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "https://cdn/new.png", resp.AvatarURL)
		assert.Equal(t, entity.GenderUnknown, resp.Gender)
	})
}
