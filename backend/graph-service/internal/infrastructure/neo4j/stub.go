// stub.go — 内存 stub 实现，仅用于 enabled=false 的离线开发与单元测试。
// 所有 stubXxx 方法在 c.mu 保护下操作 c.nodes / c.edges 内存 map，
// 行为与原 client.go 的 stub 路径一致（保持向后兼容）。

package neo4j

import (
	"encoding/json"
	"strings"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/graph-service/internal/domain/service"
)

// stubUpsertNode records the node in the in-memory map.
func (c *Client) stubUpsertNode(n service.NodePayload) error {
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
	return nil
}

// stubGetNode returns the node from the in-memory map; (nil, nil) when absent.
func (c *Client) stubGetNode(uid string) (*entity.GraphNodeView, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if n, ok := c.nodes[uid]; ok {
		cp := n
		return &cp, nil
	}
	return nil, nil
}

// stubDeleteNode removes the node and any edges referencing it.
func (c *Client) stubDeleteNode(uid string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.nodes, uid)
	for eid, e := range c.edges {
		if e.SourceUID == uid || e.TargetUID == uid {
			delete(c.edges, eid)
		}
	}
	return nil
}

// stubUpsertEdge records the edge in the in-memory map.
func (c *Client) stubUpsertEdge(e service.EdgePayload) error {
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
	return nil
}

// stubGetEdge returns the edge from the in-memory map; (nil, nil) when absent.
func (c *Client) stubGetEdge(uid string) (*entity.GraphEdgeView, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.edges[uid]; ok {
		cp := e
		return &cp, nil
	}
	return nil, nil
}

// stubDeleteEdge removes the edge from the in-memory map.
func (c *Client) stubDeleteEdge(uid string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.edges, uid)
	return nil
}

// edgeRef is a stub helper for the BFS in stubQueryPath / stubGetSubgraph.
type edgeRef struct {
	to   string
	edge entity.GraphEdgeView
}

// stubAdjacency builds an undirected adjacency list from the in-memory edges.
func (c *Client) stubAdjacency() map[string][]edgeRef {
	adj := make(map[string][]edgeRef)
	for _, e := range c.edges {
		adj[e.SourceUID] = append(adj[e.SourceUID], edgeRef{to: e.TargetUID, edge: e})
		adj[e.TargetUID] = append(adj[e.TargetUID], edgeRef{to: e.SourceUID, edge: e})
	}
	return adj
}

// stubQueryPath does BFS over the in-memory graph.
func (c *Client) stubQueryPath(startUID, endUID string, maxHops int) (*entity.GraphPath, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.nodes[startUID]; !ok {
		return nil, nil
	}
	if _, ok := c.nodes[endUID]; !ok {
		return nil, nil
	}
	if maxHops <= 0 {
		maxHops = 8
	}
	adj := c.stubAdjacency()
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
	return nil, nil
}

// stubGetSubgraph does limited-depth BFS over the in-memory graph.
func (c *Client) stubGetSubgraph(centerUID string, depth, limit int) (*entity.Subgraph, error) {
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
	adj := c.stubAdjacency()
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
	return out, nil
}

// stubGetPersonWorks traverses AUTHORED edges from personUID.
func (c *Client) stubGetPersonWorks(personUID string) []entity.GraphNodeView {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]entity.GraphNodeView, 0)
	for _, e := range c.edges {
		if e.Type != entity.RelAuthored {
			continue
		}
		if e.SourceUID == personUID {
			if n, ok := c.nodes[e.TargetUID]; ok {
				out = append(out, n)
			}
		}
	}
	return out
}

// schoolNameMatch reports whether the node properties contain a name field
// matching schoolName.
func schoolNameMatch(props json.RawMessage, schoolName string) bool {
	m := propsToMap(props)
	if v, ok := m["name"].(string); ok && v == schoolName {
		return true
	}
	return false
}

// stubGetSchoolLineage traverses BELONGS_TO + DISCIPLED in memory.
func (c *Client) stubGetSchoolLineage(schoolName string, maxDepth int) (*entity.LineagePath, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if maxDepth <= 0 {
		maxDepth = 6
	}
	var schoolUID string
	for uid, n := range c.nodes {
		if n.Label == entity.LabelSchool && (n.Name == schoolName || schoolNameMatch(n.Properties, schoolName)) {
			schoolUID = uid
			break
		}
	}
	if schoolUID == "" {
		return nil, nil
	}
	members := make([]string, 0)
	for _, e := range c.edges {
		if e.Type == entity.RelBelongsTo && e.TargetUID == schoolUID {
			members = append(members, e.SourceUID)
		}
	}
	if len(members) == 0 {
		return nil, nil
	}
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

// stubGetDynastyFigures traverses OCCURRED_IN + AUTHORED + BELONGS_TO in memory.
func (c *Client) stubGetDynastyFigures(dynastyName string) []entity.FigureWithWorks {
	c.mu.Lock()
	defer c.mu.Unlock()
	var dynastyUID string
	for uid, n := range c.nodes {
		if n.Label == entity.LabelDynasty && n.Name == dynastyName {
			dynastyUID = uid
			break
		}
	}
	if dynastyUID == "" {
		return nil
	}
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
	return out
}

// stubGetPrescriptionDetail traverses COMPOSED_OF + TREATS in memory.
func (c *Client) stubGetPrescriptionDetail(prescriptionUID string) (*entity.PrescriptionGraph, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	rx, ok := c.nodes[prescriptionUID]
	if !ok || rx.Label != entity.LabelPrescription {
		return nil, nil
	}
	medicines := make([]entity.GraphNodeView, 0)
	diseases := make([]entity.GraphNodeView, 0)
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
	return &entity.PrescriptionGraph{
		Prescription: rx,
		Medicines:    medicines,
		Diseases:     diseases,
	}, nil
}

// nodeMatchesKeyword reports whether the node name or any string property contains kw.
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

// stubSearchNodes does keyword search over the in-memory nodes.
func (c *Client) stubSearchNodes(keyword, label string, limit int) []entity.GraphNodeView {
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
	return results
}
