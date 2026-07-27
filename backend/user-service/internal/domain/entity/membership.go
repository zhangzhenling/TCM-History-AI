package entity

import (
	"encoding/json"
	"time"
)

const (
	SubscriptionStatusActive    = "active"
	SubscriptionStatusExpired   = "expired"
	SubscriptionStatusCancelled = "cancelled"

	OrderStatusPending  = "pending"
	OrderStatusPaid     = "paid"
	OrderStatusRefunded = "refunded"
)

type MembershipPlan struct {
	ID                int64           `gorm:"column:id;type:bigint;primaryKey;autoIncrement:false" json:"id"`
	Name              string          `gorm:"column:name;type:varchar(128);not null" json:"name"`
	PriceCents        int64           `gorm:"column:price_cents;type:bigint;not null;default:0" json:"price_cents"`
	DurationDays      int             `gorm:"column:duration_days;type:integer;not null;default:30" json:"duration_days"`
	MaxDailyAIRequests int            `gorm:"column:max_daily_ai_requests;type:integer;not null;default:0" json:"max_daily_ai_requests"`
	MaxTokenPerMonth  int64           `gorm:"column:max_token_per_month;type:bigint;not null;default:0" json:"max_token_per_month"`
	Features          json.RawMessage `gorm:"column:features;type:jsonb" json:"features,omitempty"`
	IsActive          bool            `gorm:"column:is_active;type:boolean;not null;default:true" json:"is_active"`
	SortOrder         int             `gorm:"column:sort_order;type:integer;not null;default:0" json:"sort_order"`
	CreatedAt         time.Time       `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt         time.Time       `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (MembershipPlan) TableName() string { return "membership_plans" }

type UserSubscription struct {
	ID          int64      `gorm:"column:id;type:bigint;primaryKey;autoIncrement:false" json:"id"`
	UserID      int64      `gorm:"column:user_id;type:bigint;not null;index:idx_user_subscriptions_user_id" json:"user_id"`
	PlanID      int64      `gorm:"column:plan_id;type:bigint;not null" json:"plan_id"`
	Status      string     `gorm:"column:status;type:varchar(32);not null;default:active" json:"status"`
	StartedAt   time.Time  `gorm:"column:started_at;type:timestamptz;not null;default:now()" json:"started_at"`
	ExpiresAt   time.Time  `gorm:"column:expires_at;type:timestamptz;not null" json:"expires_at"`
	AutoRenew   bool       `gorm:"column:auto_renew;type:boolean;not null;default:false" json:"auto_renew"`
	CancelledAt *time.Time `gorm:"column:cancelled_at;type:timestamptz" json:"cancelled_at,omitempty"`
	CreatedAt   time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (UserSubscription) TableName() string { return "user_subscriptions" }

func (s *UserSubscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive && s.ExpiresAt.After(time.Now())
}

func (s *UserSubscription) Extend(days int) {
	s.ExpiresAt = s.ExpiresAt.AddDate(0, 0, days)
}

type MembershipOrder struct {
	ID            int64      `gorm:"column:id;type:bigint;primaryKey;autoIncrement:false" json:"id"`
	UserID        int64      `gorm:"column:user_id;type:bigint;not null;index:idx_membership_orders_user_id" json:"user_id"`
	PlanID        int64      `gorm:"column:plan_id;type:bigint;not null" json:"plan_id"`
	OrderNo       string     `gorm:"column:order_no;type:varchar(64);not null;uniqueIndex:uk_membership_orders_order_no" json:"order_no"`
	AmountCents   int64      `gorm:"column:amount_cents;type:bigint;not null;default:0" json:"amount_cents"`
	Status        string     `gorm:"column:status;type:varchar(32);not null;default:pending" json:"status"`
	PaidAt        *time.Time `gorm:"column:paid_at;type:timestamptz" json:"paid_at,omitempty"`
	PaymentMethod string     `gorm:"column:payment_method;type:varchar(32)" json:"payment_method,omitempty"`
	TransactionID string     `gorm:"column:transaction_id;type:varchar(128)" json:"transaction_id,omitempty"`
	CreatedAt     time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;type:timestamptz;not null;default:now()" json:"updated_at"`
}

func (MembershipOrder) TableName() string { return "membership_orders" }
