package dto

// CreateTenantRequest is the payload for POST /api/v1/admin/tenants.
type CreateTenantRequest struct {
	Name      string `json:"name"`
	Code      string `json:"code"`
	Plan      string `json:"plan"`
	MaxUsers  int    `json:"max_users"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

// UpdateTenantRequest is the payload for PUT /api/v1/admin/tenants/:id.
type UpdateTenantRequest struct {
	Name      *string `json:"name,omitempty"`
	Plan      *string `json:"plan,omitempty"`
	Status    *string `json:"status,omitempty"`
	MaxUsers  *int    `json:"max_users,omitempty"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

// TenantResponse is the wire representation of a tenant.
type TenantResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	Plan      string `json:"plan"`
	Status    string `json:"status"`
	MaxUsers  int    `json:"max_users"`
	ExpiresAt string `json:"expires_at,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// TenantListResponse is the paginated tenant list envelope.
type TenantListResponse struct {
	Page      int              `json:"page"`
	PageSize  int              `json:"page_size"`
	Total     int              `json:"total"`
	TotalPage int              `json:"total_page"`
	Items     []TenantResponse `json:"items"`
}

// AddMemberRequest is the payload for POST /api/v1/admin/tenants/:id/members.
type AddMemberRequest struct {
	UserID    int64  `json:"user_id"`
	Role      string `json:"role"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

// MemberResponse is the wire representation of a tenant membership.
type MemberResponse struct {
	ID        int64  `json:"id"`
	TenantID  int64  `json:"tenant_id"`
	UserID    int64  `json:"user_id"`
	Role      string `json:"role"`
	JoinedAt  string `json:"joined_at"`
	ExpiredAt string `json:"expired_at,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// MemberListResponse is the tenant member list envelope.
type MemberListResponse struct {
	TenantID int64            `json:"tenant_id"`
	Total    int              `json:"total"`
	Items    []MemberResponse `json:"items"`
}

// UserTenantsResponse is the response for "which tenants does this user belong
// to".
type UserTenantsResponse struct {
	UserID int64            `json:"user_id"`
	Total  int              `json:"total"`
	Items  []MemberResponse `json:"items"`
}
