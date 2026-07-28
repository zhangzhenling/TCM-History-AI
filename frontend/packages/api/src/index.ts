// API 包入口：导出 HTTP 工厂、所有模块与类型。
export * from './types';
export * from './http';
export * from './modules/user-types';
export * from './modules/history-types';
export * from './modules/knowledge-types';
export * from './modules/graph-types';
export * from './modules/ai-types';
export * from './modules/learning-types';
export * from './modules/tenant-types';
export { AuthApi } from './modules/auth';
export { UserApi } from './modules/user';
export { HistoryApi } from './modules/history';
export { KnowledgeApi } from './modules/knowledge';
export { GraphApi } from './modules/graph';
export { AiApi } from './modules/ai';
export { LearningApi } from './modules/learning';
export { TenantApi } from './modules/tenant';

import type { AxiosInstance } from 'axios';
import { AuthApi } from './modules/auth';
import { UserApi } from './modules/user';
import { HistoryApi } from './modules/history';
import { KnowledgeApi } from './modules/knowledge';
import { GraphApi } from './modules/graph';
import { AiApi } from './modules/ai';
import { LearningApi } from './modules/learning';
import { TenantApi } from './modules/tenant';

/** 在一个 Axios 实例上装配所有 API 模块。 */
export function createApis(http: AxiosInstance) {
  return {
    auth: new AuthApi(http),
    user: new UserApi(http),
    history: new HistoryApi(http),
    knowledge: new KnowledgeApi(http),
    graph: new GraphApi(http),
    ai: new AiApi(http),
    learning: new LearningApi(http),
    tenant: new TenantApi(http),
  };
}

export type Apis = ReturnType<typeof createApis>;
