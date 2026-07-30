// User Service 类型定义，对齐 backend/user-service/internal/application/dto。

export interface RegisterRequest {
  username: string;
  password: string;
  email?: string;
  phone?: string;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface RefreshRequest {
  refresh_token: string;
}

export interface TokenResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user_id: number;
  username: string;
}

export interface ProfileResponse {
  user_id: number;
  username: string;
  email?: string;
  phone?: string;
  status: string;
  nickname?: string;
  avatar_url?: string;
  gender?: string;
  birth_date?: string;
  bio?: string;
}

export interface UpdateProfileRequest {
  nickname?: string;
  avatar_url?: string;
  gender?: string;
  birth_date?: string;
  bio?: string;
}

export interface SettingsResponse {
  user_id: number;
  locale: string;
  theme: string;
  notify_email: boolean;
  notify_push: boolean;
  preferences: unknown;
  updated_at: string;
}

export interface UpdateSettingsRequest {
  locale?: string;
  theme?: string;
  notify_email?: boolean;
  notify_push?: boolean;
  preferences?: unknown;
}

/** 管理端用户列表项（对齐设计文档 GET /admin/users 响应行）。 */
export interface UserListItem {
  id: number;
  username: string;
  nickname?: string;
  email?: string;
  phone?: string;
  status: string;
  created_at: string;
}

/** 管理端用户列表查询参数。 */
export interface UserListParams {
  page?: number;
  page_size?: number;
  keyword?: string;
  status?: string;
}

// ============================================================================
// Roles & Permissions
// ============================================================================

export interface Role {
  id: number;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface RoleRequest {
  name: string;
  description?: string;
}

export interface Permission {
  id: number;
  name: string;
  resource: string;
  action: string;
  description: string;
}

export interface AssignPermissionsRequest {
  permission_ids: number[];
}

// ============================================================================
// Membership Plans
// ============================================================================

export interface MembershipPlan {
  id: number;
  name: string;
  description: string;
  price: number;
  duration_days: number;
  features_json: unknown;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface MembershipPlanRequest {
  name: string;
  description?: string;
  price: number;
  duration_days: number;
  features_json?: unknown;
  is_active?: boolean;
}

export interface Subscription {
  id: number;
  user_id: number;
  plan_id: number;
  plan_name: string;
  status: 'active' | 'expired' | 'cancelled' | string;
  started_at: string;
  expires_at: string;
  auto_renew: boolean;
}

export interface Order {
  id: number;
  user_id: number;
  plan_id: number;
  plan_name: string;
  amount: number;
  status: string;
  created_at: string;
}

// ============================================================================
// API Keys
// ============================================================================

export interface ApiKey {
  id: number;
  user_id: number;
  name: string;
  key_prefix: string;
  is_active: boolean;
  last_used_at?: string;
  created_at: string;
  updated_at: string;
}

export interface ApiKeyCreateResponse {
  id: number;
  name: string;
  key: string;
  key_prefix: string;
  message: string;
}

export interface ApiKeyRequest {
  name: string;
  is_active?: boolean;
}
