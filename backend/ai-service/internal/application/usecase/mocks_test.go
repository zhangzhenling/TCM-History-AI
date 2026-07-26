package usecase_test

import (
	"context"
	"sync"
	"time"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/ai-service/internal/domain/event"
	"tcm-history-ai/backend/ai-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

// init seeds the snowflake generator so idgen.Next() calls in use cases do
// not panic.
func init() { idgen.Init(1) }

// ============================================================================
// ConversationRepository mock
// ============================================================================

// mockConvRepo is an in-memory ConversationRepository.
type mockConvRepo struct {
	mu     sync.Mutex
	items  map[int64]*entity.Conversation
	create func(*entity.Conversation) error
	update func(*entity.Conversation) error
	delete func(int64) error
	find   func(int64) (*entity.Conversation, error)
	list   func(int64, pagination.Params) ([]entity.Conversation, int, error)
}

func newMockConvRepo() *mockConvRepo {
	return &mockConvRepo{items: map[int64]*entity.Conversation{}}
}

func (m *mockConvRepo) Create(_ context.Context, c *entity.Conversation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.create != nil {
		return m.create(c)
	}
	m.items[c.ID] = c
	return nil
}

func (m *mockConvRepo) Update(_ context.Context, c *entity.Conversation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.update != nil {
		return m.update(c)
	}
	m.items[c.ID] = c
	return nil
}

func (m *mockConvRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.delete != nil {
		return m.delete(id)
	}
	if _, ok := m.items[id]; !ok {
		return errno.New(errno.NotFound, "conversation not found")
	}
	delete(m.items, id)
	return nil
}

func (m *mockConvRepo) FindByID(_ context.Context, id int64) (*entity.Conversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.find != nil {
		return m.find(id)
	}
	if c, ok := m.items[id]; ok {
		clone := *c
		return &clone, nil
	}
	return nil, nil
}

func (m *mockConvRepo) ListByUser(_ context.Context, userID int64, p pagination.Params) ([]entity.Conversation, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.list != nil {
		return m.list(userID, p)
	}
	all := make([]entity.Conversation, 0, len(m.items))
	for _, c := range m.items {
		if c.UserID == userID {
			all = append(all, *c)
		}
	}
	_, pageSize, offset := p.Normalise()
	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

// ============================================================================
// MessageRepository mock
// ============================================================================

type mockMsgRepo struct {
	mu       sync.Mutex
	items    map[int64]*entity.Message
	create   func(*entity.Message) error
	findHist func(int64) ([]entity.Message, error)
	list     func(int64, pagination.Params) ([]entity.Message, int, error)
}

func newMockMsgRepo() *mockMsgRepo {
	return &mockMsgRepo{items: map[int64]*entity.Message{}}
}

func (m *mockMsgRepo) Create(_ context.Context, msg *entity.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.create != nil {
		return m.create(msg)
	}
	m.items[msg.ID] = msg
	return nil
}

func (m *mockMsgRepo) FindByConversation(_ context.Context, conversationID int64) ([]entity.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.findHist != nil {
		return m.findHist(conversationID)
	}
	out := make([]entity.Message, 0)
	for _, msg := range m.items {
		if msg.ConversationID == conversationID {
			out = append(out, *msg)
		}
	}
	return out, nil
}

func (m *mockMsgRepo) ListByConversation(_ context.Context, conversationID int64, p pagination.Params) ([]entity.Message, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.list != nil {
		return m.list(conversationID, p)
	}
	all := make([]entity.Message, 0)
	for _, msg := range m.items {
		if msg.ConversationID == conversationID {
			all = append(all, *msg)
		}
	}
	_, pageSize, offset := p.Normalise()
	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

// ============================================================================
// PromptTemplateRepository mock
// ============================================================================

type mockPromptRepo struct {
	mu              sync.Mutex
	items           map[int64]*entity.PromptTemplate
	create          func(*entity.PromptTemplate) error
	update          func(*entity.PromptTemplate) error
	find            func(int64) (*entity.PromptTemplate, error)
	findByNameScene func(string, string) (*entity.PromptTemplate, error)
	listByScene     func(string, pagination.Params) ([]entity.PromptTemplate, int, error)
	findActive      func(string) (*entity.PromptTemplate, error)
	list            func(pagination.Params) ([]entity.PromptTemplate, int, error)
}

func newMockPromptRepo() *mockPromptRepo {
	return &mockPromptRepo{items: map[int64]*entity.PromptTemplate{}}
}

func (m *mockPromptRepo) Create(_ context.Context, p *entity.PromptTemplate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.create != nil {
		return m.create(p)
	}
	m.items[p.ID] = p
	return nil
}

func (m *mockPromptRepo) Update(_ context.Context, p *entity.PromptTemplate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.update != nil {
		return m.update(p)
	}
	m.items[p.ID] = p
	return nil
}

func (m *mockPromptRepo) FindByID(_ context.Context, id int64) (*entity.PromptTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.find != nil {
		return m.find(id)
	}
	if p, ok := m.items[id]; ok {
		clone := *p
		return &clone, nil
	}
	return nil, nil
}

func (m *mockPromptRepo) FindByNameAndScene(_ context.Context, name, scene string) (*entity.PromptTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.findByNameScene != nil {
		return m.findByNameScene(name, scene)
	}
	for _, p := range m.items {
		if p.Name == name && p.Scene == scene {
			clone := *p
			return &clone, nil
		}
	}
	return nil, nil
}

func (m *mockPromptRepo) ListByScene(_ context.Context, scene string, p pagination.Params) ([]entity.PromptTemplate, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listByScene != nil {
		return m.listByScene(scene, p)
	}
	all := make([]entity.PromptTemplate, 0, len(m.items))
	for _, p := range m.items {
		if p.Scene == scene {
			all = append(all, *p)
		}
	}
	_, pageSize, offset := p.Normalise()
	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (m *mockPromptRepo) FindActive(_ context.Context, scene string) (*entity.PromptTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.findActive != nil {
		return m.findActive(scene)
	}
	for _, p := range m.items {
		if p.Scene == scene && p.IsActive {
			clone := *p
			return &clone, nil
		}
	}
	return nil, nil
}

func (m *mockPromptRepo) List(_ context.Context, p pagination.Params) ([]entity.PromptTemplate, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.list != nil {
		return m.list(p)
	}
	all := make([]entity.PromptTemplate, 0, len(m.items))
	for _, p := range m.items {
		all = append(all, *p)
	}
	_, pageSize, offset := p.Normalise()
	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

// ============================================================================
// ToolRepository mock
// ============================================================================

type mockToolRepo struct {
	mu          sync.Mutex
	items       map[int64]*entity.Tool
	itemsByName map[string]*entity.Tool
	create      func(*entity.Tool) error
	update      func(*entity.Tool) error
	delete      func(int64) error
	find        func(int64) (*entity.Tool, error)
	findByName  func(string) (*entity.Tool, error)
	listEnabled func(pagination.Params) ([]entity.Tool, int, error)
	list        func(pagination.Params) ([]entity.Tool, int, error)
}

func newMockToolRepo() *mockToolRepo {
	return &mockToolRepo{items: map[int64]*entity.Tool{}, itemsByName: map[string]*entity.Tool{}}
}

func (m *mockToolRepo) Create(_ context.Context, t *entity.Tool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.create != nil {
		return m.create(t)
	}
	m.items[t.ID] = t
	m.itemsByName[t.Name] = t
	return nil
}

func (m *mockToolRepo) Update(_ context.Context, t *entity.Tool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.update != nil {
		return m.update(t)
	}
	m.items[t.ID] = t
	m.itemsByName[t.Name] = t
	return nil
}

func (m *mockToolRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.delete != nil {
		return m.delete(id)
	}
	if t, ok := m.items[id]; ok {
		delete(m.items, id)
		delete(m.itemsByName, t.Name)
	}
	return nil
}

func (m *mockToolRepo) FindByID(_ context.Context, id int64) (*entity.Tool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.find != nil {
		return m.find(id)
	}
	if t, ok := m.items[id]; ok {
		clone := *t
		return &clone, nil
	}
	return nil, nil
}

func (m *mockToolRepo) FindByName(_ context.Context, name string) (*entity.Tool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.findByName != nil {
		return m.findByName(name)
	}
	if t, ok := m.itemsByName[name]; ok {
		clone := *t
		return &clone, nil
	}
	return nil, nil
}

func (m *mockToolRepo) ListEnabled(_ context.Context, p pagination.Params) ([]entity.Tool, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listEnabled != nil {
		return m.listEnabled(p)
	}
	all := make([]entity.Tool, 0, len(m.items))
	for _, t := range m.items {
		if t.IsEnabled {
			all = append(all, *t)
		}
	}
	_, pageSize, offset := p.Normalise()
	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (m *mockToolRepo) List(_ context.Context, p pagination.Params) ([]entity.Tool, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.list != nil {
		return m.list(p)
	}
	all := make([]entity.Tool, 0, len(m.items))
	for _, t := range m.items {
		all = append(all, *t)
	}
	_, pageSize, offset := p.Normalise()
	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

// ============================================================================
// AgentRunRepository mock
// ============================================================================

type mockAgentRunRepo struct {
	mu      sync.Mutex
	items   map[int64]*entity.AgentRun
	create  func(*entity.AgentRun) error
	update  func(*entity.AgentRun) error
	find    func(int64) (*entity.AgentRun, error)
	list    func(pagination.Params) ([]entity.AgentRun, int, error)
}

func newMockAgentRunRepo() *mockAgentRunRepo {
	return &mockAgentRunRepo{items: map[int64]*entity.AgentRun{}}
}

func (m *mockAgentRunRepo) Create(_ context.Context, a *entity.AgentRun) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.create != nil {
		return m.create(a)
	}
	m.items[a.ID] = a
	return nil
}

func (m *mockAgentRunRepo) Update(_ context.Context, a *entity.AgentRun) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.update != nil {
		return m.update(a)
	}
	m.items[a.ID] = a
	return nil
}

func (m *mockAgentRunRepo) FindByID(_ context.Context, id int64) (*entity.AgentRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.find != nil {
		return m.find(id)
	}
	if a, ok := m.items[id]; ok {
		clone := *a
		return &clone, nil
	}
	return nil, nil
}

func (m *mockAgentRunRepo) ListByConversation(_ context.Context, conversationID int64, p pagination.Params) ([]entity.AgentRun, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	all := make([]entity.AgentRun, 0, len(m.items))
	for _, a := range m.items {
		if a.ConversationID == conversationID {
			all = append(all, *a)
		}
	}
	_, pageSize, offset := p.Normalise()
	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (m *mockAgentRunRepo) List(_ context.Context, p pagination.Params) ([]entity.AgentRun, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.list != nil {
		return m.list(p)
	}
	all := make([]entity.AgentRun, 0, len(m.items))
	for _, a := range m.items {
		all = append(all, *a)
	}
	_, pageSize, offset := p.Normalise()
	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

// ============================================================================
// LLMProvider mock
// ============================================================================

type mockLLMProvider struct {
	model    string
	chatFn   func(ctx context.Context, req service.LLMChatRequest) (*service.LLMChatResponse, error)
	compFn   func(ctx context.Context, prompt string) (string, error)
	chatResp *service.LLMChatResponse
	chatErr  error
	calls    int
}

func newMockLLMProvider() *mockLLMProvider {
	return &mockLLMProvider{model: "test-model"}
}

func (m *mockLLMProvider) Chat(ctx context.Context, req service.LLMChatRequest) (*service.LLMChatResponse, error) {
	m.calls++
	if m.chatFn != nil {
		return m.chatFn(ctx, req)
	}
	if m.chatErr != nil {
		return nil, m.chatErr
	}
	if m.chatResp != nil {
		return m.chatResp, nil
	}
	return &service.LLMChatResponse{
		Text:             "stub-answer",
		Model:            m.model,
		TokensPrompt:     10,
		TokensCompletion: 5,
		LatencyMs:        1,
	}, nil
}

func (m *mockLLMProvider) Complete(ctx context.Context, prompt string) (string, error) {
	if m.compFn != nil {
		return m.compFn(ctx, prompt)
	}
	resp, err := m.Chat(ctx, service.LLMChatRequest{
		Messages: []service.LLMMessage{{Role: entity.MessageRoleUser, Content: prompt}},
	})
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

func (m *mockLLMProvider) Model() string { return m.model }

// ============================================================================
// ToolExecutor mock
// ============================================================================

type mockToolExecutor struct {
	execFn func(ctx context.Context, toolName string, params map[string]any) (map[string]any, error)
	result map[string]any
	err    error
	calls  int
}

func (m *mockToolExecutor) Execute(ctx context.Context, name string, params map[string]any) (map[string]any, error) {
	m.calls++
	if m.execFn != nil {
		return m.execFn(ctx, name, params)
	}
	if m.err != nil {
		return nil, m.err
	}
	if m.result != nil {
		return m.result, nil
	}
	return map[string]any{"ok": true}, nil
}

// ============================================================================
// PromptRenderer mock (uses the real infra implementation by default).
// ============================================================================

type mockPromptRenderer struct {
	renderFn func(string, map[string]any) (string, error)
	result   string
	err      error
	calls    int
}

func (m *mockPromptRenderer) Render(tpl string, vars map[string]any) (string, error) {
	m.calls++
	if m.renderFn != nil {
		return m.renderFn(tpl, vars)
	}
	if m.err != nil {
		return "", m.err
	}
	return m.result, nil
}

// ============================================================================
// RetrievalClient mock
// ============================================================================

type mockRetrievalClient struct {
	retrieveFn   func(ctx context.Context, query string, topK int) (*service.RetrieveResult, error)
	searchFn     func(ctx context.Context, keyword, label string, limit int) (*service.GraphSearchResult, error)
	retrieveRes  *service.RetrieveResult
	retrieveErr  error
	searchRes    *service.GraphSearchResult
	searchErr    error
}

func (m *mockRetrievalClient) Retrieve(ctx context.Context, query string, topK int) (*service.RetrieveResult, error) {
	if m.retrieveFn != nil {
		return m.retrieveFn(ctx, query, topK)
	}
	if m.retrieveErr != nil {
		return nil, m.retrieveErr
	}
	if m.retrieveRes != nil {
		return m.retrieveRes, nil
	}
	return &service.RetrieveResult{Query: query, TopK: topK, Total: 0}, nil
}

func (m *mockRetrievalClient) SearchNodes(ctx context.Context, keyword, label string, limit int) (*service.GraphSearchResult, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, keyword, label, limit)
	}
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	if m.searchRes != nil {
		return m.searchRes, nil
	}
	return &service.GraphSearchResult{Keyword: keyword, Label: label, Total: 0}, nil
}

// ============================================================================
// EventPublisher mock
// ============================================================================

type mockEventPublisher struct {
	mu     sync.Mutex
	events []event.Event
	err    error
	pubFn  func(ctx context.Context, evt event.Event) error
}

func newMockEventPublisher() *mockEventPublisher {
	return &mockEventPublisher{}
}

func (m *mockEventPublisher) Publish(ctx context.Context, evt event.Event) error {
	if m.pubFn != nil {
		return m.pubFn(ctx, evt)
	}
	if m.err != nil {
		return m.err
	}
	m.mu.Lock()
	m.events = append(m.events, evt)
	m.mu.Unlock()
	return nil
}

// captureEvent returns the first event of type T published to the publisher,
// or nil if none was published. Useful for assertions in tests.
func captureEvent[T event.Event](pub *mockEventPublisher) (T, bool) {
	var zero T
	pub.mu.Lock()
	defer pub.mu.Unlock()
	for _, e := range pub.events {
		if v, ok := e.(T); ok {
			return v, true
		}
	}
	return zero, false
}

// sharedHelpers provides convenient stamps used across use case tests.
func mustTime(t time.Time) *time.Time { return &t }
