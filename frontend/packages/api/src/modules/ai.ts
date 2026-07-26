// AI API 模块：对话 / Agent 运行 / Prompt 模板 CRUD / MCP Tool CRUD / 历史查询。
// 端点对齐 backend/ai-service/internal/controller/router.go：/api/v1/ai/*。

import type { AxiosInstance } from 'axios';

import { buildQuery, type ListResponse, type PageParams } from '../types';
import type {
  AgentRequest,
  AgentResponse,
  AgentRun,
  ChatRequest,
  ChatResponse,
  Conversation,
  Message,
  PromptTemplate,
  PromptTemplateRequest,
  Tool,
  ToolExecuteRequest,
  ToolExecuteResponse,
  ToolRequest,
} from './ai-types';

export class AiApi {
  constructor(private http: AxiosInstance) {}

  // ---- Chat / Conversations ----
  chat(payload: ChatRequest): Promise<ChatResponse> {
    return this.http.post('/api/v1/ai/chat', payload) as unknown as Promise<ChatResponse>;
  }
  listConversations(params?: PageParams): Promise<ListResponse<Conversation>> {
    return this.http.get('/api/v1/ai/conversations', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<Conversation>>;
  }
  getConversation(id: number | string): Promise<Conversation> {
    return this.http.get(`/api/v1/ai/conversations/${id}`) as unknown as Promise<Conversation>;
  }
  listMessages(
    conversationId: number | string,
    params?: PageParams,
  ): Promise<ListResponse<Message>> {
    return this.http.get(`/api/v1/ai/conversations/${conversationId}/messages`, {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<Message>>;
  }
  deleteConversation(id: number | string): Promise<void> {
    return this.http.delete(`/api/v1/ai/conversations/${id}`) as unknown as Promise<void>;
  }

  // ---- Agent runs ----
  runAgent(payload: AgentRequest): Promise<AgentResponse> {
    return this.http.post('/api/v1/ai/agents/run', payload) as unknown as Promise<AgentResponse>;
  }
  listAgentRuns(params?: PageParams): Promise<ListResponse<AgentRun>> {
    return this.http.get('/api/v1/ai/agent-runs', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<AgentRun>>;
  }
  getAgentRun(id: number | string): Promise<AgentRun> {
    return this.http.get(`/api/v1/ai/agent-runs/${id}`) as unknown as Promise<AgentRun>;
  }

  // ---- Prompt templates ----
  listPrompts(params?: PageParams & { scene?: string }): Promise<ListResponse<PromptTemplate>> {
    return this.http.get('/api/v1/ai/prompts', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<PromptTemplate>>;
  }
  createPrompt(payload: PromptTemplateRequest): Promise<PromptTemplate> {
    return this.http.post('/api/v1/ai/prompts', payload) as unknown as Promise<PromptTemplate>;
  }
  getPrompt(id: number | string): Promise<PromptTemplate> {
    return this.http.get(`/api/v1/ai/prompts/${id}`) as unknown as Promise<PromptTemplate>;
  }
  updatePrompt(id: number | string, payload: PromptTemplateRequest): Promise<PromptTemplate> {
    return this.http.put(`/api/v1/ai/prompts/${id}`, payload) as unknown as Promise<PromptTemplate>;
  }
  deletePrompt(id: number | string): Promise<void> {
    return this.http.delete(`/api/v1/ai/prompts/${id}`) as unknown as Promise<void>;
  }

  // ---- Tools (MCP) ----
  listTools(params?: PageParams & { enabled?: boolean }): Promise<ListResponse<Tool>> {
    return this.http.get('/api/v1/ai/tools', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<Tool>>;
  }
  createTool(payload: ToolRequest): Promise<Tool> {
    return this.http.post('/api/v1/ai/tools', payload) as unknown as Promise<Tool>;
  }
  updateTool(id: number | string, payload: ToolRequest): Promise<Tool> {
    return this.http.put(`/api/v1/ai/tools/${id}`, payload) as unknown as Promise<Tool>;
  }
  deleteTool(id: number | string): Promise<void> {
    return this.http.delete(`/api/v1/ai/tools/${id}`) as unknown as Promise<void>;
  }
  executeTool(id: number | string, payload: ToolExecuteRequest): Promise<ToolExecuteResponse> {
    return this.http.post(
      `/api/v1/ai/tools/${id}/execute`,
      payload,
    ) as unknown as Promise<ToolExecuteResponse>;
  }
}
