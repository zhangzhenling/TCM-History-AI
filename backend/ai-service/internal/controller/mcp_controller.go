package controller

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"tcm-history-ai/backend/ai-service/internal/infrastructure/mcp"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/response"
)

// MCPController exposes the MCP protocol endpoints (SSE + JSON-RPC).
type MCPController struct {
	server *mcp.Server
}

// NewMCPController constructs an MCPController.
func NewMCPController(server *mcp.Server) *MCPController {
	return &MCPController{server: server}
}

// SSE GET /mcp/sse
// Establishes a Server-Sent Events stream for MCP communication.
func (h *MCPController) SSE(ctx context.Context, c *app.RequestContext) {
	// CORS headers for SSE.
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
	c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

	if string(c.Method()) == "OPTIONS" {
		c.SetStatusCode(204)
		return
	}

	// Create a session.
	sessionID, ch := h.server.NewSession()
	defer h.server.DropSession(sessionID)

	// Send the endpoint URL as the first event.
	c.SetContentType("text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// Write the initial endpoint event.
	endpoint := "/mcp/message?session_id=" + sessionID
	c.WriteString("event: endpoint\n")
	c.WriteString("data: " + endpoint + "\n\n")

	// Flush headers.
	if flusher, ok := c.GetWriter().(interface{ Flush() error }); ok {
		_ = flusher.Flush()
	}

	// Stream responses from the session channel.
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case resp, ok := <-ch:
			if !ok {
				return
			}
			raw, _ := json.Marshal(resp)
			c.WriteString("event: message\n")
			c.WriteString("data: " + string(raw) + "\n\n")
			if flusher, ok := c.GetWriter().(interface{ Flush() error }); ok {
				_ = flusher.Flush()
			}
		case <-ticker.C:
			// Keep-alive comment.
			c.WriteString(":keepalive\n\n")
			if flusher, ok := c.GetWriter().(interface{ Flush() error }); ok {
				_ = flusher.Flush()
			}
		case <-ctx.Done():
			return
		}
	}
}

// Message POST /mcp/message
// Receives JSON-RPC requests from the client and returns responses.
// For SSE transport, responses are pushed via the session channel.
// For simple HTTP transport, the response is returned synchronously.
func (h *MCPController) Message(ctx context.Context, c *app.RequestContext) {
	body, err := io.ReadAll(c.Request.BodyStream())
	if err != nil {
		response.FailWith(ctx, c, errno.InvalidParams, "read body: "+err.Error())
		return
	}

	req, mcpErr := mcp.ParseRequest(body)
	if mcpErr != nil {
		writeMCPResponse(ctx, c, mcp.NewErrorResponse(nil, mcpErr))
		return
	}

	// Extract scopes from Authorization header for rudimentary auth.
	scopes := scopesFromHeader(c)
	if len(scopes) > 0 {
		ctx = mcp.WithScopes(ctx, scopes)
	}

	resp := h.server.HandleRequest(ctx, req)

	// If session_id is present, push via SSE; else return directly.
	sessionID := string(c.Query("session_id"))
	if sessionID != "" {
		h.server.SendToSession(sessionID, resp)
		// Acknowledge the request.
		c.SetStatusCode(202)
		c.WriteString("Accepted")
		return
	}

	writeMCPResponse(ctx, c, resp)
}

// writeMCPResponse writes a JSON-RPC response with the correct content type.
func writeMCPResponse(ctx context.Context, c *app.RequestContext, resp *mcp.Response) {
	c.SetContentType("application/json")
	if resp.Error != nil {
		status := 200
		switch resp.Error.Code {
		case mcp.ErrMCPUnauthorized:
			status = 401
		case mcp.ErrMCPForbidden:
			status = 403
		case mcp.ErrMCPRateLimited:
			status = 429
		case mcp.ErrMCPBackendTimeout:
			status = 504
		case mcp.ErrMCPBackendUnavailable:
			status = 503
		case mcp.ErrInvalidParams:
			status = 422
		case mcp.ErrMethodNotFound:
			status = 404
		}
		c.SetStatusCode(status)
	}
	raw, _ := json.Marshal(resp)
	c.Write(raw)
}

// scopesFromHeader extracts bearer token scopes from the Authorization header.
// In production this validates the API key against the database and loads scopes.
// For the initial implementation we accept any bearer token and grant all scopes
// so that the MCP protocol can be exercised end-to-end.
func scopesFromHeader(c *app.RequestContext) []string {
	auth := string(c.GetHeader("Authorization"))
	if auth == "" {
		return nil
	}
	// Bearer token present — grant all scopes for end-to-end testing.
	if strings.HasPrefix(auth, "Bearer ") {
		return []string{
			"history:read", "person:read", "school:read", "book:read",
			"graph:read", "search:read", "medicine:read", "prescription:read",
		}
	}
	return nil
}