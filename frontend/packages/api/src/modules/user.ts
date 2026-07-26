// User API 模块：当前用户资料、设置。
// 端点对齐 backend/user-service/internal/controller/router.go：/api/v1/users/*。

import type { AxiosInstance } from 'axios';

import { buildQuery, type ListResponse } from '../types';
import type {
  ProfileResponse,
  SettingsResponse,
  UpdateProfileRequest,
  UpdateSettingsRequest,
  UserListItem,
  UserListParams,
} from './user-types';

export class UserApi {
  constructor(private http: AxiosInstance) {}

  getProfile(): Promise<ProfileResponse> {
    return this.http.get('/api/v1/users/me') as unknown as Promise<ProfileResponse>;
  }

  updateProfile(payload: UpdateProfileRequest): Promise<ProfileResponse> {
    return this.http.put('/api/v1/users/me', payload) as unknown as Promise<ProfileResponse>;
  }

  getSettings(): Promise<SettingsResponse> {
    return this.http.get('/api/v1/users/settings') as unknown as Promise<SettingsResponse>;
  }

  updateSettings(payload: UpdateSettingsRequest): Promise<SettingsResponse> {
    return this.http.put('/api/v1/users/settings', payload) as unknown as Promise<SettingsResponse>;
  }

  // ---- Admin: 用户列表（管理端，对齐设计文档 GET /admin/users） ----
  list(params?: UserListParams): Promise<ListResponse<UserListItem>> {
    return this.http.get('/api/v1/users', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<UserListItem>>;
  }
}
