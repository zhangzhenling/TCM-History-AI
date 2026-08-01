package controller_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route/param"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/ai-service/internal/application/usecase"
	"tcm-history-ai/backend/ai-service/internal/controller"
	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/ai-service/internal/domain/event"
	"tcm-history-ai/backend/ai-service/internal/domain/service"
	"tcm-history-ai/backend/ai-service/internal/infrastructure/prompt"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
	"tcm-history-ai/backend/pkg/response"
)

func init() { idgen.Init(1) }

// ============================================================================
// Mock: ConversationRepository
// ============================================================================

type mockConvRepo struct {
	mu    sync.Mutex
	items map[int64]*entity.Conversation
	create func(*entity.Conversation) error
	update func(*entity.Conversation) error
	del    func(int64) error
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
	if m.del != nil {
		return m.del(id)
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
// Mock: MessageRepository
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
	var out []entity.Message
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
// Mock: AgentRunRepository
// ============================================================================

type mockAgentRunRepo struct {
	mu     sync.Mutex
	items  map[int64]*entity.AgentRun
	create func(*entity.AgentRun) error
	update func(*entity.AgentRun) error
	find   func(int64) (*entity.AgentRun, error)
	list   func(pagination.Params) ([]entity.AgentRun, int, error)
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

func (m *mockAgentRunRepo) ListByConversation(_ context.Context, conversationID int64, _ pagination.Params) ([]entity.AgentRun, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var all []entity.AgentRun
	for _, a := range m.items {
		if a.ConversationID == conversationID {
			all = append(all, *a)
		}
	}
	return all, len(all), nil
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
// Mock: PromptTemplateRepository
// ============================================================================

type mockPromptRepo struct {
	mu         sync.Mutex
	items      map[int64]*entity.PromptTemplate
	findActive func(string) (*entity.PromptTemplate, error)
}

func newMockPromptRepo() *mockPromptRepo {
	return &mockPromptRepo{items: map[int64]*entity.PromptTemplate{}}
}

func (m *mockPromptRepo) Create(_ context.Context, p *entity.PromptTemplate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[p.ID] = p
	return nil
}

func (m *mockPromptRepo) Update(_ context.Context, p *entity.PromptTemplate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[p.ID] = p
	return nil
}

func (m *mockPromptRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, id)
	return nil
}

func (m *mockPromptRepo) FindByID(_ context.Context, id int64) (*entity.PromptTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.items[id]; ok {
		clone := *p
		return &clone, nil
	}
	return nil, nil
}

func (m *mockPromptRepo) FindByNameAndScene(_ context.Context, _, _ string) (*entity.PromptTemplate, error) {
	return nil, nil
}

func (m *mockPromptRepo) ListByScene(_ context.Context, _ string, _ pagination.Params) ([]entity.PromptTemplate, int, error) {
	return nil, 0, nil
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

func (m *mockPromptRepo) List(_ context.Context, _ pagination.Params) ([]entity.PromptTemplate, int, error) {
	return nil, 0, nil
}

func (m *mockPromptRepo) DeactivateByScene(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

// ============================================================================
// Mock: ToolRepository
// ============================================================================

type mockToolRepo struct {
	mu    sync.Mutex
	items map[int64]*entity.Tool
}

func newMockToolRepo() *mockToolRepo {
	return &mockToolRepo{items: map[int64]*entity.Tool{}}
}

func (m *mockToolRepo) Create(_ context.Context, t *entity.Tool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[t.ID] = t
	return nil
}

func (m *mockToolRepo) Update(_ context.Context, t *entity.Tool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[t.ID] = t
	return nil
}

func (m *mockToolRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, id)
	return nil
}

func (m *mockToolRepo) FindByID(_ context.Context, id int64) (*entity.Tool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.items[id]; ok {
		clone := *t
		return &clone, nil
	}
	return nil, nil
}

func (m *mockToolRepo) FindByName(_ context.Context, _ string) (*entity.Tool, error) {
	return nil, nil
}

func (m *mockToolRepo) ListEnabled(_ context.Context, _ pagination.Params) ([]entity.Tool, int, error) {
	return nil, 0, nil
}

func (m *mockToolRepo) List(_ context.Context, _ pagination.Params) ([]entity.Tool, int, error) {
	return nil, 0, nil
}

// ============================================================================
// Mock: LLMProvider
// ============================================================================

type mockLLMProvider struct {
	model    string
	chatFn   func(ctx context.Context, req service.LLMChatRequest) (*service.LLMChatResponse, error)
	chatResp *service.LLMChatResponse
	chatErr  error
}

func newMockLLMProvider() *mockLLMProvider {
	return &mockLLMProvider{model: "test-model"}
}

func (m *mockLLMProvider) Chat(ctx context.Context, req service.LLMChatRequest) (*service.LLMChatResponse, error) {
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
// Mock: ToolExecutor
// ============================================================================

type mockToolExecutor struct {
	execFn func(ctx context.Context, toolName string, params map[string]any) (map[string]any, error)
}

func (m *mockToolExecutor) Execute(ctx context.Context, name string, params map[string]any) (map[string]any, error) {
	if m.execFn != nil {
		return m.execFn(ctx, name, params)
	}
	return map[string]any{"ok": true}, nil
}

// ============================================================================
// Mock: RetrievalClient
// ============================================================================

type mockRetrievalClient struct {
	retrieveFn func(ctx context.Context, query string, topK int) (*service.RetrieveResult, error)
	searchFn   func(ctx context.Context, keyword, label string, limit int) (*service.GraphSearchResult, error)
}

func (m *mockRetrievalClient) Retrieve(ctx context.Context, query string, topK int) (*service.RetrieveResult, error) {
	if m.retrieveFn != nil {
		return m.retrieveFn(ctx, query, topK)
	}
	return &service.RetrieveResult{Query: query, TopK: topK, Total: 0}, nil
}

func (m *mockRetrievalClient) SearchNodes(ctx context.Context, keyword, label string, limit int) (*service.GraphSearchResult, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, keyword, label, limit)
	}
	return &service.GraphSearchResult{Keyword: keyword, Label: label, Total: 0}, nil
}

// ============================================================================
// Mock: EventPublisher
// ============================================================================

type mockEventPublisher struct {
	mu     sync.Mutex
	events []event.Event
	err    error
}

func newMockEventPublisher() *mockEventPublisher {
	return &mockEventPublisher{}
}

func (m *mockEventPublisher) Publish(_ context.Context, evt event.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.events = append(m.events, evt)
	return nil
}

// ============================================================================
// Helper: parse response body
// ============================================================================

func parseBody(t *testing.T, rc *app.RequestContext) response.Body {
	t.Helper()
	var body response.Body
	require.NoError(t, json.Unmarshal(rc.Response.Body(), &body))
	return body
}

// ============================================================================
// ChatController Tests
// ============================================================================

func newTestChatController(
	convRepo *mockConvRepo,
	msgRepo *mockMsgRepo,
	promptRepo *mockPromptRepo,
	llm service.LLMProvider,
	pub event.EventPublisher,
) *controller.ChatController {
	uc := usecase.NewChatUseCase(convRepo, msgRepo, promptRepo, llm, prompt.New(), pub)
	return controller.NewChatController(uc)
}

func TestChatController_Chat_HappyPath(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()

	h := newTestChatController(convRepo, msgRepo, promptRepo, llm, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("POST")
	rc.Request.SetRequestURI("/api/v1/ai/chat")
	rc.Request.Header.Set("X-User-ID", "42")
	rc.Request.SetBody([]byte(`{"message":"hello"}`))

	h.Chat(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseBody(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.NotZero(t, data["conversation_id"])
	assert.NotZero(t, data["message_id"])
	assert.Equal(t, "stub-answer", data["assistant"])
	assert.Equal(t, "test-model", data["model"])
}

func TestChatController_Chat_InvalidJSON(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()

	h := newTestChatController(convRepo, msgRepo, promptRepo, llm, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("POST")
	rc.Request.SetRequestURI("/api/v1/ai/chat")
	rc.Request.SetBody([]byte(`not-json`))

	h.Chat(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
	body := parseBody(t, rc)
	assert.Equal(t, int(errno.InvalidParams), body.Code)
}

func TestChatController_Chat_USecaseError(t *testing.T) {
	convRepo := newMockConvRepo()
	convRepo.create = func(*entity.Conversation) error {
		return errors.New("db error")
	}
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()

	h := newTestChatController(convRepo, msgRepo, promptRepo, llm, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("POST")
	rc.Request.SetRequestURI("/api/v1/ai/chat")
	rc.Request.Header.Set("X-User-ID", "1")
	rc.Request.SetBody([]byte(`{"message":"hello"}`))

	h.Chat(context.Background(), rc)

	assert.Equal(t, http.StatusInternalServerError, rc.Response.StatusCode())
	body := parseBody(t, rc)
	assert.Equal(t, int(errno.InternalError), body.Code)
}

func TestChatController_Chat_UserIDFromBody(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()

	h := newTestChatController(convRepo, msgRepo, promptRepo, llm, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("POST")
	rc.Request.SetRequestURI("/api/v1/ai/chat")
	rc.Request.SetBody([]byte(`{"user_id":99,"message":"hello"}`))

	h.Chat(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
}

func TestChatController_Chat_HeaderOverridesBody(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()

	h := newTestChatController(convRepo, msgRepo, promptRepo, llm, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("POST")
	rc.Request.SetRequestURI("/api/v1/ai/chat")
	rc.Request.Header.Set("X-User-ID", "42")
	rc.Request.SetBody([]byte(`{"user_id":99,"message":"hello"}`))

	h.Chat(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
}

func TestChatController_ListConversations_OK(t *testing.T) {
	convRepo := newMockConvRepo()
	for i := 0; i < 3; i++ {
		c := &entity.Conversation{UserID: 42, Mode: entity.ConversationModeChat, Status: entity.ConversationStatusActive}
		c.ID = idgen.Next()
		require.NoError(t, convRepo.Create(context.Background(), c))
	}
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()

	h := newTestChatController(convRepo, msgRepo, promptRepo, llm, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/ai/conversations?page=1&page_size=2")
	rc.Request.Header.Set("X-User-ID", "42")

	h.ListConversations(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseBody(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(3), data["total"])
	assert.Equal(t, float64(1), data["page"])
	assert.Equal(t, float64(2), data["page_size"])
	items, ok := data["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 2)
}

func TestChatController_ListConversations_NoHeader(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()

	h := newTestChatController(convRepo, msgRepo, promptRepo, llm, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/ai/conversations")

	h.ListConversations(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseBody(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)
}

func TestChatController_ListConversations_Error(t *testing.T) {
	convRepo := newMockConvRepo()
	convRepo.list = func(int64, pagination.Params) ([]entity.Conversation, int, error) {
		return nil, 0, errors.New("list error")
	}
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()

	h := newTestChatController(convRepo, msgRepo, promptRepo, llm, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/ai/conversations")
	rc.Request.Header.Set("X-User-ID", "1")

	h.ListConversations(context.Background(), rc)

	assert.Equal(t, http.StatusInternalServerError, rc.Response.StatusCode())
	body := parseBody(t, rc)
	assert.Equal(t, int(errno.InternalError), body.Code)
}

func TestChatController_GetConversation_OK(t *testing.T) {
	convRepo := newMockConvRepo()
	c := &entity.Conversation{UserID: 1, Title: "Test", Mode: entity.ConversationModeChat, Status: entity.ConversationStatusActive}
	c.ID = idgen.Next()
	require.NoError(t, convRepo.Create(context.Background(), c))

	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()

	h := newTestChatController(convRepo, msgRepo, promptRepo, llm, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/ai/conversations/" + itoa(c.ID))
	setParam(rc, "id", itoa(c.ID))

	h.GetConversation(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseBody(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Test", data["title"])
	assert.Equal(t, float64(c.ID), data["id"])
}

func TestChatController_GetConversation_InvalidID(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()

	h := newTestChatController(convRepo, msgRepo, promptRepo, llm, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/ai/conversations/abc")
	setParam(rc, "id", "abc")

	h.GetConversation(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
	body := parseBody(t, rc)
	assert.Equal(t, int(errno.InvalidParams), body.Code)
}

func TestChatController_GetConversation_NotFound(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()

	h := newTestChatController(convRepo, msgRepo, promptRepo, llm, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/ai/conversations/99999")
	setParam(rc, "id", "99999")

	h.GetConversation(context.Background(), rc)

	assert.Equal(t, http.StatusNotFound, rc.Response.StatusCode())
	body := parseBody(t, rc)
	assert.Equal(t, int(errno.NotFound), body.Code)
}

func TestChatController_ListMessages_OK(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	for i := 0; i < 3; i++ {
		m := &entity.Message{ConversationID: 1, Role: entity.MessageRoleUser, Content: "hello"}
		m.ID = idgen.Next()
		require.NoError(t, msgRepo.Create(context.Background(), m))
	}
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()

	h := newTestChatController(convRepo, msgRepo, promptRepo, llm, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/ai/conversations/1/messages?page=1&page_size=2")
	setParam(rc, "id", "1")

	h.ListMessages(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseBody(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(3), data["total"])
	items, ok := data["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 2)
}

func TestChatController_ListMessages_InvalidID(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()

	h := newTestChatController(convRepo, msgRepo, promptRepo, llm, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/ai/conversations/abc/messages")
	setParam(rc, "id", "abc")

	h.ListMessages(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
	body := parseBody(t, rc)
	assert.Equal(t, int(errno.InvalidParams), body.Code)
}

func TestChatController_DeleteConversation_OK(t *testing.T) {
	convRepo := newMockConvRepo()
	c := &entity.Conversation{UserID: 1, Mode: entity.ConversationModeChat, Status: entity.ConversationStatusActive}
	c.ID = idgen.Next()
	require.NoError(t, convRepo.Create(context.Background(), c))

	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()

	h := newTestChatController(convRepo, msgRepo, promptRepo, llm, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("DELETE")
	rc.Request.SetRequestURI("/api/v1/ai/conversations/" + itoa(c.ID))
	setParam(rc, "id", itoa(c.ID))

	h.DeleteConversation(context.Background(), rc)

	assert.Equal(t, http.StatusNoContent, rc.Response.StatusCode())
}

func TestChatController_DeleteConversation_InvalidID(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()

	h := newTestChatController(convRepo, msgRepo, promptRepo, llm, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("DELETE")
	rc.Request.SetRequestURI("/api/v1/ai/conversations/abc")
	setParam(rc, "id", "abc")

	h.DeleteConversation(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
	body := parseBody(t, rc)
	assert.Equal(t, int(errno.InvalidParams), body.Code)
}

func TestChatController_DeleteConversation_NotFound(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	promptRepo := newMockPromptRepo()
	llm := newMockLLMProvider()
	pub := newMockEventPublisher()

	h := newTestChatController(convRepo, msgRepo, promptRepo, llm, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("DELETE")
	rc.Request.SetRequestURI("/api/v1/ai/conversations/99999")
	setParam(rc, "id", "99999")

	h.DeleteConversation(context.Background(), rc)

	assert.Equal(t, http.StatusNotFound, rc.Response.StatusCode())
	body := parseBody(t, rc)
	assert.Equal(t, int(errno.NotFound), body.Code)
}

// ============================================================================
// AgentController Tests
// ============================================================================

func newTestAgentController(
	convRepo *mockConvRepo,
	msgRepo *mockMsgRepo,
	agentRepo *mockAgentRunRepo,
	promptRepo *mockPromptRepo,
	toolRepo *mockToolRepo,
	llm service.LLMProvider,
	toolExec service.ToolExecutor,
	retriever service.RetrievalClient,
	pub event.EventPublisher,
) *controller.AgentController {
	uc := usecase.NewAgentUseCase(
		convRepo, msgRepo, agentRepo, promptRepo, toolRepo,
		llm, toolExec, retriever,
		prompt.New(), pub,
		nil, nil,
	)
	return controller.NewAgentController(uc)
}

func TestAgentController_Run_HappyPath(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	agentRepo := newMockAgentRunRepo()
	promptRepo := newMockPromptRepo()
	toolRepo := newMockToolRepo()
	llm := newMockLLMProvider()
	toolExec := &mockToolExecutor{}
	retriever := &mockRetrievalClient{}
	pub := newMockEventPublisher()

	h := newTestAgentController(convRepo, msgRepo, agentRepo, promptRepo, toolRepo, llm, toolExec, retriever, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("POST")
	rc.Request.SetRequestURI("/api/v1/ai/agents/run")
	rc.Request.Header.Set("X-User-ID", "42")
	rc.Request.SetBody([]byte(`{"question":"What is the theory of TCm?"}`))

	h.Run(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseBody(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.NotZero(t, data["agent_run_id"])
	assert.NotZero(t, data["conversation_id"])
	assert.Equal(t, "stub-answer", data["answer"])
	assert.Equal(t, "done", data["status"])
}

func TestAgentController_Run_InvalidJSON(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	agentRepo := newMockAgentRunRepo()
	promptRepo := newMockPromptRepo()
	toolRepo := newMockToolRepo()
	llm := newMockLLMProvider()
	toolExec := &mockToolExecutor{}
	retriever := &mockRetrievalClient{}
	pub := newMockEventPublisher()

	h := newTestAgentController(convRepo, msgRepo, agentRepo, promptRepo, toolRepo, llm, toolExec, retriever, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("POST")
	rc.Request.SetRequestURI("/api/v1/ai/agents/run")
	rc.Request.SetBody([]byte(`not-json`))

	h.Run(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
	body := parseBody(t, rc)
	assert.Equal(t, int(errno.InvalidParams), body.Code)
}

func TestAgentController_Run_HeaderUserID(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	agentRepo := newMockAgentRunRepo()
	promptRepo := newMockPromptRepo()
	toolRepo := newMockToolRepo()
	llm := newMockLLMProvider()
	toolExec := &mockToolExecutor{}
	retriever := &mockRetrievalClient{}
	pub := newMockEventPublisher()

	h := newTestAgentController(convRepo, msgRepo, agentRepo, promptRepo, toolRepo, llm, toolExec, retriever, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("POST")
	rc.Request.SetRequestURI("/api/v1/ai/agents/run")
	rc.Request.Header.Set("X-User-ID", "7")
	rc.Request.SetBody([]byte(`{"question":"test"}`))

	h.Run(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseBody(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)
}

func TestAgentController_Run_USecaseError(t *testing.T) {
	convRepo := newMockConvRepo()
	convRepo.create = func(*entity.Conversation) error {
		return errors.New("db error")
	}
	msgRepo := newMockMsgRepo()
	agentRepo := newMockAgentRunRepo()
	promptRepo := newMockPromptRepo()
	toolRepo := newMockToolRepo()
	llm := newMockLLMProvider()
	toolExec := &mockToolExecutor{}
	retriever := &mockRetrievalClient{}
	pub := newMockEventPublisher()

	h := newTestAgentController(convRepo, msgRepo, agentRepo, promptRepo, toolRepo, llm, toolExec, retriever, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("POST")
	rc.Request.SetRequestURI("/api/v1/ai/agents/run")
	rc.Request.Header.Set("X-User-ID", "1")
	rc.Request.SetBody([]byte(`{"question":"test"}`))

	h.Run(context.Background(), rc)

	assert.Equal(t, http.StatusInternalServerError, rc.Response.StatusCode())
	body := parseBody(t, rc)
	assert.Equal(t, int(errno.InternalError), body.Code)
}

func TestAgentController_ListAgentRuns_OK(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	agentRepo := newMockAgentRunRepo()
	for i := 0; i < 3; i++ {
		a := &entity.AgentRun{
			ConversationID: 1,
			UserID:         42,
			Status:         entity.AgentRunStatusDone,
			FinalAnswer:    "answer",
		}
		a.ID = idgen.Next()
		require.NoError(t, agentRepo.Create(context.Background(), a))
	}
	promptRepo := newMockPromptRepo()
	toolRepo := newMockToolRepo()
	llm := newMockLLMProvider()
	toolExec := &mockToolExecutor{}
	retriever := &mockRetrievalClient{}
	pub := newMockEventPublisher()

	h := newTestAgentController(convRepo, msgRepo, agentRepo, promptRepo, toolRepo, llm, toolExec, retriever, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/ai/agent-runs?page=1&page_size=2")

	h.ListAgentRuns(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseBody(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(3), data["total"])
	items, ok := data["items"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 2)
}

func TestAgentController_ListAgentRuns_Error(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	agentRepo := newMockAgentRunRepo()
	agentRepo.list = func(pagination.Params) ([]entity.AgentRun, int, error) {
		return nil, 0, errors.New("list error")
	}
	promptRepo := newMockPromptRepo()
	toolRepo := newMockToolRepo()
	llm := newMockLLMProvider()
	toolExec := &mockToolExecutor{}
	retriever := &mockRetrievalClient{}
	pub := newMockEventPublisher()

	h := newTestAgentController(convRepo, msgRepo, agentRepo, promptRepo, toolRepo, llm, toolExec, retriever, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/ai/agent-runs")

	h.ListAgentRuns(context.Background(), rc)

	assert.Equal(t, http.StatusInternalServerError, rc.Response.StatusCode())
	body := parseBody(t, rc)
	assert.Equal(t, int(errno.InternalError), body.Code)
}

func TestAgentController_GetAgentRun_OK(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	agentRepo := newMockAgentRunRepo()
	a := &entity.AgentRun{
		ConversationID: 1,
		UserID:         42,
		Status:         entity.AgentRunStatusDone,
		FinalAnswer:    "answer",
	}
	a.ID = idgen.Next()
	require.NoError(t, agentRepo.Create(context.Background(), a))

	promptRepo := newMockPromptRepo()
	toolRepo := newMockToolRepo()
	llm := newMockLLMProvider()
	toolExec := &mockToolExecutor{}
	retriever := &mockRetrievalClient{}
	pub := newMockEventPublisher()

	h := newTestAgentController(convRepo, msgRepo, agentRepo, promptRepo, toolRepo, llm, toolExec, retriever, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/ai/agent-runs/" + itoa(a.ID))
	setParam(rc, "id", itoa(a.ID))

	h.GetAgentRun(context.Background(), rc)

	assert.Equal(t, http.StatusOK, rc.Response.StatusCode())
	body := parseBody(t, rc)
	assert.Equal(t, int(errno.OK), body.Code)

	data, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(a.ID), data["id"])
	assert.Equal(t, "answer", data["final_answer"])
	assert.Equal(t, "done", data["status"])
}

func TestAgentController_GetAgentRun_InvalidID(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	agentRepo := newMockAgentRunRepo()
	promptRepo := newMockPromptRepo()
	toolRepo := newMockToolRepo()
	llm := newMockLLMProvider()
	toolExec := &mockToolExecutor{}
	retriever := &mockRetrievalClient{}
	pub := newMockEventPublisher()

	h := newTestAgentController(convRepo, msgRepo, agentRepo, promptRepo, toolRepo, llm, toolExec, retriever, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/ai/agent-runs/abc")
	setParam(rc, "id", "abc")

	h.GetAgentRun(context.Background(), rc)

	assert.Equal(t, http.StatusBadRequest, rc.Response.StatusCode())
	body := parseBody(t, rc)
	assert.Equal(t, int(errno.InvalidParams), body.Code)
}

func TestAgentController_GetAgentRun_NotFound(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	agentRepo := newMockAgentRunRepo()
	promptRepo := newMockPromptRepo()
	toolRepo := newMockToolRepo()
	llm := newMockLLMProvider()
	toolExec := &mockToolExecutor{}
	retriever := &mockRetrievalClient{}
	pub := newMockEventPublisher()

	h := newTestAgentController(convRepo, msgRepo, agentRepo, promptRepo, toolRepo, llm, toolExec, retriever, pub)

	rc := app.NewContext(0)
	rc.Request.SetMethod("GET")
	rc.Request.SetRequestURI("/api/v1/ai/agent-runs/99999")
	setParam(rc, "id", "99999")

	h.GetAgentRun(context.Background(), rc)

	assert.Equal(t, http.StatusNotFound, rc.Response.StatusCode())
	body := parseBody(t, rc)
	assert.Equal(t, int(errno.NotFound), body.Code)
}

// itoa converts int64 to string.
func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}

// setParam sets a path parameter on the request context.
func setParam(rc *app.RequestContext, key, value string) {
	rc.Params = param.Params{{Key: key, Value: value}}
}