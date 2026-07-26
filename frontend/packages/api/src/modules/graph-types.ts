// Graph Service 类型定义，对齐 backend/graph-service/internal/application/dto。
// 字段命名与后端 json tag 完全一致（snake_case）。
//
// 节点 Label 与关系 Type 的常量集与 backend/graph-service/internal/domain/entity
// 中的枚举保持同步，便于前端在调用 create/update 时做客户端校验。

// ---- 节点 Label 枚举（doc/05 §5.2）----
export type GraphNodeLabel =
  | 'Person' // 人物
  | 'Classic' // 经典（著作/理论）
  | 'School' // 学派
  | 'Prescription' // 方剂
  | 'Medicine' // 药物
  | 'Disease' // 疾病
  | 'Dynasty' // 朝代
  | 'HistoricalEvent'; // 历史事件

export const GRAPH_NODE_LABELS: readonly GraphNodeLabel[] = [
  'Person',
  'Classic',
  'School',
  'Prescription',
  'Medicine',
  'Disease',
  'Dynasty',
  'HistoricalEvent',
] as const;

// ---- 关系 Type 枚举（doc/05 §5.3）----
export type GraphEdgeType =
  | 'AUTHORED' // Person → Classic 著作
  | 'DISCIPLED' // Person → Person 师承（弟子→师父）
  | 'INFLUENCED' // 任意 → 任意 影响
  | 'BELONGS_TO' // Person/Classic/Prescription → School 属于
  | 'OCCURRED_IN' // Person/Classic/HistoricalEvent → Dynasty 发生于
  | 'CITED' // Classic → Classic 引用
  | 'PROPOSED' // Person/Classic → Classic(理论) 提出
  | 'OPPOSED' // Person/Classic → Person/Classic/School 反对
  | 'INHERITED'; // Person/School → Person/Classic/School 继承

export const GRAPH_EDGE_TYPES: readonly GraphEdgeType[] = [
  'AUTHORED',
  'DISCIPLED',
  'INFLUENCED',
  'BELONGS_TO',
  'OCCURRED_IN',
  'CITED',
  'PROPOSED',
  'OPPOSED',
  'INHERITED',
] as const;

// ---- 同步来源 / 动作 / 状态枚举（graph_sync_logs）----
export type GraphSyncSource = 'history' | 'knowledge';
export type GraphSyncAction = 'upsert' | 'delete';
export type GraphSyncStatus = 'pending' | 'done' | 'failed';

// ---- 节点 CRUD ----

/** POST/PUT /api/v1/graph/nodes 请求体，对应 dto.NodeRequest。 */
export interface GraphNodeRequest {
  uid: string;
  label: GraphNodeLabel;
  name: string;
  properties_json?: Record<string, unknown> | null;
}

/** 节点完整视图，对应 dto.NodeResponse。 */
export interface GraphNodeResponse {
  id: number;
  uid: string;
  label: GraphNodeLabel;
  name: string;
  properties_json: Record<string, unknown> | null;
  synced_at: string;
  created_at: string;
  updated_at: string;
}

// ---- 关系 CRUD ----

/** POST/PUT /api/v1/graph/edges 请求体，对应 dto.EdgeRequest。 */
export interface GraphEdgeRequest {
  uid: string;
  type: GraphEdgeType;
  source_uid: string;
  target_uid: string;
  properties_json?: Record<string, unknown> | null;
}

/** 关系完整视图，对应 dto.EdgeResponse。 */
export interface GraphEdgeResponse {
  id: number;
  uid: string;
  type: GraphEdgeType;
  source_uid: string;
  target_uid: string;
  properties_json: Record<string, unknown> | null;
  synced_at: string;
  created_at: string;
  updated_at: string;
}

// ---- 图查询轻量视图（dto.NodeView / dto.EdgeView）----

/** 图查询返回的节点轻量视图，仅包含 uid/label/name/properties。 */
export interface GraphNodeView {
  uid: string;
  label: GraphNodeLabel;
  name: string;
  properties: Record<string, unknown> | null;
}

/** 图查询返回的边轻量视图。 */
export interface GraphEdgeView {
  uid: string;
  type: GraphEdgeType;
  source_uid: string;
  target_uid: string;
  properties: Record<string, unknown> | null;
}

// ---- 复杂查询响应 ----

/** 最短路径 / 子图共用的图路径视图，对应 dto.GraphPath。 */
export interface GraphPath {
  nodes: GraphNodeView[];
  edges: GraphEdgeView[];
  hops: number;
}

/** 子图查询响应，对应 dto.Subgraph。 */
export interface GraphSubgraph {
  nodes: GraphNodeView[];
  edges: GraphEdgeView[];
}

/** 学派师承链查询响应，对应 dto.LineageResponse。 */
export interface GraphLineageResponse {
  path: GraphPath;
  generations: number[];
}

/** 朝代代表人物与著作聚合响应，对应 dto.FigureWithWorksResponse。 */
export interface GraphFigureWithWorksResponse {
  person: GraphNodeView;
  works: GraphNodeView[];
  schools: GraphNodeView[];
}

/** 方剂全貌查询响应，对应 dto.PrescriptionDetailResponse。 */
export interface GraphPrescriptionDetailResponse {
  prescription: GraphNodeView;
  medicines: GraphNodeView[];
  diseases: GraphNodeView[];
}

/** 节点检索响应，对应 dto.SearchResponse。 */
export interface GraphSearchResponse {
  keyword: string;
  label: string;
  total: number;
  items: GraphNodeView[];
}

/** 手动触发同步响应，对应 dto.SyncResponse。 */
export interface GraphSyncResponse {
  succeeded: number;
  failed: number;
  pending: number;
}

// ---- 查询参数 ----

/** GET /api/v1/graph/nodes 查询参数。 */
export interface GraphNodeListParams {
  label?: GraphNodeLabel | '';
  keyword?: string;
}

/** GET /api/v1/graph/edges 查询参数。 */
export interface GraphEdgeListParams {
  source_uid?: string;
  target_uid?: string;
  type?: GraphEdgeType | '';
}

/** GET /api/v1/graph/paths/shortest 查询参数。 */
export interface GraphShortestPathParams {
  start_uid: string;
  end_uid: string;
  max_hops?: number;
}

/** GET /api/v1/graph/subgraph 查询参数。 */
export interface GraphSubgraphParams {
  center_uid: string;
  depth?: number;
  limit?: number;
}

/** GET /api/v1/graph/search 查询参数。 */
export interface GraphSearchParams {
  keyword: string;
  label?: GraphNodeLabel | '';
  limit?: number;
}

/** GET /api/v1/graph/schools/:name/lineage 查询参数。 */
export interface GraphSchoolLineageParams {
  max_depth?: number;
}

/** POST /api/v1/graph/sync 查询参数。 */
export interface GraphSyncParams {
  limit?: number;
}
