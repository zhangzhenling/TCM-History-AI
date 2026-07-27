package repository

import (
	"context"

	"tcm-history-ai/backend/pkg/pagination"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
)

// TenantRepository is the port for tenants persistence.
type TenantRepository interface {
	Create(ctx context.Context, t *entity.Tenant) error
	Update(ctx context.Context, t *entity.Tenant) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*entity.Tenant, error)
	FindByCode(ctx context.Context, code string) (*entity.Tenant, error)
	List(ctx context.Context, p pagination.Params, status string) ([]entity.Tenant, int64, error)
}

// TenantMemberRepository is the port for tenant_members persistence.
type TenantMemberRepository interface {
	AddMember(ctx context.Context, m *entity.TenantMember) error
	RemoveMember(ctx context.Context, tenantID, userID int64) error
	FindMembers(ctx context.Context, tenantID int64) ([]entity.TenantMember, error)
	FindUserTenants(ctx context.Context, userID int64) ([]entity.TenantMember, error)
	IsMember(ctx context.Context, tenantID, userID int64) (*entity.TenantMember, bool, error)
	CountMembers(ctx context.Context, tenantID int64) (int64, error)
}
