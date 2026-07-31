// User Store：承载当前登录态、用户资料与权限。
// 持久化 token + profile + permissions，刷新页面后无需重新登录。
// 通过 configureAuth 将 token 读取/刷新回调注入 @tcm/api 的 HTTP 拦截器，闭环 401 自动刷新。

import 'pinia-plugin-persistedstate';

import { computed, ref } from 'vue';
import { defineStore } from 'pinia';

import { type Apis, configureAuth } from '@tcm/api';

import type { ProfileResponse, TokenResponse } from '@tcm/api';

export interface LoginPayload {
  username: string;
  password: string;
}

export const useUserStore = defineStore(
  'user',
  () => {
    const accessToken = ref<string>('');
    const refreshToken = ref<string>('');
    const profile = ref<ProfileResponse | null>(null);
    const permissions = ref<string[]>([]);
    const expiresAt = ref<number>(0); // access token 到期时间戳（ms）

    const isLogged = computed(() => !!accessToken.value);
    const userId = computed(() => profile.value?.user_id ?? null);
    const username = computed(() => profile.value?.username ?? '');
    const nickname = computed(() => profile.value?.nickname ?? profile.value?.username ?? '');

    /** 注入 HTTP 拦截器所需的 token 读取与刷新回调。应用启动时调用一次。 */
    function bindToHttp(opts?: { baseURL?: string }): void {
      const baseURL = opts?.baseURL || '';
      configureAuth({
        tokenAccessor: () => ({
          accessToken: accessToken.value || null,
          refreshToken: refreshToken.value || null,
        }),
        refresher: () => doRefresh(baseURL),
        onLogout: logout,
      });
    }

    async function login(payload: LoginPayload, apis: Apis): Promise<void> {
      const res: TokenResponse = await apis.auth.login({
        username: payload.username,
        password: payload.password,
      });
      applyTokens(res);
      await fetchProfile(apis);
    }

    async function register(
      payload: LoginPayload & { email?: string; phone?: string },
      apis: Apis,
    ): Promise<void> {
      const res = await apis.auth.register({
        username: payload.username,
        password: payload.password,
        email: payload.email,
        phone: payload.phone,
      });
      applyTokens(res);
      await fetchProfile(apis);
    }

    async function fetchProfile(apis: Apis): Promise<void> {
      profile.value = await apis.user.getProfile();
    }

    /** 用 refresh token 换取新的 access token，返回新 token 字符串供 http 重放。 */
    async function doRefresh(baseURL: string): Promise<string | null> {
      if (!refreshToken.value) return null;
      try {
        // 这里直接走 fetch 而非 apis.auth.refresh，避免触发拦截器再次走刷新流程。
        const res = await fetch(`${baseURL}/api/v1/auth/refresh`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: refreshToken.value }),
        });
        if (!res.ok) return null;
        const env = await res.json();
        if (env.code !== 0) return null;
        const t: TokenResponse = env.data;
        applyTokens(t);
        return t.access_token;
      } catch {
        return null;
      }
    }

    function applyTokens(res: TokenResponse): void {
      accessToken.value = res.access_token;
      refreshToken.value = res.refresh_token;
      expiresAt.value = Date.now() + (res.expires_in || 3600) * 1000;
      // 用户名兜底，profile 拉取前可用。
      if (!profile.value) {
        profile.value = {
          user_id: res.user_id,
          username: res.username,
          status: 'active',
        };
      }
    }

    function hasPermission(code: string): boolean {
      return permissions.value.includes(code) || permissions.value.includes('*');
    }

    function logout(): void {
      accessToken.value = '';
      refreshToken.value = '';
      profile.value = null;
      permissions.value = [];
      expiresAt.value = 0;
    }

    return {
      accessToken,
      refreshToken,
      profile,
      permissions,
      expiresAt,
      isLogged,
      userId,
      username,
      nickname,
      bindToHttp,
      login,
      register,
      fetchProfile,
      doRefresh,
      hasPermission,
      logout,
    };
  },
  {
    persist: {
      key: 'tcm-user',
      storage: typeof localStorage !== 'undefined' ? localStorage : undefined,
      paths: ['accessToken', 'refreshToken', 'profile', 'permissions', 'expiresAt'],
    },
  },
);
