package dto

type CreateApiKeyRequest struct {
	Name               string   `json:"name"`
	Scopes             []string `json:"scopes,omitempty"`
	QuotaDaily         int64    `json:"quota_daily,omitempty"`
	QuotaMonthly       int64    `json:"quota_monthly,omitempty"`
	RateLimitPerMinute int      `json:"rate_limit_per_minute,omitempty"`
	ExpiresAt          string   `json:"expires_at,omitempty"`
}

type ApiKeyResponse struct {
	ID                 int64    `json:"id"`
	Name               string   `json:"name"`
	KeyPrefix          string   `json:"key_prefix"`
	Scopes             []string `json:"scopes"`
	QuotaDaily         int64    `json:"quota_daily"`
	QuotaMonthly       int64    `json:"quota_monthly"`
	RateLimitPerMinute int      `json:"rate_limit_per_minute"`
	Status             string   `json:"status"`
	LastUsedAt         string   `json:"last_used_at,omitempty"`
	ExpiresAt          string   `json:"expires_at,omitempty"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
}

type ApiKeyListResponse struct {
	Page      int              `json:"page"`
	PageSize  int              `json:"page_size"`
	Total     int              `json:"total"`
	TotalPage int              `json:"total_page"`
	Items     []ApiKeyResponse `json:"items"`
}

type UpdateApiKeyRequest struct {
	Name               *string  `json:"name,omitempty"`
	Scopes             []string `json:"scopes,omitempty"`
	QuotaDaily         *int64   `json:"quota_daily,omitempty"`
	QuotaMonthly       *int64   `json:"quota_monthly,omitempty"`
	RateLimitPerMinute *int     `json:"rate_limit_per_minute,omitempty"`
	Status             *string  `json:"status,omitempty"`
}

type RegenerateResponse struct {
	ApiKeyResponse
	PlainKey string `json:"plain_key"`
}
