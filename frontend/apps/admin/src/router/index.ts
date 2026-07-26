// Router 实例与全局守卫。
// 守卫逻辑：未登录访问 requiresAuth 路由时跳转登录页并携带 redirect。

import { createRouter, createWebHistory } from 'vue-router';

import { adminRoutes } from './routes';
import { useUserStore } from '@tcm/stores';

export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL || '/'),
  routes: adminRoutes,
  scrollBehavior(_to, _from, savedPosition) {
    return savedPosition ?? { top: 0 };
  },
});

router.beforeEach((to, _from, next) => {
  // 设置页面标题。
  if (to.meta?.title) {
    document.title = `${String(to.meta.title)} · 中医发展史 AI 管理后台`;
  }
  const userStore = useUserStore();
  if (to.meta?.requiresAuth && !userStore.isLogged) {
    return next({ name: 'AdminLogin', query: { redirect: to.fullPath } });
  }
  // 已登录用户访问登录页时跳回仪表盘。
  if (to.name === 'AdminLogin' && userStore.isLogged) {
    return next({ path: '/admin/dashboard' });
  }
  next();
});
