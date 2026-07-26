// Auth API 模块：登录、注册、刷新 token。
// 端点对齐 backend/user-service/internal/controller/router.go：/api/v1/auth/*。

import type { AxiosInstance } from 'axios';

import type { LoginRequest, RefreshRequest, RegisterRequest, TokenResponse } from './user-types';

export class AuthApi {
  constructor(private http: AxiosInstance) {}

  register(payload: RegisterRequest): Promise<TokenResponse> {
    return this.http.post('/api/v1/auth/register', payload) as unknown as Promise<TokenResponse>;
  }

  login(payload: LoginRequest): Promise<TokenResponse> {
    return this.http.post('/api/v1/auth/login', payload) as unknown as Promise<TokenResponse>;
  }

  refresh(refreshToken: string): Promise<TokenResponse> {
    const body: RefreshRequest = { refresh_token: refreshToken };
    return this.http.post('/api/v1/auth/refresh', body) as unknown as Promise<TokenResponse>;
  }
}
