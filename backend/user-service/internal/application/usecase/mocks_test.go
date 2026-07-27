package usecase_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
	"tcm-history-ai/backend/user-service/internal/domain/entity"
	"tcm-history-ai/backend/user-service/internal/domain/event"
	"tcm-history-ai/backend/user-service/internal/domain/service"
)

// TestMain seeds the snowflake id generator so entity IDs are deterministic
// and non-zero across the usecase test suite.
func TestMain(m *testing.M) {
	idgen.Init(1)
	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// UserRepository mock
// ---------------------------------------------------------------------------

// mockUserRepo is an in-memory fake repository.UserRepository. Every method
// has an optional function field; when set, the field is called instead of the
// default in-memory behaviour. This lets individual tests inject errors or
// custom return values without touching the shared state.
type mockUserRepo struct {
	items           map[int64]*entity.User
	create          func(*entity.User) error
	findByID        func(int64) (*entity.User, error)
	findByUsername  func(string) (*entity.User, error)
	findByEmail     func(string) (*entity.User, error)
	findByPhone     func(string) (*entity.User, error)
	updateLastLogin func(int64, time.Time, string) error
	update          func(*entity.User) error
	delete          func(int64) error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{items: map[int64]*entity.User{}}
}

func (m *mockUserRepo) Create(_ context.Context, u *entity.User) error {
	if m.create != nil {
		return m.create(u)
	}
	if _, ok := m.items[u.ID]; ok {
		return errno.New(errno.AlreadyExists, "user exists")
	}
	m.items[u.ID] = u
	return nil
}

func (m *mockUserRepo) FindByID(_ context.Context, id int64) (*entity.User, error) {
	if m.findByID != nil {
		return m.findByID(id)
	}
	if u, ok := m.items[id]; ok {
		return cloneUser(u), nil
	}
	return nil, nil
}

func (m *mockUserRepo) FindByUsername(_ context.Context, username string) (*entity.User, error) {
	if m.findByUsername != nil {
		return m.findByUsername(username)
	}
	for _, u := range m.items {
		if u.Username == username {
			return cloneUser(u), nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) FindByEmail(_ context.Context, email string) (*entity.User, error) {
	if m.findByEmail != nil {
		return m.findByEmail(email)
	}
	for _, u := range m.items {
		if u.Email != nil && *u.Email == email {
			return cloneUser(u), nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) FindByPhone(_ context.Context, phone string) (*entity.User, error) {
	if m.findByPhone != nil {
		return m.findByPhone(phone)
	}
	for _, u := range m.items {
		if u.Phone != nil && *u.Phone == phone {
			return cloneUser(u), nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) UpdateLastLogin(_ context.Context, id int64, at time.Time, ip string) error {
	if m.updateLastLogin != nil {
		return m.updateLastLogin(id, at, ip)
	}
	u, ok := m.items[id]
	if !ok {
		return errno.New(errno.NotFound, "user not found")
	}
	u.LastLoginAt = &at
	u.LastLoginIP = ip
	return nil
}

func (m *mockUserRepo) Update(_ context.Context, u *entity.User) error {
	if m.update != nil {
		return m.update(u)
	}
	if _, ok := m.items[u.ID]; !ok {
		return errno.New(errno.NotFound, "user not found")
	}
	m.items[u.ID] = u
	return nil
}

func (m *mockUserRepo) Delete(_ context.Context, id int64) error {
	if m.delete != nil {
		return m.delete(id)
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "user not found")
	}
	delete(m.items, id)
	return nil
}

func (m *mockUserRepo) List(_ context.Context, p pagination.Params, status string) ([]entity.User, int64, error) {
	out := make([]entity.User, 0, len(m.items))
	for _, u := range m.items {
		if status == "" || u.Status == status {
			out = append(out, *cloneUser(u))
		}
	}
	return out, int64(len(out)), nil
}

// cloneUser returns a deep copy of u so callers cannot mutate the stored row.
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
// RoleRepository mock
// ---------------------------------------------------------------------------

type mockRoleRepo struct {
	roles       map[string]*entity.Role // keyed by code
	userRoles   map[int64][]entity.Role // keyed by userID
	assignments map[int64][]int64       // userID → roleIDs assigned
	findByCode  func(string) (*entity.Role, error)
	findByUser  func(int64) ([]entity.Role, error)
	listAll     func() ([]entity.Role, error)
	assignRole  func(int64, int64) error
}

func newMockRoleRepo() *mockRoleRepo {
	return &mockRoleRepo{
		roles:       map[string]*entity.Role{},
		userRoles:   map[int64][]entity.Role{},
		assignments: map[int64][]int64{},
	}
}

func (m *mockRoleRepo) FindByCode(_ context.Context, code string) (*entity.Role, error) {
	if m.findByCode != nil {
		return m.findByCode(code)
	}
	if r, ok := m.roles[code]; ok {
		c := *r
		return &c, nil
	}
	return nil, nil
}

func (m *mockRoleRepo) FindByUserID(_ context.Context, userID int64) ([]entity.Role, error) {
	if m.findByUser != nil {
		return m.findByUser(userID)
	}
	out := make([]entity.Role, len(m.userRoles[userID]))
	copy(out, m.userRoles[userID])
	return out, nil
}

func (m *mockRoleRepo) ListAll(_ context.Context) ([]entity.Role, error) {
	if m.listAll != nil {
		return m.listAll()
	}
	out := make([]entity.Role, 0, len(m.roles))
	for _, r := range m.roles {
		out = append(out, *r)
	}
	return out, nil
}

func (m *mockRoleRepo) AssignRole(_ context.Context, userID, roleID int64) error {
	if m.assignRole != nil {
		return m.assignRole(userID, roleID)
	}
	m.assignments[userID] = append(m.assignments[userID], roleID)
	return nil
}

func (m *mockRoleRepo) FindByID(_ context.Context, id int64) (*entity.Role, error) {
	for _, r := range m.roles {
		if r.ID == id {
			c := *r
			return &c, nil
		}
	}
	return nil, nil
}

func (m *mockRoleRepo) Create(_ context.Context, r *entity.Role) error {
	m.roles[r.Code] = r
	return nil
}

func (m *mockRoleRepo) Update(_ context.Context, r *entity.Role) error {
	m.roles[r.Code] = r
	return nil
}

func (m *mockRoleRepo) Delete(_ context.Context, id int64) error {
	for code, r := range m.roles {
		if r.ID == id {
			delete(m.roles, code)
			return nil
		}
	}
	return errno.New(errno.NotFound, "role not found")
}

func (m *mockRoleRepo) SetUserRoles(_ context.Context, userID int64, roleIDs []int64) error {
	m.assignments[userID] = roleIDs
	roles := make([]entity.Role, 0, len(roleIDs))
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
	return nil
}

func (m *mockRoleRepo) RevokePermission(_ context.Context, roleID, permissionID int64) error {
	return nil
}

// seedStudentRole inserts the built-in student role into the mock and returns
// it, so tests don't have to repeat the boilerplate.
func (m *mockRoleRepo) seedStudentRole() *entity.Role {
	r := &entity.Role{ID: 100, Code: entity.RoleStudent, Name: "Student", IsBuiltin: true}
	m.roles[entity.RoleStudent] = r
	return r
}

// ---------------------------------------------------------------------------
// ProfileRepository mock
// ---------------------------------------------------------------------------

type mockProfileRepo struct {
	items        map[int64]*entity.UserProfile // keyed by userID
	create       func(*entity.UserProfile) error
	findByUserID func(int64) (*entity.UserProfile, error)
	update       func(*entity.UserProfile) error
}

func newMockProfileRepo() *mockProfileRepo {
	return &mockProfileRepo{items: map[int64]*entity.UserProfile{}}
}

func (m *mockProfileRepo) Create(_ context.Context, p *entity.UserProfile) error {
	if m.create != nil {
		return m.create(p)
	}
	if _, ok := m.items[p.UserID]; ok {
		return errno.New(errno.AlreadyExists, "profile exists")
	}
	m.items[p.UserID] = p
	return nil
}

func (m *mockProfileRepo) FindByUserID(_ context.Context, userID int64) (*entity.UserProfile, error) {
	if m.findByUserID != nil {
		return m.findByUserID(userID)
	}
	if p, ok := m.items[userID]; ok {
		c := *p
		return &c, nil
	}
	return nil, nil
}

func (m *mockProfileRepo) Update(_ context.Context, p *entity.UserProfile) error {
	if m.update != nil {
		return m.update(p)
	}
	m.items[p.UserID] = p
	return nil
}

// ---------------------------------------------------------------------------
// SettingsRepository mock
// ---------------------------------------------------------------------------

type mockSettingsRepo struct {
	items        map[int64]*entity.UserSettings // keyed by userID
	create       func(*entity.UserSettings) error
	findByUserID func(int64) (*entity.UserSettings, error)
	update       func(*entity.UserSettings) error
}

func newMockSettingsRepo() *mockSettingsRepo {
	return &mockSettingsRepo{items: map[int64]*entity.UserSettings{}}
}

func (m *mockSettingsRepo) Create(_ context.Context, s *entity.UserSettings) error {
	if m.create != nil {
		return m.create(s)
	}
	if _, ok := m.items[s.UserID]; ok {
		return errno.New(errno.AlreadyExists, "settings exists")
	}
	m.items[s.UserID] = s
	return nil
}

func (m *mockSettingsRepo) FindByUserID(_ context.Context, userID int64) (*entity.UserSettings, error) {
	if m.findByUserID != nil {
		return m.findByUserID(userID)
	}
	if s, ok := m.items[userID]; ok {
		c := *s
		return &c, nil
	}
	return nil, nil
}

func (m *mockSettingsRepo) Update(_ context.Context, s *entity.UserSettings) error {
	if m.update != nil {
		return m.update(s)
	}
	m.items[s.UserID] = s
	return nil
}

// ---------------------------------------------------------------------------
// PermissionRepository mock
// ---------------------------------------------------------------------------

type mockPermissionRepo struct {
	findByRoleID func(int64) ([]entity.Permission, error)
	findByUser   func(int64) ([]entity.Permission, error)
	listAll      func() ([]entity.Permission, error)
}

func (m *mockPermissionRepo) FindByRoleID(_ context.Context, roleID int64) ([]entity.Permission, error) {
	if m.findByRoleID != nil {
		return m.findByRoleID(roleID)
	}
	return nil, nil
}

func (m *mockPermissionRepo) FindByUserID(_ context.Context, userID int64) ([]entity.Permission, error) {
	if m.findByUser != nil {
		return m.findByUser(userID)
	}
	return nil, nil
}

func (m *mockPermissionRepo) ListAll(_ context.Context) ([]entity.Permission, error) {
	if m.listAll != nil {
		return m.listAll()
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// PasswordHasher mock
// ---------------------------------------------------------------------------

// fakeHasher implements service.PasswordHasher with a deterministic,
// non-cryptographic hash so tests stay fast.
type fakeHasher struct {
	hashFn   func(string) (string, error)
	verifyFn func(string, string) bool
}

func (f *fakeHasher) Hash(password string) (string, error) {
	if f.hashFn != nil {
		return f.hashFn(password)
	}
	return "hash:" + password, nil
}

func (f *fakeHasher) Verify(password, hash string) bool {
	if f.verifyFn != nil {
		return f.verifyFn(password, hash)
	}
	return hash == "hash:"+password
}

// ---------------------------------------------------------------------------
// TokenManager mock
// ---------------------------------------------------------------------------

// fakeTokenManager implements service.TokenManager using simple string tokens
// of the form "<prefix>:<userID>:<comma-joined-roles>". The prefix encodes
// the issuer so access and refresh tokens cannot be cross-used.
type fakeTokenManager struct {
	accessTTL      time.Duration
	refreshTTL     time.Duration
	issueAccessFn  func(int64, []string) (string, error)
	issueRefreshFn func(int64, []string) (string, error)
	parseAccessFn  func(string) (*service.Claims, error)
	parseRefreshFn func(string) (*service.Claims, error)
}

func newFakeTokenManager() *fakeTokenManager {
	return &fakeTokenManager{
		accessTTL:  15 * time.Minute,
		refreshTTL: 24 * time.Hour,
	}
}

func (m *fakeTokenManager) IssueAccessToken(userID int64, roles []string) (string, error) {
	if m.issueAccessFn != nil {
		return m.issueAccessFn(userID, roles)
	}
	return issueFake("access", userID, roles), nil
}

func (m *fakeTokenManager) IssueRefreshToken(userID int64, roles []string) (string, error) {
	if m.issueRefreshFn != nil {
		return m.issueRefreshFn(userID, roles)
	}
	return issueFake("refresh", userID, roles), nil
}

func (m *fakeTokenManager) ParseAccessToken(token string) (*service.Claims, error) {
	if m.parseAccessFn != nil {
		return m.parseAccessFn(token)
	}
	return parseFake(token, "access")
}

func (m *fakeTokenManager) ParseRefreshToken(token string) (*service.Claims, error) {
	if m.parseRefreshFn != nil {
		return m.parseRefreshFn(token)
	}
	return parseFake(token, "refresh")
}

func (m *fakeTokenManager) AccessTokenTTL() time.Duration  { return m.accessTTL }
func (m *fakeTokenManager) RefreshTokenTTL() time.Duration { return m.refreshTTL }

func issueFake(prefix string, userID int64, roles []string) string {
	return fmt.Sprintf("%s:%d:%s", prefix, userID, strings.Join(roles, ","))
}

func parseFake(token, expectedPrefix string) (*service.Claims, error) {
	parts := strings.SplitN(token, ":", 3)
	if len(parts) < 2 || parts[0] != expectedPrefix {
		return nil, fmt.Errorf("invalid token: expected prefix %q", expectedPrefix)
	}
	if _, err := strconv.ParseInt(parts[1], 10, 64); err != nil {
		return nil, fmt.Errorf("invalid subject: %w", err)
	}
	var roles []string
	if len(parts) == 3 && parts[2] != "" {
		roles = strings.Split(parts[2], ",")
	}
	now := time.Now()
	return &service.Claims{
		Sub:   parts[1],
		Roles: roles,
		Exp:   now.Add(time.Hour).Unix(),
		Iat:   now.Unix(),
	}, nil
}

// ---------------------------------------------------------------------------
// RefreshTokenStore mock (in-memory map)
// ---------------------------------------------------------------------------

type memoryRefreshStore struct {
	mu    sync.Mutex
	items map[int64]string
	setFn func(int64, string, time.Duration) error
	getFn func(int64) (string, error)
	delFn func(int64) error
}

func newMemoryRefreshStore() *memoryRefreshStore {
	return &memoryRefreshStore{items: map[int64]string{}}
}

func (s *memoryRefreshStore) Set(_ context.Context, userID int64, token string, _ time.Duration) error {
	if s.setFn != nil {
		return s.setFn(userID, token, 0)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[userID] = token
	return nil
}

func (s *memoryRefreshStore) Get(_ context.Context, userID int64) (string, error) {
	if s.getFn != nil {
		return s.getFn(userID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tok, ok := s.items[userID]
	if !ok {
		return "", fmt.Errorf("refresh token not found for user %d", userID)
	}
	return tok, nil
}

func (s *memoryRefreshStore) Delete(_ context.Context, userID int64) error {
	if s.delFn != nil {
		return s.delFn(userID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, userID)
	return nil
}

// ---------------------------------------------------------------------------
// EventPublisher mock
// ---------------------------------------------------------------------------

type mockEventPublisher struct {
	mu        sync.Mutex
	events    []event.Event
	publishFn func(event.Event) error
}

func newMockEventPublisher() *mockEventPublisher {
	return &mockEventPublisher{}
}

func (p *mockEventPublisher) Publish(_ context.Context, e event.Event) error {
	if p.publishFn != nil {
		return p.publishFn(e)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, e)
	return nil
}

func (p *mockEventPublisher) collected() []event.Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]event.Event, len(p.events))
	copy(out, p.events)
	return out
}
