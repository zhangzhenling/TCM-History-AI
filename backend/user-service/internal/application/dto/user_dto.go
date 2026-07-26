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

// AdminUserResponse is the admin view of a user (includes status, roles,
// timestamps). Used by admin list/get endpoints.
type AdminUserResponse struct {
	ID          int64    `json:"id"`
	Username    string   `json:"username"`
	Email       string   `json:"email,omitempty"`
	Phone       string   `json:"phone,omitempty"`
	Status      string   `json:"status"`
	Roles       []string `json:"roles,omitempty"`
	LastLoginAt string   `json:"last_login_at,omitempty"`
	LastLoginIP string   `json:"last_login_ip,omitempty"`
	CreatedAt   string   `json:"created_at,omitempty"`
	UpdatedAt   string   `json:"updated_at,omitempty"`
}

// UpdateUserStatusRequest is the payload for PATCH /admin/users/:id/status.
type UpdateUserStatusRequest struct {
	Status string `json:"status"`
}

// AssignRolesRequest is the payload for PUT /admin/users/:id/roles.
type AssignRolesRequest struct {
	RoleIDs []int64 `json:"role_ids"`
}

// RoleResponse is the wire representation of a role.
type RoleResponse struct {
	ID          int64  `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsBuiltin   bool   `json:"is_builtin"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

// RoleDetailResponse extends RoleResponse with the list of permission codes.
type RoleDetailResponse struct {
	RoleResponse
	Permissions []string `json:"permissions,omitempty"`
}

// CreateRoleRequest is the payload for POST /admin/roles.
type CreateRoleRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// UpdateRoleRequest is the payload for PUT /admin/roles/:id.
type UpdateRoleRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// AssignPermissionsRequest is the payload for PUT /admin/roles/:id/permissions.
type AssignPermissionsRequest struct {
	PermissionIDs []int64 `json:"permission_ids"`
}

// PermissionResponse is the wire representation of a permission.
type PermissionResponse struct {
	ID          int64  `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description,omitempty"`
}
