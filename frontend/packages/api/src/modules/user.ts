// User API 模块：当前用户资料、设置、角色、会员、API Key。
// 端点对齐 backend/user-service/internal/controller/router.go。

import type { AxiosInstance } from 'axios';

import { buildQuery, type ListResponse } from '../types';
import type {
  ApiKey,
  ApiKeyCreateResponse,
  ApiKeyRequest,
  MembershipPlan,
  MembershipPlanRequest,
  Permission,
  ProfileResponse,
  Role,
  RoleRequest,
  SettingsResponse,
  UpdateProfileRequest,
  UpdateSettingsRequest,
  UserListItem,
  UserListParams,
} from './user-types';

export class UserApi {
  constructor(private http: AxiosInstance) {}

  // ---- Profile ----
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

  // ---- Admin: 用户列表 ----
  list(params?: UserListParams): Promise<ListResponse<UserListItem>> {
    return this.http.get('/api/v1/admin/users', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<UserListItem>>;
  }

  // ---- Admin: Roles ----
  listRoles(params?: { page?: number; page_size?: number }): Promise<ListResponse<Role>> {
    return this.http.get('/api/v1/admin/roles', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<Role>>;
  }
  getRole(id: number | string): Promise<Role> {
    return this.http.get(`/api/v1/admin/roles/${id}`) as unknown as Promise<Role>;
  }
  createRole(payload: RoleRequest): Promise<Role> {
    return this.http.post('/api/v1/admin/roles', payload) as unknown as Promise<Role>;
  }
  updateRole(id: number | string, payload: RoleRequest): Promise<Role> {
    return this.http.put(`/api/v1/admin/roles/${id}`, payload) as unknown as Promise<Role>;
  }
  deleteRole(id: number | string): Promise<void> {
    return this.http.delete(`/api/v1/admin/roles/${id}`) as unknown as Promise<void>;
  }
  listPermissions(): Promise<Permission[]> {
    return this.http.get('/api/v1/admin/permissions') as unknown as Promise<Permission[]>;
  }

  // ---- Admin: Membership Plans ----
  listMembershipPlans(params?: {
    page?: number;
    page_size?: number;
  }): Promise<ListResponse<MembershipPlan>> {
    return this.http.get('/api/v1/admin/membership/plans', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<MembershipPlan>>;
  }
  createMembershipPlan(payload: MembershipPlanRequest): Promise<MembershipPlan> {
    return this.http.post(
      '/api/v1/admin/membership/plans',
      payload,
    ) as unknown as Promise<MembershipPlan>;
  }
  updateMembershipPlan(
    id: number | string,
    payload: MembershipPlanRequest,
  ): Promise<MembershipPlan> {
    return this.http.put(
      `/api/v1/admin/membership/plans/${id}`,
      payload,
    ) as unknown as Promise<MembershipPlan>;
  }
  deleteMembershipPlan(id: number | string): Promise<void> {
    return this.http.delete(`/api/v1/admin/membership/plans/${id}`) as unknown as Promise<void>;
  }

  // ---- API Keys ----
  listApiKeys(params?: { page?: number; page_size?: number }): Promise<ListResponse<ApiKey>> {
    return this.http.get('/api/v1/api-keys', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<ApiKey>>;
  }
  createApiKey(payload: ApiKeyRequest): Promise<ApiKeyCreateResponse> {
    return this.http.post('/api/v1/api-keys', payload) as unknown as Promise<ApiKeyCreateResponse>;
  }
  getApiKey(id: number | string): Promise<ApiKey> {
    return this.http.get(`/api/v1/api-keys/${id}`) as unknown as Promise<ApiKey>;
  }
  updateApiKey(id: number | string, payload: ApiKeyRequest): Promise<ApiKey> {
    return this.http.put(`/api/v1/api-keys/${id}`, payload) as unknown as Promise<ApiKey>;
  }
  deleteApiKey(id: number | string): Promise<void> {
    return this.http.delete(`/api/v1/api-keys/${id}`) as unknown as Promise<void>;
  }
  regenerateApiKey(id: number | string): Promise<ApiKeyCreateResponse> {
    return this.http.post(
      `/api/v1/api-keys/${id}/regenerate`,
    ) as unknown as Promise<ApiKeyCreateResponse>;
  }
}
