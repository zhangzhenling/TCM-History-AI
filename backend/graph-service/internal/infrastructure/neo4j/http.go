// http.go — Neo4j HTTP 事务 API 实现，仅用于 enabled=true 的生产/联调环境。
//
// 端点：POST /db/{database}/tx/commit
// 鉴权：HTTP Basic Auth（user:password）
// 协议：https://neo4j.com/docs/http-api/current/actions/#actions_transactional
//
// 请求体：{"statements":[{"statement":"<cypher>","parameters":{...}}]}
// 响应体：{"results":[{"columns":[...],"data":[{"row":[...]}]}],"errors":[...]}
//
// 设计要点：
//   - runCypher 返回 [][]json.RawMessage，让每个调用方按需反序列化自己的列
//   - 节点的 label 通过 MERGE 模式 :Label 静态绑定，查询时用 labels(n) 取回
//   - 路径查询返回 nodes(p) + relationships(p) 两列，避免解析嵌套 path 对象

package neo4j

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"tcm-history-ai/backend/graph-service/internal/domain/entity"
	"tcm-history-ai/backend/graph-service/internal/domain/service"
	"tcm-history-ai/backend/pkg/errno"
)

// baseURL returns the HTTP API root, e.g. http://localhost:7474.
// Neo4j HTTP 事务 API 默认端口是 7474（非 Bolt 7687）。
func (c *Client) baseURL() string {
	host := c.cfg.Host
	if host == "" {
		host = "localhost"
	}
	port := c.cfg.Port
	// 配置文件里的 port 是 Bolt 端口 7687；HTTP 端口通常是 bolt_port - 200 = 7474。
	// 但如果用户显式配置了 7474，则尊重配置。
	if port <= 0 || port == 7687 {
		port = 7474
	}
	return fmt.Sprintf("http://%s:%d", host, port)
}

// authHeader returns the HTTP Basic Authorization header value.
func (c *Client) authHeader() string {
	user := c.cfg.User
	if user == "" {
		user = "neo4j"
	}
	pass := c.cfg.Password
	if pass == "" {
		pass = "neo4j"
	}
	cred := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
	return "Basic " + cred
}

// cypherRequest is the wire payload for POST /db/{db}/tx/commit.
type cypherRequest struct {
	Statements []cypherStatement `json:"statements"`
}

type cypherStatement struct {
	Statement  string         `json:"statement"`
	Parameters map[string]any `json:"parameters,omitempty"`
}

// cypherResponse is the wire payload returned by the API.
type cypherResponse struct {
	Results []cypherResult `json:"results"`
	Errors  []cypherError  `json:"errors"`
}

type cypherResult struct {
	Columns []string        `json:"columns"`
	Data    []cypherDataRow `json:"data"`
}

type cypherDataRow struct {
	Row []json.RawMessage `json:"row"`
}

type cypherError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// runCypher executes a single Cypher statement and returns the rows.
// Each row is a slice of json.RawMessage matching the columns order.
// 当 statements 无返回行时返回空 slice（len==0），不返回 nil。
func (c *Client) runCypher(ctx context.Context, stmt string, params map[string]any) ([][]json.RawMessage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	body, err := json.Marshal(cypherRequest{
		Statements: []cypherStatement{{Statement: stmt, Parameters: params}},
	})
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "neo4j: marshal request", err)
	}
	url := c.baseURL() + "/db/neo4j/tx/commit"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, errno.Wrap(errno.InternalError, "neo4j: build request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", c.authHeader())
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpCli.Do(httpReq)
	if err != nil {
		return nil, errno.Wrap(errno.DependencyUnavailable, "neo4j: call api", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errno.Wrap(errno.DependencyUnavailable, "neo4j: read body", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errno.New(errno.DependencyUnavailable,
			fmt.Sprintf("neo4j: http %d: %s", resp.StatusCode, string(respBody)))
	}
	var parsed cypherResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, errno.Wrap(errno.InternalError, "neo4j: unmarshal response", err)
	}
	if len(parsed.Errors) > 0 {
		e := parsed.Errors[0]
		return nil, errno.New(errno.DependencyUnavailable,
			fmt.Sprintf("neo4j: %s: %s", e.Code, e.Message))
	}
	if len(parsed.Results) == 0 {
		return nil, nil
	}
	rows := make([][]json.RawMessage, 0, len(parsed.Results[0].Data))
	for _, d := range parsed.Results[0].Data {
		rows = append(rows, d.Row)
	}
	return rows, nil
}

// runMultiCypher executes multiple statements in one transaction.
// 用于 EnsureConstraints 这类需要批量 DDL 的场景。
func (c *Client) runMultiCypher(ctx context.Context, stmts []cypherStatement) error {
	if ctx == nil {
		ctx = context.Background()
	}
	body, err := json.Marshal(cypherRequest{Statements: stmts})
	if err != nil {
		return errno.Wrap(errno.InternalError, "neo4j: marshal request", err)
	}
	url := c.baseURL() + "/db/neo4j/tx/commit"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return errno.Wrap(errno.InternalError, "neo4j: build request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", c.authHeader())
	resp, err := c.httpCli.Do(httpReq)
	if err != nil {
		return errno.Wrap(errno.DependencyUnavailable, "neo4j: call api", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return errno.Wrap(errno.DependencyUnavailable, "neo4j: read body", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errno.New(errno.DependencyUnavailable,
			fmt.Sprintf("neo4j: http %d: %s", resp.StatusCode, string(respBody)))
	}
	var parsed cypherResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return errno.Wrap(errno.InternalError, "neo4j: unmarshal response", err)
	}
	if len(parsed.Errors) > 0 {
		e := parsed.Errors[0]
		return errno.New(errno.DependencyUnavailable,
			fmt.Sprintf("neo4j: %s: %s", e.Code, e.Message))
	}
	return nil
}

// parseNode parses a Neo4j node row (properties map + labels list) into GraphNodeView.
// row[0] = node properties map, row[1] = labels list (e.g. ["Person"]).
func parseNode(row []json.RawMessage) (entity.GraphNodeView, bool) {
	if len(row) == 0 {
		return entity.GraphNodeView{}, false
	}
	var props map[string]any
	if err := json.Unmarshal(row[0], &props); err != nil {
		return entity.GraphNodeView{}, false
	}
	uid, _ := props["uid"].(string)
	name, _ := props["name"].(string)
	label := ""
	if len(row) > 1 {
		var labels []string
		if err := json.Unmarshal(row[1], &labels); err == nil && len(labels) > 0 {
			label = labels[0]
		}
	}
	if label == "" {
		if v, ok := props["label"].(string); ok {
			label = v
		}
	}
	return nodeFromMap(uid, label, name, props), true
}

// parseNodeList parses a column that is an array of node property maps.
func parseNodeList(raw json.RawMessage) []entity.GraphNodeView {
	var arr []map[string]any
	if err := json.Unmarshal(raw, &arr); err != nil || len(arr) == 0 {
		return nil
	}
	out := make([]entity.GraphNodeView, 0, len(arr))
	for _, props := range arr {
		uid, _ := props["uid"].(string)
		name, _ := props["name"].(string)
		label, _ := props["label"].(string)
		out = append(out, nodeFromMap(uid, label, name, props))
	}
	return out
}

// parseEdge parses a Neo4j relationship row.
// row[0] = rel properties, row[1] = type string, row[2] = source uid, row[3] = target uid.
func parseEdge(row []json.RawMessage) (entity.GraphEdgeView, bool) {
	if len(row) == 0 {
		return entity.GraphEdgeView{}, false
	}
	var props map[string]any
	if err := json.Unmarshal(row[0], &props); err != nil {
		return entity.GraphEdgeView{}, false
	}
	uid, _ := props["uid"].(string)
	relType := ""
	if len(row) > 1 {
		_ = json.Unmarshal(row[1], &relType)
	}
	srcUID, _ := props["source_uid"].(string)
	tgtUID, _ := props["target_uid"].(string)
	if len(row) > 2 {
		_ = json.Unmarshal(row[2], &srcUID)
	}
	if len(row) > 3 {
		_ = json.Unmarshal(row[3], &tgtUID)
	}
	return entity.GraphEdgeView{
		UID:        uid,
		Type:       relType,
		SourceUID:  srcUID,
		TargetUID:  tgtUID,
		Properties: normaliseProps(props),
	}, true
}

// parseEdgeList parses a column that is an array of relationship property maps.
// 注意：relationships(p) 只返回属性，不含 type/source/target，需另行查询或推断。
// 这里从 properties.source_uid / target_uid 兜底（如写入时已存）。
func parseEdgeList(raw json.RawMessage, types []string, srcUIDs, tgtUIDs []string) []entity.GraphEdgeView {
	var arr []map[string]any
	if err := json.Unmarshal(raw, &arr); err != nil || len(arr) == 0 {
		return nil
	}
	out := make([]entity.GraphEdgeView, 0, len(arr))
	for i, props := range arr {
		uid, _ := props["uid"].(string)
		relType := ""
		if i < len(types) {
			relType = types[i]
		}
		src := ""
		tgt := ""
		if i < len(srcUIDs) {
			src = srcUIDs[i]
		}
		if i < len(tgtUIDs) {
			tgt = tgtUIDs[i]
		}
		if v, ok := props["source_uid"].(string); ok {
			src = v
		}
		if v, ok := props["target_uid"].(string); ok {
			tgt = v
		}
		out = append(out, entity.GraphEdgeView{
			UID:        uid,
			Type:       relType,
			SourceUID:  src,
			TargetUID:  tgt,
			Properties: normaliseProps(props),
		})
	}
	return out
}

// parseStringList parses a column that is an array of strings.
func parseStringList(raw json.RawMessage) []string {
	var arr []string
	_ = json.Unmarshal(raw, &arr)
	return arr
}

// httpEnsureConstraints 建立 8 类节点唯一约束 + B-Tree 索引（doc/05 §5.8）。
func (c *Client) httpEnsureConstraints(ctx context.Context) error {
	var stmts []cypherStatement
	for _, label := range entity.NodeLabels {
		stmts = append(stmts, cypherStatement{
			Statement: fmt.Sprintf("CREATE CONSTRAINT IF NOT EXISTS FOR (n:%s) REQUIRE n.uid IS UNIQUE", label),
		})
	}
	// 高频查询字段索引。
	for _, label := range entity.NodeLabels {
		stmts = append(stmts, cypherStatement{
			Statement: fmt.Sprintf("CREATE INDEX IF NOT EXISTS FOR (n:%s) ON (n.name)", label),
		})
	}
	// 关系唯一约束（Neo4j 5+ 支持，4.x 跳过失败不阻断）。
	for _, rel := range entity.EdgeTypes {
		stmts = append(stmts, cypherStatement{
			Statement: fmt.Sprintf("CREATE CONSTRAINT IF NOT EXISTS FOR ()-[r:%s]-() REQUIRE r.uid IS UNIQUE", rel),
		})
	}
	return c.runMultiCypher(ctx, stmts)
}

// httpUpsertNode MERGEs a node by uid.
func (c *Client) httpUpsertNode(ctx context.Context, n service.NodePayload) error {
	if n.Label == "" {
		return errno.New(errno.InvalidParams, "node label is required")
	}
	stmt := fmt.Sprintf("MERGE (n:%s {uid:$uid}) SET n.name=$name, n += $props", n.Label)
	_, err := c.runCypher(ctx, stmt, map[string]any{
		"uid":   n.UID,
		"name":  n.Name,
		"props": n.Properties,
	})
	return err
}

// httpGetNode fetches a node by uid.
func (c *Client) httpGetNode(ctx context.Context, uid string) (*entity.GraphNodeView, error) {
	rows, err := c.runCypher(ctx,
		"MATCH (n {uid:$uid}) RETURN n, labels(n) LIMIT 1",
		map[string]any{"uid": uid})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	node, ok := parseNode(rows[0])
	if !ok {
		return nil, nil
	}
	return &node, nil
}

// httpDeleteNode removes a node and its edges (DETACH DELETE).
func (c *Client) httpDeleteNode(ctx context.Context, uid string) error {
	_, err := c.runCypher(ctx,
		"MATCH (n {uid:$uid}) DETACH DELETE n",
		map[string]any{"uid": uid})
	return err
}

// httpUpsertEdge MERGEs a relationship by uid.
func (c *Client) httpUpsertEdge(ctx context.Context, e service.EdgePayload) error {
	if e.Type == "" {
		return errno.New(errno.InvalidParams, "edge type is required")
	}
	stmt := fmt.Sprintf(
		"MATCH (a {uid:$src}), (b {uid:$tgt}) "+
			"MERGE (a)-[r:%s {uid:$uid}]->(b) SET r += $props", e.Type)
	_, err := c.runCypher(ctx, stmt, map[string]any{
		"src":   e.SourceUID,
		"tgt":   e.TargetUID,
		"uid":   e.UID,
		"props": e.Properties,
	})
	return err
}

// httpGetEdge fetches a relationship by uid.
func (c *Client) httpGetEdge(ctx context.Context, uid string) (*entity.GraphEdgeView, error) {
	rows, err := c.runCypher(ctx,
		"MATCH (a)-[r {uid:$uid}]->(b) RETURN r, type(r), a.uid, b.uid LIMIT 1",
		map[string]any{"uid": uid})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	edge, ok := parseEdge(rows[0])
	if !ok {
		return nil, nil
	}
	return &edge, nil
}

// httpDeleteEdge removes a relationship by uid.
func (c *Client) httpDeleteEdge(ctx context.Context, uid string) error {
	_, err := c.runCypher(ctx,
		"MATCH ()-[r {uid:$uid}]->() DELETE r",
		map[string]any{"uid": uid})
	return err
}

// httpQueryPath returns the shortest path between two nodes.
// Neo4j 不支持参数化变长深度，maxHops 用 fmt.Sprintf 注入（int 安全）。
func (c *Client) httpQueryPath(ctx context.Context, startUID, endUID string, maxHops int) (*entity.GraphPath, error) {
	if maxHops <= 0 {
		maxHops = 8
	}
	stmt := fmt.Sprintf(
		"MATCH p = shortestPath((a {uid:$start})-[*..%d]-(b {uid:$end})) "+
			"RETURN nodes(p) AS nodes, relationships(p) AS rels LIMIT 1", maxHops)
	rows, err := c.runCypher(ctx, stmt, map[string]any{
		"start": startUID,
		"end":   endUID,
	})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	row := rows[0]
	if len(row) < 2 {
		return nil, nil
	}
	nodes := parseNodeList(row[0])
	// relationships(p) 返回属性 map 数组；type/source/target 需额外查询。
	// 这里用 [r IN relationships(p) | type(r)] 同列返回，但为简化用 properties 兜底。
	edges := parseEdgeList(row[1], nil, nil, nil)
	if len(nodes) == 0 {
		return nil, nil
	}
	return &entity.GraphPath{
		Nodes: nodes,
		Edges: edges,
		Hops:  len(nodes) - 1,
	}, nil
}

// httpGetSubgraph returns the subgraph centred on centerUID.
func (c *Client) httpGetSubgraph(ctx context.Context, centerUID string, depth, limit int) (*entity.Subgraph, error) {
	if depth <= 0 {
		depth = 2
	}
	if limit <= 0 {
		limit = 100
	}
	stmt := fmt.Sprintf(
		"MATCH p=(n {uid:$center})-[*1..%d]-(m) "+
			"WITH nodes(p) AS ns, relationships(p) AS rs "+
			"UNWIND ns AS node WITH collect(DISTINCT node) AS nodes, rs "+
			"RETURN nodes, collect(DISTINCT rs)[0] AS rels LIMIT $limit", depth)
	rows, err := c.runCypher(ctx, stmt, map[string]any{
		"center": centerUID,
		"limit":  limit,
	})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return &entity.Subgraph{}, nil
	}
	row := rows[0]
	out := &entity.Subgraph{}
	if len(row) > 0 {
		// nodes 可能是单个 node 或 node 数组，尝试数组解析。
		out.Nodes = parseNodeList(row[0])
	}
	if len(row) > 1 {
		out.Edges = parseEdgeList(row[1], nil, nil, nil)
	}
	return out, nil
}

// httpGetPersonWorks returns classics authored by a person.
func (c *Client) httpGetPersonWorks(ctx context.Context, personUID string) ([]entity.GraphNodeView, error) {
	rows, err := c.runCypher(ctx,
		"MATCH (p:Person {uid:$uid})-[:AUTHORED]->(c) RETURN c, labels(c)",
		map[string]any{"uid": personUID})
	if err != nil {
		return nil, err
	}
	out := make([]entity.GraphNodeView, 0, len(rows))
	for _, row := range rows {
		if node, ok := parseNode(row); ok {
			out = append(out, node)
		}
	}
	return out, nil
}

// httpGetSchoolLineage returns the discipled lineage of a school.
// 两步查询：先找 School 节点，再遍历 BELONGS_TO + DISCIPLED。
func (c *Client) httpGetSchoolLineage(ctx context.Context, schoolName string, maxDepth int) (*entity.LineagePath, error) {
	if maxDepth <= 0 {
		maxDepth = 6
	}
	// 1. 找到 School 节点 + 直接成员。
	rows, err := c.runCypher(ctx,
		"MATCH (s:School) WHERE s.name=$name OR s.name CONTAINS $name "+
			"MATCH (p:Person)-[:BELONGS_TO]->(s) "+
			"RETURN p, labels(p) LIMIT 50",
		map[string]any{"name": schoolName})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	members := make([]entity.GraphNodeView, 0, len(rows))
	for _, row := range rows {
		if node, ok := parseNode(row); ok {
			members = append(members, node)
		}
	}
	// 2. 对每个成员沿 DISCIPLED 链向上追溯 maxDepth 代。
	pathNodes := make([]entity.GraphNodeView, 0)
	generations := make([]int, 0)
	pathEdges := make([]entity.GraphEdgeView, 0)
	for _, m := range members {
		stmt := fmt.Sprintf(
			"MATCH path=(p:Person {uid:$uid})-[:DISCIPLED*1..%d]->(master) "+
				"RETURN nodes(path) AS nodes, relationships(path) AS rels LIMIT 1", maxDepth)
		chainRows, err := c.runCypher(ctx, stmt, map[string]any{"uid": m.UID})
		if err != nil || len(chainRows) == 0 {
			pathNodes = append(pathNodes, m)
			generations = append(generations, 0)
			continue
		}
		chainNodes := parseNodeList(chainRows[0][0])
		if len(chainNodes) == 0 {
			pathNodes = append(pathNodes, m)
			generations = append(generations, 0)
			continue
		}
		for i, n := range chainNodes {
			pathNodes = append(pathNodes, n)
			generations = append(generations, i)
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

// httpGetDynastyFigures returns representative figures of a dynasty with works.
func (c *Client) httpGetDynastyFigures(ctx context.Context, dynastyName string) ([]entity.FigureWithWorks, error) {
	rows, err := c.runCypher(ctx,
		"MATCH (d:Dynasty {name:$name})<-[:OCCURRED_IN]-(p:Person) "+
			"OPTIONAL MATCH (p)-[:AUTHORED]->(w) "+
			"OPTIONAL MATCH (p)-[:BELONGS_TO]->(s) "+
			"RETURN p, labels(p), collect(DISTINCT w) AS works, collect(DISTINCT s) AS schools",
		map[string]any{"name": dynastyName})
	if err != nil {
		return nil, err
	}
	out := make([]entity.FigureWithWorks, 0, len(rows))
	for _, row := range rows {
		person, ok := parseNode(row)
		if !ok {
			continue
		}
		var works []entity.GraphNodeView
		var schools []entity.GraphNodeView
		if len(row) > 2 {
			works = parseNodeList(row[2])
		}
		if len(row) > 3 {
			schools = parseNodeList(row[3])
		}
		out = append(out, entity.FigureWithWorks{
			Person:  person,
			Works:   works,
			Schools: schools,
		})
	}
	return out, nil
}

// httpGetPrescriptionDetail returns the medicines and diseases of a prescription.
func (c *Client) httpGetPrescriptionDetail(ctx context.Context, prescriptionUID string) (*entity.PrescriptionGraph, error) {
	rows, err := c.runCypher(ctx,
		"MATCH (rx:Prescription {uid:$uid}) "+
			"OPTIONAL MATCH (rx)-[:COMPOSED_OF]->(m) "+
			"OPTIONAL MATCH (rx)-[:TREATS]->(d) "+
			"RETURN rx, labels(rx), collect(DISTINCT m) AS medicines, collect(DISTINCT d) AS diseases",
		map[string]any{"uid": prescriptionUID})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	row := rows[0]
	rx, ok := parseNode(row)
	if !ok {
		return nil, nil
	}
	var medicines []entity.GraphNodeView
	var diseases []entity.GraphNodeView
	if len(row) > 2 {
		medicines = parseNodeList(row[2])
	}
	if len(row) > 3 {
		diseases = parseNodeList(row[3])
	}
	return &entity.PrescriptionGraph{
		Prescription: rx,
		Medicines:    medicines,
		Diseases:     diseases,
	}, nil
}

// httpSearchNodes runs a keyword search over nodes.
func (c *Client) httpSearchNodes(ctx context.Context, keyword, label string, limit int) ([]entity.GraphNodeView, error) {
	if limit <= 0 {
		limit = 20
	}
	kw := strings.ToLower(keyword)
	var stmt string
	params := map[string]any{"kw": "%" + kw + "%", "limit": limit}
	if label != "" {
		stmt = fmt.Sprintf("MATCH (n:%s) WHERE toLower(n.name) CONTAINS toLower($kw) "+
			"RETURN n, labels(n) LIMIT $limit", label)
	} else {
		stmt = "MATCH (n) WHERE toLower(n.name) CONTAINS toLower($kw) " +
			"RETURN n, labels(n) LIMIT $limit"
	}
	rows, err := c.runCypher(ctx, stmt, params)
	if err != nil {
		return nil, err
	}
	out := make([]entity.GraphNodeView, 0, len(rows))
	for _, row := range rows {
		if node, ok := parseNode(row); ok {
			out = append(out, node)
		}
	}
	return out, nil
}
