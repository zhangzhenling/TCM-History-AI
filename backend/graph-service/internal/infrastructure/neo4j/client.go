// Package neo4j provides the Neo4j-backed GraphStore adapter.
//
// 设计依据：doc/05-知识图谱设计.md §5.2 / §5.3 / §5.5 / §5.8
//   - 8 类节点 Label（Person/Classic/School/Prescription/Medicine/Disease/Dynasty/HistoricalEvent）
//   - 9 类关系 Type（AUTHORED/DISCIPLED/INFLUENCED/BELONGS_TO/OCCURRED_IN/CITED/PROPOSED/OPPOSED/INHERITED）
//   - 唯一约束：每类节点 uid + 关系 uid；B-Tree 索引覆盖高频查询字段
//
// 实现策略（ADR-21-01）：
//   - enabled=true 时走 Neo4j HTTP 事务 API（POST /db/{db}/tx/commit + Cypher，net/http 直连）
//   - enabled=false 时退化为内存 stub，保证离线开发与单元测试可运行（ADR-21-02）
//
// HTTP 事务 API 参考：https://neo4j.com/docs/http-api/current/actions/
package neo4j

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/graph-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
)

// Config captures the Neo4j client coordinates.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Enabled  bool
	Timeout  int // HTTP call timeout in seconds, defaults to 30
}

// Client implements service.GraphStore. enabled=true 时走 HTTP 事务 API，
// enabled=false 时退化为内存 stub（与既有离线开发模式一致）。
type Client struct {
	cfg     Config
	httpCli *http.Client // per-instance HTTP client, replaces package-level singleton
	// stub 字段：仅用于 enabled=false 的离线开发与单元测试。
	mu    sync.Mutex
	nodes map[string]entity.GraphNodeView // uid -> view
	edges map[string]entity.GraphEdgeView // uid -> view
}

// New constructs a Client. 连接延迟到首次调用时建立。
func New(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30
	}
	return &Client{
		cfg:     cfg,
		httpCli: &http.Client{Timeout: time.Duration(timeout) * time.Second},
		nodes:   make(map[string]entity.GraphNodeView),
		edges:   make(map[string]entity.GraphEdgeView),
	}
}

// EnsureConstraints 建立 8 个节点唯一约束 + B-Tree 索引 + 关系唯一约束（doc/05 §5.8）。
func (c *Client) EnsureConstraints(ctx context.Context) error {
	if !c.cfg.Enabled {
		return nil
	}
	return c.httpEnsureConstraints(ctx)
}

// UpsertNode upserts a node by uid (MERGE semantics, doc/05 §5.6).
func (c *Client) UpsertNode(ctx context.Context, n service.NodePayload) error {
	if n.UID == "" {
		return errno.New(errno.InvalidParams, "uid is required")
	}
	if !c.cfg.Enabled {
		return c.stubUpsertNode(n)
	}
	return c.httpUpsertNode(ctx, n)
}

// GetNode fetches a single node by uid; returns (nil, nil) when not found.
func (c *Client) GetNode(ctx context.Context, uid string) (*entity.GraphNodeView, error) {
	if !c.cfg.Enabled {
		return c.stubGetNode(uid)
	}
	return c.httpGetNode(ctx, uid)
}

// DeleteNode removes a node by uid.
func (c *Client) DeleteNode(ctx context.Context, uid string) error {
	if !c.cfg.Enabled {
		return c.stubDeleteNode(uid)
	}
	return c.httpDeleteNode(ctx, uid)
}

// UpsertEdge upserts an edge by uid (MERGE semantics).
func (c *Client) UpsertEdge(ctx context.Context, e service.EdgePayload) error {
	if e.UID == "" {
		return errno.New(errno.InvalidParams, "uid is required")
	}
	if !c.cfg.Enabled {
		return c.stubUpsertEdge(e)
	}
	return c.httpUpsertEdge(ctx, e)
}

// GetEdge fetches a single edge by uid; returns (nil, nil) when not found.
func (c *Client) GetEdge(ctx context.Context, uid string) (*entity.GraphEdgeView, error) {
	if !c.cfg.Enabled {
		return c.stubGetEdge(uid)
	}
	return c.httpGetEdge(ctx, uid)
}

// DeleteEdge removes an edge by uid.
func (c *Client) DeleteEdge(ctx context.Context, uid string) error {
	if !c.cfg.Enabled {
		return c.stubDeleteEdge(uid)
	}
	return c.httpDeleteEdge(ctx, uid)
}

// QueryPath returns the shortest path between two nodes (doc/05 §5.5.3).
func (c *Client) QueryPath(ctx context.Context, startUID, endUID string, maxHops int) (*entity.GraphPath, error) {
	if !c.cfg.Enabled {
		return c.stubQueryPath(startUID, endUID, maxHops)
	}
	return c.httpQueryPath(ctx, startUID, endUID, maxHops)
}

// GetSubgraph returns the subgraph centred on centerUID (doc/05 §5.9).
func (c *Client) GetSubgraph(ctx context.Context, centerUID string, depth, limit int) (*entity.Subgraph, error) {
	if !c.cfg.Enabled {
		return c.stubGetSubgraph(centerUID, depth, limit)
	}
	return c.httpGetSubgraph(ctx, centerUID, depth, limit)
}

// GetPersonWorks returns the classics authored by a person (doc/05 §5.5.1).
func (c *Client) GetPersonWorks(ctx context.Context, personUID string) ([]entity.GraphNodeView, error) {
	if !c.cfg.Enabled {
		return c.stubGetPersonWorks(personUID), nil
	}
	return c.httpGetPersonWorks(ctx, personUID)
}

// GetSchoolLineage returns the discipled lineage of a school (doc/05 §5.5.2).
func (c *Client) GetSchoolLineage(ctx context.Context, schoolName string, maxDepth int) (*entity.LineagePath, error) {
	if !c.cfg.Enabled {
		return c.stubGetSchoolLineage(schoolName, maxDepth)
	}
	return c.httpGetSchoolLineage(ctx, schoolName, maxDepth)
}

// GetDynastyFigures returns the representative figures of a dynasty (doc/05 §5.5.4).
func (c *Client) GetDynastyFigures(ctx context.Context, dynastyName string) ([]entity.FigureWithWorks, error) {
	if !c.cfg.Enabled {
		return c.stubGetDynastyFigures(dynastyName), nil
	}
	return c.httpGetDynastyFigures(ctx, dynastyName)
}

// GetPrescriptionDetail returns the medicines and diseases of a prescription (doc/05 §5.5.5).
func (c *Client) GetPrescriptionDetail(ctx context.Context, prescriptionUID string) (*entity.PrescriptionGraph, error) {
	if !c.cfg.Enabled {
		return c.stubGetPrescriptionDetail(prescriptionUID)
	}
	return c.httpGetPrescriptionDetail(ctx, prescriptionUID)
}

// SearchNodes runs a keyword search over nodes, optionally restricted to a label.
func (c *Client) SearchNodes(ctx context.Context, keyword, label string, limit int) ([]entity.GraphNodeView, error) {
	if keyword == "" {
		return nil, errno.New(errno.InvalidParams, "keyword is required")
	}
	if !c.cfg.Enabled {
		return c.stubSearchNodes(keyword, label, limit), nil
	}
	return c.httpSearchNodes(ctx, keyword, label, limit)
}

// normaliseProps serialises a map[string]any to a stable json.RawMessage.
func normaliseProps(m map[string]any) json.RawMessage {
	if len(m) == 0 {
		return json.RawMessage("{}")
	}
	body, err := json.Marshal(m)
	if err != nil {
		return json.RawMessage("{}")
	}
	return body
}

// propsToMap unmarshals a JSON properties blob into a map.
func propsToMap(raw json.RawMessage) map[string]any {
	props := map[string]any{}
	if len(raw) == 0 || string(raw) == "null" {
		return props
	}
	_ = json.Unmarshal(raw, &props)
	return props
}

// nodeFromMap builds a GraphNodeView from a properties map + label.
func nodeFromMap(uid, label, name string, props map[string]any) entity.GraphNodeView {
	if name == "" {
		if v, ok := props["name"].(string); ok {
			name = v
		}
	}
	if uid == "" {
		if v, ok := props["uid"].(string); ok {
			uid = v
		}
	}
	if label == "" {
		if v, ok := props["label"].(string); ok {
			label = v
		}
	}
	return entity.GraphNodeView{
		UID:        uid,
		Label:      label,
		Name:       name,
		Properties: normaliseProps(props),
	}
}

// String returns a debug representation.
func (c *Client) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return fmt.Sprintf("neo4j.Client{host=%s:%d, enabled=%v, nodes=%d, edges=%d}",
		c.cfg.Host, c.cfg.Port, c.cfg.Enabled, len(c.nodes), len(c.edges))
}

// Compile-time check: the Client satisfies the domain GraphStore port.
var _ service.GraphStore = (*Client)(nil)
