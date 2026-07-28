package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	"tcm-history-ai/backend/ai-service/internal/infrastructure/mcp"
)

// BenchmarkServer_HandleRequest_ToolsList benchmarks the tools/list method.
func BenchmarkServer_HandleRequest_ToolsList(b *testing.B) {
	reg := mcp.NewRegistry(nil)
	for _, t := range mcp.BuiltInTools() {
		reg.Register(t)
	}
	srv := mcp.NewServer(reg, &noopExecutor{})
	ctx := context.Background()
	req := &mcp.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
		Params:  nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := srv.HandleRequest(ctx, req)
		if resp.Error != nil {
			b.Fatal(resp.Error)
		}
	}
}

// BenchmarkServer_HandleRequest_ToolsCall benchmarks the tools/call method.
func BenchmarkServer_HandleRequest_ToolsCall(b *testing.B) {
	reg := mcp.NewRegistry(nil)
	for _, t := range mcp.BuiltInTools() {
		reg.Register(t)
	}
	srv := mcp.NewServer(reg, &noopExecutor{})
	ctx := mcp.WithScopes(context.Background(), []string{
		"history:read", "person:read", "school:read", "book:read",
		"graph:read", "search:read", "medicine:read", "prescription:read",
	})
	params, _ := json.Marshal(map[string]any{
		"name":      "tcm.person.query",
		"arguments": map[string]any{"name": "张仲景"},
	})
	req := &mcp.Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  params,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := srv.HandleRequest(ctx, req)
		if resp.Error != nil {
			b.Fatal(resp.Error)
		}
	}
}

// noopExecutor is a stub executor for benchmarking.
type noopExecutor struct{}

func (n *noopExecutor) Execute(_ context.Context, _ string, _ map[string]any) (map[string]any, bool, error) {
	return map[string]any{"result": "ok"}, false, nil
}