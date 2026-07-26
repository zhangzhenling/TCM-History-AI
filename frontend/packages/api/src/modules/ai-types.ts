// AI Service 类型定义，对齐 backend/ai-service/internal/application/dto/ai.go。
// 字段命名与后端 json tag 完全一致（snake_case）。

/** ChatRequest POST /api/v1/ai/chat */
export interface ChatRequest {
  conversation_id?: number;
  user_id?: number;
  mode?: 'chat' | 'agent' | 'reasoning';
  message: string;
  variables?: Record<string, unknown>;
  template_scene?: 'chat' | 'agent' | 'reasoning' | 'summarize';
}

/** ChatResponse */
export interface ChatResponse {
  conversation_id: number;
  message_id: number;
  assistant: string;
  model: string;
  tokens_prompt: number;
  tokens_completion: number;
  latency_ms: number;
  metadata?: Record<string, unknown>;
}

/** AgentRequest POST /api/v1/ai/agents/run */
export interface AgentRequest {
  conversation_id?: number;
  user_id?: number;
  question: string;
  variables?: Record<string, unknown>;
}

/** AgentStep 单个 Agent 步骤 */
export interface AgentStep {
  sub_task_id: string;
  intent_type?: string;
  channel?: 'rag' | 'graph' | 'tool' | 'direct';
  query?: string;
  tool_name?: string;
  result?: Record<string, unknown>;
}

/** AgentResponse */
export interface AgentResponse {
  agent_run_id: number;
  conversation_id: number;
  answer: string;
  steps?: AgentStep[];
  total_tokens: number;
  total_latency_ms: number;
  status: 'pending' | 'running' | 'done' | 'failed';
}

/** PromptTemplateRequest */
export interface PromptTemplateRequest {
  name: string;
  scene: 'chat' | 'agent' | 'reasoning' | 'summarize';
  system_prompt: string;
  template?: string;
  variables_json?: unknown;
  model?: string;
  temperature?: number;
  max_tokens?: number;
  top_p?: number;
  is_active?: boolean;
  version?: number;
}

/** PromptTemplate 响应 */
export interface PromptTemplate {
  id: number;
  name: string;
  scene: 'chat' | 'agent' | 'reasoning' | 'summarize';
  system_prompt: string;
  template: string;
  variables_json: unknown;
  model: string;
  temperature: number;
  max_tokens: number;
  top_p: number;
  is_active: boolean;
  version: number;
  created_at: string;
  updated_at: string;
}

/** ToolRequest */
export interface ToolRequest {
  name: string;
  description?: string;
  endpoint?: string;
  method?: 'GET' | 'POST';
  parameters_json?: unknown;
  category?: string;
  is_enabled?: boolean;
  version?: string;
}

/** Tool 响应 */
export interface Tool {
  id: number;
  name: string;
  description: string;
  endpoint: string;
  method: 'GET' | 'POST';
  parameters_json: unknown;
  category: string;
  is_enabled: boolean;
  version: string;
  created_at: string;
  updated_at: string;
}

/** ToolExecuteRequest */
export interface ToolExecuteRequest {
  params?: Record<string, unknown>;
}

/** ToolExecuteResponse */
export interface ToolExecuteResponse {
  tool_name: string;
  result: Record<string, unknown>;
}

/** Message 响应（ai_messages 行） */
export interface Message {
  id: number;
  conversation_id: number;
  role: 'user' | 'assistant' | 'system' | 'tool';
  content: string;
  tool_calls_json?: unknown;
  tool_call_id?: string;
  tokens_prompt: number;
  tokens_completion: number;
  latency_ms: number;
  model_name: string;
  created_at: string;
}

/** Conversation 响应（ai_conversations 行） */
export interface Conversation {
  id: number;
  user_id: number;
  title: string;
  mode: 'chat' | 'agent' | 'reasoning';
  status: 'active' | 'archived';
  message_count: number;
  metadata_json: Record<string, unknown> | null;
  created_at: string;
  updated_at: string;
}

/** AgentRun 响应（ai_agent_runs 行） */
export interface AgentRun {
  id: number;
  conversation_id: number;
  user_id: number;
  plan_json?: unknown;
  steps_json?: unknown;
  final_answer: string;
  status: 'pending' | 'running' | 'done' | 'failed';
  error_msg?: string;
  total_tokens: number;
  total_latency_ms: number;
  created_at: string;
  updated_at: string;
}
