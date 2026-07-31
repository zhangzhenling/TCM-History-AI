// Package retrieval provides HTTP clients for AI Service to call
// Knowledge Service (RAG) and Graph Service (Cypher) over the gateway.
//
// 当 services.knowledge_url / services.graph_url 为空时返回空结果，
// 与 llm stub / milvus stub 模式一致，保证离线开发链路可运行。
package retrieval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tcm-history-ai/backend/ai-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
)

// Config captures the upstream service endpoints.
type Config struct {
	KnowledgeURL string // 例: http://gateway:8080/api/v1/knowledge
	GraphURL     string // 例: http://gateway:8080/api/v1/graph
	Timeout      int    // HTTP 超时秒
}

// Client implements service.RetrievalClient via HTTP.
type Client struct {
	knowledgeBase string
	graphBase     string
	httpCli       *http.Client
}

// New constructs a RetrievalClient. 任一 base 为空时对应方法返回空结果。
func New(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10
	}
	return &Client{
		knowledgeBase: strings.TrimRight(cfg.KnowledgeURL, "/"),
		graphBase:     strings.TrimRight(cfg.GraphURL, "/"),
		httpCli:       &http.Client{Timeout: time.Duration(timeout) * time.Second},
	}
}

// knowledgeRetrieveRequest mirrors knowledge-service dto.RetrieveRequest.
type knowledgeRetrieveRequest struct {
	Query         string   `json:"query"`
	TopK          int      `json:"top_k,omitempty"`
	ClassicCodes  []string `json:"classic_codes,omitempty"`
	Dynasties     []string `json:"dynasties,omitempty"`
	Schools       []string `json:"schools,omitempty"`
	ContentTypes  []string `json:"content_types,omitempty"`
}

// apiEnvelope mirrors pkg/response 标准响应包络。
type apiEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// Retrieve calls Knowledge Service POST /api/v1/knowledge/retrieve.
func (c *Client) Retrieve(ctx context.Context, query string, topK int) (*service.RetrieveResult, error) {
	if c.knowledgeBase == "" {
		return &service.RetrieveResult{Query: query, TopK: topK, Total: 0}, nil
	}
	if query == "" {
		return nil, errno.New(errno.InvalidParams, "retrieval: query is required")
	}
	if topK <= 0 {
		topK = 5
	}

	body := knowledgeRetrieveRequest{Query: query, TopK: topK}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "retrieval: marshal request", err)
	}

	reqURL := c.knowledgeBase + "/retrieve"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(raw))
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "retrieval: build request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	respBody, err := c.do(httpReq)
	if err != nil {
		return nil, err
	}

	var result service.RetrieveResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, errno.Wrap(errno.InternalError, "retrieval: unmarshal response", err)
	}
	return &result, nil
}

// SearchNodes calls Graph Service GET /api/v1/graph/search.
func (c *Client) SearchNodes(ctx context.Context, keyword, label string, limit int) (*service.GraphSearchResult, error) {
	if c.graphBase == "" {
		return &service.GraphSearchResult{Keyword: keyword, Label: label, Total: 0}, nil
	}
	if keyword == "" {
		return nil, errno.New(errno.InvalidParams, "retrieval: keyword is required")
	}
	if limit <= 0 {
		limit = 20
	}

	q := url.Values{}
	q.Set("keyword", keyword)
	if label != "" {
		q.Set("label", label)
	}
	q.Set("limit", fmt.Sprintf("%d", limit))

	reqURL := c.graphBase + "/search?" + q.Encode()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "retrieval: build request", err)
	}

	respBody, err := c.do(httpReq)
	if err != nil {
		return nil, err
	}

	var result service.GraphSearchResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, errno.Wrap(errno.InternalError, "retrieval: unmarshal response", err)
	}
	return &result, nil
}

// do executes the HTTP request, unwraps the standard API envelope, and returns
// the data field as raw JSON bytes.
func (c *Client) do(httpReq *http.Request) ([]byte, error) {
	httpResp, err := c.httpCli.Do(httpReq)
	if err != nil {
		return nil, errno.Wrap(errno.DependencyUnavailable, "retrieval: call upstream", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, errno.Wrap(errno.DependencyUnavailable, "retrieval: read body", err)
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, errno.New(errno.DependencyUnavailable,
			fmt.Sprintf("retrieval: http %d: %s", httpResp.StatusCode, string(body)))
	}

	var env apiEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		// 兼容未封装的直返 JSON
		return body, nil
	}
	if env.Code != 0 && env.Code != 200 {
		return nil, errno.New(errno.DependencyUnavailable,
			fmt.Sprintf("retrieval: upstream code=%d msg=%s", env.Code, env.Message))
	}
	// 若 data 字段缺失（直返未封装 JSON），回退到原始 body。
	if len(env.Data) == 0 {
		return body, nil
	}
	return env.Data, nil
}

// Compile-time check.
var _ service.RetrievalClient = (*Client)(nil)
