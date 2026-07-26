// Package neo4j provides the Neo4j-backed GraphStore adapter.
//
// 设计依据：doc/05-知识图谱设计.md §5.2 / §5.3 / §5.5 / §5.8
//   - 8 类节点 Label（Person/Classic/School/Prescription/Medicine/Disease/Dynasty/HistoricalEvent）
//   - 9 类关系 Type（AUTHORED/DISCIPLED/INFLUENCED/BELONGS_TO/OCCURRED_IN/CITED/PROPOSED/OPPOSED/INHERITED）
//   - 唯一约束：每类节点 uid + 关系 uid；B-Tree 索引覆盖高频查询字段
//
// 注意：本文件目前以「SDK 接入占位」形式实现，未引入 neo4j-go-driver 依赖。
// 原因：离线开发环境下 go get 无法拉取新模块；此处先保证接口契约与可编译性，
// 待联网环境下补全 SDK 调用（标记为 TODO(neo4j-sdk)）。在 neo4j.enabled=false
// 时该实现返回空结果，内存 map 仅用于本地开发联调。与 knowledge-service 的
// milvus stub 模式一致。
package neo4j

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

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
}

// Client implements service.GraphStore with the Neo4j SDK.
// 在 SDK 未接入前，所有写操作记录在内存（用于单元测试），读操作返回空。
type Client struct {
	cfg   Config
	mu    sync.Mutex
	nodes map[string]entity.GraphNodeView // uid -> view，仅用于离线 stub
	edges map[string]entity.GraphEdgeView // uid -> view
}

// New constructs a Client. 连接延迟到首次调用时建立。
func New(cfg Config) *Client {
	return &Client{
		cfg:   cfg,
		nodes: make(map[string]entity.GraphNodeView),
		edges: make(map[string]entity.GraphEdgeView),
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

// UpsertNode upserts a node by uid (MERGE semantics, doc/05 §5.6).
// TODO(neo4j-sdk): 替换为 session.Run("MERGE (n:Label {uid:$uid}) SET n += $props")。
func (c *Client) UpsertNode(ctx context.Context, n service.NodePayload) error {
	if !c.cfg.Enabled {
		return nil
	}
	if n.UID == "" {
		return errno.New(errno.InvalidParams, "uid is required")
	}
	props := normaliseProps(n.Properties)
	view := entity.GraphNodeView{
		UID:        n.UID,
		Label:      n.Label,
		Name:       n.Name,
		Properties: props,
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nodes[n.UID] = view
	_ = ctx
	return nil
}

// GetNode fetches a single node by uid; returns (nil, nil) when not found.
// TODO(neo4j-sdk): 替换为 session.Run("MATCH (n {uid:$uid}) RETURN n")。
func (c *Client) GetNode(ctx context.Context, uid string) (*entity.GraphNodeView, error) {
	if !c.cfg.Enabled {
		return nil, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if n, ok := c.nodes[uid]; ok {
		cp := n
		return &cp, nil
	}
	_ = ctx
	return nil, nil
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
	// 同步删除引用该节点的边，保持 stub 内部一致。
	for eid, e := range c.edges {
		if e.SourceUID == uid || e.TargetUID == uid {
			delete(c.edges, eid)
		}
	}
	_ = ctx
	return nil
}

// UpsertEdge upserts an edge by uid (MERGE semantics).
// TODO(neo4j-sdk): 替换为 session.Run("MATCH (a {uid:$from}), (b {uid:$to}) MERGE (a)-[r:TYPE {uid:$uid}]->(b) SET r += $props")。
func (c *Client) UpsertEdge(ctx context.Context, e service.EdgePayload) error {
	if !c.cfg.Enabled {
		return nil
	}
	if e.UID == "" {
		return errno.New(errno.InvalidParams, "uid is required")
	}
	view := entity.GraphEdgeView{
		UID:        e.UID,
		Type:       e.Type,
		SourceUID:  e.SourceUID,
		TargetUID:  e.TargetUID,
		Properties: normaliseProps(e.Properties),
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.edges[e.UID] = view
	_ = ctx
	return nil
}

// GetEdge fetches a single edge by uid; returns (nil, nil) when not found.
// TODO(neo4j-sdk): 替换为 session.Run("MATCH ()-[r {uid:$uid}]->() RETURN r")。
func (c *Client) GetEdge(ctx context.Context, uid string) (*entity.GraphEdgeView, error) {
	if !c.cfg.Enabled {
		return nil, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.edges[uid]; ok {
		cp := e
		return &cp, nil
	}
	_ = ctx
	return nil, nil
}

// DeleteEdge removes an edge by uid.
// TODO(neo4j-sdk): 替换为 session.Run("MATCH ()-[r {uid:$uid}]->() DELETE r")。
func (c *Client) DeleteEdge(ctx context.Context, uid string) error {
	if !c.cfg.Enabled {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.edges, uid)
	_ = ctx
	return nil
}

// QueryPath returns the shortest path between two nodes (doc/05 §5.5.3).
// Stub: 返回 nil 表示路径不可达，待 SDK 接入后替换为 shortestPath Cypher。
// TODO(neo4j-sdk): 替换为 session.Run("MATCH p = shortestPath((a {uid:$start})-[*..$max]-(b {uid:$end})) RETURN p")。
func (c *Client) QueryPath(ctx context.Context, startUID, endUID string, maxHops int) (*entity.GraphPath, error) {
	if !c.cfg.Enabled {
		return nil, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// Stub: BFS over in-memory map，maxHops 限定深度。
	if _, ok := c.nodes[startUID]; !ok {
		return nil, nil
	}
	if _, ok := c.nodes[endUID]; !ok {
		return nil, nil
	}
	if maxHops <= 0 {
		maxHops = 8
	}
	// 邻接表（无向，因为 shortestPath Cypher 使用 -[*..n]-）。
	adj := make(map[string][]edgeRef)
	for _, e := range c.edges {
		adj[e.SourceUID] = append(adj[e.SourceUID], edgeRef{to: e.TargetUID, edge: e})
		adj[e.TargetUID] = append(adj[e.TargetUID], edgeRef{to: e.SourceUID, edge: e})
	}
	type state struct {
		uid   string
		path  []string
		edges []entity.GraphEdgeView
	}
	queue := []state{{uid: startUID, path: []string{startUID}}}
	visited := map[string]bool{startUID: true}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if len(cur.path)-1 > maxHops {
			break
		}
		if cur.uid == endUID {
			nodes := make([]entity.GraphNodeView, 0, len(cur.path))
			for _, uid := range cur.path {
				nodes = append(nodes, c.nodes[uid])
			}
			return &entity.GraphPath{
				Nodes: nodes,
				Edges: cur.edges,
				Hops:  len(cur.edges),
			}, nil
		}
		for _, ref := range adj[cur.uid] {
			if visited[ref.to] {
				continue
			}
			visited[ref.to] = true
			queue = append(queue, state{
				uid:   ref.to,
				path:  append(append([]string{}, cur.path...), ref.to),
				edges: append(append([]entity.GraphEdgeView{}, cur.edges...), ref.edge),
			})
		}
	}
	_ = ctx
	return nil, nil
}

// edgeRef is a stub helper for the BFS in QueryPath.
type edgeRef struct {
	to   string
	edge entity.GraphEdgeView
}

// GetSubgraph returns the subgraph centred on centerUID (doc/05 §5.9).
// Stub: 在内存图上做有限深度 BFS。
// TODO(neo4j-sdk): 替换为 session.Run("MATCH p=(n {uid:$center})-[*1..$depth]-(m) RETURN p LIMIT $limit")。
func (c *Client) GetSubgraph(ctx context.Context, centerUID string, depth, limit int) (*entity.Subgraph, error) {
	if !c.cfg.Enabled {
		return &entity.Subgraph{}, nil
	}
	if limit <= 0 {
		limit = 100
	}
	if depth <= 0 {
		depth = 2
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.nodes[centerUID]; !ok {
		return &entity.Subgraph{}, nil
	}
	// 邻接表（无向）。
	adj := make(map[string][]edgeRef)
	for _, e := range c.edges {
		adj[e.SourceUID] = append(adj[e.SourceUID], edgeRef{to: e.TargetUID, edge: e})
		adj[e.TargetUID] = append(adj[e.TargetUID], edgeRef{to: e.SourceUID, edge: e})
	}
	visited := map[string]bool{centerUID: true}
	queue := []struct {
		uid   string
		depth int
	}{{uid: centerUID, depth: 0}}
	nodeSet := map[string]entity.GraphNodeView{centerUID: c.nodes[centerUID]}
	edgeSet := map[string]entity.GraphEdgeView{}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.depth >= depth {
			continue
		}
		if len(nodeSet) >= limit {
			break
		}
		for _, ref := range adj[cur.uid] {
			if !visited[ref.to] {
				visited[ref.to] = true
				nodeSet[ref.to] = c.nodes[ref.to]
				queue = append(queue, struct {
					uid   string
					depth int
				}{uid: ref.to, depth: cur.depth + 1})
			}
			if _, ok := edgeSet[ref.edge.UID]; !ok {
				edgeSet[ref.edge.UID] = ref.edge
			}
			if len(nodeSet) >= limit {
				break
			}
		}
	}
	out := &entity.Subgraph{
		Nodes: make([]entity.GraphNodeView, 0, len(nodeSet)),
		Edges: make([]entity.GraphEdgeView, 0, len(edgeSet)),
	}
	for _, n := range nodeSet {
		out.Nodes = append(out.Nodes, n)
	}
	for _, e := range edgeSet {
		out.Edges = append(out.Edges, e)
	}
	_ = ctx
	return out, nil
}

// GetPersonWorks returns the classics authored by a person (doc/05 §5.5.1).
// TODO(neo4j-sdk): 替换为 session.Run("MATCH (p:Person {uid:$uid})-[:AUTHORED]->(c:Classic) RETURN c")。
func (c *Client) GetPersonWorks(ctx context.Context, personUID string) ([]entity.GraphNodeView, error) {
	if !c.cfg.Enabled {
		return nil, nil
	}
	_ = ctx
	return c.collectRelatedNodes(personUID, entity.RelAuthored, true), nil
}

// GetSchoolLineage returns the discipled lineage of a school (doc/05 §5.5.2).
// Stub: 内存中遍历 BELONGS_TO 反向定位学派成员，再沿 DISCIPLED 边展开。
// TODO(neo4j-sdk): 替换为 apoc.path.expandConfig 变长路径遍历。
func (c *Client) GetSchoolLineage(ctx context.Context, schoolName string, maxDepth int) (*entity.LineagePath, error) {
	if !c.cfg.Enabled {
		return nil, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if maxDepth <= 0 {
		maxDepth = 6
	}
	// 找到 School 节点。
	var schoolUID string
	for uid, n := range c.nodes {
		if n.Label == entity.LabelSchool && (n.Name == schoolName || schoolNameMatch(n.Properties, schoolName)) {
			schoolUID = uid
			break
		}
	}
	if schoolUID == "" {
		_ = ctx
		return nil, nil
	}
	// 反向找 BELONGS_TO 关联人物。
	members := make([]string, 0)
	for _, e := range c.edges {
		if e.Type == entity.RelBelongsTo && e.TargetUID == schoolUID {
			members = append(members, e.SourceUID)
		}
	}
	if len(members) == 0 {
		_ = ctx
		return nil, nil
	}
	// 构建师承有向图（DISCIPLED: 弟子 → 师父）。
	discipled := make(map[string]string)
	for _, e := range c.edges {
		if e.Type == entity.RelDiscipled {
			discipled[e.SourceUID] = e.TargetUID
		}
	}
	pathNodes := make([]entity.GraphNodeView, 0)
	pathEdges := make([]entity.GraphEdgeView, 0)
	generations := make([]int, 0)
	for _, m := range members {
		chain := []string{m}
		visited := map[string]bool{m: true}
		cur := m
		for i := 0; i < maxDepth; i++ {
			next, ok := discipled[cur]
			if !ok || visited[next] {
				break
			}
			chain = append(chain, next)
			visited[next] = true
			cur = next
		}
		for i, uid := range chain {
			if n, ok := c.nodes[uid]; ok {
				pathNodes = append(pathNodes, n)
				generations = append(generations, i)
			}
		}
	}
	_ = ctx
	if len(pathNodes) == 0 {
		return nil, nil
	}
	return &entity.LineagePath{
		Path: entity.GraphPath{
			Nodes: pathNodes,
			Edges: pathEdges,
			Hops:  len(pathNodes) - 1,
		},
		Generations: generations,
	}, nil
}

// GetDynastyFigures returns the representative figures of a dynasty (doc/05 §5.5.4).
// TODO(neo4j-sdk): 替换为反向遍历 OCCURRED_IN + 联动 AUTHORED/BELONGS_TO。
func (c *Client) GetDynastyFigures(ctx context.Context, dynastyName string) ([]entity.FigureWithWorks, error) {
	if !c.cfg.Enabled {
		return nil, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// 找到 Dynasty 节点。
	var dynastyUID string
	for uid, n := range c.nodes {
		if n.Label == entity.LabelDynasty && n.Name == dynastyName {
			dynastyUID = uid
			break
		}
	}
	if dynastyUID == "" {
		_ = ctx
		return nil, nil
	}
	// 反向找 OCCURRED_IN 关联人物。
	out := make([]entity.FigureWithWorks, 0)
	for _, e := range c.edges {
		if e.Type != entity.RelOccurredIn || e.TargetUID != dynastyUID {
			continue
		}
		person, ok := c.nodes[e.SourceUID]
		if !ok || person.Label != entity.LabelPerson {
			continue
		}
		works := make([]entity.GraphNodeView, 0)
		schools := make([]entity.GraphNodeView, 0)
		for _, re := range c.edges {
			if re.SourceUID == person.UID && re.Type == entity.RelAuthored {
				if w, ok := c.nodes[re.TargetUID]; ok {
					works = append(works, w)
				}
			}
			if re.SourceUID == person.UID && re.Type == entity.RelBelongsTo {
				if s, ok := c.nodes[re.TargetUID]; ok {
					schools = append(schools, s)
				}
			}
		}
		out = append(out, entity.FigureWithWorks{
			Person:  person,
			Works:   works,
			Schools: schools,
		})
	}
	_ = ctx
	return out, nil
}

// GetPrescriptionDetail returns the medicines and diseases of a prescription (doc/05 §5.5.5).
// TODO(neo4j-sdk): 替换为 OPTIONAL MATCH 组成与主治关系遍历。
func (c *Client) GetPrescriptionDetail(ctx context.Context, prescriptionUID string) (*entity.PrescriptionGraph, error) {
	if !c.cfg.Enabled {
		return nil, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	rx, ok := c.nodes[prescriptionUID]
	if !ok || rx.Label != entity.LabelPrescription {
		_ = ctx
		return nil, nil
	}
	medicines := make([]entity.GraphNodeView, 0)
	diseases := make([]entity.GraphNodeView, 0)
	// 注：doc/05 §5.5.5 提到 COMPOSED_OF / TREATS 为方剂子图支撑关系。
	// stub 中通过 properties.composition / indication 近似匹配，不引入新关系类型。
	for _, e := range c.edges {
		if e.SourceUID != prescriptionUID {
			continue
		}
		switch e.Type {
		case "COMPOSED_OF":
			if m, ok := c.nodes[e.TargetUID]; ok && m.Label == entity.LabelMedicine {
				medicines = append(medicines, m)
			}
		case "TREATS":
			if d, ok := c.nodes[e.TargetUID]; ok && d.Label == entity.LabelDisease {
				diseases = append(diseases, d)
			}
		}
	}
	_ = ctx
	return &entity.PrescriptionGraph{
		Prescription: rx,
		Medicines:    medicines,
		Diseases:     diseases,
	}, nil
}

// SearchNodes runs a keyword search over nodes, optionally restricted to a label.
// TODO(neo4j-sdk): 替换为全文索引检索（§5.8.3 node_fulltext）。
func (c *Client) SearchNodes(ctx context.Context, keyword, label string, limit int) ([]entity.GraphNodeView, error) {
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
	results := make([]entity.GraphNodeView, 0, limit)
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

// nodeMatchesKeyword reports whether the node name or any string property of n contains kw.
func nodeMatchesKeyword(n entity.GraphNodeView, kw string) bool {
	if strings.Contains(strings.ToLower(n.Name), kw) {
		return true
	}
	for _, v := range propsToMap(n.Properties) {
		if s, ok := v.(string); ok && strings.Contains(strings.ToLower(s), kw) {
			return true
		}
	}
	return false
}

// schoolNameMatch reports whether the node properties contain a name field
// matching schoolName. Used by GetSchoolLineage when only the schoolName is
// provided.
func schoolNameMatch(props json.RawMessage, schoolName string) bool {
	m := propsToMap(props)
	if v, ok := m["name"].(string); ok && v == schoolName {
		return true
	}
	return false
}

// collectRelatedNodes is a stub helper that traverses the in-memory edges map
// to return nodes connected to srcUID via the given edge type. When
// outgoing is true, srcUID is treated as the source side.
func (c *Client) collectRelatedNodes(srcUID, edgeType string, outgoing bool) []entity.GraphNodeView {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]entity.GraphNodeView, 0)
	for _, e := range c.edges {
		if e.Type != edgeType {
			continue
		}
		if outgoing && e.SourceUID == srcUID {
			if n, ok := c.nodes[e.TargetUID]; ok {
				out = append(out, n)
			}
		}
		if !outgoing && e.TargetUID == srcUID {
			if n, ok := c.nodes[e.SourceUID]; ok {
				out = append(out, n)
			}
		}
	}
	return out
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

// String returns a debug representation.
func (c *Client) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return fmt.Sprintf("neo4j.Client{host=%s:%d, enabled=%v, nodes=%d, edges=%d}",
		c.cfg.Host, c.cfg.Port, c.cfg.Enabled, len(c.nodes), len(c.edges))
}

// Compile-time check: the Client satisfies the domain GraphStore port.
var _ service.GraphStore = (*Client)(nil)
