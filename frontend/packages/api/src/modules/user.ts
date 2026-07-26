// User API 模块：当前用户资料、设置。
// 端点对齐 backend/user-service/internal/controller/router.go：/api/v1/users/*。

import type { AxiosInstance } from 'axios';

import type {
  ProfileResponse,
  SettingsResponse,
  UpdateProfileRequest,
  UpdateSettingsRequest,
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
}
