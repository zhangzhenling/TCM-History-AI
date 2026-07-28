package controller

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"tcm-history-ai/backend/pkg/response"
)

// Deps bundles every controller the router needs. It is populated by wire.
type Deps struct {
	Chat   *ChatController
	Agent  *AgentController
	Prompt *PromptController
	Tool   *ToolController
	Token  *TokenController
	MCP    *MCPController
}

// RegisterRoutes wires every AI Service route onto the Hertz server.
// Routes follow RESTful conventions under /api/v1/ai.
func RegisterRoutes(h *server.Hertz, deps *Deps) {
	h.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		response.OKWith(ctx, c, "ai-service up", map[string]any{
			"service": "ai-service",
			"status":  "ok",
		})
	})

	v1 := h.Group("/api/v1/ai")

	// Chat / Conversations
	v1.POST("/chat", deps.Chat.Chat)
	v1.GET("/conversations", deps.Chat.ListConversations)
	v1.GET("/conversations/:id", deps.Chat.GetConversation)
	v1.GET("/conversations/:id/messages", deps.Chat.ListMessages)
	v1.DELETE("/conversations/:id", deps.Chat.DeleteConversation)

	// Agent runs
	v1.POST("/agents/run", deps.Agent.Run)
	v1.GET("/agent-runs", deps.Agent.ListAgentRuns)
	v1.GET("/agent-runs/:id", deps.Agent.GetAgentRun)

	// Prompt templates
	v1.GET("/prompts", deps.Prompt.List)
	v1.POST("/prompts", deps.Prompt.Create)
	v1.GET("/prompts/:id", deps.Prompt.Get)
	v1.PUT("/prompts/:id", deps.Prompt.Update)
	v1.DELETE("/prompts/:id", deps.Prompt.Delete)
	v1.PATCH("/prompts/:id/activate", deps.Prompt.Activate)

	// Tools (MCP)
	v1.GET("/tools", deps.Tool.List)
	v1.POST("/tools", deps.Tool.Create)
	v1.PUT("/tools/:id", deps.Tool.Update)
	v1.DELETE("/tools/:id", deps.Tool.Delete)
	v1.POST("/tools/:id/execute", deps.Tool.Execute)

	// Token usage & quota
	v1.GET("/token/quota", deps.Token.GetQuota)
	v1.GET("/token/usage", deps.Token.GetUsage)
	v1.GET("/token/records", deps.Token.GetRecords)

	// MCP protocol endpoints (SSE + JSON-RPC)
	if deps.MCP != nil {
		h.GET("/mcp/sse", deps.MCP.SSE)
		h.POST("/mcp/message", deps.MCP.Message)
	}

	// Suppress unused-import warning for consts.
	_ = consts.StatusOK
}