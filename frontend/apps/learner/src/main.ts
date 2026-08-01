// Learner SPA 入口：创建 Pinia + Router，注入 ant-design-vue 错误提示，
// 在应用启动时将 user store 的 token 读取/刷新回调绑定到 HTTP 拦截器。

import { createApp } from 'vue';
import { createPinia } from 'pinia';
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate';
import { message } from 'ant-design-vue';
import 'ant-design-vue/dist/reset.css';

import App from './App.vue';
import { router } from './router';
import { createApis, createHttp, configureErrorHandler } from '@tcm/api';
import { useUserStore } from '@tcm/stores';
import { ErrorBoundary } from '@tcm/shared';
import { useErrorHandler } from './composables/useErrorHandler';

import './styles/global.less';

const app = createApp(App);
const pinia = createPinia();
pinia.use(piniaPluginPersistedstate);
app.use(pinia);

// 全局错误提示：HTTP 拦截器与业务层共用。
configureErrorHandler((msg) => message.error(msg));

// 注册全局错误边界组件。
app.component('ErrorBoundary', ErrorBoundary);

// 创建 API 实例（API 模块已带 /api/v1 前缀，baseURL 留空即可）。
const http = createHttp({ baseURL: import.meta.env.VITE_API_BASE || '' });
const apis = createApis(http);

// 绑定 user store 的 token 读取/刷新回调到 HTTP 拦截器。
const userStore = useUserStore();
userStore.bindToHttp({ baseURL: import.meta.env.VITE_API_BASE || '' });

// 全局提供 apis，组件用 inject('apis') 取用。
app.provide('apis', apis);

// 注册全局错误处理 composable。
const errorHandler = useErrorHandler();
app.provide('errorHandler', errorHandler);

app.use(router);
app.mount('#app');

// PWA Service Worker 注册（生产环境由 vite-plugin-pwa 自动生成 sw.js）
if ('serviceWorker' in navigator && import.meta.env.PROD) {
  window.addEventListener('load', () => {
    navigator.serviceWorker
      .register('/sw.js')
      .catch(() => {
        /* 静默失败：不影响用户体验 */
      });
  });
}
