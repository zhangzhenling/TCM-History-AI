// Package search wraps the meilisearch-go client to provide a small surface
// for index lifecycle and BM25 document search.
//
// 索引设计（doc/06-RAG设计.md §6.6）：
//   - 索引名: knowledge_chunks
//   - PrimaryKey: chunk_id
//   - searchableAttributes: text
//   - filterableAttributes: classic_code, dynasty, school, content_type
package search

import (
	"context"
	"fmt"
	"strings"

	"github.com/meilisearch/meilisearch-go"

	"tcm-history-ai/backend/knowledge-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
)

// MeiliClient wraps the official meilisearch-go client.
type MeiliClient struct {
	client      *meilisearch.Client
	indexPrefix string
	indexName   string
}

// NewMeiliClient constructs a MeiliClient. It does not connect eagerly so
// callers may construct it even if the broker is not yet reachable.
func NewMeiliClient(host string, port int, apiKey, indexPrefix string) *MeiliClient {
	url := fmt.Sprintf("http://%s:%d", host, port)
	if port == 0 {
		url = host
	}
	if indexPrefix == "" {
		indexPrefix = "knowledge_"
	}
	return &MeiliClient{
		client:      meilisearch.NewClient(meilisearch.ClientConfig{Host: url, APIKey: apiKey}),
		indexPrefix: indexPrefix,
		indexName:   indexPrefix + "chunks",
	}
}

// EnsureIndex creates the index if absent and configures its searchable
// and filterable attributes. Idempotent.
func (c *MeiliClient) EnsureIndex() error {
	idx := c.client.Index(c.indexName)
	if _, err := idx.FetchInfo(); err == nil {
		// already exists; refresh attributes
		_, _ = idx.UpdateSearchableAttributes(&[]string{"text"})
		_, _ = idx.UpdateFilterableAttributes(&[]string{"classic_code", "dynasty", "school", "content_type"})
		return nil
	}
	task, err := c.client.CreateIndex(&meilisearch.IndexConfig{
		Uid:        c.indexName,
		PrimaryKey: "chunk_id",
	})
	if err != nil {
		return errno.Wrap(errno.DependencyUnavailable, "create meili index", err)
	}
	if err := c.waitForTask(task.TaskUID); err != nil {
		return err
	}
	if _, err := idx.UpdateSearchableAttributes(&[]string{"text"}); err != nil {
		return errno.Wrap(errno.DependencyUnavailable, "update searchable attributes", err)
	}
	if _, err := idx.UpdateFilterableAttributes(&[]string{"classic_code", "dynasty", "school", "content_type"}); err != nil {
		return errno.Wrap(errno.DependencyUnavailable, "update filterable attributes", err)
	}
	return nil
}

// Index upserts a batch of chunks into the full-text index.
func (c *MeiliClient) Index(ctx context.Context, docs []service.FullTextDoc) error {
	if len(docs) == 0 {
		return nil
	}
	payload := make([]map[string]any, 0, len(docs))
	for i := range docs {
		payload = append(payload, map[string]any{
			"chunk_id":     docs[i].ChunkID,
			"doc_id":       docs[i].DocID,
			"classic_code": docs[i].ClassicCode,
			"dynasty":      docs[i].Dynasty,
			"school":       docs[i].School,
			"volume":       docs[i].Volume,
			"clause_no":    docs[i].ClauseNo,
			"content_type": docs[i].ContentType,
			"text":         docs[i].Text,
		})
	}
	task, err := c.client.Index(c.indexName).AddDocuments(payload, "chunk_id")
	if err != nil {
		return errno.Wrap(errno.DependencyUnavailable, "index documents", err)
	}
	return c.waitForTask(task.TaskUID)
}

// Search runs a BM25 query with optional filters and returns ranked hits.
func (c *MeiliClient) Search(ctx context.Context, query string, topK int, filters service.SearchFilter) ([]service.FullTextHit, error) {
	if topK <= 0 {
		topK = 20
	}
	req := &meilisearch.SearchRequest{
		Limit:            int64(topK),
		ShowRankingScore: true,
	}
	if filterExpr := buildFilter(filters); filterExpr != "" {
		req.Filter = &filterExpr
	}
	resp, err := c.client.Index(c.indexName).Search(query, req)
	if err != nil {
		if strings.Contains(err.Error(), "index not found") || strings.Contains(err.Error(), "404") {
			return []service.FullTextHit{}, nil
		}
		return nil, errno.Wrap(errno.DependencyUnavailable, "search", err)
	}
	out := make([]service.FullTextHit, 0, len(resp.Hits))
	for _, hit := range resp.Hits {
		m, ok := hit.(map[string]any)
		if !ok {
			continue
		}
		chunkID, _ := m["chunk_id"].(string)
		docID, _ := m["doc_id"].(float64)
		var score float64
		if rs, ok := m["_rankingScore"].(float64); ok {
			score = rs
		}
		out = append(out, service.FullTextHit{
			ChunkID: chunkID,
			Score:   score,
			DocID:   int64(docID),
		})
	}
	return out, nil
}

// buildFilter constructs a Meilisearch filter expression from SearchFilter.
// Multiple values within a field are OR'd; multiple fields are AND'd.
func buildFilter(f service.SearchFilter) string {
	var parts []string
	if len(f.ClassicCodes) > 0 {
		parts = append(parts, orExpr("classic_code", f.ClassicCodes))
	}
	if len(f.Dynasties) > 0 {
		parts = append(parts, orExpr("dynasty", f.Dynasties))
	}
	if len(f.Schools) > 0 {
		parts = append(parts, orExpr("school", f.Schools))
	}
	if len(f.ContentTypes) > 0 {
		parts = append(parts, orExpr("content_type", f.ContentTypes))
	}
	return strings.Join(parts, " AND ")
}

// orExpr builds `field = "v1" OR field = "v2"`.
func orExpr(field string, values []string) string {
	quoted := make([]string, 0, len(values))
	for _, v := range values {
		quoted = append(quoted, fmt.Sprintf("%s = %q", field, v))
	}
	return "(" + strings.Join(quoted, " OR ") + ")"
}

// waitForTask polls the task queue until the given task completes.
func (c *MeiliClient) waitForTask(taskUID int64) error {
	if taskUID == 0 {
		return nil
	}
	for i := 0; i < 60; i++ {
		task, err := c.client.GetTask(taskUID)
		if err != nil {
			return errno.Wrap(errno.DependencyUnavailable, "get meili task", err)
		}
		switch task.Status {
		case meilisearch.TaskStatusSucceeded:
			return nil
		case meilisearch.TaskStatusFailed:
			return errno.New(errno.DependencyUnavailable,
				fmt.Sprintf("meili task %d failed: %s", taskUID, task.Error.Message))
		}
	}
	return errno.New(errno.DependencyUnavailable, "meili task timed out")
}

// Compile-time check: MeiliClient implements service.FullTextSearcher.
var _ service.FullTextSearcher = (*MeiliClient)(nil)
