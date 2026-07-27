package dto

type TokenUsageResponse struct {
	ID                 int64  `json:"id"`
	UserID             int64  `json:"user_id"`
	ConversationID     int64  `json:"conversation_id"`
	Model              string `json:"model"`
	Provider           string `json:"provider"`
	PromptTokens       int    `json:"prompt_tokens"`
	CompletionTokens   int    `json:"completion_tokens"`
	TotalTokens        int    `json:"total_tokens"`
	EstimatedCostCents int64  `json:"estimated_cost_cents"`
	CreatedAt          string `json:"created_at"`
}

type QuotaResponse struct {
	UserID          int64  `json:"user_id"`
	Month           string `json:"month"`
	TotalTokens     int64  `json:"total_tokens"`
	UsedTokens      int64  `json:"used_tokens"`
	AvailableTokens int64  `json:"available_tokens"`
	UpdatedAt       string `json:"updated_at"`
}

type UsageSummaryResponse struct {
	Today     UsagePeriodSummary `json:"today"`
	ThisWeek  UsagePeriodSummary `json:"this_week"`
	ThisMonth UsagePeriodSummary `json:"this_month"`
}

type UsagePeriodSummary struct {
	TotalTokens        int   `json:"total_tokens"`
	PromptTokens       int   `json:"prompt_tokens"`
	CompletionTokens   int   `json:"completion_tokens"`
	Requests           int   `json:"requests"`
	EstimatedCostCents int64 `json:"estimated_cost_cents"`
}
