// Axios 实例与拦截器：注入 JWT、追踪 ID，统一错误处理与 Access Token 自动刷新。
//
// 与 stores 包解耦：通过 setTokenAccessor 注入 token 读取回调，避免循环依赖。
// Access Token 过期（业务码 40161）触发 onAccessTokenExpired 回调，由 stores
// 负责调用 refresh，并以 refreshing Promise 串行化避免并发刷新。

import axios, {
  type AxiosInstance,
  type AxiosRequestConfig,
  type AxiosError,
  type InternalAxiosRequestConfig,
} from 'axios';

import { ErrorCode, isAccessTokenExpired } from '@tcm/shared';

import type { ApiEnvelope } from './types';

/** Token 读取回调，由 stores 在应用启动时注入。 */
export type TokenAccessor = () => {
  accessToken: string | null;
  refreshToken: string | null;
};

/** Access Token 过期回调，返回新 token 后由调用方重放原请求。 */
export type AccessTokenRefresher = () => Promise<string | null>;

/** 退出登录回调（refresh token 也失效时触发）。 */
export type LogoutHandler = () => void;

export interface HttpOptions {
  baseURL: string;
  timeout?: number;
  onError?: (message: string) => void;
}

interface RetryableConfig extends InternalAxiosRequestConfig {
  _retried?: boolean;
}

let tokenAccessor: TokenAccessor = () => ({ accessToken: null, refreshToken: null });
let refresher: AccessTokenRefresher | null = null;
let logoutHandler: LogoutHandler | null = null;
let onError: (msg: string) => void = (msg) => console.error('[http]', msg);

let refreshing: Promise<string | null> | null = null;

/** 注入 token 与刷新回调。在应用入口调用一次。 */
export function configureAuth(opts: {
  tokenAccessor: TokenAccessor;
  refresher?: AccessTokenRefresher;
  onLogout?: LogoutHandler;
}): void {
  tokenAccessor = opts.tokenAccessor;
  if (opts.refresher) refresher = opts.refresher;
  if (opts.onLogout) logoutHandler = opts.onLogout;
}

/** 注入全局错误提示回调（如 ant-design-vue 的 message.error）。 */
export function configureErrorHandler(handler: (msg: string) => void): void {
  onError = handler;
}

/** 创建一个配置好的 Axios 实例。 */
export function createHttp(opts: HttpOptions): AxiosInstance {
  const instance = axios.create({
    baseURL: opts.baseURL,
    timeout: opts.timeout ?? 30000,
    headers: { 'Content-Type': 'application/json' },
  });

  // 请求拦截：注入 JWT + Request-Id。
  instance.interceptors.request.use((config) => {
    const { accessToken } = tokenAccessor();
    if (accessToken) {
      config.headers.Authorization = `Bearer ${accessToken}`;
    }
    // crypto.randomUUID 在 HTTPS 或 localhost 下可用，否则降级。
    config.headers['X-Request-Id'] =
      typeof crypto !== 'undefined' && 'randomUUID' in crypto
        ? crypto.randomUUID()
        : `rid-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
    return config;
  });

  // 响应拦截：拆 envelope、统一错误处理、401 自动刷新重放。
  instance.interceptors.response.use(
    (res) => {
      const env = res.data as ApiEnvelope;
      // 非 envelope 响应（如文件下载）原样返回。
      if (!env || typeof env.code !== 'number') return res.data;
      if (env.code === ErrorCode.OK) return env.data;
      if (isAccessTokenExpired(env.code)) {
        return replayAfterRefresh(instance, res.config as RetryableConfig);
      }
      onError(env.message || '请求失败');
      return Promise.reject(new Error(env.message || `业务错误 ${env.code}`));
    },
    async (err: AxiosError<ApiEnvelope>) => {
      const config = err.config as RetryableConfig | undefined;
      const status = err.response?.status;
      // HTTP 401（无 envelope，如网关直接拒）也走刷新流程。
      if (
        status === 401 &&
        config &&
        !config._retried &&
        !String(config.url).includes('/auth/refresh')
      ) {
        return replayAfterRefresh(instance, config);
      }
      const msg = err.response?.data?.message || err.message || '网络异常';
      onError(msg);
      return Promise.reject(err);
    },
  );

  return instance;
}

async function replayAfterRefresh(
  instance: AxiosInstance,
  config: RetryableConfig,
): Promise<unknown> {
  if (!refresher) {
    triggerLogout();
    return Promise.reject(new Error('access token expired and no refresher configured'));
  }
  if (config._retried) {
    triggerLogout();
    return Promise.reject(new Error('access token expired after retry'));
  }
  config._retried = true;
  try {
    refreshing ??= refresher();
    const newToken = await refreshing;
    refreshing = null;
    if (!newToken) {
      triggerLogout();
      return Promise.reject(new Error('refresh failed'));
    }
    config.headers.Authorization = `Bearer ${newToken}`;
    return instance.request(config as AxiosRequestConfig);
  } catch (e) {
    refreshing = null;
    triggerLogout();
    return Promise.reject(e);
  }
}

function triggerLogout(): void {
  if (logoutHandler) logoutHandler();
}
