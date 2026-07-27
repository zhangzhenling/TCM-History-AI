package persistence

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
	"tcm-history-ai/backend/user-service/internal/domain/repository"
)

// TenantRepo implements repository.TenantRepository with GORM.
type TenantRepo struct {
	baseRepo
}

// NewTenantRepo constructs a TenantRepo.
func NewTenantRepo(db *gorm.DB) *TenantRepo {
	return &TenantRepo{baseRepo{db: db}}
}

// Ensure TenantRepo satisfies the interface at compile time.
var _ repository.TenantRepository = (*TenantRepo)(nil)

// Create inserts a new tenant row.
func (r *TenantRepo) Create(ctx context.Context, t *entity.Tenant) error {
	if err := txFrom(ctx, r.db).Create(t).Error; err != nil {
		return errno.Wrap(errno.InternalError, "create tenant", err)
	}
	return nil
}

// Update saves changes to an existing tenant row.
func (r *TenantRepo) Update(ctx context.Context, t *entity.Tenant) error {
	if err := txFrom(ctx, r.db).Save(t).Error; err != nil {
		return errno.Wrap(errno.InternalError, "update tenant", err)
	}
	return nil
}

// Delete soft-deletes a tenant by id.
func (r *TenantRepo) Delete(ctx context.Context, id int64) error {
	res := txFrom(ctx, r.db).Where("id = ?", id).Delete(&entity.Tenant{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "delete tenant", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "tenant not found")
	}
	return nil
}

// FindByID fetches a single tenant by id; returns (nil, nil) when not found.
func (r *TenantRepo) FindByID(ctx context.Context, id int64) (*entity.Tenant, error) {
	var t entity.Tenant
	err := txFrom(ctx, r.db).First(&t, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find tenant by id", err)
	}
	return &t, nil
}

// FindByCode fetches a tenant by code; returns (nil, nil) when not found.
func (r *TenantRepo) FindByCode(ctx context.Context, code string) (*entity.Tenant, error) {
	var t entity.Tenant
	err := txFrom(ctx, r.db).First(&t, "code = ?", code).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errno.Wrap(errno.InternalError, "find tenant by code", err)
	}
	return &t, nil
}

// List returns paginated tenants, optionally filtered by status.
func (r *TenantRepo) List(ctx context.Context, p pagination.Params, status string) ([]entity.Tenant, int64, error) {
	q := txFrom(ctx, r.db).Model(&entity.Tenant{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "count tenants", err)
	}
	var tenants []entity.Tenant
	_, limit, offset := p.Normalise()
	if limit > 100 {
		limit = 100
		page, _, _ := p.Normalise()
		offset = (page - 1) * limit
	}
	if err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&tenants).Error; err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "list tenants", err)
	}
	return tenants, total, nil
}

// TenantMemberRepo implements repository.TenantMemberRepository with GORM.
type TenantMemberRepo struct {
	baseRepo
}

// NewTenantMemberRepo constructs a TenantMemberRepo.
func NewTenantMemberRepo(db *gorm.DB) *TenantMemberRepo {
	return &TenantMemberRepo{baseRepo{db: db}}
}

// Ensure TenantMemberRepo satisfies the interface at compile time.
var _ repository.TenantMemberRepository = (*TenantMemberRepo)(nil)

// AddMember inserts a new tenant_members row.
func (r *TenantMemberRepo) AddMember(ctx context.Context, m *entity.TenantMember) error {
	if err := txFrom(ctx, r.db).Create(m).Error; err != nil {
		return errno.Wrap(errno.InternalError, "add tenant member", err)
	}
	return nil
}

// RemoveMember soft-deletes a tenant_members row by (tenant_id, user_id).
func (r *TenantMemberRepo) RemoveMember(ctx context.Context, tenantID, userID int64) error {
	res := txFrom(ctx, r.db).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Delete(&entity.TenantMember{})
	if res.Error != nil {
		return errno.Wrap(errno.InternalError, "remove tenant member", res.Error)
	}
	if res.RowsAffected == 0 {
		return errno.New(errno.NotFound, "tenant member not found")
	}
	return nil
}

// FindMembers returns all members for the given tenant.
func (r *TenantMemberRepo) FindMembers(ctx context.Context, tenantID int64) ([]entity.TenantMember, error) {
	var members []entity.TenantMember
	if err := txFrom(ctx, r.db).
		Where("tenant_id = ?", tenantID).
		Order("joined_at ASC, id ASC").
		Find(&members).Error; err != nil {
		return nil, errno.Wrap(errno.InternalError, "find tenant members", err)
	}
	return members, nil
}

// FindUserTenants returns all tenant memberships for the given user.
func (r *TenantMemberRepo) FindUserTenants(ctx context.Context, userID int64) ([]entity.TenantMember, error) {
	var members []entity.TenantMember
	if err := txFrom(ctx, r.db).
		Where("user_id = ?", userID).
		Order("joined_at DESC, id DESC").
		Find(&members).Error; err != nil {
		return nil, errno.Wrap(errno.InternalError, "find user tenants", err)
	}
	return members, nil
}

// IsMember reports whether a user belongs to a tenant. Returns the
// membership row (so the caller can inspect role / expiry) and a bool.
func (r *TenantMemberRepo) IsMember(ctx context.Context, tenantID, userID int64) (*entity.TenantMember, bool, error) {
	var m entity.TenantMember
	err := txFrom(ctx, r.db).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, errno.Wrap(errno.InternalError, "check tenant membership", err)
	}
	return &m, true, nil
}

// CountMembers returns the number of active members for the given tenant.
func (r *TenantMemberRepo) CountMembers(ctx context.Context, tenantID int64) (int64, error) {
	var total int64
	if err := txFrom(ctx, r.db).
		Model(&entity.TenantMember{}).
		Where("tenant_id = ?", tenantID).
		Count(&total).Error; err != nil {
		return 0, errno.Wrap(errno.InternalError, "count tenant members", err)
	}
	return total, nil
}
