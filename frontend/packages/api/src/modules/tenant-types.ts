// Tenant（学校/机构多租户）类型定义，对齐 backend/user-service/internal/application/dto/tenant_dto.go。
// 端点：/api/v1/admin/tenants/*（由 gateway RBAC 强制 admin 角色）。

/** 租户实体（响应体）。 */
export interface Tenant {
  id: number;
  name: string;
  code: string;
  plan: string;
  status: string;
  max_users: number;
  expires_at?: string;
  created_at?: string;
  updated_at?: string;
}

/** 创建租户请求体。 */
export interface CreateTenantRequest {
  name: string;
  code: string;
  plan: string;
  max_users: number;
  expires_at?: string;
}

/** 更新租户请求体（所有字段可选，支持 PATCH 语义）。 */
export interface UpdateTenantRequest {
  name?: string;
  plan?: string;
  status?: string;
  max_users?: number;
  expires_at?: string;
}

/** 租户列表查询参数。 */
export interface TenantListParams {
  page?: number;
  page_size?: number;
  status?: string;
}

/** 租户成员实体（响应体）。 */
export interface TenantMember {
  id: number;
  tenant_id: number;
  user_id: number;
  role: string;
  joined_at: string;
  expired_at?: string;
  created_at?: string;
  updated_at?: string;
}

/** 添加成员请求体。 */
export interface AddMemberRequest {
  user_id: number;
  role: string;
  expires_at?: string;
}

/** 租户成员列表响应。 */
export interface MemberListResponse {
  tenant_id: number;
  total: number;
  items: TenantMember[];
}

/** 用户所属租户列表响应。 */
export interface UserTenantsResponse {
  user_id: number;
  total: number;
  items: TenantMember[];
}
