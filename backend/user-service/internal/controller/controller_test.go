package controller_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route/param"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/gormutil"
	"tcm-history-ai/backend/pkg/pagination"
	"tcm-history-ai/backend/pkg/response"
	"tcm-history-ai/backend/user-service/internal/application/usecase"
	"tcm-history-ai/backend/user-service/internal/controller"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
	"tcm-history-ai/backend/user-service/internal/domain/event"
	"tcm-history-ai/backend/user-service/internal/domain/service"
)

// ---------------------------------------------------------------------------
// Mock: PasswordHasher
// ---------------------------------------------------------------------------

type mockHasher struct {
	hashFn   func(string) (string, error)
	verifyFn func(string, string) bool
}

func (m *mockHasher) Hash(pw string) (string, error) {
	if m.hashFn != nil {
		return m.hashFn(pw)
	}
	return "hash:" + pw, nil
}

func (m *mockHasher) Verify(pw, h string) bool {
	if m.verifyFn != nil {
		return m.verifyFn(pw, h)
	}
	return h == "hash:"+pw
}

// ---------------------------------------------------------------------------
// Mock: TokenManager
// ---------------------------------------------------------------------------

type mockTokenMgr struct {
	accessTTL  time.Duration
	refreshTTL time.Duration
	issueAcc   func(int64, []string) (string, error)
	issueRef   func(int64, []string) (string, error)
	parseAcc   func(string) (*service.Claims, error)
	parseRef   func(string) (*service.Claims, error)
}

func (m *mockTokenMgr) IssueAccessToken(uid int64, roles []string) (string, error) {
	if m.issueAcc != nil {
		return m.issueAcc(uid, roles)
	}
	return "access:" + strconv.FormatInt(uid, 10), nil
}

func (m *mockTokenMgr) IssueRefreshToken(uid int64, roles []string) (string, error) {
	if m.issueRef != nil {
		return m.issueRef(uid, roles)
	}
	return "refresh:" + strconv.FormatInt(uid, 10), nil
}

func (m *mockTokenMgr) ParseAccessToken(token string) (*service.Claims, error) {
	if m.parseAcc != nil {
		return m.parseAcc(token)
	}
	return &service.Claims{Sub: "1", Roles: []string{"student"}, Exp: time.Now().Add(time.Hour).Unix()}, nil
}

func (m *mockTokenMgr) ParseRefreshToken(token string) (*service.Claims, error) {
	if m.parseRef != nil {
		return m.parseRef(token)
	}
	return &service.Claims{Sub: "1", Roles: []string{"student"}, Exp: time.Now().Add(time.Hour).Unix()}, nil
}

func (m *mockTokenMgr) AccessTokenTTL() time.Duration {
	if m.accessTTL > 0 {
		return m.accessTTL
	}
	return 15 * time.Minute
}

func (m *mockTokenMgr) RefreshTokenTTL() time.Duration {
	if m.refreshTTL > 0 {
		return m.refreshTTL
	}
	return 24 * time.Hour
}

// ---------------------------------------------------------------------------
// Mock: RefreshTokenStore
// ---------------------------------------------------------------------------

type mockRefreshStore struct {
	setFn func(int64, string, time.Duration) error
	getFn func(int64) (string, error)
	delFn func(int64) error
}

func (m *mockRefreshStore) Set(_ context.Context, uid int64, token string, ttl time.Duration) error {
	if m.setFn != nil {
		return m.setFn(uid, token, ttl)
	}
	return nil
}

func (m *mockRefreshStore) Get(_ context.Context, uid int64) (string, error) {
	if m.getFn != nil {
		return m.getFn(uid)
	}
	return "", errno.New(errno.NotFound, "not found")
}

func (m *mockRefreshStore) Delete(_ context.Context, uid int64) error {
	if m.delFn != nil {
		return m.delFn(uid)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: EventPublisher
// ---------------------------------------------------------------------------

type mockPublisher struct {
	publishFn func(event.Event) error
}

func (m *mockPublisher) Publish(_ context.Context, e event.Event) error {
	if m.publishFn != nil {
		return m.publishFn(e)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: UserRepository
// ---------------------------------------------------------------------------

type mockUserRepo struct {
	users map[int64]*entity.User

	createFn         func(*entity.User) error
	findByIDFn       func(int64) (*entity.User, error)
	findByUsernameFn func(string) (*entity.User, error)
	findByEmailFn    func(string) (*entity.User, error)
	findByPhoneFn    func(string) (*entity.User, error)
	updateLastLoginFn func(int64, time.Time, string) error
	updateFn         func(*entity.User) error
	deleteFn         func(int64) error
	listFn           func(pagination.Params, string) ([]entity.User, int64, error)
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: map[int64]*entity.User{}}
}

func (m *mockUserRepo) Create(_ context.Context, u *entity.User) error {
	if m.createFn != nil {
		return m.createFn(u)
	}
	m.users[u.ID] = u
	return nil
}

func (m *mockUserRepo) FindByID(_ context.Context, id int64) (*entity.User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id)
	}
	if u, ok := m.users[id]; ok {
		return cloneUser(u), nil
	}
	return nil, nil
}

func (m *mockUserRepo) FindByUsername(_ context.Context, username string) (*entity.User, error) {
	if m.findByUsernameFn != nil {
		return m.findByUsernameFn(username)
	}
	for _, u := range m.users {
		if u.Username == username {
			return cloneUser(u), nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) FindByEmail(_ context.Context, email string) (*entity.User, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(email)
	}
	for _, u := range m.users {
		if u.Email != nil && *u.Email == email {
			return cloneUser(u), nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) FindByPhone(_ context.Context, phone string) (*entity.User, error) {
	if m.findByPhoneFn != nil {
		return m.findByPhoneFn(phone)
	}
	for _, u := range m.users {
		if u.Phone != nil && *u.Phone == phone {
			return cloneUser(u), nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) UpdateLastLogin(_ context.Context, id int64, at time.Time, ip string) error {
	if m.updateLastLoginFn != nil {
		return m.updateLastLoginFn(id, at, ip)
	}
	if u, ok := m.users[id]; ok {
		u.LastLoginAt = &at
		u.LastLoginIP = ip
		return nil
	}
	return errno.New(errno.NotFound, "user not found")
}

func (m *mockUserRepo) Update(_ context.Context, u *entity.User) error {
	if m.updateFn != nil {
		return m.updateFn(u)
	}
	m.users[u.ID] = u
	return nil
}

func (m *mockUserRepo) Delete(_ context.Context, id int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(id)
	}
	delete(m.users, id)
	return nil
}

func (m *mockUserRepo) List(_ context.Context, p pagination.Params, status string) ([]entity.User, int64, error) {
	if m.listFn != nil {
		return m.listFn(p, status)
	}
	var out []entity.User
	for _, u := range m.users {
		if status == "" || u.Status == status {
			out = append(out, *cloneUser(u))
		}
	}
	return out, int64(len(out)), nil
}

func cloneUser(u *entity.User) *entity.User {
	if u == nil {
		return nil
	}
	c := *u
	if u.Email != nil {
		v := *u.Email
		c.Email = &v
	}
	if u.Phone != nil {
		v := *u.Phone
		c.Phone = &v
	}
	if u.LastLoginAt != nil {
		v := *u.LastLoginAt
		c.LastLoginAt = &v
	}
	return &c
}

// ---------------------------------------------------------------------------
// Mock: RoleRepository
// ---------------------------------------------------------------------------

type mockRoleRepo struct {
	roles     map[string]*entity.Role
	userRoles map[int64][]entity.Role

	findByCodeFn func(string) (*entity.Role, error)
	findByUserFn func(int64) ([]entity.Role, error)
	listAllFn    func() ([]entity.Role, error)
	assignRoleFn func(int64, int64) error
	findByIDFn   func(int64) (*entity.Role, error)
	createFn     func(*entity.Role) error
	updateFn     func(*entity.Role) error
	deleteFn     func(int64) error
	setUserRolesFn    func(int64, []int64) error
	setRolePermsFn    func(int64, []int64) error
}

func newMockRoleRepo() *mockRoleRepo {
	return &mockRoleRepo{
		roles:     map[string]*entity.Role{},
		userRoles: map[int64][]entity.Role{},
	}
}

func (m *mockRoleRepo) FindByCode(_ context.Context, code string) (*entity.Role, error) {
	if m.findByCodeFn != nil {
		return m.findByCodeFn(code)
	}
	if r, ok := m.roles[code]; ok {
		c := *r
		return &c, nil
	}
	return nil, nil
}

func (m *mockRoleRepo) FindByUserID(_ context.Context, userID int64) ([]entity.Role, error) {
	if m.findByUserFn != nil {
		return m.findByUserFn(userID)
	}
	out := make([]entity.Role, len(m.userRoles[userID]))
	copy(out, m.userRoles[userID])
	return out, nil
}

func (m *mockRoleRepo) ListAll(_ context.Context) ([]entity.Role, error) {
	if m.listAllFn != nil {
		return m.listAllFn()
	}
	var out []entity.Role
	for _, r := range m.roles {
		out = append(out, *r)
	}
	return out, nil
}

func (m *mockRoleRepo) AssignRole(_ context.Context, userID, roleID int64) error {
	if m.assignRoleFn != nil {
		return m.assignRoleFn(userID, roleID)
	}
	for _, r := range m.roles {
		if r.ID == roleID {
			m.userRoles[userID] = append(m.userRoles[userID], *r)
		}
	}
	return nil
}

func (m *mockRoleRepo) FindByID(_ context.Context, id int64) (*entity.Role, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id)
	}
	for _, r := range m.roles {
		if r.ID == id {
			c := *r
			return &c, nil
		}
	}
	return nil, nil
}

func (m *mockRoleRepo) Create(_ context.Context, r *entity.Role) error {
	if m.createFn != nil {
		return m.createFn(r)
	}
	m.roles[r.Code] = r
	return nil
}

func (m *mockRoleRepo) Update(_ context.Context, r *entity.Role) error {
	if m.updateFn != nil {
		return m.updateFn(r)
	}
	m.roles[r.Code] = r
	return nil
}

func (m *mockRoleRepo) Delete(_ context.Context, id int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(id)
	}
	for code, r := range m.roles {
		if r.ID == id {
			delete(m.roles, code)
			return nil
		}
	}
	return errno.New(errno.NotFound, "role not found")
}

func (m *mockRoleRepo) SetUserRoles(_ context.Context, userID int64, roleIDs []int64) error {
	if m.setUserRolesFn != nil {
		return m.setUserRolesFn(userID, roleIDs)
	}
	var roles []entity.Role
	for _, rid := range roleIDs {
		for _, r := range m.roles {
			if r.ID == rid {
				roles = append(roles, *r)
				break
			}
		}
	}
	m.userRoles[userID] = roles
	return nil
}

func (m *mockRoleRepo) AssignPermission(_ context.Context, roleID, permissionID int64) error {
	return nil
}

func (m *mockRoleRepo) SetRolePermissions(_ context.Context, roleID int64, permissionIDs []int64) error {
	if m.setRolePermsFn != nil {
		return m.setRolePermsFn(roleID, permissionIDs)
	}
	return nil
}

func (m *mockRoleRepo) RevokePermission(_ context.Context, roleID, permissionID int64) error {
	return nil
}

func (m *mockRoleRepo) seedStudentRole() *entity.Role {
	r := &entity.Role{ID: 100, Code: entity.RoleStudent, Name: "Student", IsBuiltin: true}
	m.roles[entity.RoleStudent] = r
	return r
}

// ---------------------------------------------------------------------------
// Mock: ProfileRepository
// ---------------------------------------------------------------------------

type mockProfileRepo struct {
	items map[int64]*entity.UserProfile

	createFn       func(*entity.UserProfile) error
	findByUserIDFn func(int64) (*entity.UserProfile, error)
	updateFn       func(*entity.UserProfile) error
}

func newMockProfileRepo() *mockProfileRepo {
	return &mockProfileRepo{items: map[int64]*entity.UserProfile{}}
}

func (m *mockProfileRepo) Create(_ context.Context, p *entity.UserProfile) error {
	if m.createFn != nil {
		return m.createFn(p)
	}
	m.items[p.UserID] = p
	return nil
}

func (m *mockProfileRepo) FindByUserID(_ context.Context, userID int64) (*entity.UserProfile, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(userID)
	}
	if p, ok := m.items[userID]; ok {
		c := *p
		return &c, nil
	}
	return nil, nil
}

func (m *mockProfileRepo) Update(_ context.Context, p *entity.UserProfile) error {
	if m.updateFn != nil {
		return m.updateFn(p)
	}
	m.items[p.UserID] = p
	return nil
}

// ---------------------------------------------------------------------------
// Mock: SettingsRepository
// ---------------------------------------------------------------------------

type mockSettingsRepo struct {
	items map[int64]*entity.UserSettings

	createFn       func(*entity.UserSettings) error
	findByUserIDFn func(int64) (*entity.UserSettings, error)
	updateFn       func(*entity.UserSettings) error
}

func newMockSettingsRepo() *mockSettingsRepo {
	return &mockSettingsRepo{items: map[int64]*entity.UserSettings{}}
}

func (m *mockSettingsRepo) Create(_ context.Context, s *entity.UserSettings) error {
	if m.createFn != nil {
		return m.createFn(s)
	}
	m.items[s.UserID] = s
	return nil
}

func (m *mockSettingsRepo) FindByUserID(_ context.Context, userID int64) (*entity.UserSettings, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(userID)
	}
	if s, ok := m.items[userID]; ok {
		c := *s
		return &c, nil
	}
	return nil, nil
}

func (m *mockSettingsRepo) Update(_ context.Context, s *entity.UserSettings) error {
	if m.updateFn != nil {
		return m.updateFn(s)
	}
	m.items[s.UserID] = s
	return nil
}

// ---------------------------------------------------------------------------
// Mock: PermissionRepository
// ---------------------------------------------------------------------------

type mockPermRepo struct {
	perms []entity.Permission

	findByRoleIDFn func(int64) ([]entity.Permission, error)
	findByUserFn   func(int64) ([]entity.Permission, error)
	listAllFn      func() ([]entity.Permission, error)
	findByIDFn     func(int64) (*entity.Permission, error)
}

func (m *mockPermRepo) FindByRoleID(_ context.Context, roleID int64) ([]entity.Permission, error) {
	if m.findByRoleIDFn != nil {
		return m.findByRoleIDFn(roleID)
	}
	return nil, nil
}

func (m *mockPermRepo) FindByUserID(_ context.Context, userID int64) ([]entity.Permission, error) {
	if m.findByUserFn != nil {
		return m.findByUserFn(userID)
	}
	return nil, nil
}

func (m *mockPermRepo) ListAll(_ context.Context) ([]entity.Permission, error) {
	if m.listAllFn != nil {
		return m.listAllFn()
	}
	return m.perms, nil
}

func (m *mockPermRepo) FindByID(_ context.Context, id int64) (*entity.Permission, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id)
	}
	for _, p := range m.perms {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: TenantRepository
// ---------------------------------------------------------------------------

type mockTenantRepo struct {
	tenants map[int64]*entity.Tenant

	createFn    func(*entity.Tenant) error
	updateFn    func(*entity.Tenant) error
	deleteFn    func(int64) error
	findByIDFn  func(int64) (*entity.Tenant, error)
	findByIDForUpdateFn func(int64) (*entity.Tenant, error)
	findByCodeFn func(string) (*entity.Tenant, error)
	listFn      func(pagination.Params, string) ([]entity.Tenant, int64, error)
}

func newMockTenantRepo() *mockTenantRepo {
	return &mockTenantRepo{tenants: map[int64]*entity.Tenant{}}
}

func (m *mockTenantRepo) Create(_ context.Context, t *entity.Tenant) error {
	if m.createFn != nil {
		return m.createFn(t)
	}
	m.tenants[t.ID] = t
	return nil
}

func (m *mockTenantRepo) Update(_ context.Context, t *entity.Tenant) error {
	if m.updateFn != nil {
		return m.updateFn(t)
	}
	m.tenants[t.ID] = t
	return nil
}

func (m *mockTenantRepo) Delete(_ context.Context, id int64) error {
	if m.deleteFn != nil {
		return m.deleteFn(id)
	}
	delete(m.tenants, id)
	return nil
}

func (m *mockTenantRepo) FindByID(_ context.Context, id int64) (*entity.Tenant, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id)
	}
	if t, ok := m.tenants[id]; ok {
		c := *t
		return &c, nil
	}
	return nil, nil
}

func (m *mockTenantRepo) FindByIDForUpdate(_ context.Context, id int64) (*entity.Tenant, error) {
	if m.findByIDForUpdateFn != nil {
		return m.findByIDForUpdateFn(id)
	}
	return m.FindByID(context.Background(), id)
}

func (m *mockTenantRepo) FindByCode(_ context.Context, code string) (*entity.Tenant, error) {
	if m.findByCodeFn != nil {
		return m.findByCodeFn(code)
	}
	for _, t := range m.tenants {
		if t.Code == code {
			c := *t
			return &c, nil
		}
	}
	return nil, nil
}

func (m *mockTenantRepo) List(_ context.Context, p pagination.Params, status string) ([]entity.Tenant, int64, error) {
	if m.listFn != nil {
		return m.listFn(p, status)
	}
	var out []entity.Tenant
	for _, t := range m.tenants {
		if status == "" || t.Status == status {
			out = append(out, *t)
		}
	}
	return out, int64(len(out)), nil
}

// ---------------------------------------------------------------------------
// Mock: TenantMemberRepository
// ---------------------------------------------------------------------------

type mockTenantMemberRepo struct {
	members map[int64]map[int64]*entity.TenantMember // tenantID -> userID -> member

	addMemberFn    func(*entity.TenantMember) error
	removeMemberFn func(int64, int64) error
	findMembersFn  func(int64) ([]entity.TenantMember, error)
	findUserTenantsFn func(int64) ([]entity.TenantMember, error)
	isMemberFn     func(int64, int64) (*entity.TenantMember, bool, error)
	countMembersFn func(int64) (int64, error)
}

func newMockTenantMemberRepo() *mockTenantMemberRepo {
	return &mockTenantMemberRepo{members: map[int64]map[int64]*entity.TenantMember{}}
}

func (m *mockTenantMemberRepo) AddMember(_ context.Context, mem *entity.TenantMember) error {
	if m.addMemberFn != nil {
		return m.addMemberFn(mem)
	}
	if m.members[mem.TenantID] == nil {
		m.members[mem.TenantID] = map[int64]*entity.TenantMember{}
	}
	m.members[mem.TenantID][mem.UserID] = mem
	return nil
}

func (m *mockTenantMemberRepo) RemoveMember(_ context.Context, tenantID, userID int64) error {
	if m.removeMemberFn != nil {
		return m.removeMemberFn(tenantID, userID)
	}
	if m.members[tenantID] != nil {
		delete(m.members[tenantID], userID)
	}
	return nil
}

func (m *mockTenantMemberRepo) FindMembers(_ context.Context, tenantID int64) ([]entity.TenantMember, error) {
	if m.findMembersFn != nil {
		return m.findMembersFn(tenantID)
	}
	var out []entity.TenantMember
	if users, ok := m.members[tenantID]; ok {
		for _, mem := range users {
			out = append(out, *mem)
		}
	}
	return out, nil
}

func (m *mockTenantMemberRepo) FindUserTenants(_ context.Context, userID int64) ([]entity.TenantMember, error) {
	if m.findUserTenantsFn != nil {
		return m.findUserTenantsFn(userID)
	}
	var out []entity.TenantMember
	for _, users := range m.members {
		if mem, ok := users[userID]; ok {
			out = append(out, *mem)
		}
	}
	return out, nil
}

func (m *mockTenantMemberRepo) IsMember(_ context.Context, tenantID, userID int64) (*entity.TenantMember, bool, error) {
	if m.isMemberFn != nil {
		return m.isMemberFn(tenantID, userID)
	}
	if users, ok := m.members[tenantID]; ok {
		if mem, ok := users[userID]; ok {
			return mem, true, nil
		}
	}
	return nil, false, nil
}

func (m *mockTenantMemberRepo) CountMembers(_ context.Context, tenantID int64) (int64, error) {
	if m.countMembersFn != nil {
		return m.countMembersFn(tenantID)
	}
	if users, ok := m.members[tenantID]; ok {
		return int64(len(users)), nil
	}
	return 0, nil
}

// ===========================================================================
// Test Helpers
// ===========================================================================

// setupAuth creates an AuthController backed by mocked dependencies.
func setupAuth() (*controller.AuthController, *mockUserRepo, *mockRoleRepo, *mockProfileRepo, *mockSettingsRepo, *mockHasher, *mockTokenMgr, *mockRefreshStore, *mockPublisher) {
	userRepo := newMockUserRepo()
	roleRepo := newMockRoleRepo()
	roleRepo.seedStudentRole()
	profileRepo := newMockProfileRepo()
	settingsRepo := newMockSettingsRepo()
	hasher := &mockHasher{}
	tokens := &mockTokenMgr{}
	store := &mockRefreshStore{}
	pub := &mockPublisher{}

	uc := usecase.NewAuthUseCase(userRepo, roleRepo, profileRepo, settingsRepo, hasher, tokens, store, pub)
	ctl := controller.NewAuthController(uc)
	return ctl, userRepo, roleRepo, profileRepo, settingsRepo, hasher, tokens, store, pub
}

// setupProfile creates a ProfileController backed by mocked dependencies.
func setupProfile() (*controller.ProfileController, *mockUserRepo, *mockProfileRepo, *mockPublisher) {
	userRepo := newMockUserRepo()
	profileRepo := newMockProfileRepo()
	pub := &mockPublisher{}
	uc := usecase.NewProfileUseCase(userRepo, profileRepo, pub)
	ctl := controller.NewProfileController(uc)
	return ctl, userRepo, profileRepo, pub
}

// setupAdmin creates an AdminController backed by mocked dependencies.
func setupAdmin() (*controller.AdminController, *mockUserRepo, *mockRoleRepo, *mockPermRepo) {
	userRepo := newMockUserRepo()
	roleRepo := newMockRoleRepo()
	permRepo := &mockPermRepo{}
	uc := usecase.NewAdminUseCase(userRepo, roleRepo, permRepo)
	ctl := controller.NewAdminController(uc)
	return ctl, userRepo, roleRepo, permRepo
}

// setupRole creates a RoleController backed by mocked dependencies.
func setupRole() (*controller.RoleController, *mockRoleRepo, *mockPermRepo) {
	roleRepo := newMockRoleRepo()
	permRepo := &mockPermRepo{}
	uc := usecase.NewRoleUseCase(roleRepo, permRepo)
	ctl := controller.NewRoleController(uc)
	return ctl, roleRepo, permRepo
}

// setupTenant creates a TenantController backed by mocked dependencies.
func setupTenant() (*controller.TenantController, *mockTenantRepo, *mockTenantMemberRepo) {
	tenantRepo := newMockTenantRepo()
	memberRepo := newMockTenantMemberRepo()
	uc := usecase.NewTenantUseCase(nil, tenantRepo, memberRepo)
	ctl := controller.NewTenantController(uc)
	return ctl, tenantRepo, memberRepo
}

// newCtx creates a minimal app.RequestContext for handler testing.
func newCtx(method, uri string, body string) *app.RequestContext {
	rc := app.NewContext(0)
	rc.Request.SetMethod(method)
	rc.Request.SetRequestURI(uri)
	if body != "" {
		rc.Request.SetBodyString(body)
	}
	return rc
}

// parseResponse parses the response body into a response.Body.
func parseResponse(t *testing.T, rc *app.RequestContext) response.Body {
	var body response.Body
	require.NoError(t, json.Unmarshal(rc.Response.Body(), &body))
	return body
}

// setParam sets a single path parameter on the RequestContext.
func setParam(rc *app.RequestContext, key, value string) {
	rc.Params = param.Params{{Key: key, Value: value}}
}

// newTenant creates a Tenant entity with the given fields, wrapping BaseModel.
func newTenant(id int64, name, code, plan, status string, maxUsers int, now time.Time) *entity.Tenant {
	return &entity.Tenant{
		BaseModel: gormutil.BaseModel{ID: id, CreatedAt: now, UpdatedAt: now},
		Name:      name,
		Code:      code,
		Plan:      plan,
		Status:    status,
		MaxUsers:  maxUsers,
	}
}

// ===========================================================================
// AuthController Tests
// ===========================================================================

func TestAuthController_Register_Success(t *testing.T) {
	ctl, userRepo, _, _, _, hasher, _, _, _ := setupAuth()

	hasher.hashFn = func(pw string) (string, error) { return "hash:" + pw, nil }

	rc := newCtx("POST", "/api/v1/auth/register", `{"username":"alice","password":"secret123","email":"alice@example.com"}`)
	ctl.Register(context.Background(), rc)

	assert.Equal(t, http.StatusCreated, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)
	assert.Equal(t, "created", body.Message)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, data["access_token"])
	assert.NotEmpty(t, data["refresh_token"])
	userID, ok := data["user_id"].(float64)
	require.True(t, ok)
	assert.NotZero(t, userID)
	assert.Equal(t, "alice", data["username"])
	assert.NotNil(t, userRepo)
}

func TestAuthController_Register_InvalidJSON(t *testing.T) {
	ctl, _, _, _, _, _, _, _, _ := setupAuth()

	rc := newCtx("POST", "/api/v1/auth/register", `not-json`)
	ctl.Register(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.InvalidParams), body.Code)
}

func TestAuthController_Register_MissingFields(t *testing.T) {
	ctl, _, _, _, _, _, _, _, _ := setupAuth()

	rc := newCtx("POST", "/api/v1/auth/register", `{"username":"alice"}`)
	ctl.Register(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.InvalidParams), body.Code)
}

func TestAuthController_Register_DuplicateUsername(t *testing.T) {
	ctl, userRepo, _, _, _, _, _, _, _ := setupAuth()

	existing := &entity.User{}
	existing.ID = 1
	existing.Username = "alice"
	existing.Status = entity.StatusActive
	userRepo.users[1] = existing

	rc := newCtx("POST", "/api/v1/auth/register", `{"username":"alice","password":"secret123"}`)
	ctl.Register(context.Background(), rc)

	assert.Equal(t, http.StatusConflict, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.AlreadyExists), body.Code)
}

func TestAuthController_Login_Success(t *testing.T) {
	ctl, userRepo, _, _, _, hasher, tokens, store, _ := setupAuth()

	hasher.hashFn = func(pw string) (string, error) { return "hash:" + pw, nil }

	user := &entity.User{}
	user.ID = 1
	user.Username = "alice"
	user.PasswordHash = "hash:secret123"
	user.Status = entity.StatusActive
	userRepo.users[1] = user

	store.setFn = func(_ int64, _ string, _ time.Duration) error { return nil }
	tokens.issueAcc = func(uid int64, roles []string) (string, error) {
		return "access-token", nil
	}
	tokens.issueRef = func(uid int64, roles []string) (string, error) {
		return "refresh-token", nil
	}

	rc := newCtx("POST", "/api/v1/auth/login", `{"username":"alice","password":"secret123"}`)
	ctl.Login(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "access-token", data["access_token"])
	assert.Equal(t, "refresh-token", data["refresh_token"])
	assert.Equal(t, "alice", data["username"])
}

func TestAuthController_Login_InvalidCredentials(t *testing.T) {
	ctl, userRepo, _, _, _, _, _, _, _ := setupAuth()

	user := &entity.User{}
	user.ID = 1
	user.Username = "alice"
	user.PasswordHash = "hash:correct"
	user.Status = entity.StatusActive
	userRepo.users[1] = user

	rc := newCtx("POST", "/api/v1/auth/login", `{"username":"alice","password":"wrong"}`)
	ctl.Login(context.Background(), rc)

	assert.Equal(t, http.StatusUnauthorized, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.Unauthorized), body.Code)
}

func TestAuthController_Login_InvalidJSON(t *testing.T) {
	ctl, _, _, _, _, _, _, _, _ := setupAuth()

	rc := newCtx("POST", "/api/v1/auth/login", `{invalid}`)
	ctl.Login(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
}

func TestAuthController_Login_MissingFields(t *testing.T) {
	ctl, _, _, _, _, _, _, _, _ := setupAuth()

	rc := newCtx("POST", "/api/v1/auth/login", `{}`)
	ctl.Login(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.InvalidParams), body.Code)
}

func TestAuthController_Refresh_Success(t *testing.T) {
	ctl, userRepo, _, _, _, _, tokens, store, _ := setupAuth()

	user := &entity.User{}
	user.ID = 1
	user.Username = "alice"
	user.Status = entity.StatusActive
	userRepo.users[1] = user

	store.getFn = func(uid int64) (string, error) { return "refresh:token", nil }
	tokens.parseRef = func(token string) (*service.Claims, error) {
		return &service.Claims{Sub: "1", Roles: []string{"student"}, Exp: time.Now().Add(time.Hour).Unix()}, nil
	}
	tokens.issueAcc = func(uid int64, roles []string) (string, error) {
		return "new-access", nil
	}
	tokens.issueRef = func(uid int64, roles []string) (string, error) {
		return "new-refresh", nil
	}

	rc := newCtx("POST", "/api/v1/auth/refresh", `{"refresh_token":"refresh:token"}`)
	ctl.Refresh(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "new-access", data["access_token"])
	assert.Equal(t, "new-refresh", data["refresh_token"])
}

func TestAuthController_Refresh_InvalidToken(t *testing.T) {
	ctl, _, _, _, _, _, tokens, _, _ := setupAuth()

	tokens.parseRef = func(token string) (*service.Claims, error) {
		return nil, errno.New(errno.Unauthorized, "invalid token")
	}

	rc := newCtx("POST", "/api/v1/auth/refresh", `{"refresh_token":"bad"}`)
	ctl.Refresh(context.Background(), rc)

	assert.Equal(t, http.StatusUnauthorized, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.Unauthorized), body.Code)
}

func TestAuthController_Refresh_MissingToken(t *testing.T) {
	ctl, _, _, _, _, _, _, _, _ := setupAuth()

	rc := newCtx("POST", "/api/v1/auth/refresh", `{}`)
	ctl.Refresh(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.InvalidParams), body.Code)
}

// ===========================================================================
// ProfileController Tests
// ===========================================================================

func TestProfileController_Get_Success(t *testing.T) {
	ctl, userRepo, profileRepo, _ := setupProfile()

	user := &entity.User{}
	user.ID = 1
	user.Username = "alice"
	user.Status = entity.StatusActive
	email := "alice@example.com"
	user.Email = &email
	userRepo.users[1] = user

	profile := &entity.UserProfile{UserID: 1, Nickname: "Alice"}
	profileRepo.items[1] = profile

	rc := newCtx("GET", "/api/v1/users/me", "")
	rc.Request.Header.Set("X-User-ID", "1")
	ctl.Get(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), data["user_id"])
	assert.Equal(t, "alice", data["username"])
	assert.Equal(t, "alice@example.com", data["email"])
	assert.Equal(t, "active", data["status"])
	assert.Equal(t, "Alice", data["nickname"])
}

func TestProfileController_Get_Unauthorized(t *testing.T) {
	ctl, _, _, _ := setupProfile()

	rc := newCtx("GET", "/api/v1/users/me", "")
	ctl.Get(context.Background(), rc)

	assert.Equal(t, http.StatusUnauthorized, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.Unauthorized), body.Code)
}

func TestProfileController_Get_UserNotFound(t *testing.T) {
	ctl, _, _, _ := setupProfile()

	rc := newCtx("GET", "/api/v1/users/me", "")
	rc.Request.Header.Set("X-User-ID", "999")
	ctl.Get(context.Background(), rc)

	assert.Equal(t, http.StatusNotFound, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.NotFound), body.Code)
}

func TestProfileController_Update_Success(t *testing.T) {
	ctl, userRepo, profileRepo, _ := setupProfile()

	user := &entity.User{}
	user.ID = 1
	user.Username = "alice"
	user.Status = entity.StatusActive
	userRepo.users[1] = user

	profile := &entity.UserProfile{UserID: 1, Nickname: "OldName"}
	profileRepo.items[1] = profile

	rc := newCtx("PUT", "/api/v1/users/me", `{"nickname":"NewName"}`)
	rc.Request.Header.Set("X-User-ID", "1")
	ctl.Update(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "NewName", data["nickname"])
}

func TestProfileController_Update_Unauthorized(t *testing.T) {
	ctl, _, _, _ := setupProfile()

	rc := newCtx("PUT", "/api/v1/users/me", `{"nickname":"test"}`)
	ctl.Update(context.Background(), rc)

	assert.Equal(t, http.StatusUnauthorized, rc.Response.StatusCode())
}

func TestProfileController_Update_InvalidJSON(t *testing.T) {
	ctl, _, _, _ := setupProfile()

	rc := newCtx("PUT", "/api/v1/users/me", `invalid`)
	rc.Request.Header.Set("X-User-ID", "1")
	ctl.Update(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
}

func TestProfileController_Update_UserNotFound(t *testing.T) {
	ctl, _, _, _ := setupProfile()

	rc := newCtx("PUT", "/api/v1/users/me", `{"nickname":"test"}`)
	rc.Request.Header.Set("X-User-ID", "999")
	ctl.Update(context.Background(), rc)

	assert.Equal(t, http.StatusNotFound, rc.Response.StatusCode())
}

// ===========================================================================
// AdminController Tests
// ===========================================================================

func TestAdminController_ListUsers_Success(t *testing.T) {
	ctl, userRepo, roleRepo, _ := setupAdmin()

	email := "user1@test.com"
	user1 := &entity.User{}
	user1.ID = 1
	user1.Username = "user1"
	user1.Email = &email
	user1.Status = entity.StatusActive
	user1.CreatedAt = time.Now()
	user1.UpdatedAt = time.Now()
	userRepo.users[1] = user1

	role := entity.Role{ID: 100, Code: entity.RoleStudent, Name: "Student"}
	roleRepo.roles[entity.RoleStudent] = &role
	roleRepo.userRoles[1] = []entity.Role{role}

	rc := newCtx("GET", "/api/v1/admin/users?page=1&page_size=10", "")
	ctl.ListUsers(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), data["page"])
	assert.Equal(t, float64(10), data["page_size"])
	assert.Equal(t, float64(1), data["total"])
	items, ok := data["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 1)
	item := items[0].(map[string]interface{})
	assert.Equal(t, "user1", item["username"])
	assert.Equal(t, "active", item["status"])
}

func TestAdminController_ListUsers_WithStatusFilter(t *testing.T) {
	ctl, userRepo, _, _ := setupAdmin()

	user1 := &entity.User{}
	user1.ID = 1
	user1.Username = "user1"
	user1.Status = entity.StatusActive
	userRepo.users[1] = user1

	user2 := &entity.User{}
	user2.ID = 2
	user2.Username = "user2"
	user2.Status = entity.StatusDisabled
	userRepo.users[2] = user2

	rc := newCtx("GET", "/api/v1/admin/users?status=disabled", "")
	ctl.ListUsers(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	data := body.Data.(map[string]interface{})
	items := data["items"].([]interface{})
	assert.Len(t, items, 1)
	assert.Equal(t, "user2", items[0].(map[string]interface{})["username"])
}

func TestAdminController_ListUsers_DefaultPagination(t *testing.T) {
	ctl, _, _, _ := setupAdmin()

	rc := newCtx("GET", "/api/v1/admin/users", "")
	ctl.ListUsers(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	data := body.Data.(map[string]interface{})
	assert.Equal(t, float64(1), data["page"])
	assert.Equal(t, float64(20), data["page_size"])
}

func TestAdminController_GetUser_Success(t *testing.T) {
	ctl, userRepo, _, _ := setupAdmin()

	email := "user1@test.com"
	user := &entity.User{}
	user.ID = 1
	user.Username = "user1"
	user.Email = &email
	user.Status = entity.StatusActive
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	userRepo.users[1] = user

	rc := newCtx("GET", "/api/v1/admin/users/1", "")
	// Set path param since Hertz uses c.Param for path params
	setParam(rc, "id", "1")
	ctl.GetUser(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), data["id"])
	assert.Equal(t, "user1", data["username"])
}

func TestAdminController_GetUser_NotFound(t *testing.T) {
	ctl, _, _, _ := setupAdmin()

	rc := newCtx("GET", "/api/v1/admin/users/999", "")
	setParam(rc, "id", "999")
	ctl.GetUser(context.Background(), rc)

	assert.Equal(t, http.StatusNotFound, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.NotFound), body.Code)
}

func TestAdminController_GetUser_InvalidID(t *testing.T) {
	ctl, _, _, _ := setupAdmin()

	rc := newCtx("GET", "/api/v1/admin/users/abc", "")
	setParam(rc, "id", "abc")
	ctl.GetUser(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.InvalidParams), body.Code)
}

func TestAdminController_UpdateStatus_Success(t *testing.T) {
	ctl, userRepo, _, _ := setupAdmin()

	user := &entity.User{}
	user.ID = 1
	user.Username = "user1"
	user.Status = entity.StatusActive
	userRepo.users[1] = user

	rc := newCtx("PATCH", "/api/v1/admin/users/1/status", `{"status":"disabled"}`)
	setParam(rc, "id", "1")
	ctl.UpdateStatus(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)
	assert.Equal(t, "ok", body.Message)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), data["id"])
	assert.Equal(t, "disabled", data["status"])
}

func TestAdminController_UpdateStatus_InvalidStatus(t *testing.T) {
	ctl, userRepo, _, _ := setupAdmin()

	user := &entity.User{}
	user.ID = 1
	user.Username = "user1"
	user.Status = entity.StatusActive
	userRepo.users[1] = user

	rc := newCtx("PATCH", "/api/v1/admin/users/1/status", `{"status":"invalid_status"}`)
	setParam(rc, "id", "1")
	ctl.UpdateStatus(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
}

func TestAdminController_UpdateStatus_InvalidID(t *testing.T) {
	ctl, _, _, _ := setupAdmin()

	rc := newCtx("PATCH", "/api/v1/admin/users/abc/status", `{"status":"active"}`)
	setParam(rc, "id", "abc")
	ctl.UpdateStatus(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
}

func TestAdminController_UpdateStatus_InvalidJSON(t *testing.T) {
	ctl, _, _, _ := setupAdmin()

	rc := newCtx("PATCH", "/api/v1/admin/users/1/status", `bad`)
	setParam(rc, "id", "1")
	ctl.UpdateStatus(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
}

// ===========================================================================
// RoleController Tests
// ===========================================================================

func TestRoleController_List_Success(t *testing.T) {
	ctl, roleRepo, _ := setupRole()

	now := time.Now()
	role1 := entity.Role{ID: 1, Code: "admin", Name: "Admin", IsBuiltin: true, CreatedAt: now, UpdatedAt: now}
	role2 := entity.Role{ID: 2, Code: "teacher", Name: "Teacher", IsBuiltin: true, CreatedAt: now, UpdatedAt: now}
	roleRepo.roles["admin"] = &role1
	roleRepo.roles["teacher"] = &role2

	rc := newCtx("GET", "/api/v1/admin/roles", "")
	ctl.List(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)

	data, ok := body.Data.([]interface{})
	require.True(t, ok)
	assert.Len(t, data, 2)
}

func TestRoleController_List_Empty(t *testing.T) {
	ctl, _, _ := setupRole()

	rc := newCtx("GET", "/api/v1/admin/roles", "")
	ctl.List(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	data, ok := body.Data.([]interface{})
	require.True(t, ok)
	assert.Len(t, data, 0)
}

func TestRoleController_Get_Success(t *testing.T) {
	ctl, roleRepo, _ := setupRole()

	now := time.Now()
	role := entity.Role{ID: 1, Code: "admin", Name: "Admin", IsBuiltin: true, CreatedAt: now, UpdatedAt: now}
	roleRepo.roles["admin"] = &role

	rc := newCtx("GET", "/api/v1/admin/roles/1", "")
	setParam(rc, "id", "1")
	ctl.Get(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), data["id"])
	assert.Equal(t, "admin", data["code"])
	assert.Equal(t, "Admin", data["name"])
	assert.Equal(t, true, data["is_builtin"])
}

func TestRoleController_Get_NotFound(t *testing.T) {
	ctl, _, _ := setupRole()

	rc := newCtx("GET", "/api/v1/admin/roles/999", "")
	setParam(rc, "id", "999")
	ctl.Get(context.Background(), rc)

	assert.Equal(t, http.StatusNotFound, rc.Response.StatusCode())
}

func TestRoleController_Get_InvalidID(t *testing.T) {
	ctl, _, _ := setupRole()

	rc := newCtx("GET", "/api/v1/admin/roles/abc", "")
	setParam(rc, "id", "abc")
	ctl.Get(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
}

func TestRoleController_Create_Success(t *testing.T) {
	ctl, roleRepo, _ := setupRole()

	rc := newCtx("POST", "/api/v1/admin/roles", `{"code":"manager","name":"Manager","description":"Department manager"}`)
	ctl.Create(context.Background(), rc)

	assert.Equal(t, http.StatusCreated, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)
	assert.Equal(t, "created", body.Message)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "manager", data["code"])
	assert.Equal(t, "Manager", data["name"])
	assert.Equal(t, "Department manager", data["description"])

	_, exists := roleRepo.roles["manager"]
	assert.True(t, exists)
}

func TestRoleController_Create_InvalidJSON(t *testing.T) {
	ctl, _, _ := setupRole()

	rc := newCtx("POST", "/api/v1/admin/roles", `not-json`)
	ctl.Create(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
}

func TestRoleController_Create_MissingFields(t *testing.T) {
	ctl, _, _ := setupRole()

	rc := newCtx("POST", "/api/v1/admin/roles", `{}`)
	ctl.Create(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.InvalidParams), body.Code)
}

func TestRoleController_Create_DuplicateCode(t *testing.T) {
	ctl, roleRepo, _ := setupRole()

	now := time.Now()
	existing := entity.Role{ID: 1, Code: "admin", Name: "Admin", IsBuiltin: true, CreatedAt: now, UpdatedAt: now}
	roleRepo.roles["admin"] = &existing

	rc := newCtx("POST", "/api/v1/admin/roles", `{"code":"admin","name":"NewAdmin"}`)
	ctl.Create(context.Background(), rc)

	assert.Equal(t, http.StatusConflict, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.AlreadyExists), body.Code)
}

func TestRoleController_Update_Success(t *testing.T) {
	ctl, roleRepo, _ := setupRole()

	now := time.Now()
	role := entity.Role{ID: 5, Code: "custom", Name: "OldName", IsBuiltin: false, CreatedAt: now, UpdatedAt: now}
	roleRepo.roles["custom"] = &role

	rc := newCtx("PUT", "/api/v1/admin/roles/5", `{"name":"NewName"}`)
	setParam(rc, "id", "5")
	ctl.Update(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "NewName", data["name"])
}

func TestRoleController_Update_NotFound(t *testing.T) {
	ctl, _, _ := setupRole()

	rc := newCtx("PUT", "/api/v1/admin/roles/999", `{"name":"Test"}`)
	setParam(rc, "id", "999")
	ctl.Update(context.Background(), rc)

	assert.Equal(t, http.StatusNotFound, rc.Response.StatusCode())
}

func TestRoleController_Update_InvalidJSON(t *testing.T) {
	ctl, _, _ := setupRole()

	rc := newCtx("PUT", "/api/v1/admin/roles/1", `invalid`)
	setParam(rc, "id", "1")
	ctl.Update(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
}

func TestRoleController_Delete_Success(t *testing.T) {
	ctl, roleRepo, _ := setupRole()

	now := time.Now()
	role := entity.Role{ID: 10, Code: "temp", Name: "Temp", IsBuiltin: false, CreatedAt: now, UpdatedAt: now}
	roleRepo.roles["temp"] = &role

	rc := newCtx("DELETE", "/api/v1/admin/roles/10", "")
	setParam(rc, "id", "10")
	ctl.Delete(context.Background(), rc)

	assert.Equal(t, http.StatusNoContent, rc.Response.StatusCode())

	_, exists := roleRepo.roles["temp"]
	assert.False(t, exists)
}

func TestRoleController_Delete_NotFound(t *testing.T) {
	ctl, _, _ := setupRole()

	rc := newCtx("DELETE", "/api/v1/admin/roles/999", "")
	setParam(rc, "id", "999")
	ctl.Delete(context.Background(), rc)

	assert.Equal(t, http.StatusNotFound, rc.Response.StatusCode())
}

func TestRoleController_Delete_Builtin(t *testing.T) {
	ctl, roleRepo, _ := setupRole()

	now := time.Now()
	role := entity.Role{ID: 100, Code: "admin", Name: "Admin", IsBuiltin: true, CreatedAt: now, UpdatedAt: now}
	roleRepo.roles["admin"] = &role

	rc := newCtx("DELETE", "/api/v1/admin/roles/100", "")
	setParam(rc, "id", "100")
	ctl.Delete(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.ValidationFailed), body.Code)
}

// ===========================================================================
// TenantController Tests
// ===========================================================================

func TestTenantController_CreateTenant_Success(t *testing.T) {
	ctl, tenantRepo, _ := setupTenant()

	rc := newCtx("POST", "/api/v1/admin/tenants", `{"name":"Test School","code":"test_school","plan":"standard","max_users":100}`)
	ctl.CreateTenant(context.Background(), rc)

	assert.Equal(t, http.StatusCreated, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)
	assert.Equal(t, "created", body.Message)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Test School", data["name"])
	assert.Equal(t, "test_school", data["code"])
	assert.Equal(t, "standard", data["plan"])
	assert.Equal(t, float64(100), data["max_users"])
	assert.NotEmpty(t, tenantRepo.tenants)
}

func TestTenantController_CreateTenant_InvalidJSON(t *testing.T) {
	ctl, _, _ := setupTenant()

	rc := newCtx("POST", "/api/v1/admin/tenants", `{bad}`)
	ctl.CreateTenant(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
}

func TestTenantController_CreateTenant_MissingFields(t *testing.T) {
	ctl, _, _ := setupTenant()

	rc := newCtx("POST", "/api/v1/admin/tenants", `{}`)
	ctl.CreateTenant(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.InvalidParams), body.Code)
}

func TestTenantController_CreateTenant_InvalidPlan(t *testing.T) {
	ctl, _, _ := setupTenant()

	rc := newCtx("POST", "/api/v1/admin/tenants", `{"name":"School","code":"sch","plan":"invalid_plan"}`)
	ctl.CreateTenant(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.InvalidParams), body.Code)
}

func TestTenantController_CreateTenant_DuplicateCode(t *testing.T) {
	ctl, tenantRepo, _ := setupTenant()

	existing := &entity.Tenant{}
	existing.ID = 1
	existing.Name = "Old"
	existing.Code = "dup"
	existing.Plan = entity.PlanStandard
	existing.Status = entity.TenantStatusActive
	existing.MaxUsers = 50
	tenantRepo.tenants[1] = existing

	rc := newCtx("POST", "/api/v1/admin/tenants", `{"name":"New","code":"dup","plan":"standard"}`)
	ctl.CreateTenant(context.Background(), rc)

	assert.Equal(t, http.StatusConflict, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.AlreadyExists), body.Code)
}

func TestTenantController_ListTenants_Success(t *testing.T) {
	ctl, tenantRepo, _ := setupTenant()

	now := time.Now()
	t1 := newTenant(1, "School1", "s1", entity.PlanStandard, entity.TenantStatusActive, 50, now)
	t2 := newTenant(2, "School2", "s2", entity.PlanPremium, entity.TenantStatusActive, 100, now)
	tenantRepo.tenants[1] = t1
	tenantRepo.tenants[2] = t2

	rc := newCtx("GET", "/api/v1/admin/tenants?page=1&page_size=10", "")
	ctl.ListTenants(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), data["page"])
	assert.Equal(t, float64(10), data["page_size"])
	assert.Equal(t, float64(2), data["total"])
	assert.Len(t, data["items"].([]interface{}), 2)
}

func TestTenantController_ListTenants_WithStatusFilter(t *testing.T) {
	ctl, tenantRepo, _ := setupTenant()

	now := time.Now()
	t1 := newTenant(1, "Active", "a", "", entity.TenantStatusActive, 10, now)
	t2 := newTenant(2, "Suspended", "s", "", entity.TenantStatusSuspended, 10, now)
	tenantRepo.tenants[1] = t1
	tenantRepo.tenants[2] = t2

	rc := newCtx("GET", "/api/v1/admin/tenants?status=suspended", "")
	ctl.ListTenants(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	data := body.Data.(map[string]interface{})
	assert.Equal(t, float64(1), data["total"])
}

func TestTenantController_ListTenants_DefaultPagination(t *testing.T) {
	ctl, _, _ := setupTenant()

	rc := newCtx("GET", "/api/v1/admin/tenants", "")
	ctl.ListTenants(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	data := body.Data.(map[string]interface{})
	assert.Equal(t, float64(1), data["page"])
	assert.Equal(t, float64(20), data["page_size"])
}

func TestTenantController_GetTenant_Success(t *testing.T) {
	ctl, tenantRepo, _ := setupTenant()

	now := time.Now()
	tenant := newTenant(1, "Test", "test", entity.PlanStandard, entity.TenantStatusActive, 100, now)
	tenantRepo.tenants[1] = tenant

	rc := newCtx("GET", "/api/v1/admin/tenants/1", "")
	setParam(rc, "id", "1")
	ctl.GetTenant(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), data["id"])
	assert.Equal(t, "Test", data["name"])
	assert.Equal(t, "test", data["code"])
}

func TestTenantController_GetTenant_NotFound(t *testing.T) {
	ctl, _, _ := setupTenant()

	rc := newCtx("GET", "/api/v1/admin/tenants/999", "")
	setParam(rc, "id", "999")
	ctl.GetTenant(context.Background(), rc)

	assert.Equal(t, http.StatusNotFound, rc.Response.StatusCode())
}

func TestTenantController_GetTenant_InvalidID(t *testing.T) {
	ctl, _, _ := setupTenant()

	rc := newCtx("GET", "/api/v1/admin/tenants/abc", "")
	setParam(rc, "id", "abc")
	ctl.GetTenant(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
}

func TestTenantController_UpdateTenant_Success(t *testing.T) {
	ctl, tenantRepo, _ := setupTenant()

	now := time.Now()
	tenant := newTenant(1, "Old", "test", entity.PlanStandard, entity.TenantStatusActive, 50, now)
	tenantRepo.tenants[1] = tenant

	rc := newCtx("PUT", "/api/v1/admin/tenants/1", `{"name":"New","max_users":200}`)
	setParam(rc, "id", "1")
	ctl.UpdateTenant(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseResponse(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "New", data["name"])
	assert.Equal(t, float64(200), data["max_users"])
}

func TestTenantController_UpdateTenant_NotFound(t *testing.T) {
	ctl, _, _ := setupTenant()

	rc := newCtx("PUT", "/api/v1/admin/tenants/999", `{"name":"Test"}`)
	setParam(rc, "id", "999")
	ctl.UpdateTenant(context.Background(), rc)

	assert.Equal(t, http.StatusNotFound, rc.Response.StatusCode())
}

func TestTenantController_UpdateTenant_InvalidJSON(t *testing.T) {
	ctl, _, _ := setupTenant()

	rc := newCtx("PUT", "/api/v1/admin/tenants/1", `{bad}`)
	setParam(rc, "id", "1")
	ctl.UpdateTenant(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
}

func TestTenantController_UpdateTenant_InvalidID(t *testing.T) {
	ctl, _, _ := setupTenant()

	rc := newCtx("PUT", "/api/v1/admin/tenants/abc", `{"name":"Test"}`)
	setParam(rc, "id", "abc")
	ctl.UpdateTenant(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
}

func TestTenantController_DeleteTenant_Success(t *testing.T) {
	ctl, tenantRepo, _ := setupTenant()

	now := time.Now()
	tenant := newTenant(1, "Test", "test", entity.PlanStandard, entity.TenantStatusActive, 50, now)
	tenantRepo.tenants[1] = tenant

	rc := newCtx("DELETE", "/api/v1/admin/tenants/1", "")
	setParam(rc, "id", "1")
	ctl.DeleteTenant(context.Background(), rc)

	assert.Equal(t, http.StatusNoContent, rc.Response.StatusCode())

	_, exists := tenantRepo.tenants[1]
	assert.False(t, exists)
}

func TestTenantController_DeleteTenant_NotFound(t *testing.T) {
	ctl, _, _ := setupTenant()

	rc := newCtx("DELETE", "/api/v1/admin/tenants/999", "")
	setParam(rc, "id", "999")
	ctl.DeleteTenant(context.Background(), rc)

	assert.Equal(t, http.StatusNotFound, rc.Response.StatusCode())
}

func TestTenantController_DeleteTenant_InvalidID(t *testing.T) {
	ctl, _, _ := setupTenant()

	rc := newCtx("DELETE", "/api/v1/admin/tenants/abc", "")
	setParam(rc, "id", "abc")
	ctl.DeleteTenant(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
}