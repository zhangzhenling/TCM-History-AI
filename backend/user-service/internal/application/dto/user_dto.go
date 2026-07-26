package dto

import (
	"encoding/json"
	"time"
)

// UpdateProfileRequest is the payload for PUT /api/v1/users/me.
type UpdateProfileRequest struct {
	Nickname  *string `json:"nickname,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	Gender    *string `json:"gender,omitempty"`
	BirthDate *string `json:"birth_date,omitempty"` // RFC3339
	Bio       *string `json:"bio,omitempty"`
}

// ProfileResponse is the wire representation of a user profile.
type ProfileResponse struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	Email     string `json:"email,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Status    string `json:"status"`
	Nickname  string `json:"nickname,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Gender    string `json:"gender,omitempty"`
	BirthDate string `json:"birth_date,omitempty"`
	Bio       string `json:"bio,omitempty"`
}

// UpdateSettingsRequest is the payload for PUT /api/v1/users/settings.
type UpdateSettingsRequest struct {
	Locale      *string         `json:"locale,omitempty"`
	Theme       *string         `json:"theme,omitempty"`
	NotifyEmail *bool           `json:"notify_email,omitempty"`
	NotifyPush  *bool           `json:"notify_push,omitempty"`
	Preferences json.RawMessage `json:"preferences,omitempty"`
}

// SettingsResponse is the wire representation of user settings.
type SettingsResponse struct {
	UserID      int64           `json:"user_id"`
	Locale      string          `json:"locale"`
	Theme       string          `json:"theme"`
	NotifyEmail bool            `json:"notify_email"`
	NotifyPush  bool            `json:"notify_push"`
	Preferences json.RawMessage `json:"preferences"`
	UpdatedAt   time.Time       `json:"updated_at"`
}
