// Tenant API 模块：学校/机构多租户管理。
// 端点对齐 backend/user-service/internal/controller/router.go：/api/v1/admin/tenants/*。

import type { AxiosInstance } from 'axios';

import { buildQuery, type ListResponse } from '../types';
import type {
  Tenant,
  CreateTenantRequest,
  UpdateTenantRequest,
  TenantListParams,
  TenantMember,
  AddMemberRequest,
  MemberListResponse,
  UserTenantsResponse,
} from './tenant-types';

export class TenantApi {
  constructor(private http: AxiosInstance) {}

  // ---- 租户 CRUD ----

  list(params?: TenantListParams): Promise<ListResponse<Tenant>> {
    return this.http.get('/api/v1/admin/tenants', {
      params: buildQuery(params ?? {}),
    }) as unknown as Promise<ListResponse<Tenant>>;
  }

  get(id: number | string): Promise<Tenant> {
    return this.http.get(`/api/v1/admin/tenants/${id}`) as unknown as Promise<Tenant>;
  }

  create(payload: CreateTenantRequest): Promise<Tenant> {
    return this.http.post('/api/v1/admin/tenants', payload) as unknown as Promise<Tenant>;
  }

  update(id: number | string, payload: UpdateTenantRequest): Promise<Tenant> {
    return this.http.put(`/api/v1/admin/tenants/${id}`, payload) as unknown as Promise<Tenant>;
  }

  delete(id: number | string): Promise<void> {
    return this.http.delete(`/api/v1/admin/tenants/${id}`) as unknown as Promise<void>;
  }

  // ---- 成员管理 ----

  listMembers(tenantId: number | string): Promise<MemberListResponse> {
    return this.http.get(
      `/api/v1/admin/tenants/${tenantId}/members`,
    ) as unknown as Promise<MemberListResponse>;
  }

  addMember(tenantId: number | string, payload: AddMemberRequest): Promise<TenantMember> {
    return this.http.post(
      `/api/v1/admin/tenants/${tenantId}/members`,
      payload,
    ) as unknown as Promise<TenantMember>;
  }

  removeMember(tenantId: number | string, userId: number | string): Promise<void> {
    return this.http.delete(
      `/api/v1/admin/tenants/${tenantId}/members/${userId}`,
    ) as unknown as Promise<void>;
  }

  // ---- 用户租户查询 ----

  listUserTenants(userId: number | string): Promise<UserTenantsResponse> {
    return this.http.get(
      `/api/v1/admin/tenants-for-user/${userId}`,
    ) as unknown as Promise<UserTenantsResponse>;
  }
}
