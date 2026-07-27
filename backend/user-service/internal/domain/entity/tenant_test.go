package entity_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// TestTenantPlanConstants asserts the wire values of the plan enumeration
// stay stable; downstream code (DB defaults, admin UI) depends on them.
func TestTenantPlanConstants(t *testing.T) {
	assert.Equal(t, "standard", entity.PlanStandard)
	assert.Equal(t, "premium", entity.PlanPremium)
	assert.Equal(t, "enterprise", entity.PlanEnterprise)
}

// TestTenantStatusConstants asserts the wire values of the status
// enumeration stay stable.
func TestTenantStatusConstants(t *testing.T) {
	assert.Equal(t, "active", entity.TenantStatusActive)
	assert.Equal(t, "suspended", entity.TenantStatusSuspended)
	assert.Equal(t, "expired", entity.TenantStatusExpired)
}

// TestTenantMemberRoleConstants asserts the wire values of the member role
// enumeration stay stable.
func TestTenantMemberRoleConstants(t *testing.T) {
	assert.Equal(t, "school_admin", entity.TenantRoleSchoolAdmin)
	assert.Equal(t, "teacher", entity.TenantRoleTeacher)
	assert.Equal(t, "student", entity.TenantRoleStudent)
}

// TestTenant_TableName verifies the GORM table overrides.
func TestTenant_TableName(t *testing.T) {
	assert.Equal(t, "tenants", entity.Tenant{}.TableName())
	assert.Equal(t, "tenant_members", entity.TenantMember{}.TableName())
}

// TestTenant_ValidatePlan covers every supported plan plus the rejected cases.
func TestTenant_ValidatePlan(t *testing.T) {
	for _, p := range []string{
		entity.PlanStandard,
		entity.PlanPremium,
		entity.PlanEnterprise,
	} {
		assert.True(t, entity.ValidatePlan(p), "expected %q to be valid", p)
	}
	for _, p := range []string{"", "free", "STANDARD", "pro", "ultimate"} {
		assert.False(t, entity.ValidatePlan(p), "expected %q to be invalid", p)
	}
}

// TestTenant_ValidateStatus covers every supported status plus the rejected cases.
func TestTenant_ValidateStatus(t *testing.T) {
	for _, s := range []string{
		entity.TenantStatusActive,
		entity.TenantStatusSuspended,
		entity.TenantStatusExpired,
	} {
		assert.True(t, entity.ValidateStatus(s), "expected %q to be valid", s)
	}
	for _, s := range []string{"", "pending", "ACTIVE", "deleted"} {
		assert.False(t, entity.ValidateStatus(s), "expected %q to be invalid", s)
	}
}

// TestTenant_ValidateMemberRole covers every supported role plus the rejected
// cases.
func TestTenant_ValidateMemberRole(t *testing.T) {
	for _, r := range []string{
		entity.TenantRoleSchoolAdmin,
		entity.TenantRoleTeacher,
		entity.TenantRoleStudent,
	} {
		assert.True(t, entity.ValidateTenantMemberRole(r), "expected %q to be valid", r)
	}
	for _, r := range []string{"", "admin", "TEACHER", "principal"} {
		assert.False(t, entity.ValidateTenantMemberRole(r), "expected %q to be invalid", r)
	}
}

// TestTenant_IsActive exercises each status branch.
func TestTenant_IsActive(t *testing.T) {
	cases := []struct {
		status string
		active bool
	}{
		{entity.TenantStatusActive, true},
		{entity.TenantStatusSuspended, false},
		{entity.TenantStatusExpired, false},
		{"", false},
		{"bogus", false},
	}
	for _, tc := range cases {
		t.Run(tc.status, func(t *testing.T) {
			tenant := &entity.Tenant{Status: tc.status}
			assert.Equal(t, tc.active, tenant.IsActive())
		})
	}
}

// TestTenant_IsExpired covers the nil-expiry, future-expiry and past-expiry
// branches.
func TestTenant_IsExpired(t *testing.T) {
	t.Run("nil expires_at is never expired", func(t *testing.T) {
		tenant := &entity.Tenant{}
		assert.False(t, tenant.IsExpired())
	})

	t.Run("future expires_at is not expired", func(t *testing.T) {
		future := time.Now().Add(24 * time.Hour)
		tenant := &entity.Tenant{ExpiresAt: &future}
		assert.False(t, tenant.IsExpired())
	})

	t.Run("past expires_at is expired", func(t *testing.T) {
		past := time.Now().Add(-24 * time.Hour)
		tenant := &entity.Tenant{ExpiresAt: &past}
		assert.True(t, tenant.IsExpired())
	})
}

// TestTenantMember_IsMemberActive covers the nil-expiry and past-expiry
// branches.
func TestTenantMember_IsMemberActive(t *testing.T) {
	t.Run("nil expired_at is active", func(t *testing.T) {
		m := &entity.TenantMember{}
		assert.True(t, m.IsMemberActive())
	})

	t.Run("future expired_at is active", func(t *testing.T) {
		future := time.Now().Add(24 * time.Hour)
		m := &entity.TenantMember{ExpiredAt: &future}
		assert.True(t, m.IsMemberActive())
	})

	t.Run("past expired_at is inactive", func(t *testing.T) {
		past := time.Now().Add(-24 * time.Hour)
		m := &entity.TenantMember{ExpiredAt: &past}
		assert.False(t, m.IsMemberActive())
	})
}
