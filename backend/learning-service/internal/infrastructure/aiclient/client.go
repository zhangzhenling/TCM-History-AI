package aiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"tcm-history-ai/backend/learning-service/internal/conf"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/logger"
	"go.uber.org/zap"
)

type ChatRequest struct {
	UserID        int64          `json:"user_id,omitempty"`
	Message       string         `json:"message"`
	TemplateScene string         `json:"template_scene,omitempty"`
	Variables     map[string]any `json:"variables,omitempty"`
	Mode          string         `json:"mode,omitempty"`
}

type ChatResponse struct {
	ConversationID int64           `json:"conversation_id"`
	MessageID      int64           `json:"message_id"`
	Assistant      string          `json:"assistant"`
	Model          string          `json:"model"`
	TokensPrompt   int             `json:"tokens_prompt"`
	TokensCompletion int           `json:"tokens_completion"`
	LatencyMs      int             `json:"latency_ms"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

type baseResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// Client is a thin HTTP client for the AI Service chat endpoint.
// When the base URL is empty all methods return (zero, nil) so callers
// can degrade gracefully when AI is unavailable.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(cfg conf.AIServiceConfig) *Client {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	return &Client{
		baseURL:    cfg.BaseURL,
		httpClient: client,
	}
}

// Chat calls the AI service chat endpoint. Returns ("", nil) when the
// client is configured without a base URL (offline mode).
func (c *Client) Chat(ctx context.Context, req ChatRequest) (string, error) {
	if c.baseURL == "" {
		return "", nil
	}
	body, err := json.Marshal(req)
	if err != nil {
		return "", errno.Wrap(errno.InvalidParams, "marshal ai chat request", err)
	}
	url := fmt.Sprintf("%s/api/v1/ai/chat", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", errno.Wrap(errno.InternalError, "build ai chat request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if req.UserID > 0 {
		httpReq.Header.Set("X-User-ID", fmt.Sprintf("%d", req.UserID))
	}
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		logger.Default().Warn("ai service chat call failed", zap.Error(err))
		return "", nil
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil
	}
	if resp.StatusCode != http.StatusOK {
		logger.Default().Warn("ai service chat returned non-200",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(respBody)))
		return "", nil
	}
	var base baseResponse
	if err := json.Unmarshal(respBody, &base); err != nil {
		return "", nil
	}
	if base.Code != 0 {
		logger.Default().Warn("ai service chat returned error",
			zap.Int("code", base.Code),
			zap.String("message", base.Message))
		return "", nil
	}
	var chatResp ChatResponse
	if err := json.Unmarshal(base.Data, &chatResp); err != nil {
		return "", nil
	}
	return chatResp.Assistant, nil
}

// GenerateStudyPlan asks the AI to generate a study plan for the given
// goal and target days. Returns the AI text response or an empty string
// when the AI service is unavailable.
func (c *Client) GenerateStudyPlan(ctx context.Context, userID int64, goal string, targetDays int) (string, error) {
	prompt := fmt.Sprintf(
		"请为我生成一个中医历史学习计划。\n学习目标：%s\n目标天数：%d天\n\n请按以下JSON格式输出（不要输出其他内容）：\n{\n  \"title\": \"计划标题\",\n  \"description\": \"计划描述\",\n  \"courses\": [\"课程1\", \"课程2\"]\n}",
		goal, targetDays)
	return c.Chat(ctx, ChatRequest{
		UserID:  userID,
		Message: prompt,
		Mode:    "chat",
	})
}
