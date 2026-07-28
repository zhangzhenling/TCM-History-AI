package usecase_test

import (
	"context"
	"fmt"
	"testing"

	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// BenchmarkTenantUseCase_AddMember benchmarks adding a member under normal
// conditions (no quota pressure).
func BenchmarkTenantUseCase_AddMember(b *testing.B) {
	h := newTenantHarness()
	t := seedTenant(h.tenantRepo, 1, "bench", 100)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Each iteration needs a fresh user ID to avoid duplicate member errors.
		req := &dto.AddMemberRequest{UserID: int64(1000 + i), Role: "student"}
		_, _ = h.uc.AddMember(ctx, t.ID, req)
	}
}

// BenchmarkTenantUseCase_AddMember_QuotaPressure benchmarks adding members
// when the tenant is near its max_users limit.
func BenchmarkTenantUseCase_AddMember_QuotaPressure(b *testing.B) {
	h := newTenantHarness()
	// Small quota so most iterations hit the limit quickly.
	t := seedTenant(h.tenantRepo, 1, "bench", 5)
	ctx := context.Background()

	// Pre-fill to 4 members so the next add is the last allowed.
	for i := 1; i <= 4; i++ {
		_, _ = h.uc.AddMember(ctx, t.ID, &dto.AddMemberRequest{UserID: int64(i), Role: "student"})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &dto.AddMemberRequest{UserID: int64(1000 + i), Role: "student"}
		_, _ = h.uc.AddMember(ctx, t.ID, req)
	}
}

// BenchmarkTenantUseCase_CreateTenant benchmarks tenant creation.
func BenchmarkTenantUseCase_CreateTenant(b *testing.B) {
	h := newTenantHarness()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		in := &dto.CreateTenantRequest{
			Code:     fmt.Sprintf("bench-%d", i),
			Name:     "Benchmark Tenant",
			Plan:     string(entity.PlanStandard),
			MaxUsers: 100,
		}
		_, _ = h.uc.CreateTenant(ctx, in)
	}
}

