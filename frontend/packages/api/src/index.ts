// API 包入口：导出 HTTP 工厂、所有模块与类型。
export * from './types';
export * from './http';
export * from './modules/user-types';
export * from './modules/history-types';
export { AuthApi } from './modules/auth';
export { UserApi } from './modules/user';
export { HistoryApi } from './modules/history';

import type { AxiosInstance } from 'axios';
import { AuthApi } from './modules/auth';
import { UserApi } from './modules/user';
import { HistoryApi } from './modules/history';

/** 在一个 Axios 实例上装配所有 API 模块。 */
export function createApis(http: AxiosInstance) {
  return {
    auth: new AuthApi(http),
    user: new UserApi(http),
    history: new HistoryApi(http),
  };
}

export type Apis = ReturnType<typeof createApis>;
