package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tcm-history-ai/backend/ai-service/internal/domain/service"
)

// TestAnthropicProvider_Chat verifies the Anthropic Messages API client
// correctly serialises the request, attaches the x-api-key header, and
// stitches together the multi-block content response.
func TestAnthropicProvider_Chat(t *testing.T) {
	var gotBody anthropicRequest
	var gotKey, gotVer string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotKey = r.Header.Get("x-api-key")
		gotVer = r.Header.Get("anthropic-version")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)

		resp := anthropicResponse{
			ID:    "msg_test",
			Model: gotBody.Model,
			Role:  "assistant",
			Content: []anthropicContentBlock{
				{Type: "text", Text: "伤寒论由"},
				{Type: "text", Text: "张仲景所著。"},
			},
			StopReason: "end_turn",
			Usage:      anthropicUsage{InputTokens: 15, OutputTokens: 10},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := NewAnthropicProvider(srv.URL, "sk-ant-test", "claude-3-5-sonnet-20240620", 5)
	resp, err := p.Chat(context.Background(), service.LLMChatRequest{
		System:    "你是中医助手。",
		Messages:  []service.LLMMessage{{Role: "user", Content: "伤寒论是谁写的？"}},
		MaxTokens: 200,
	})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if resp.Text != "伤寒论由张仲景所著。" {
		t.Errorf("expected concatenated text, got %q", resp.Text)
	}
	if resp.TokensPrompt != 15 || resp.TokensCompletion != 10 {
		t.Errorf("unexpected token usage: prompt=%d completion=%d", resp.TokensPrompt, resp.TokensCompletion)
	}
	if gotKey != "sk-ant-test" {
		t.Errorf("expected x-api-key 'sk-ant-test', got %q", gotKey)
	}
	if gotVer != "2023-06-01" {
		t.Errorf("expected anthropic-version '2023-06-01', got %q", gotVer)
	}
	if gotBody.System != "你是中医助手。" {
		t.Errorf("expected system field set, got %q", gotBody.System)
	}
	if gotBody.MaxTokens != 200 {
		t.Errorf("expected max_tokens=200, got %d", gotBody.MaxTokens)
	}
}

// TestAnthropicProvider_DefaultMaxTokens verifies the 1024 fallback.
func TestAnthropicProvider_DefaultMaxTokens(t *testing.T) {
	var gotBody anthropicRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		resp := anthropicResponse{
			Model:   "claude-3-5-sonnet-20240620",
			Content: []anthropicContentBlock{{Type: "text", Text: "ok"}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := NewAnthropicProvider(srv.URL, "k", "claude-3-5-sonnet-20240620", 5)
	_, err := p.Chat(context.Background(), service.LLMChatRequest{
		Messages: []service.LLMMessage{{Role: "user", Content: "ping"}},
	})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if gotBody.MaxTokens != 1024 {
		t.Errorf("expected default max_tokens=1024, got %d", gotBody.MaxTokens)
	}
}

// TestAnthropicProvider_Error verifies HTTP errors are propagated.
func TestAnthropicProvider_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"max_tokens is required"}}`))
	}))
	defer srv.Close()

	p := NewAnthropicProvider(srv.URL, "k", "claude-3-5-sonnet-20240620", 5)
	_, err := p.Chat(context.Background(), service.LLMChatRequest{
		Messages: []service.LLMMessage{{Role: "user", Content: "x"}},
		MaxTokens: 10,
	})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if !strings.Contains(err.Error(), "max_tokens is required") {
		t.Errorf("expected error to mention 'max_tokens is required', got %v", err)
	}
}

// TestAnthropicProvider_SystemRoleMapping verifies that a "system" role
// message in the chat history is rewritten to "user" (Anthropic constraint).
func TestAnthropicProvider_SystemRoleMapping(t *testing.T) {
	var gotBody anthropicRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		resp := anthropicResponse{
			Content: []anthropicContentBlock{{Type: "text", Text: "ok"}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := NewAnthropicProvider(srv.URL, "k", "claude-3-5-sonnet-20240620", 5)
	_, err := p.Chat(context.Background(), service.LLMChatRequest{
		Messages: []service.LLMMessage{
			{Role: "system", Content: "ignored"},
			{Role: "user", Content: "hi"},
		},
		MaxTokens: 10,
	})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if len(gotBody.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(gotBody.Messages))
	}
	if gotBody.Messages[0].Role != "user" {
		t.Errorf("expected first message role rewritten to 'user', got %q", gotBody.Messages[0].Role)
	}
}
