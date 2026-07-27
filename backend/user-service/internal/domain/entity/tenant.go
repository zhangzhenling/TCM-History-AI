package entity

import (
	"time"

	"tcm-history-ai/backend/pkg/gormutil"
)

// Tenant plan enumeration: 套餐档位。
const (
	PlanStandard   = "standard"
	PlanPremium    = "premium"
	PlanEnterprise = "enterprise"
)

// Tenant status enumeration: 租户生命周期状态。
const (
	TenantStatusActive   = "active"
	TenantStatusSuspended = "suspended"
	TenantStatusExpired  = "expired"
)

// Tenant member role enumeration: 院校成员角色.
// The Tenant* prefix avoids collision with the built-in role codes
// (RoleTeacher / RoleStudent) declared in role.go.
const (
	TenantRoleSchoolAdmin = "school_admin"
	TenantRoleTeacher     = "teacher"
	TenantRoleStudent     = "student"
)

// validPlans is the allow-list for Tenant.Plan. Kept as a package-level
// variable so ValidatePlan can be reused by usecase tests without
// re-allocating on every call.
var validPlans = map[string]struct{}{
	PlanStandard:   {},
	PlanPremium:    {},
	PlanEnterprise: {},
}

// validTenantStatuses is the allow-list for Tenant.Status.
var validTenantStatuses = map[string]struct{}{
	TenantStatusActive:    {},
	TenantStatusSuspended: {},
	TenantStatusExpired:   {},
}

// validTenantMemberRoles is the allow-list for TenantMember.Role.
var validTenantMemberRoles = map[string]struct{}{
	TenantRoleSchoolAdmin: {},
	TenantRoleTeacher:     {},
	TenantRoleStudent:     {},
}

// Tenant corresponds to the tenants table.
type Tenant struct {
	gormutil.BaseModel
	Name      string     `gorm:"column:name;type:varchar(128);not null" json:"name"`
	Code      string     `gorm:"column:code;type:varchar(64);not null;uniqueIndex:uk_tenants_code" json:"code"`
	Plan      string     `gorm:"column:plan;type:varchar(32);not null;default:standard" json:"plan"`
	Status    string     `gorm:"column:status;type:varchar(32);not null;default:active;index:idx_tenants_status" json:"status"`
	MaxUsers  int        `gorm:"column:max_users;type:integer;not null;default:0" json:"max_users"`
	ExpiresAt *time.Time `gorm:"column:expires_at;type:timestamptz" json:"expires_at,omitempty"`
}

// TableName overrides the default GORM table name.
func (Tenant) TableName() string { return "tenants" }

// ValidatePlan reports whether the plan value is one of the supported
// commercial tiers (standard / premium / enterprise).
func ValidatePlan(plan string) bool {
	_, ok := validPlans[plan]
	return ok
}

// ValidateStatus reports whether the status value is one of the supported
// tenant lifecycle states (active / suspended / expired).
func ValidateStatus(status string) bool {
	_, ok := validTenantStatuses[status]
	return ok
}

// ValidateTenantMemberRole reports whether the role value is one of the
// supported tenant member roles (school_admin / teacher / student).
func ValidateTenantMemberRole(role string) bool {
	_, ok := validTenantMemberRoles[role]
	return ok
}

// IsActive reports whether the tenant is in the active status.
func (t *Tenant) IsActive() bool { return t.Status == TenantStatusActive }

// IsExpired reports whether the tenant's expiry time has passed. A nil
// ExpiresAt is treated as "no expiry" (returns false).
func (t *Tenant) IsExpired() bool {
	return t.ExpiresAt != nil && t.ExpiresAt.Before(time.Now())
}

// TenantMember corresponds to the tenant_members table.
type TenantMember struct {
	gormutil.BaseModel
	TenantID  int64      `gorm:"column:tenant_id;type:bigint;not null;uniqueIndex:uk_tenant_members_tenant_user" json:"tenant_id"`
	UserID    int64      `gorm:"column:user_id;type:bigint;not null;uniqueIndex:uk_tenant_members_tenant_user" json:"user_id"`
	Role      string     `gorm:"column:role;type:varchar(32);not null;default:student" json:"role"`
	JoinedAt  time.Time  `gorm:"column:joined_at;type:timestamptz;not null;default:now()" json:"joined_at"`
	ExpiredAt *time.Time `gorm:"column:expired_at;type:timestamptz" json:"expired_at,omitempty"`
}

// TableName overrides the default GORM table name.
func (TenantMember) TableName() string { return "tenant_members" }

// IsMemberActive reports whether the membership is currently in effect.
// A nil ExpiredAt means the membership does not expire.
func (m *TenantMember) IsMemberActive() bool {
	return m.ExpiredAt == nil || m.ExpiredAt.After(time.Now())
}
