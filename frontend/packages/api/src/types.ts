// 统一响应包络与分页结构，与后端 backend/pkg/response、backend/pkg/pagination 对齐。
import type { AxiosRequestConfig } from 'axios';

/** 后端统一响应体。 */
export interface ApiEnvelope<T = unknown> {
  code: number;
  message: string;
  data?: T;
  trace_id?: string;
}

/** 分页列表响应（后端 dto.ListResponse）。 */
export interface ListResponse<T> {
  page: number;
  page_size: number;
  total: number;
  total_page: number;
  items: T[];
}

/** 分页查询参数。 */
export interface PageParams {
  page?: number;
  page_size?: number;
}

/** 请求附加选项，透传给 axios，用于取消请求等。 */
export interface RequestOptions extends Pick<AxiosRequestConfig, 'signal' | 'timeout'> {}

/** 通用查询字符串构造，自动剔除 null/undefined/空字符串。 */
export function buildQuery(params: object): Record<string, string> {
  const q: Record<string, string> = {};
  for (const [k, v] of Object.entries(params as Record<string, unknown>)) {
    if (v === null || v === undefined || v === '') continue;
    if (Array.isArray(v)) {
      if (v.length === 0) continue;
      q[k] = v.join(',');
    } else {
      q[k] = String(v);
    }
  }
  return q;
}
