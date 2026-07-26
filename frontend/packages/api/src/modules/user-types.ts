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
