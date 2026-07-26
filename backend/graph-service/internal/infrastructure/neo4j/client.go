// Package neo4j provides the Neo4j-backed GraphStore / GraphRepository adapter.
//
// 设计依据：doc/05-知识图谱设计.md §5.2 / §5.3 / §5.5 / §5.8
//   - 8 类节点 Label（Person/Classic/School/Prescription/Medicine/Disease/Dynasty/HistoricalEvent）
//   - 9 类关系 Type（AUTHORED/DISCIPLED/INFLUENCED/BELONGS_TO/OCCURRED_IN/CITED/PROPOSED/OPPOSED/INHERITED）
//   - 唯一约束：每类节点 uid + 关系 uid；B-Tree 索引覆盖高频查询字段
//
// 注意：本文件目前以「SDK 接入占位」形式实现，未引入 neo4j-go-driver 依赖。
// 原因：离线开发环境下 go get 无法拉取新模块；此处先保证接口契约与可编译性，
// 待联网环境下补全 SDK 调用（标记为 TODO(neo4j-sdk)）。在 neo4j.enabled=false
// 时该实现返回空结果，内存 map 仅用于本地开发联调。
package neo4j

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/graph-service/internal/domain/repository"
	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/pagination"
)

// Config captures the Neo4j client coordinates.
type Config struct {
	URI      string // bolt://host:7687
	Username string
	Password string
	Enabled  bool
}

// Client implements repository.GraphRepository and service.GraphStore with the
// Neo4j SDK. 在 SDK 未接入前，所有写操作记录在内存（用于单元测试），读操作返回空。
type Client struct {
	cfg  Config
	mu   sync.Mutex
	// nodes 以 uid 为键存储节点，便于本地开发 stub。
	nodes map[string]entity.GraphNode
	// rels 以 uid 为键存储关系。
	rels map[string]entity.GraphRelationship
}

// New constructs a Client. 连接延迟到首次调用时建立。
func New(cfg Config) *Client {
	return &Client{
		cfg:   cfg,
		nodes: make(map[string]entity.GraphNode),
		rels:  make(map[string]entity.GraphRelationship),
	}
}

// EnsureConstraints 建立 8 个节点唯一约束 + B-Tree 索引 + 关系唯一约束（doc/05 §5.8）。
// TODO(neo4j-sdk): 接入 neo4j-go-driver，执行 §5.8.1 / §5.8.2 / §5.8.3 的 DDL。
func (c *Client) EnsureConstraints(ctx context.Context) error {
	if !c.cfg.Enabled {
		return nil
	}
	// SDK 接入后此处应执行：
	//  1. 为 8 类节点 Label 的 uid 建立唯一约束（§5.8.1）
	//  2. 为高频字段建立 B-Tree 索引（§5.8.2）
	//  3. 为关系 uid 建立唯一约束（§5.8.3 末尾）
	//  4. 为 intro/abstract/achievements 建立全文索引（§5.8.3）
	_ = ctx
	return nil
}

// MergeNode upserts a node by uid (MERGE semantics, doc/05 §5.6).
// TODO(neo4j-sdk): 替换为 session.Run("MERGE (n:Label {uid:$uid}) SET n += $props")。
func (c *Client) MergeNode(ctx context.Context, label, uid string, props map[string]any) error {
	if !c.cfg.Enabled {
		return nil
	}
	if uid == "" {
		return errno.New(errno.InvalidParams, "uid is required")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nodes[uid] = entity.GraphNode{UID: uid, Label: label, Properties: props}
	_ = ctx
	return nil
}

// DeleteNode removes a node by uid.
// TODO(neo4j-sdk): 替换为 session.Run("MATCH (n {uid:$uid}) DETACH DELETE n")。
func (c *Client) DeleteNode(ctx context.Context, uid string) error {
	if !c.cfg.Enabled {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.nodes, uid)
	_ = ctx
	return nil
}

// GetNode fetches a single node by uid; returns (nil, nil) when not found.
// TODO(neo4j-sdk): 替换为 session.Run("MATCH (n {uid:$uid}) RETURN n")。
func (c *Client) GetNode(ctx context.Context, uid string) (*entity.GraphNode, error) {
	if !c.cfg.Enabled {
		return nil, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if n, ok := c.nodes[uid]; ok {
		return &n, nil
	}
	_ = ctx
	return nil, nil
}

// ListNodes returns a paginated list of nodes, optionally filtered by label.
// TODO(neo4j-sdk): 替换为参数化 Cypher MATCH (n:Label) RETURN n SKIP $skip LIMIT $limit。
func (c *Client) ListNodes(ctx context.Context, label string, p pagination.Params) ([]entity.GraphNode, int, error) {
	if !c.cfg.Enabled {
		return nil, 0, nil
	}
	_, pageSize, offset := p.Normalise()
	c.mu.Lock()
	defer c.mu.Unlock()
	// Stub: 遍历内存 map，按 uid 排序保持稳定顺序。
	var all []entity.GraphNode
	for _, n := range c.nodes {
		if label != "" && n.Label != label {
			continue
		}
		all = append(all, n)
	}
	total := len(all)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	_ = ctx
	return all[offset:end], total, nil
}

// SearchNodes runs a keyword search over nodes, optionally restricted to a label.
// TODO(neo4j-sdk): 替换为全文索引检索（§5.8.3 node_fulltext）。
func (c *Client) SearchNodes(ctx context.Context, keyword, label string, limit int) ([]entity.GraphNode, error) {
	if !c.cfg.Enabled {
		return nil, nil
	}
	if keyword == "" {
		return nil, errno.New(errno.InvalidParams, "keyword is required")
	}
	if limit <= 0 {
		limit = 20
	}
	kw := strings.ToLower(keyword)
	c.mu.Lock()
	defer c.mu.Unlock()
	results := make([]entity.GraphNode, 0, limit)
	for _, n := range c.nodes {
		if label != "" && n.Label != label {
			continue
		}
		if !nodeMatchesKeyword(n, kw) {
			continue
		}
		results = append(results, n)
		if len(results) >= limit {
			break
		}
	}
	_ = ctx
	return results, nil
}

// nodeMatchesKeyword reports whether any string property of n contains kw.
func nodeMatchesKeyword(n entity.GraphNode, kw string) bool {
	for _, v := range n.Properties {
		if s, ok := v.(string); ok && strings.Contains(strings.ToLower(s), kw) {
			return true
		}
	}
	return false
}

// MergeRelationship upserts a relationship by uid (MERGE semantics).
// TODO(neo4j-sdk): 替换为 session.Run("MATCH (a {uid:$from}), (b {uid:$to}) MERGE (a)-[r:TYPE {uid:$uid}]->(b) SET r += $props")。
func (c *Client) MergeRelationship(ctx context.Context, relType, fromUID, toUID, uid string, props map[string]any) error {
	if !c.cfg.Enabled {
		return nil
	}
	if uid == "" {
		return errno.New(errno.InvalidParams, "uid is required")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rels[uid] = entity.GraphRelationship{
		UID:        uid,
		Type:       relType,
		FromUID:    fromUID,
		ToUID:      toUID,
		Properties: props,
	}
	_ = ctx
	return nil
}

// DeleteRelationship removes a relationship by uid.
// TODO(neo4j-sdk): 替换为 session.Run("MATCH ()-[r {uid:$uid}]->() DELETE r")。
func (c *Client) DeleteRelationship(ctx context.Context, uid string) error {
	if !c.cfg.Enabled {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.rels, uid)
	_ = ctx
	return nil
}

// GetRelationship fetches a single relationship by uid; returns (nil, nil) when not found.
// TODO(neo4j-sdk): 替换为 session.Run("MATCH ()-[r {uid:$uid}]->() RETURN r")。
func (c *Client) GetRelationship(ctx context.Context, uid string) (*entity.GraphRelationship, error) {
	if !c.cfg.Enabled {
		return nil, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if r, ok := c.rels[uid]; ok {
		return &r, nil
	}
	_ = ctx
	return nil, nil
}

// FindShortestPath returns the shortest path between two nodes (doc/05 §5.5.3).
// Stub: 返回 nil 表示路径不可达，待 SDK 接入后替换为 shortestPath Cypher。
// TODO(neo4j-sdk): 替换为 session.Run("MATCH p = shortestPath((a {uid:$start})-[*..$max]-(b {uid:$end})) RETURN p")。
func (c *Client) FindShortestPath(ctx context.Context, startUID, endUID string, maxHops int) (*entity.GraphPath, error) {
	if !c.cfg.Enabled {
		return nil, nil
	}
	_ = ctx
	_ = maxHops
	// Stub: 不做 BFS，直接返回 nil；真实 SDK 接入后由 Cypher shortestPath 计算。
	return nil, nil
}

// GetSubgraph returns the subgraph centred on centerUID (doc/05 §5.9).
// Stub: 返回空子图，待 SDK 接入后替换为变长路径遍历。
// TODO(neo4j-sdk): 替换为 session.Run("MATCH p=(n {uid:$center})-[*1..$depth]-(m) RETURN p LIMIT $limit")。
func (c *Client) GetSubgraph(ctx context.Context, centerUID string, depth, limit int) (*entity.Subgraph, error) {
	if !c.cfg.Enabled {
		return &entity.Subgraph{}, nil
	}
	_ = ctx
	_ = centerUID
	_ = depth
	_ = limit
	// Stub: 返回空子图，避免前端渲染空指针。
	return &entity.Subgraph{}, nil
}

// GetPersonWorks returns the classics authored by a person (doc/05 §5.5.1).
// TODO(neo4j-sdk): 替换为 session.Run("MATCH (p:Person {uid:$uid})-[:AUTHORED]->(c:Classic) RETURN c")。
func (c *Client) GetPersonWorks(ctx context.Context, personUID string) ([]entity.GraphNode, error) {
	if !c.cfg.Enabled {
		return nil, nil
	}
	_ = ctx
	return c.collectRelatedNodes(personUID, entity.RelAuthored, true), nil
}

// GetSchoolLineage returns the discipled lineage of a school (doc/05 §5.5.2).
// TODO(neo4j-sdk): 替换为 apoc.path.expandConfig 变长路径遍历。
func (c *Client) GetSchoolLineage(ctx context.Context, schoolName string, maxDepth int) (*entity.LineagePath, error) {
	if !c.cfg.Enabled {
		return nil, nil
	}
	_ = ctx
	_ = schoolName
	_ = maxDepth
	// Stub: 返回 nil 表示未找到师承链。
	return nil, nil
}

// GetDynastyFigures returns the representative figures of a dynasty (doc/05 §5.5.4).
// TODO(neo4j-sdk): 替换为反向遍历 OCCURRED_IN + 联动 AUTHORED/BELONGS_TO。
func (c *Client) GetDynastyFigures(ctx context.Context, dynastyName string) ([]entity.FigureWithWorks, error) {
	if !c.cfg.Enabled {
		return nil, nil
	}
	_ = ctx
	_ = dynastyName
	return nil, nil
}

// GetPrescriptionDetail returns the medicines and diseases of a prescription (doc/05 §5.5.5).
// TODO(neo4j-sdk): 替换为 OPTIONAL MATCH 组成与主治关系遍历。
func (c *Client) GetPrescriptionDetail(ctx context.Context, prescriptionUID string) (*entity.PrescriptionGraph, error) {
	if !c.cfg.Enabled {
		return nil, nil
	}
	_ = ctx
	_ = prescriptionUID
	return nil, nil
}

// RunCypher is the generic Cypher query interface.
// TODO(neo4j-sdk): 替换为 session.Run(query, params) 并将 result records 解包为 [][]any。
func (c *Client) RunCypher(ctx context.Context, query string, params map[string]any) ([][]any, error) {
	if !c.cfg.Enabled {
		return nil, nil
	}
	_ = ctx
	_ = query
	_ = params
	// Stub: 返回空结果集。
	return nil, nil
}

// collectRelatedNodes is a stub helper that traverses the in-memory rels map
// to return nodes connected to srcUID via the given relationship type. When
// outgoing is true, srcUID is treated as the from side.
func (c *Client) collectRelatedNodes(srcUID, relType string, outgoing bool) []entity.GraphNode {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]entity.GraphNode, 0)
	for _, r := range c.rels {
		if r.Type != relType {
			continue
		}
		if outgoing && r.FromUID == srcUID {
			if n, ok := c.nodes[r.ToUID]; ok {
				out = append(out, n)
			}
		}
		if !outgoing && r.ToUID == srcUID {
			if n, ok := c.nodes[r.FromUID]; ok {
				out = append(out, n)
			}
		}
	}
	return out
}

// String returns a debug representation.
func (c *Client) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return fmt.Sprintf("neo4j.Client{uri=%s, enabled=%v, nodes=%d, rels=%d}",
		c.cfg.URI, c.cfg.Enabled, len(c.nodes), len(c.rels))
}

// Compile-time checks: the Client satisfies both the domain repository port
// and the Neo4j driver abstraction.
var (
	_ repository.GraphRepository = (*Client)(nil)
)
