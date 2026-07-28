package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"tcm-history-ai/backend/pkg/errno"
)

// Executor is the port for actually running a Tool call.
// Implementations route to the appropriate backend service.
type Executor interface {
	// Execute runs the named tool with the given parameters.
	// The returned map is the tool result; the degraded flag indicates
	// whether the result was produced by a fallback path.
	Execute(ctx context.Context, toolName string, params map[string]any) (result map[string]any, degraded bool, err error)
}

// Server is the MCP JSON-RPC handler.
// It is transport-agnostic: SSE, stdio, or plain HTTP can feed it
// Requests and emit Responses.
type Server struct {
	registry *Registry
	executor Executor

	// sessionID counter for SSE sessions.
	sessionSeq atomic.Int64
	// active sessions (sessionID -> send channel).
	sessions   sync.Map // string -> chan *Response
}

// NewServer constructs an MCP Server backed by the given registry and executor.
func NewServer(registry *Registry, executor Executor) *Server {
	return &Server{
		registry: registry,
		executor: executor,
	}
}

// RegisterBuiltinTools adds the 8 pre-defined TCM tools to the registry.
func (s *Server) RegisterBuiltinTools() {
	for _, t := range BuiltInTools() {
		s.registry.Register(t)
	}
}

// HandleRequest processes a single JSON-RPC request and returns the response.
// It is safe for concurrent use.
func (s *Server) HandleRequest(ctx context.Context, req *Request) *Response {
	switch req.Method {
	case "tools/list":
		return s.handleToolsList(ctx, req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	case "initialize":
		return s.handleInitialize(ctx, req)
	default:
		return NewErrorResponse(req.ID, NewError(ErrMethodNotFound, "unknown method: "+req.Method))
	}
}

// handleInitialize responds to the MCP initialize handshake.
func (s *Server) handleInitialize(_ context.Context, req *Request) *Response {
	result := map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]any{
			"tools": map[string]any{"listChanged": true},
		},
		"serverInfo": map[string]any{
			"name":    "tcm-history-ai-mcp",
			"version": "1.0.0",
		},
	}
	return NewResponse(req.ID, result)
}

// handleToolsList returns the list of enabled tools visible to the caller.
func (s *Server) handleToolsList(ctx context.Context, req *Request) *Response {
	// Scope filtering: in a real deployment the caller's scopes are injected
	// into ctx by auth middleware.  For now we return all enabled tools.
	scopes := scopesFromContext(ctx)
	tools := s.registry.List(scopes)

	// Convert to MCP wire format (name, description, inputSchema).
	type wireTool struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		InputSchema json.RawMessage `json:"inputSchema"`
	}
	list := make([]wireTool, 0, len(tools))
	for _, t := range tools {
		list = append(list, wireTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}
	return NewResponse(req.ID, map[string]any{"tools": list})
}

// handleToolsCall executes a single tool call.
func (s *Server) handleToolsCall(ctx context.Context, req *Request) *Response {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return NewErrorResponse(req.ID, NewError(ErrInvalidParams, "invalid params: "+err.Error()))
	}
	if params.Name == "" {
		return NewErrorResponse(req.ID, NewError(ErrInvalidParams, "name is required"))
	}

	meta, ok := s.registry.Get(params.Name)
	if !ok {
		return NewErrorResponse(req.ID, NewError(ErrInvalidParams, "tool not found: "+params.Name))
	}

	// Validate scopes.
	scopes := scopesFromContext(ctx)
	set := make(map[string]struct{}, len(scopes))
	for _, s := range scopes {
		set[s] = struct{}{}
	}
	if !scopesSatisfy(set, meta.RequiredScopes) {
		return NewErrorResponse(req.ID, NewError(ErrMCPForbidden,
			fmt.Sprintf("missing scope for %s", params.Name)))
	}

	// Execute via the executor port.
	result, degraded, err := s.executor.Execute(ctx, params.Name, params.Arguments)
	if err != nil {
		code := ErrInternalError
		msg := err.Error()
		if e, ok := err.(*errno.Error); ok {
			switch e.Code {
			case errno.DependencyUnavailable:
				code = ErrMCPBackendUnavailable
			case errno.InvalidParams:
				code = ErrInvalidParams
			case errno.Forbidden:
				code = ErrMCPForbidden
			}
			msg = e.Message
		}
		return NewErrorResponse(req.ID, NewError(code, msg))
	}

	if degraded {
		result["_degraded"] = true
	}
	return NewResponse(req.ID, result)
}

// scopesFromContext extracts caller scopes from context.
// In production this reads a value injected by auth middleware.
func scopesFromContext(ctx context.Context) []string {
	if ctx == nil {
		return nil
	}
	v := ctx.Value(scopesKey)
	if v == nil {
		return nil
	}
	if s, ok := v.([]string); ok {
		return s
	}
	return nil
}

// WithScopes injects scopes into a context for testing / middleware.
func WithScopes(ctx context.Context, scopes []string) context.Context {
	return context.WithValue(ctx, scopesKey, scopes)
}

type scopeKey struct{}

var scopesKey = scopeKey{}

// SSE session management -----------------------------------------------------

// NewSession creates a new SSE session and returns its ID and receive channel.
func (s *Server) NewSession() (sessionID string, ch <-chan *Response) {
	id := fmt.Sprintf("sess-%d", s.sessionSeq.Add(1))
	c := make(chan *Response, 16)
	s.sessions.Store(id, c)
	return id, c
}

// DropSession closes and removes an SSE session.
func (s *Server) DropSession(sessionID string) {
	if v, ok := s.sessions.LoadAndDelete(sessionID); ok {
		close(v.(chan *Response))
	}
}

// SendToSession pushes a response to an SSE session (non-blocking).
func (s *Server) SendToSession(sessionID string, resp *Response) bool {
	v, ok := s.sessions.Load(sessionID)
	if !ok {
		return false
	}
	select {
	case v.(chan *Response) <- resp:
		return true
	default:
		return false
	}
}
