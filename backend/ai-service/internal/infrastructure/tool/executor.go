// Package tool provides the ToolExecutor adapter for the AI Service.
//
// 当前实现：HTTPExecutor —— 通过 net/http 调用注册的 Tool endpoint。
// 在 enabled=false 时返回固定桩结果，与 milvus stub / llm stub 模式一致。
// 参考 doc/08-MCP设计.md §8.2 / §8.3。
package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"tcm-history-ai/backend/ai-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
)

// Config captures the tool executor coordinates.
type Config struct {
	HTTPTimeout int  // 单次调用超时秒
	Enabled     bool // false 时强制走 stub
}

// HTTPExecutor implements repository-independent Tool execution by calling
// the registered endpoint over HTTP. 在 enabled=false 时直接返回桩结果。
type HTTPExecutor struct {
	toolRepo repository.ToolRepository
	cfg      Config
	client   *http.Client
}

// New constructs an HTTPExecutor.
func New(toolRepo repository.ToolRepository, cfg Config) *HTTPExecutor {
	timeout := cfg.HTTPTimeout
	if timeout <= 0 {
		timeout = 5
	}
	return &HTTPExecutor{
		toolRepo: toolRepo,
		cfg:     cfg,
		client:  &http.Client{Timeout: time.Duration(timeout) * time.Second},
	}
}

// Execute looks up the tool by name and dispatches the call. 在 enabled=false
// 或 tool 未配置 endpoint 时返回桩结果，便于离线开发联调。
//
// TODO(mcp-sdk): 接入完整 MCP 协议（SSE/stdio 传输、Schema 校验、限流）。
func (e *HTTPExecutor) Execute(ctx context.Context, toolName string, params map[string]any) (map[string]any, error) {
	if toolName == "" {
		return nil, errno.New(errno.InvalidParams, "tool: empty name")
	}
	t, err := e.toolRepo.FindByName(ctx, toolName)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errno.New(errno.NotFound, "tool not found: "+toolName)
	}
	if !t.IsEnabled {
		return nil, errno.New(errno.Forbidden, "tool disabled: "+toolName)
	}

	// Stub path: endpoint 为空或 enabled=false 时返回桩结果。
	if !e.cfg.Enabled || t.Endpoint == "" {
		return stubResult(t.Name, params), nil
	}

	body, err := json.Marshal(params)
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "marshal params", err)
	}
	method := strings.ToUpper(t.Method)
	if method == "" {
		method = "POST"
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, t.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "build http request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(httpReq)
	if err != nil {
		// 调用失败时回退到桩结果，避免阻塞 Agent 链路（参考 doc/07 §9.2 降级策略）。
		return stubResult(t.Name, params), nil
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)
	var parsed map[string]any
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		parsed = map[string]any{
			"raw":       string(respBody),
			"tool_name": t.Name,
		}
	}
	if resp.StatusCode >= 400 {
		return nil, errno.Wrap(errno.DependencyUnavailable, fmt.Sprintf("tool %s returned %d", t.Name, resp.StatusCode), nil)
	}
	return parsed, nil
}

// stubResult returns a deterministic placeholder for offline development.
func stubResult(toolName string, params map[string]any) map[string]any {
	return map[string]any{
		"tool_name": toolName,
		"params":    params,
		"result":    "[stub-tool] tool executor 处于离线 stub 模式，未调用真实 endpoint",
		"degraded":  true,
	}
}

// Compile-time check.
var _ = (*HTTPExecutor)(nil)
