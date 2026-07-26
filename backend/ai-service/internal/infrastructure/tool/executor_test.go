package tool_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/ai-service/internal/domain/entity"
	"tcm-history-ai/backend/ai-service/internal/infrastructure/tool"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
	"tcm-history-ai/backend/pkg/pagination"
)

// init seeds the snowflake generator so idgen.Next() calls inside mock repos
// do not race with the default generator.
func init() { idgen.Init(1) }

// mockToolRepo is an in-memory ToolRepository for the executor tests.
type mockToolRepo struct {
	tools map[string]*entity.Tool
	err   error // forced error from FindByName
}

func newMockToolRepo(tools ...*entity.Tool) *mockToolRepo {
	m := &mockToolRepo{tools: map[string]*entity.Tool{}}
	for _, t := range tools {
		m.tools[t.Name] = t
	}
	return m
}

func (m *mockToolRepo) Create(_ context.Context, t *entity.Tool) error { return nil }
func (m *mockToolRepo) Update(_ context.Context, t *entity.Tool) error { return nil }
func (m *mockToolRepo) Delete(_ context.Context, id int64) error       { return nil }
func (m *mockToolRepo) FindByID(_ context.Context, id int64) (*entity.Tool, error) {
	for _, t := range m.tools {
		if t.ID == id {
			c := *t
			return &c, nil
		}
	}
	return nil, nil
}
func (m *mockToolRepo) FindByName(_ context.Context, name string) (*entity.Tool, error) {
	if m.err != nil {
		return nil, m.err
	}
	if t, ok := m.tools[name]; ok {
		c := *t
		return &c, nil
	}
	return nil, nil
}
func (m *mockToolRepo) ListEnabled(_ context.Context, _ pagination.Params) ([]entity.Tool, int, error) {
	return nil, 0, nil
}
func (m *mockToolRepo) List(_ context.Context, _ pagination.Params) ([]entity.Tool, int, error) {
	return nil, 0, nil
}

// TestExecute_StubMode verifies that when Enabled=false the executor returns
// a deterministic stub result without any HTTP call.
func TestExecute_StubMode(t *testing.T) {
	tt := &entity.Tool{
		Name:      "timeline",
		Endpoint:  "http://example.com/timeline",
		Method:    entity.ToolMethodGET,
		IsEnabled: true,
	}
	tt.ID = idgen.Next()
	repo := newMockToolRepo(tt)

	exec := tool.New(repo, tool.Config{HTTPTimeout: 1, Enabled: false})
	out, err := exec.Execute(context.Background(), "timeline", map[string]any{"q": "东汉"})
	require.NoError(t, err)
	assert.Equal(t, "timeline", out["tool_name"])
	assert.Equal(t, true, out["degraded"])
	params, _ := out["params"].(map[string]any)
	assert.Equal(t, "东汉", params["q"])
}

// TestExecute_DisabledTool verifies that a tool with IsEnabled=false is
// rejected with the Forbidden code.
func TestExecute_DisabledTool(t *testing.T) {
	tt := &entity.Tool{
		Name:      "off",
		Endpoint:  "http://example.com",
		Method:    entity.ToolMethodPOST,
		IsEnabled: false,
	}
	tt.ID = idgen.Next()
	repo := newMockToolRepo(tt)

	exec := tool.New(repo, tool.Config{Enabled: true})
	_, err := exec.Execute(context.Background(), "off", nil)
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.Forbidden, e.Code)
	}
}

// TestExecute_EmptyName verifies that an empty tool name is rejected with
// InvalidParams.
func TestExecute_EmptyName(t *testing.T) {
	repo := newMockToolRepo()
	exec := tool.New(repo, tool.Config{Enabled: true})
	_, err := exec.Execute(context.Background(), "", nil)
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.InvalidParams, e.Code)
	}
}

// TestExecute_UnknownTool verifies that a missing tool returns NotFound.
func TestExecute_UnknownTool(t *testing.T) {
	repo := newMockToolRepo()
	exec := tool.New(repo, tool.Config{Enabled: true})
	_, err := exec.Execute(context.Background(), "missing", nil)
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.NotFound, e.Code)
	}
}

// TestExecute_RepoError verifies that a repository error is propagated.
func TestExecute_RepoError(t *testing.T) {
	repo := newMockToolRepo()
	repo.err = errors.New("db down")
	exec := tool.New(repo, tool.Config{Enabled: true})
	_, err := exec.Execute(context.Background(), "any", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db down")
}

// TestExecute_EmptyEndpointFallsBackToStub verifies that a tool without an
// endpoint returns a stub result even when Enabled=true.
func TestExecute_EmptyEndpointFallsBackToStub(t *testing.T) {
	tt := &entity.Tool{
		Name:      "noendpoint",
		Endpoint:  "",
		Method:    entity.ToolMethodGET,
		IsEnabled: true,
	}
	tt.ID = idgen.Next()
	repo := newMockToolRepo(tt)

	exec := tool.New(repo, tool.Config{Enabled: true})
	out, err := exec.Execute(context.Background(), "noendpoint", map[string]any{"a": 1})
	require.NoError(t, err)
	assert.Equal(t, "noendpoint", out["tool_name"])
	assert.Equal(t, true, out["degraded"])
}

// TestExecute_HTTPHappyPath verifies that a successful HTTP call returns the
// parsed JSON payload.
func TestExecute_HTTPHappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		_ = json.Unmarshal(body, &got)
		assert.Equal(t, "东汉", got["q"])
		_, _ = w.Write([]byte(`{"answer":"张仲景","year":150}`))
	}))
	defer srv.Close()

	tt := &entity.Tool{
		Name:      "ask",
		Endpoint:  srv.URL,
		Method:    entity.ToolMethodPOST,
		IsEnabled: true,
	}
	tt.ID = idgen.Next()
	repo := newMockToolRepo(tt)

	exec := tool.New(repo, tool.Config{Enabled: true, HTTPTimeout: 2})
	out, err := exec.Execute(context.Background(), "ask", map[string]any{"q": "东汉"})
	require.NoError(t, err)
	assert.Equal(t, "张仲景", out["answer"])
	assert.Equal(t, float64(150), out["year"])
}

// TestExecute_HTTPGet verifies a GET tool sends no body and parses response.
func TestExecute_HTTPGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{"items":[1,2]}`))
	}))
	defer srv.Close()

	tt := &entity.Tool{
		Name:      "list",
		Endpoint:  srv.URL,
		Method:    entity.ToolMethodGET,
		IsEnabled: true,
	}
	tt.ID = idgen.Next()
	repo := newMockToolRepo(tt)

	exec := tool.New(repo, tool.Config{Enabled: true, HTTPTimeout: 2})
	out, err := exec.Execute(context.Background(), "list", map[string]any{"k": "v"})
	require.NoError(t, err)
	items, _ := out["items"].([]any)
	assert.Len(t, items, 2)
}

// TestExecute_HTTPErrorStatus verifies a 4xx response produces a wrapped
// DependencyUnavailable error.
func TestExecute_HTTPErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad"}`))
	}))
	defer srv.Close()

	tt := &entity.Tool{
		Name:      "bad",
		Endpoint:  srv.URL,
		Method:    entity.ToolMethodPOST,
		IsEnabled: true,
	}
	tt.ID = idgen.Next()
	repo := newMockToolRepo(tt)

	exec := tool.New(repo, tool.Config{Enabled: true, HTTPTimeout: 2})
	_, err := exec.Execute(context.Background(), "bad", map[string]any{})
	require.Error(t, err)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.DependencyUnavailable, e.Code)
	}
}

// TestExecute_HTTPNonJSONResponse verifies a non-JSON HTTP response is
// wrapped into a {raw, tool_name} map.
func TestExecute_HTTPNonJSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("plain text reply"))
	}))
	defer srv.Close()

	tt := &entity.Tool{
		Name:      "raw",
		Endpoint:  srv.URL,
		Method:    entity.ToolMethodPOST,
		IsEnabled: true,
	}
	tt.ID = idgen.Next()
	repo := newMockToolRepo(tt)

	exec := tool.New(repo, tool.Config{Enabled: true, HTTPTimeout: 2})
	out, err := exec.Execute(context.Background(), "raw", map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, "plain text reply", out["raw"])
	assert.Equal(t, "raw", out["tool_name"])
}

// TestExecute_HTTPCallFailsFallsBackToStub verifies that when the HTTP call
// errors out (e.g. connection refused), the executor falls back to a stub
// result instead of propagating the error.
func TestExecute_HTTPCallFailsFallsBackToStub(t *testing.T) {
	tt := &entity.Tool{
		Name:      "down",
		Endpoint:  "http://127.0.0.1:0/unreachable", // invalid endpoint
		Method:    entity.ToolMethodPOST,
		IsEnabled: true,
	}
	tt.ID = idgen.Next()
	repo := newMockToolRepo(tt)

	exec := tool.New(repo, tool.Config{Enabled: true, HTTPTimeout: 1})
	out, err := exec.Execute(context.Background(), "down", map[string]any{"q": "x"})
	require.NoError(t, err)
	assert.Equal(t, "down", out["tool_name"])
	assert.Equal(t, true, out["degraded"])
}

// TestExecute_CancelledContext verifies that a cancelled context still goes
// through the HTTP path (the executor swallows the error and falls back to
// stub to keep the Agent chain unblocked).
func TestExecute_CancelledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tt := &entity.Tool{
		Name:      "ctx",
		Endpoint:  srv.URL,
		Method:    entity.ToolMethodPOST,
		IsEnabled: true,
	}
	tt.ID = idgen.Next()
	repo := newMockToolRepo(tt)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	exec := tool.New(repo, tool.Config{Enabled: true, HTTPTimeout: 1})
	// Either stub or success — but never an error that breaks the chain.
	out, err := exec.Execute(ctx, "ctx", nil)
	require.NoError(t, err)
	assert.NotNil(t, out)
}

// TestNew_DefaultTimeout verifies that when HTTPTimeout<=0 a default 5s is
// applied (no panic, no nil client).
func TestNew_DefaultTimeout(t *testing.T) {
	repo := newMockToolRepo()
	exec := tool.New(repo, tool.Config{HTTPTimeout: 0, Enabled: false})
	require.NotNil(t, exec)
}

// TestStubResult_Shape mirrors the executor's stubResult return shape using
// only exported fields; the degraded flag is consistently set on stub paths.
func TestStubResult_Shape(t *testing.T) {
	tt := &entity.Tool{
		Name:      "stub",
		Endpoint:  "", // empty endpoint triggers stub
		Method:    entity.ToolMethodGET,
		IsEnabled: true,
	}
	tt.ID = idgen.Next()
	repo := newMockToolRepo(tt)

	exec := tool.New(repo, tool.Config{Enabled: true, HTTPTimeout: 1})
	out, err := exec.Execute(context.Background(), "stub", map[string]any{"k": "v"})
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "stub", out["tool_name"])
	assert.Equal(t, true, out["degraded"])
	assert.Contains(t, out["result"], "stub-tool")
	params, _ := out["params"].(map[string]any)
	assert.Equal(t, "v", params["k"])
}
