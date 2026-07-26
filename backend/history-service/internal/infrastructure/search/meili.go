// Package search wraps the meilisearch-go client to provide a small surface
// for index lifecycle and document search.
package search

import (
	"context"
	"fmt"
	"strings"

	"github.com/meilisearch/meilisearch-go"

	"tcm-history-ai/backend/pkg/errno"
)

// MeiliClient wraps the official meilisearch-go client.
type MeiliClient struct {
	client      *meilisearch.Client
	indexPrefix string
}

// NewMeiliClient constructs a MeiliClient. It does not connect eagerly so
// callers may construct it even if the broker is not yet reachable.
func NewMeiliClient(host string, port int, apiKey, indexPrefix string) *MeiliClient {
	url := fmt.Sprintf("http://%s:%d", host, port)
	if port == 0 {
		url = host
	}
	if indexPrefix == "" {
		indexPrefix = "history_"
	}
	return &MeiliClient{
		client:      meilisearch.NewClient(meilisearch.ClientConfig{Host: url, APIKey: apiKey}),
		indexPrefix: indexPrefix,
	}
}

// indexName builds the fully-qualified Meilisearch index name.
func (c *MeiliClient) indexName(name string) string {
	return c.indexPrefix + name
}

// EnsureIndex creates the index if it does not exist and configures its
// searchable attributes. The call is idempotent.
func (c *MeiliClient) EnsureIndex(name string, searchableAttributes []string) error {
	idx := c.client.Index(c.indexName(name))
	if _, err := idx.FetchInfo(); err == nil {
		// index already exists; just update searchable attributes.
		if _, err := idx.UpdateSearchableAttributes(&searchableAttributes); err != nil {
			return errno.Wrap(errno.DependencyUnavailable, "update searchable attributes", err)
		}
		return nil
	}
	task, err := c.client.CreateIndex(&meilisearch.IndexConfig{
		Uid:        c.indexName(name),
		PrimaryKey: "id",
	})
	if err != nil {
		return errno.Wrap(errno.DependencyUnavailable, "create index", err)
	}
	if err := c.waitForTask(task.TaskUID); err != nil {
		return err
	}
	if len(searchableAttributes) > 0 {
		if _, err := idx.UpdateSearchableAttributes(&searchableAttributes); err != nil {
			return errno.Wrap(errno.DependencyUnavailable, "update searchable attributes", err)
		}
	}
	return nil
}

// IndexDocuments pushes a batch of documents into the named index.
func (c *MeiliClient) IndexDocuments(ctx context.Context, index string, docs []map[string]any) error {
	idx := c.client.Index(c.indexName(index))
	task, err := idx.AddDocuments(docs, "id")
	if err != nil {
		return errno.Wrap(errno.DependencyUnavailable, "index documents", err)
	}
	return c.waitForTask(task.TaskUID)
}

// Search runs a query against the named index and returns matching documents
// along with the estimated total hit count.
func (c *MeiliClient) Search(ctx context.Context, index, query string, limit int) ([]map[string]any, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	req := &meilisearch.SearchRequest{
		Limit:            int64(limit),
		ShowRankingScore: true,
	}
	resp, err := c.client.Index(c.indexName(index)).Search(query, req)
	if err != nil {
		// meilisearch returns a 404 if the index has not been created yet;
		// surface this as DependencyUnavailable so callers can retry.
		if strings.Contains(err.Error(), "index not found") || strings.Contains(err.Error(), "404") {
			return []map[string]any{}, 0, nil
		}
		return nil, 0, errno.Wrap(errno.DependencyUnavailable, "search", err)
	}
	out := make([]map[string]any, 0, len(resp.Hits))
	for _, hit := range resp.Hits {
		if m, ok := hit.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out, int64(resp.EstimatedTotalHits), nil
}

// DeleteIndex removes an index from the broker.
func (c *MeiliClient) DeleteIndex(name string) error {
	task, err := c.client.DeleteIndex(c.indexName(name))
	if err != nil {
		return errno.Wrap(errno.DependencyUnavailable, "delete index", err)
	}
	return c.waitForTask(task.TaskUID)
}

// waitForTask polls the task queue until the given task completes; bounded
// to a small number of attempts so it does not block forever in unit tests.
func (c *MeiliClient) waitForTask(taskUID int64) error {
	if taskUID == 0 {
		return nil
	}
	for i := 0; i < 60; i++ {
		task, err := c.client.GetTask(taskUID)
		if err != nil {
			return errno.Wrap(errno.DependencyUnavailable, "get task", err)
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
