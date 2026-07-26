// Knowledge API 模块：文档/切片/任务 CRUD + RAG 检索 + 反馈。
// 端点对齐 backend/knowledge-service/internal/controller/router.go：/api/v1/knowledge/*。

import type { AxiosInstance } from 'axios';

import { buildQuery, type ListResponse, type PageParams } from '../types';
import type {
  Document,
  DocumentRequest,
  DocumentChunk,
  EmbeddingTask,
  FeedbackRequest,
  RetrieveRequest,
  RetrieveResponse,
} from './knowledge-types';

export class KnowledgeApi {
  constructor(private http: AxiosInstance) {}

  // ---- Documents ----
  listDocuments(params?: PageParams & { classic_code?: string }): Promise<ListResponse<Document>> {
    return this.http.get('/api/v1/knowledge/documents', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<Document>>;
  }
  getDocument(id: number | string): Promise<Document> {
    return this.http.get(`/api/v1/knowledge/documents/${id}`) as unknown as Promise<Document>;
  }
  createDocument(payload: DocumentRequest): Promise<Document> {
    return this.http.post('/api/v1/knowledge/documents', payload) as unknown as Promise<Document>;
  }
  updateDocument(id: number | string, payload: DocumentRequest): Promise<Document> {
    return this.http.put(
      `/api/v1/knowledge/documents/${id}`,
      payload,
    ) as unknown as Promise<Document>;
  }
  deleteDocument(id: number | string): Promise<void> {
    return this.http.delete(`/api/v1/knowledge/documents/${id}`) as unknown as Promise<void>;
  }

  // ---- Chunks ----
  listChunksByDocument(
    documentId: number | string,
    params?: PageParams,
  ): Promise<ListResponse<DocumentChunk>> {
    return this.http.get(`/api/v1/knowledge/documents/${documentId}/chunks`, {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<DocumentChunk>>;
  }
  getChunk(id: number | string): Promise<DocumentChunk> {
    return this.http.get(`/api/v1/knowledge/chunks/${id}`) as unknown as Promise<DocumentChunk>;
  }

  // ---- Embedding Tasks ----
  listTasks(params?: PageParams & { status?: string }): Promise<ListResponse<EmbeddingTask>> {
    return this.http.get('/api/v1/knowledge/tasks', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<EmbeddingTask>>;
  }
  getTask(id: number | string): Promise<EmbeddingTask> {
    return this.http.get(`/api/v1/knowledge/tasks/${id}`) as unknown as Promise<EmbeddingTask>;
  }
  listTasksByDocument(documentId: number | string): Promise<EmbeddingTask[]> {
    return this.http.get(`/api/v1/knowledge/documents/${documentId}/tasks`) as unknown as Promise<
      EmbeddingTask[]
    >;
  }

  // ---- RAG Retrieval ----
  retrieve(payload: RetrieveRequest): Promise<RetrieveResponse> {
    return this.http.post(
      '/api/v1/knowledge/retrieve',
      payload,
    ) as unknown as Promise<RetrieveResponse>;
  }
  submitFeedback(queryLogId: number | string, payload: FeedbackRequest): Promise<void> {
    return this.http.post(
      `/api/v1/knowledge/queries/${queryLogId}/feedback`,
      payload,
    ) as unknown as Promise<void>;
  }
}
