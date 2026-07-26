// Graph API 模块：节点/关系 CRUD + 复杂图查询 + 同步触发。
// 端点对齐 backend/graph-service/internal/controller/router.go：/api/v1/graph/*。
//
// 注意：本模块未在 src/index.ts 中注册，需调用方自行 new GraphApi(http) 装配，
// 或在 index.ts 升级时再统一接入（避免改动既有导出表面）。

import type { AxiosInstance } from 'axios';

import { buildQuery, type ListResponse, type PageParams } from '../types';
import type {
  GraphEdgeListParams,
  GraphEdgeRequest,
  GraphEdgeResponse,
  GraphFigureWithWorksResponse,
  GraphLineageResponse,
  GraphNodeListParams,
  GraphNodeRequest,
  GraphNodeResponse,
  GraphNodeView,
  GraphPath,
  GraphPrescriptionDetailResponse,
  GraphSchoolLineageParams,
  GraphSearchParams,
  GraphSearchResponse,
  GraphShortestPathParams,
  GraphSubgraph,
  GraphSubgraphParams,
  GraphSyncParams,
  GraphSyncResponse,
} from './graph-types';

export class GraphApi {
  constructor(private http: AxiosInstance) {}

  // ---- Nodes ----

  /** GET /api/v1/graph/nodes?label=&keyword=&page=&page_size= */
  listNodes(params?: PageParams & GraphNodeListParams): Promise<ListResponse<GraphNodeResponse>> {
    return this.http.get('/api/v1/graph/nodes', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<GraphNodeResponse>>;
  }

  /** POST /api/v1/graph/nodes （MERGE 语义，按 uid 去重） */
  createNode(payload: GraphNodeRequest): Promise<GraphNodeResponse> {
    return this.http.post('/api/v1/graph/nodes', payload) as unknown as Promise<GraphNodeResponse>;
  }

  /** GET /api/v1/graph/nodes/:uid */
  getNode(uid: string): Promise<GraphNodeResponse> {
    return this.http.get(
      `/api/v1/graph/nodes/${encodeURIComponent(uid)}`,
    ) as unknown as Promise<GraphNodeResponse>;
  }

  /** PUT /api/v1/graph/nodes/:uid */
  updateNode(uid: string, payload: GraphNodeRequest): Promise<GraphNodeResponse> {
    return this.http.put(
      `/api/v1/graph/nodes/${encodeURIComponent(uid)}`,
      payload,
    ) as unknown as Promise<GraphNodeResponse>;
  }

  /** DELETE /api/v1/graph/nodes/:uid */
  deleteNode(uid: string): Promise<void> {
    return this.http.delete(
      `/api/v1/graph/nodes/${encodeURIComponent(uid)}`,
    ) as unknown as Promise<void>;
  }

  // ---- Edges ----

  /** GET /api/v1/graph/edges?source_uid=&target_uid=&type=&page=&page_size= */
  listEdges(params?: PageParams & GraphEdgeListParams): Promise<ListResponse<GraphEdgeResponse>> {
    return this.http.get('/api/v1/graph/edges', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<GraphEdgeResponse>>;
  }

  /** POST /api/v1/graph/edges （MERGE 语义，按 uid 去重） */
  createEdge(payload: GraphEdgeRequest): Promise<GraphEdgeResponse> {
    return this.http.post('/api/v1/graph/edges', payload) as unknown as Promise<GraphEdgeResponse>;
  }

  /** GET /api/v1/graph/edges/:uid */
  getEdge(uid: string): Promise<GraphEdgeResponse> {
    return this.http.get(
      `/api/v1/graph/edges/${encodeURIComponent(uid)}`,
    ) as unknown as Promise<GraphEdgeResponse>;
  }

  /** PUT /api/v1/graph/edges/:uid */
  updateEdge(uid: string, payload: GraphEdgeRequest): Promise<GraphEdgeResponse> {
    return this.http.put(
      `/api/v1/graph/edges/${encodeURIComponent(uid)}`,
      payload,
    ) as unknown as Promise<GraphEdgeResponse>;
  }

  /** DELETE /api/v1/graph/edges/:uid */
  deleteEdge(uid: string): Promise<void> {
    return this.http.delete(
      `/api/v1/graph/edges/${encodeURIComponent(uid)}`,
    ) as unknown as Promise<void>;
  }

  // ---- 复杂查询（doc/05 §5.5）----

  /** GET /api/v1/graph/persons/:uid/works —— 人物著作（doc/05 §5.5.1） */
  getPersonWorks(uid: string): Promise<GraphNodeView[]> {
    return this.http.get(
      `/api/v1/graph/persons/${encodeURIComponent(uid)}/works`,
    ) as unknown as Promise<GraphNodeView[]>;
  }

  /** GET /api/v1/graph/schools/:name/lineage?max_depth= —— 学派师承链（doc/05 §5.5.2） */
  getSchoolLineage(name: string, params?: GraphSchoolLineageParams): Promise<GraphLineageResponse> {
    return this.http.get(`/api/v1/graph/schools/${encodeURIComponent(name)}/lineage`, {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<GraphLineageResponse>;
  }

  /** GET /api/v1/graph/paths/shortest?start_uid=&end_uid=&max_hops= —— 最短路径（doc/05 §5.5.3） */
  findShortestPath(params: GraphShortestPathParams): Promise<GraphPath> {
    return this.http.get('/api/v1/graph/paths/shortest', {
      params: buildQuery(params),
    }) as unknown as Promise<GraphPath>;
  }

  /** GET /api/v1/graph/dynasties/:name/figures —— 朝代代表人物与著作（doc/05 §5.5.4） */
  getDynastyFigures(name: string): Promise<GraphFigureWithWorksResponse[]> {
    return this.http.get(
      `/api/v1/graph/dynasties/${encodeURIComponent(name)}/figures`,
    ) as unknown as Promise<GraphFigureWithWorksResponse[]>;
  }

  /** GET /api/v1/graph/prescriptions/:uid/detail —— 方剂全貌（doc/05 §5.5.5） */
  getPrescriptionDetail(uid: string): Promise<GraphPrescriptionDetailResponse> {
    return this.http.get(
      `/api/v1/graph/prescriptions/${encodeURIComponent(uid)}/detail`,
    ) as unknown as Promise<GraphPrescriptionDetailResponse>;
  }

  /** GET /api/v1/graph/subgraph?center_uid=&depth=&limit= —— 子图可视化（doc/05 §5.9） */
  getSubgraph(params: GraphSubgraphParams): Promise<GraphSubgraph> {
    return this.http.get('/api/v1/graph/subgraph', {
      params: buildQuery(params),
    }) as unknown as Promise<GraphSubgraph>;
  }

  /** GET /api/v1/graph/search?keyword=&label=&limit= —— 节点全文检索（doc/05 §5.8.3） */
  searchNodes(params: GraphSearchParams): Promise<GraphSearchResponse> {
    return this.http.get('/api/v1/graph/search', {
      params: buildQuery(params),
    }) as unknown as Promise<GraphSearchResponse>;
  }

  // ---- 同步 ----

  /** POST /api/v1/graph/sync?limit= —— 手动触发 ETL 重试（doc/05 §5.6） */
  triggerSync(params?: GraphSyncParams): Promise<GraphSyncResponse> {
    return this.http.post('/api/v1/graph/sync', undefined, {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<GraphSyncResponse>;
  }
}
