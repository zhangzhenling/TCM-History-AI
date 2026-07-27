package dto

import (
	"encoding/json"
	"time"
)

type MembershipPlanResponse struct {
	ID                 int64           `json:"id"`
	Name               string          `json:"name"`
	PriceCents         int64           `json:"price_cents"`
	DurationDays       int             `json:"duration_days"`
	MaxDailyAIRequests int             `json:"max_daily_ai_requests"`
	MaxTokenPerMonth   int64           `json:"max_token_per_month"`
	Features           json.RawMessage `json:"features,omitempty"`
	IsActive           bool            `json:"is_active"`
	SortOrder          int             `json:"sort_order"`
	CreatedAt          string          `json:"created_at,omitempty"`
	UpdatedAt          string          `json:"updated_at,omitempty"`
}

type CreateMembershipPlanRequest struct {
	Name               string          `json:"name"`
	PriceCents         int64           `json:"price_cents"`
	DurationDays       int             `json:"duration_days"`
	MaxDailyAIRequests int             `json:"max_daily_ai_requests"`
	MaxTokenPerMonth   int64           `json:"max_token_per_month"`
	Features           json.RawMessage `json:"features,omitempty"`
	IsActive           bool            `json:"is_active"`
	SortOrder          int             `json:"sort_order"`
}

type UpdateMembershipPlanRequest struct {
	Name               *string          `json:"name,omitempty"`
	PriceCents         *int64           `json:"price_cents,omitempty"`
	DurationDays       *int             `json:"duration_days,omitempty"`
	MaxDailyAIRequests *int             `json:"max_daily_ai_requests,omitempty"`
	MaxTokenPerMonth   *int64           `json:"max_token_per_month,omitempty"`
	Features           *json.RawMessage `json:"features,omitempty"`
	IsActive           *bool            `json:"is_active,omitempty"`
	SortOrder          *int             `json:"sort_order,omitempty"`
}

type UserSubscriptionResponse struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	PlanID      int64  `json:"plan_id"`
	PlanName    string `json:"plan_name,omitempty"`
	Status      string `json:"status"`
	StartedAt   string `json:"started_at"`
	ExpiresAt   string `json:"expires_at"`
	AutoRenew   bool   `json:"auto_renew"`
	CancelledAt string `json:"cancelled_at,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

type SubscribeRequest struct {
	PlanID int64 `json:"plan_id"`
}

type MembershipOrderResponse struct {
	ID            int64  `json:"id"`
	UserID        int64  `json:"user_id"`
	PlanID        int64  `json:"plan_id"`
	PlanName      string `json:"plan_name,omitempty"`
	OrderNo       string `json:"order_no"`
	AmountCents   int64  `json:"amount_cents"`
	Status        string `json:"status"`
	PaidAt        string `json:"paid_at,omitempty"`
	PaymentMethod string `json:"payment_method,omitempty"`
	TransactionID string `json:"transaction_id,omitempty"`
	CreatedAt     string `json:"created_at,omitempty"`
	UpdatedAt     string `json:"updated_at,omitempty"`
}

type PaymentCallbackRequest struct {
	OrderNo       string `json:"order_no"`
	TransactionID string `json:"transaction_id"`
	PaymentMethod string `json:"payment_method"`
	Status        string `json:"status"`
	Signature     string `json:"signature"`
}

func formatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
