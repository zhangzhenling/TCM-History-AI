// Package milvus provides the Milvus-backed VectorStore adapter.
//
// 设计依据：doc/06-RAG设计.md §6.5
//   - Collection: tcm_chunks
//   - 向量字段: embedding FLOAT_VECTOR(1024), HNSW(M=16, efConstruction=200, metric=IP)
//   - 标量字段: classic_code/dynasty/school/volume/clause_no/content_type/doc_id
//   - 6 Partition: p_{classic_code}，按经典编码划分
//
// 实现策略（ADR-21-01）：
//   - enabled=true 时走 Milvus REST API v2（net/http 直连，不引入 milvus-sdk-go）
//   - enabled=false 时退化为内存 stub，保证离线开发与单元测试可运行（ADR-21-02）
//
// REST API 参考：https://milvus.io/api-reference/rest/v2.x/v2/About.md
package milvus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"tcm-history-ai/backend/knowledge-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
)

// ClassicCodes 是六个核心经典编码，对应 Milvus 的六个 Partition。
var ClassicCodes = []string{
	"huangdi_neijing",   // 黄帝内经
	"shanghan_lun",      // 伤寒论
	"jingui_yaolue",     // 金匮要略
	"zhenjiu_yiijing",   // 针灸甲乙经
	"wenbing_tiaobian",  // 温病条辨
	"bencao_gangmu",     // 本草纲目
}

// Config captures the Milvus client coordinates.
type Config struct {
	Host       string
	Port       int
	Collection string
	Dim        int
	Username   string // REST API 鉴权用户名（默认 root）
	Password   string // REST API 鉴权密码（默认 Milvus）
	Enabled    bool
}

// Client implements service.VectorStore. enabled=true 时走 REST API，
// enabled=false 时退化为内存 stub（与既有离线开发模式一致）。
type Client struct {
	cfg     Config
	httpCli *http.Client
	// stub 字段：仅用于 enabled=false 的离线开发与单元测试。
	mu      sync.Mutex
	records map[string]service.VectorRecord // chunk_id -> record
}

// New constructs a Client. 连接延迟到 EnsureCollection / 首次调用时建立。
func New(cfg Config) *Client {
	return &Client{
		cfg:     cfg,
		httpCli: &http.Client{Timeout: 30 * time.Second},
		records: make(map[string]service.VectorRecord),
	}
}

// baseURL returns the REST API root, e.g. http://localhost:19530.
func (c *Client) baseURL() string {
	host := c.cfg.Host
	if host == "" {
		host = "localhost"
	}
	port := c.cfg.Port
	if port <= 0 {
		port = 19530
	}
	return fmt.Sprintf("http://%s:%d", host, port)
}

// authToken returns the Authorization header value. Milvus REST API v2 接受
// `Bearer <user>:<password>` 形式（明文，非 base64）。
func (c *Client) authToken() string {
	user := c.cfg.Username
	if user == "" {
		user = "root"
	}
	pass := c.cfg.Password
	if pass == "" {
		pass = "Milvus"
	}
	return "Bearer " + user + ":" + pass
}

// doPOST issues a POST to the given REST path with the JSON body.
func (c *Client) doPOST(ctx context.Context, path string, body any) ([]byte, int, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "milvus: marshal request", err)
	}
	url := c.baseURL() + path
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, 0, errno.Wrap(errno.InternalError, "milvus: build request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", c.authToken())
	resp, err := c.httpCli.Do(httpReq)
	if err != nil {
		return nil, 0, errno.Wrap(errno.DependencyUnavailable, "milvus: call api", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, errno.Wrap(errno.DependencyUnavailable, "milvus: read body", err)
	}
	return respBody, resp.StatusCode, nil
}

// milvusError is the error envelope returned by the REST API.
type milvusError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Reason  string `json:"reason"`
}

func (e milvusError) String() string {
	if e.Reason != "" {
		return fmt.Sprintf("code=%d message=%s reason=%s", e.Code, e.Message, e.Reason)
	}
	return fmt.Sprintf("code=%d message=%s", e.Code, e.Message)
}

// checkRESTError parses a non-2xx response into an errno.Error.
func checkRESTError(statusCode int, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}
	var me milvusError
	_ = json.Unmarshal(body, &me)
	msg := me.Message
	if msg == "" {
		msg = fmt.Sprintf("milvus: http %d: %s", statusCode, string(body))
	} else {
		msg = "milvus: " + msg
	}
	return errno.New(errno.DependencyUnavailable, msg)
}

// EnsureCollection creates the collection + schema if absent.
// 走 describe 探测，不存在则 create。schema 与 doc/06 §6.5 对齐。
func (c *Client) EnsureCollection(ctx context.Context) error {
	if !c.cfg.Enabled {
		return nil
	}
	collection := c.cfg.Collection
	if collection == "" {
		collection = "tcm_chunks"
	}
	// 1. describe：探测 collection 是否已存在。
	_, descStatus, err := c.doPOST(ctx, "/v2/vectordb/collections/describe",
		map[string]string{"collectionName": collection})
	if err != nil {
		return err
	}
	if descStatus >= 200 && descStatus < 300 {
		// collection 已存在，直接复用。
		return nil
	}
	// 2. create：describe 返回非 2xx（通常是 404 / 500 表示不存在），尝试创建。
	dim := c.cfg.Dim
	if dim <= 0 {
		dim = 1024
	}
	createReq := map[string]any{
		"collectionName": collection,
		"schema": map[string]any{
			"autoId": false,
			"fields": []map[string]any{
				{"fieldName": "chunk_id", "dataType": "VarChar", "elementTypeParams": map[string]string{"max_length": "64"}, "isPrimary": true},
				{"fieldName": "embedding", "dataType": "FloatVector", "elementTypeParams": map[string]string{"dim": fmt.Sprintf("%d", dim)}},
				{"fieldName": "classic_code", "dataType": "VarChar", "elementTypeParams": map[string]string{"max_length": "64"}},
				{"fieldName": "dynasty", "dataType": "VarChar", "elementTypeParams": map[string]string{"max_length": "32"}},
				{"fieldName": "school", "dataType": "VarChar", "elementTypeParams": map[string]string{"max_length": "64"}},
				{"fieldName": "volume", "dataType": "VarChar", "elementTypeParams": map[string]string{"max_length": "64"}},
				{"fieldName": "clause_no", "dataType": "Int64"},
				{"fieldName": "content_type", "dataType": "VarChar", "elementTypeParams": map[string]string{"max_length": "32"}},
				{"fieldName": "doc_id", "dataType": "Int64"},
			},
		},
		"indexParams": []map[string]any{
			{"fieldName": "embedding", "indexType": "HNSW", "metricType": "IP", "params": map[string]int{"M": 16, "efConstruction": 200}},
		},
	}
	_, createStatus, err := c.doPOST(ctx, "/v2/vectordb/collections/create", createReq)
	if err != nil {
		return err
	}
	if err := checkRESTError(createStatus, nil); err != nil {
		return err
	}
	// 3. 为 6 个 classic_code 创建 partition（best-effort，失败不阻断）。
	for _, code := range ClassicCodes {
		_, _, _ = c.doPOST(ctx, "/v2/vectordb/partitions/create", map[string]string{
			"collectionName": collection,
			"partitionName":  "p_" + code,
		})
	}
	return nil
}

// Insert upserts a batch of vector records.
// enabled=false 时记录到内存 map（stub）；enabled=true 时走 REST API。
func (c *Client) Insert(ctx context.Context, records []service.VectorRecord) error {
	if !c.cfg.Enabled {
		return c.stubInsert(records)
	}
	if len(records) == 0 {
		return nil
	}
	collection := c.cfg.Collection
	if collection == "" {
		collection = "tcm_chunks"
	}
	data := make([]map[string]any, 0, len(records))
	for _, r := range records {
		data = append(data, map[string]any{
			"chunk_id":      r.ChunkID,
			"embedding":     r.Embedding,
			"classic_code":  r.ClassicCode,
			"dynasty":       r.Dynasty,
			"school":        r.School,
			"volume":        r.Volume,
			"clause_no":     r.ClauseNo,
			"content_type":  r.ContentType,
			"doc_id":        r.DocID,
		})
	}
	body, status, err := c.doPOST(ctx, "/v2/vectordb/entities/upsert", map[string]any{
		"collectionName": collection,
		"data":           data,
	})
	if err != nil {
		return err
	}
	return checkRESTError(status, body)
}

// stubInsert is the in-memory fallback used when enabled=false.
func (c *Client) stubInsert(records []service.VectorRecord) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range records {
		c.records[records[i].ChunkID] = records[i]
	}
	return nil
}

// DeleteByDoc removes all vectors belonging to a document.
func (c *Client) DeleteByDoc(ctx context.Context, docID int64) error {
	if !c.cfg.Enabled {
		return c.stubDeleteByDoc(docID)
	}
	collection := c.cfg.Collection
	if collection == "" {
		collection = "tcm_chunks"
	}
	body, status, err := c.doPOST(ctx, "/v2/vectordb/entities/delete", map[string]any{
		"collectionName": collection,
		"filter":         fmt.Sprintf("doc_id == %d", docID),
	})
	if err != nil {
		return err
	}
	return checkRESTError(status, body)
}

// stubDeleteByDoc is the in-memory fallback.
func (c *Client) stubDeleteByDoc(docID int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, r := range c.records {
		if r.DocID == docID {
			delete(c.records, id)
		}
	}
	return nil
}

// Search runs an ANN query with optional scalar filters.
func (c *Client) Search(ctx context.Context, query []float32, topK int, filters service.SearchFilter) ([]service.VectorSearchResult, error) {
	if !c.cfg.Enabled {
		return c.stubSearch(query, topK, filters)
	}
	if topK <= 0 {
		topK = 20
	}
	if len(query) == 0 {
		return nil, errno.New(errno.InvalidParams, "empty query vector")
	}
	collection := c.cfg.Collection
	if collection == "" {
		collection = "tcm_chunks"
	}
	req := map[string]any{
		"collectionName": collection,
		"data":           [][]float32{query},
		"limit":          topK,
		"outputFields":   []string{"chunk_id", "doc_id"},
	}
	if expr := buildFilterExpr(filters); expr != "" {
		req["filter"] = expr
	}
	body, status, err := c.doPOST(ctx, "/v2/vectordb/entities/search", req)
	if err != nil {
		return nil, err
	}
	if err := checkRESTError(status, body); err != nil {
		return nil, err
	}
	var parsed struct {
		Code int `json:"code"`
		Data []struct {
			ChunkID   string  `json:"chunk_id"`
			DocID     int64   `json:"doc_id"`
			Distance  float32 `json:"distance"`
			Score     float32 `json:"score"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, errno.Wrap(errno.InternalError, "milvus: unmarshal search response", err)
	}
	out := make([]service.VectorSearchResult, 0, len(parsed.Data))
	for _, d := range parsed.Data {
		score := d.Score
		if score == 0 {
			score = d.Distance // 不同 Milvus 版本返回字段不同
		}
		out = append(out, service.VectorSearchResult{
			ChunkID: d.ChunkID,
			Score:   score,
			DocID:   d.DocID,
		})
	}
	return out, nil
}

// stubSearch is the in-memory fallback.
func (c *Client) stubSearch(query []float32, topK int, filters service.SearchFilter) ([]service.VectorSearchResult, error) {
	if topK <= 0 {
		topK = 20
	}
	if len(query) == 0 {
		return nil, errno.New(errno.InvalidParams, "empty query vector")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	results := make([]service.VectorSearchResult, 0, topK)
	for _, r := range c.records {
		if !matchFilter(r, filters) {
			continue
		}
		results = append(results, service.VectorSearchResult{
			ChunkID: r.ChunkID,
			Score:   0.5, // stub score，真实场景由 HNSW IP 距离计算
			DocID:   r.DocID,
		})
		if len(results) >= topK {
			break
		}
	}
	return results, nil
}

// buildFilterExpr translates SearchFilter into a Milvus boolean expression.
// 空字段不参与过滤；同一字段内为 OR，字段间为 AND。
func buildFilterExpr(f service.SearchFilter) string {
	var parts []string
	if expr := inExpr("classic_code", f.ClassicCodes); expr != "" {
		parts = append(parts, expr)
	}
	if expr := inExpr("dynasty", f.Dynasties); expr != "" {
		parts = append(parts, expr)
	}
	if expr := inExpr("school", f.Schools); expr != "" {
		parts = append(parts, expr)
	}
	if expr := inExpr("content_type", f.ContentTypes); expr != "" {
		parts = append(parts, expr)
	}
	return strings.Join(parts, " and ")
}

// inExpr builds a `field in ['a','b']` expression; empty values yield "".
func inExpr(field string, vals []string) string {
	if len(vals) == 0 {
		return ""
	}
	quoted := make([]string, 0, len(vals))
	for _, v := range vals {
		v = strings.ReplaceAll(v, "'", "\\'")
		quoted = append(quoted, "'"+v+"'")
	}
	return fmt.Sprintf("%s in [%s]", field, strings.Join(quoted, ","))
}

// matchFilter applies the scalar filter to a record (stub path only).
func matchFilter(r service.VectorRecord, f service.SearchFilter) bool {
	if len(f.ClassicCodes) > 0 && !contains(f.ClassicCodes, r.ClassicCode) {
		return false
	}
	if len(f.Dynasties) > 0 && !contains(f.Dynasties, r.Dynasty) {
		return false
	}
	if len(f.Schools) > 0 && !contains(f.Schools, r.School) {
		return false
	}
	if len(f.ContentTypes) > 0 && !contains(f.ContentTypes, r.ContentType) {
		return false
	}
	return true
}

// contains reports whether ss contains v.
func contains(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}

// String returns a debug representation.
func (c *Client) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return fmt.Sprintf("milvus.Client{collection=%s, dim=%d, enabled=%v, records=%d}",
		c.cfg.Collection, c.cfg.Dim, c.cfg.Enabled, len(c.records))
}
