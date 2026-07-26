// Knowledge Service 类型定义，对齐 backend/knowledge-service/internal/application/dto。
// 字段命名与后端 json tag 完全一致（snake_case）。

export interface Document {
  id: number;
  classic_code: string;
  title: string;
  version: string;
  dynasty: string;
  school: string;
  author: string;
  source_type: string;
  source_ref: string;
  file_url: string;
  pdf_object_key: string;
  markdown_object_key: string;
  mime_type: string;
  content_hash: string;
  status: string;
  chunk_count: number;
  volume_count: number;
  clause_count: number;
  metadata_json: Record<string, unknown> | null;
  created_at: string;
  updated_at: string;
}

export interface DocumentRequest {
  classic_code: string;
  title: string;
  version?: string;
  dynasty?: string;
  school?: string;
  author?: string;
  source_type?: string;
  source_ref?: string;
  file_url?: string;
  pdf_object_key?: string;
  markdown_object_key?: string;
  mime_type?: string;
  content_hash?: string;
  volume_count?: number;
  clause_count?: number;
  metadata_json?: Record<string, unknown>;
}

export interface DocumentChunk {
  id: number;
  document_id: number;
  chunk_id: string;
  chunk_index: number;
  classic_code: string;
  volume: string;
  clause_no: number;
  content_type: string;
  content: string;
  text_original: string;
  text_translation: string;
  token_count: number;
  embedding_id: string;
  embedding_model: string;
}

export interface EmbeddingTask {
  id: number;
  document_id: number;
  chunk_id: number;
  task_type: string;
  stage: string;
  status: string;
  progress: number;
  model: string;
  chunk_count: number;
  vector_count: number;
  error_message?: string;
  retry_count: number;
  started_at?: string;
  finished_at?: string;
  created_at: string;
  updated_at: string;
}

export interface RetrieveRequest {
  query: string;
  topk?: number;
  classic_codes?: string[];
  dynasties?: string[];
  schools?: string[];
  content_types?: string[];
  session_id?: string;
}

export interface RetrievedChunk {
  chunk_id: string;
  document_id: number;
  classic_code: string;
  volume: string;
  clause_no: number;
  content_type: string;
  content: string;
  text_original: string;
  text_translation: string;
  score: number;
  source: string;
}

export interface RetrieveResponse {
  query: string;
  topk: number;
  latency_ms: number;
  total: number;
  chunks: RetrievedChunk[];
  query_log_id?: number;
}

export interface FeedbackRequest {
  feedback: 'good' | 'bad';
}
