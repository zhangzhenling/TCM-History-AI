package usecase

import (
	"context"
	"strconv"
	"time"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
	"tcm-history-ai/backend/user-service/internal/application/dto"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
	"tcm-history-ai/backend/user-service/internal/domain/repository"
)

// TenantUseCase provides admin-driven operations for the school multi-tenant
// edition: tenant CRUD and tenant member management.
type TenantUseCase struct {
	tenantRepo repository.TenantRepository
	memberRepo repository.TenantMemberRepository
}

// NewTenantUseCase constructs a TenantUseCase.
func NewTenantUseCase(tenantRepo repository.TenantRepository, memberRepo repository.TenantMemberRepository) *TenantUseCase {
	return &TenantUseCase{
		tenantRepo: tenantRepo,
		memberRepo: memberRepo,
	}
}

// CreateTenant creates a new tenant after validating uniqueness of code and
// the plan/status enumerations.
func (uc *TenantUseCase) CreateTenant(ctx context.Context, in *dto.CreateTenantRequest) (*dto.TenantResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "request body is required")
	}
	if in.Name == "" {
		return nil, errno.New(errno.InvalidParams, "name is required")
	}
	if in.Code == "" {
		return nil, errno.New(errno.InvalidParams, "code is required")
	}
	plan := in.Plan
	if plan == "" {
		plan = entity.PlanStandard
	}
	if !entity.ValidatePlan(plan) {
		return nil, errno.New(errno.InvalidParams, "invalid plan: "+plan)
	}
	if in.MaxUsers < 0 {
		return nil, errno.New(errno.InvalidParams, "max_users must be >= 0")
	}

	// Uniqueness check on code (treat existing row as conflict even if soft-
	// deleted; the unique index is partial on deleted_at IS NULL but a code
	// reuse is still a red flag the use case wants to surface).
	existing, err := uc.tenantRepo.FindByCode(ctx, in.Code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errno.New(errno.AlreadyExists, "tenant code already exists: "+in.Code)
	}

	var expiresAt *time.Time
	if in.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, in.ExpiresAt)
		if err != nil {
			return nil, errno.New(errno.InvalidParams, "invalid expires_at format")
		}
		expiresAt = &t
	}

	tenant := &entity.Tenant{
		Name:      in.Name,
		Code:      in.Code,
		Plan:      plan,
		Status:    entity.TenantStatusActive,
		MaxUsers:  in.MaxUsers,
		ExpiresAt: expiresAt,
	}
	tenant.ID = idgen.Next()

	if err := uc.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, err
	}
	resp := toTenantResponse(tenant)
	return &resp, nil
}

// UpdateTenant patches an existing tenant.
func (uc *TenantUseCase) UpdateTenant(ctx context.Context, id int64, in *dto.UpdateTenantRequest) (*dto.TenantResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "request body is required")
	}
	tenant, err := uc.tenantRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, errno.New(errno.NotFound, "tenant not found: "+strconv.FormatInt(id, 10))
	}

	if in.Name != nil {
		if *in.Name == "" {
			return nil, errno.New(errno.InvalidParams, "name must not be empty")
		}
		tenant.Name = *in.Name
	}
	if in.Plan != nil {
		if !entity.ValidatePlan(*in.Plan) {
			return nil, errno.New(errno.InvalidParams, "invalid plan: "+*in.Plan)
		}
		tenant.Plan = *in.Plan
	}
	if in.Status != nil {
		if !entity.ValidateStatus(*in.Status) {
			return nil, errno.New(errno.InvalidParams, "invalid status: "+*in.Status)
		}
		tenant.Status = *in.Status
	}
	if in.MaxUsers != nil {
		if *in.MaxUsers < 0 {
			return nil, errno.New(errno.InvalidParams, "max_users must be >= 0")
		}
		tenant.MaxUsers = *in.MaxUsers
	}
	if in.ExpiresAt != nil {
		if *in.ExpiresAt == "" {
			tenant.ExpiresAt = nil
		} else {
			t, err := time.Parse(time.RFC3339, *in.ExpiresAt)
			if err != nil {
				return nil, errno.New(errno.InvalidParams, "invalid expires_at format")
			}
			tenant.ExpiresAt = &t
		}
	}

	if err := uc.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, err
	}
	resp := toTenantResponse(tenant)
	return &resp, nil
}

// GetTenant returns a single tenant by id.
func (uc *TenantUseCase) GetTenant(ctx context.Context, id int64) (*dto.TenantResponse, error) {
	tenant, err := uc.tenantRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, errno.New(errno.NotFound, "tenant not found: "+strconv.FormatInt(id, 10))
	}
	resp := toTenantResponse(tenant)
	return &resp, nil
}

// ListTenants returns paginated tenants, optionally filtered by status.
func (uc *TenantUseCase) ListTenants(ctx context.Context, p pagination.Params, status string) (dto.TenantListResponse, error) {
	tenants, total, err := uc.tenantRepo.List(ctx, p, status)
	if err != nil {
		return dto.TenantListResponse{}, err
	}
	page, pageSize, _ := p.Normalise()
	items := make([]dto.TenantResponse, 0, len(tenants))
	for i := range tenants {
		items = append(items, toTenantResponse(&tenants[i]))
	}
	pageObj := pagination.NewPage(page, pageSize, int(total), items)
	return dto.TenantListResponse{
		Page:      pageObj.Page,
		PageSize:  pageObj.PageSize,
		Total:     pageObj.Total,
		TotalPage: pageObj.TotalPage,
		Items:     pageObj.Items,
	}, nil
}

// DeleteTenant soft-deletes a tenant.
func (uc *TenantUseCase) DeleteTenant(ctx context.Context, id int64) error {
	tenant, err := uc.tenantRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if tenant == nil {
		return errno.New(errno.NotFound, "tenant not found")
	}
	return uc.tenantRepo.Delete(ctx, id)
}

// AddMember adds a user to a tenant with the given role, enforcing the
// max_users quota and rejecting duplicates.
func (uc *TenantUseCase) AddMember(ctx context.Context, tenantID int64, in *dto.AddMemberRequest) (*dto.MemberResponse, error) {
	if in == nil {
		return nil, errno.New(errno.InvalidParams, "request body is required")
	}
	if in.UserID <= 0 {
		return nil, errno.New(errno.InvalidParams, "user_id is required")
	}
	role := in.Role
	if role == "" {
		role = entity.TenantRoleStudent
	}
	if !entity.ValidateTenantMemberRole(role) {
		return nil, errno.New(errno.InvalidParams, "invalid role: "+role)
	}

	tenant, err := uc.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, errno.New(errno.NotFound, "tenant not found: "+strconv.FormatInt(tenantID, 10))
	}
	if !tenant.IsActive() {
		return nil, errno.New(errno.InvalidParams, "tenant is not active")
	}

	// Reject duplicates (covers both active and soft-deleted rows; the repo's
	// unique index is partial on deleted_at IS NULL, but a re-add should
	// surface as a clearer error than a DB constraint violation).
	if _, exists, err := uc.memberRepo.IsMember(ctx, tenantID, in.UserID); err != nil {
		return nil, err
	} else if exists {
		return nil, errno.New(errno.AlreadyExists, "user already a member of this tenant")
	}

	// Enforce max_users quota when set (>0). A 0 value disables the quota
	// check; this is convenient for early pilots where limits are not yet
	// enforced.
	if tenant.MaxUsers > 0 {
		count, err := uc.memberRepo.CountMembers(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		if count >= int64(tenant.MaxUsers) {
			return nil, errno.New(errno.Forbidden, "tenant member quota exceeded")
		}
	}

	var expiredAt *time.Time
	if in.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, in.ExpiresAt)
		if err != nil {
			return nil, errno.New(errno.InvalidParams, "invalid expires_at format")
		}
		expiredAt = &t
	}

	member := &entity.TenantMember{
		TenantID:  tenantID,
		UserID:    in.UserID,
		Role:      role,
		JoinedAt:  time.Now(),
		ExpiredAt: expiredAt,
	}
	member.ID = idgen.Next()

	if err := uc.memberRepo.AddMember(ctx, member); err != nil {
		return nil, err
	}
	resp := toMemberResponse(member)
	return &resp, nil
}

// RemoveMember soft-deletes a tenant membership.
func (uc *TenantUseCase) RemoveMember(ctx context.Context, tenantID, userID int64) error {
	if tenantID <= 0 {
		return errno.New(errno.InvalidParams, "tenant_id is required")
	}
	if userID <= 0 {
		return errno.New(errno.InvalidParams, "user_id is required")
	}
	return uc.memberRepo.RemoveMember(ctx, tenantID, userID)
}

// ListMembers returns all members of a tenant.
func (uc *TenantUseCase) ListMembers(ctx context.Context, tenantID int64) (dto.MemberListResponse, error) {
	members, err := uc.memberRepo.FindMembers(ctx, tenantID)
	if err != nil {
		return dto.MemberListResponse{}, err
	}
	items := make([]dto.MemberResponse, 0, len(members))
	for i := range members {
		items = append(items, toMemberResponse(&members[i]))
	}
	return dto.MemberListResponse{
		TenantID: tenantID,
		Total:    len(items),
		Items:    items,
	}, nil
}

// ListUserTenants returns every tenant the user belongs to.
func (uc *TenantUseCase) ListUserTenants(ctx context.Context, userID int64) (dto.UserTenantsResponse, error) {
	if userID <= 0 {
		return dto.UserTenantsResponse{}, errno.New(errno.InvalidParams, "user_id is required")
	}
	members, err := uc.memberRepo.FindUserTenants(ctx, userID)
	if err != nil {
		return dto.UserTenantsResponse{}, err
	}
	items := make([]dto.MemberResponse, 0, len(members))
	for i := range members {
		items = append(items, toMemberResponse(&members[i]))
	}
	return dto.UserTenantsResponse{
		UserID: userID,
		Total:  len(items),
		Items:  items,
	}, nil
}

// toTenantResponse converts a Tenant entity to its wire DTO.
func toTenantResponse(t *entity.Tenant) dto.TenantResponse {
	return dto.TenantResponse{
		ID:        t.ID,
		Name:      t.Name,
		Code:      t.Code,
		Plan:      t.Plan,
		Status:    t.Status,
		MaxUsers:  t.MaxUsers,
		ExpiresAt: formatTenantTimePtr(t.ExpiresAt),
		CreatedAt: t.CreatedAt.Format(time.RFC3339),
		UpdatedAt: t.UpdatedAt.Format(time.RFC3339),
	}
}

// toMemberResponse converts a TenantMember entity to its wire DTO.
func toMemberResponse(m *entity.TenantMember) dto.MemberResponse {
	return dto.MemberResponse{
		ID:        m.ID,
		TenantID:  m.TenantID,
		UserID:    m.UserID,
		Role:      m.Role,
		JoinedAt:  m.JoinedAt.Format(time.RFC3339),
		ExpiredAt: formatTenantTimePtr(m.ExpiredAt),
		CreatedAt: m.CreatedAt.Format(time.RFC3339),
		UpdatedAt: m.UpdatedAt.Format(time.RFC3339),
	}
}

// formatTenantTimePtr renders a *time.Time as RFC3339, returning "" for nil.
// It is a local twin of dto.formatTimePtr so the usecase can stay terse
// without forcing the dto package to export its helper.
func formatTenantTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
