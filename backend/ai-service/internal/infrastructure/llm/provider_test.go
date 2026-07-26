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

// TestOpenAIProvider_Chat verifies the OpenAI-compatible client correctly
// serialises the request, attaches the Bearer token, and unwraps the response.
func TestOpenAIProvider_Chat(t *testing.T) {
	var gotBody openAIRequest
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)

		resp := openAIResponse{
			ID:    "chatcmpl-test",
			Model: gotBody.Model,
			Choices: []openAIChoice{
				{Index: 0, Message: openAIMessage{Role: "assistant", Content: "张仲景是东汉医学家。"}, FinishReason: "stop"},
			},
			Usage: openAIUsage{PromptTokens: 12, CompletionTokens: 8, TotalTokens: 20},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "sk-test", "gpt-4o-mini", 5)
	resp, err := p.Chat(context.Background(), service.LLMChatRequest{
		System: "你是中医助手。",
		Messages: []service.LLMMessage{
			{Role: "user", Content: "张仲景是谁？"},
		},
		Temperature: 0.3,
		MaxTokens:   100,
	})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if !strings.Contains(resp.Text, "张仲景") {
		t.Errorf("expected response text to mention 张仲景, got %q", resp.Text)
	}
	if resp.Model != "gpt-4o-mini" {
		t.Errorf("expected model gpt-4o-mini, got %s", resp.Model)
	}
	if resp.TokensPrompt != 12 || resp.TokensCompletion != 8 {
		t.Errorf("unexpected token usage: prompt=%d completion=%d", resp.TokensPrompt, resp.TokensCompletion)
	}
	if gotAuth != "Bearer sk-test" {
		t.Errorf("expected Authorization 'Bearer sk-test', got %q", gotAuth)
	}
	if len(gotBody.Messages) != 2 {
		t.Errorf("expected 2 messages (system+user), got %d", len(gotBody.Messages))
	}
	if gotBody.Messages[0].Role != "system" || gotBody.Messages[0].Content != "你是中医助手。" {
		t.Errorf("system message mismatch: %+v", gotBody.Messages[0])
	}
	if gotBody.Messages[1].Role != "user" || gotBody.Messages[1].Content != "张仲景是谁？" {
		t.Errorf("user message mismatch: %+v", gotBody.Messages[1])
	}
}

// TestOpenAIProvider_Error verifies HTTP errors are propagated.
func TestOpenAIProvider_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid api key","type":"invalid_request_error","code":"invalid_api_key"}}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "bad", "gpt-4o-mini", 5)
	_, err := p.Chat(context.Background(), service.LLMChatRequest{
		Messages: []service.LLMMessage{{Role: "user", Content: "hello"}},
	})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "invalid api key") {
		t.Errorf("expected error to mention 'invalid api key', got %v", err)
	}
}

// TestOpenAIProvider_EmptyMessages verifies parameter validation.
func TestOpenAIProvider_EmptyMessages(t *testing.T) {
	p := NewOpenAIProvider("", "key", "gpt-4o-mini", 5)
	_, err := p.Chat(context.Background(), service.LLMChatRequest{})
	if err == nil {
		t.Fatal("expected error for empty messages")
	}
}

// TestOpenAIProvider_Complete verifies the convenience wrapper.
func TestOpenAIProvider_Complete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIResponse{
			Model: "gpt-4o-mini",
			Choices: []openAIChoice{
				{Message: openAIMessage{Role: "assistant", Content: "ok"}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "key", "gpt-4o-mini", 5)
	out, err := p.Complete(context.Background(), "ping")
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if out != "ok" {
		t.Errorf("expected 'ok', got %q", out)
	}
}

// TestNew_Dispatch verifies the provider factory dispatches correctly.
func TestNew_Dispatch(t *testing.T) {
	cases := []struct {
		name      string
		cfg       Config
		wantStub  bool
		wantOpenAI bool
		wantAnth  bool
	}{
		{name: "stub disabled", cfg: Config{Provider: "openai", Enabled: false}, wantStub: true},
		{name: "stub explicit", cfg: Config{Provider: "stub", Enabled: true}, wantStub: true},
		{name: "openai", cfg: Config{Provider: "openai", APIKey: "k", Enabled: true}, wantOpenAI: true},
		{name: "deepseek", cfg: Config{Provider: "deepseek", APIKey: "k", Enabled: true}, wantOpenAI: true},
		{name: "qwen", cfg: Config{Provider: "qwen", APIKey: "k", Enabled: true}, wantOpenAI: true},
		{name: "kimi", cfg: Config{Provider: "kimi", APIKey: "k", Enabled: true}, wantOpenAI: true},
		{name: "glm", cfg: Config{Provider: "glm", APIKey: "k", Enabled: true}, wantOpenAI: true},
		{name: "anthropic", cfg: Config{Provider: "anthropic", APIKey: "k", Enabled: true}, wantAnth: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, err := New(c.cfg)
			if err != nil {
				t.Fatalf("New returned error: %v", err)
			}
			_, isStub := p.(*StubProvider)
			_, isOpenAI := p.(*OpenAIProvider)
			_, isAnth := p.(*AnthropicProvider)
			if c.wantStub && !isStub {
				t.Errorf("expected *StubProvider, got %T", p)
			}
			if c.wantOpenAI && !isOpenAI {
				t.Errorf("expected *OpenAIProvider, got %T", p)
			}
			if c.wantAnth && !isAnth {
				t.Errorf("expected *AnthropicProvider, got %T", p)
			}
		})
	}
}

// TestNew_UnknownProvider verifies unknown providers error out.
func TestNew_UnknownProvider(t *testing.T) {
	_, err := New(Config{Provider: "unknown", Enabled: true})
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

// TestNew_DefaultModels verifies the per-vendor fallback model is applied.
func TestNew_DefaultModels(t *testing.T) {
	cases := []struct {
		provider string
		want     string
	}{
		{"deepseek", "deepseek-chat"},
		{"qwen", "qwen-turbo"},
		{"kimi", "moonshot-v1-8k"},
		{"glm", "glm-4-flash"},
		{"anthropic", "claude-3-5-sonnet-20240620"},
	}
	for _, c := range cases {
		t.Run(c.provider, func(t *testing.T) {
			p, err := New(Config{Provider: c.provider, APIKey: "k", Enabled: true})
			if err != nil {
				t.Fatalf("New returned error: %v", err)
			}
			if p.Model() != c.want {
				t.Errorf("expected model %s, got %s", c.want, p.Model())
			}
		})
	}
}
