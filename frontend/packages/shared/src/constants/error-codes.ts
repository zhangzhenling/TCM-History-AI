// 业务错误码：与后端 backend/pkg/errno/errno.go 保持同步。
// 0 表示成功；4xxxx 客户端错误；5xxxx 服务端错误。
// 40161 专门用于 Access Token 过期，触发前端自动 refresh。

export const ErrorCode = {
  OK: 0,
  // 客户端错误 (4xxxx)
  InvalidParams: 40000,
  Unauthorized: 40100,
  AccessTokenExpired: 40161,
  RefreshTokenExpired: 40162,
  Forbidden: 40300,
  NotFound: 40400,
  Conflict: 40900,
  TooManyRequests: 42900,
  // 服务端错误 (5xxxx)
  Internal: 50000,
  NotImplemented: 50100,
  BadGateway: 50200,
  ServiceUnavailable: 50300,
} as const;

export type ErrorCode = (typeof ErrorCode)[keyof typeof ErrorCode];

export function isAccessTokenExpired(code: number): boolean {
  return code === ErrorCode.AccessTokenExpired;
}

export function isRefreshTokenExpired(code: number): boolean {
  return code === ErrorCode.RefreshTokenExpired;
}
