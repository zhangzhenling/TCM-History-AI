// Package milvus provides the Milvus-backed VectorStore adapter.
//
// 设计依据：doc/06-RAG设计.md §6.5
//   - Collection: tcm_chunks
//   - 向量字段: embedding FLOAT_VECTOR(1024), HNSW(M=16, efConstruction=200, metric=IP)
//   - 标量字段: classic_code/dynasty/school/volume/clause_no/content_type/doc_id
//   - 6 Partition: p_{classic_code}，按经典编码划分
//
// 注意：本文件目前以「SDK 接入占位」形式实现，未引入 milvus-sdk-go 依赖。
// 原因：离线开发环境下 go get 无法拉取新模块；此处先保证接口契约与可编译性，
// 待联网环境下补全 SDK 调用（标记为 TODO(milvus-sdk)）。在 milvus.enabled=false
// 时该实现返回空结果，不影响其他服务流程。
package milvus

import (
	"context"
	"fmt"
	"sync"

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
	Enabled    bool
}

// Client implements service.VectorStore with the Milvus SDK.
// 在 SDK 未接入前，所有写操作记录在内存（用于单元测试），读操作返回空。
type Client struct {
	cfg     Config
	mu      sync.Mutex
	records map[string]service.VectorRecord // chunk_id -> record，仅用于离线 stub
}

// New constructs a Client. 连接延迟到 EnsureCollection 时建立。
func New(cfg Config) *Client {
	return &Client{
		cfg:     cfg,
		records: make(map[string]service.VectorRecord),
	}
}

// EnsureCollection creates the collection + 6 partitions if absent.
// TODO(milvus-sdk): 接入 milvus-sdk-go，按设计文档 Schema 创建 Collection 与 Partition。
func (c *Client) EnsureCollection(ctx context.Context) error {
	if !c.cfg.Enabled {
		return nil
	}
	// SDK 接入后此处应：
	// 1. 检查 Collection 是否存在，不存在则按 Schema 创建
	// 2. 为 embedding 字段创建 HNSW 索引 (M=16, efConstruction=200, metric=IP)
	// 3. 为 6 个 classic_code 创建 Partition p_{code}
	return nil
}

// Insert upserts a batch of vector records.
// TODO(milvus-sdk): 替换为 client.Insert(ctx, collection, partition, records...)。
func (c *Client) Insert(ctx context.Context, records []service.VectorRecord) error {
	if !c.cfg.Enabled {
		return nil
	}
	if len(records) == 0 {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range records {
		c.records[records[i].ChunkID] = records[i]
	}
	return nil
}

// DeleteByDoc removes all vectors belonging to a document.
// TODO(milvus-sdk): 替换为 client.Delete(ctx, collection, expr: doc_id == {docID})。
func (c *Client) DeleteByDoc(ctx context.Context, docID int64) error {
	if !c.cfg.Enabled {
		return nil
	}
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
// TODO(milvus-sdk): 替换为 client.Search(ctx, collection, partitions, vector, topK, expr)。
// 当前 stub 在内存中遍历，仅用于本地开发联调；正式环境必须接入 SDK。
func (c *Client) Search(ctx context.Context, query []float32, topK int, filters service.SearchFilter) ([]service.VectorSearchResult, error) {
	if !c.cfg.Enabled {
		return nil, nil
	}
	if topK <= 0 {
		topK = 20
	}
	if len(query) == 0 {
		return nil, errno.New(errno.InvalidParams, "empty query vector")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	// Stub: 按插入顺序返回前 topK 条匹配过滤条件的记录。
	// 真实 SDK 接入后此段代码会被 HNSW 检索替换。
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

// matchFilter applies the scalar filter to a record.
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
	return fmt.Sprintf("milvus.Client{collection=%s, dim=%d, enabled=%v, records=%d}",
		c.cfg.Collection, c.cfg.Dim, c.cfg.Enabled, len(c.records))
}
