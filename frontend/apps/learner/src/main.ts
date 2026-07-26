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

import './styles/global.less';

const app = createApp(App);
const pinia = createPinia();
pinia.use(piniaPluginPersistedstate);
app.use(pinia);

// 全局错误提示：HTTP 拦截器与业务层共用。
configureErrorHandler((msg) => message.error(msg));

// 创建 API 实例（baseURL 走 Vite 代理 /api）。
const http = createHttp({ baseURL: import.meta.env.VITE_API_BASE || '/api' });
const apis = createApis(http);

// 绑定 user store 的 token 读取/刷新回调到 HTTP 拦截器。
const userStore = useUserStore();
userStore.bindToHttp({ baseURL: import.meta.env.VITE_API_BASE || '/api' });

// 全局提供 apis，组件用 inject('apis') 取用。
app.provide('apis', apis);

app.use(router);
app.mount('#app');
