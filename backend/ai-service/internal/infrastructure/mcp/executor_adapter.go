package mcp

import (
	"context"

	"tcm-history-ai/backend/ai-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
)

// ExecutorAdapter bridges the mcp.Executor port to the existing
// service.ToolExecutor (HTTPExecutor) so that the MCP Server can reuse
// the current tool execution pipeline.
type ExecutorAdapter struct {
	inner service.ToolExecutor
}

// NewExecutorAdapter wraps the existing ToolExecutor.
func NewExecutorAdapter(inner service.ToolExecutor) *ExecutorAdapter {
	return &ExecutorAdapter{inner: inner}
}

// Execute implements mcp.Executor by delegating to the inner ToolExecutor.
// The degraded flag is inferred from the result payload.
func (a *ExecutorAdapter) Execute(ctx context.Context, toolName string, params map[string]any) (map[string]any, bool, error) {
	if a.inner == nil {
		return nil, false, errno.New(errno.DependencyUnavailable, "tool executor not configured")
	}
	result, err := a.inner.Execute(ctx, toolName, params)
	if err != nil {
		return nil, false, err
	}
	// Detect degraded flag from the stub / fallback result.
	degraded := false
	if d, ok := result["degraded"].(bool); ok {
		degraded = d
	}
	if d, ok := result["_degraded"].(bool); ok {
		degraded = d
	}
	// If the inner executor returns a stub marker, treat as degraded.
	if _, ok := result["result"].(string); ok {
		if s, ok := result["result"].(string); ok && len(s) > 6 && s[:6] == "[stub-" {
			degraded = true
		}
	}
	return result, degraded, nil
}

// Compile-time check.
var _ Executor = (*ExecutorAdapter)(nil)
