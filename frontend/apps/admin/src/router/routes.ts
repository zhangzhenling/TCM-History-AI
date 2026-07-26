// 路由表：管理端采用静态路由，前缀 /admin，登录页 /admin/login 独立。

import type { RouteRecordRaw } from 'vue-router';

export const adminRoutes: RouteRecordRaw[] = [
  {
    path: '/admin/login',
    name: 'AdminLogin',
    component: () => import('@/views/auth/login.vue'),
    meta: { title: '管理后台登录' },
  },
  {
    path: '/admin',
    component: () => import('@/layouts/AdminLayout.vue'),
    redirect: '/admin/dashboard',
    meta: { requiresAuth: true },
    children: [
      {
        path: 'dashboard',
        name: 'AdminDashboard',
        component: () => import('@/views/dashboard/index.vue'),
        meta: { title: '仪表盘', keepAlive: true },
      },
      {
        path: 'users',
        name: 'AdminUsers',
        component: () => import('@/views/users/list.vue'),
        meta: { title: '用户管理', keepAlive: true },
      },
      {
        path: 'history/dynasties',
        name: 'AdminDynasties',
        component: () => import('@/views/history/dynasties.vue'),
        meta: { title: '朝代管理', keepAlive: true },
      },
      {
        path: 'history/persons',
        name: 'AdminPersons',
        component: () => import('@/views/history/persons.vue'),
        meta: { title: '人物管理', keepAlive: true },
      },
      {
        path: 'history/books',
        name: 'AdminBooks',
        component: () => import('@/views/history/books.vue'),
        meta: { title: '著作管理', keepAlive: true },
      },
      {
        path: 'knowledge/documents',
        name: 'AdminDocuments',
        component: () => import('@/views/knowledge/documents.vue'),
        meta: { title: '文献管理', keepAlive: true },
      },
      {
        path: 'knowledge/queries',
        name: 'AdminQueries',
        component: () => import('@/views/knowledge/queries.vue'),
        meta: { title: '检索日志', keepAlive: true },
      },
      {
        path: 'ai/conversations',
        name: 'AdminConversations',
        component: () => import('@/views/ai/conversations.vue'),
        meta: { title: '对话记录', keepAlive: true },
      },
      {
        path: 'ai/prompts',
        name: 'AdminPrompts',
        component: () => import('@/views/ai/prompts.vue'),
        meta: { title: 'Prompt 模板', keepAlive: true },
      },
      {
        path: 'learning/courses',
        name: 'AdminCourses',
        component: () => import('@/views/learning/courses.vue'),
        meta: { title: '课程管理', keepAlive: true },
      },
      {
        path: 'learning/exams',
        name: 'AdminExams',
        component: () => import('@/views/learning/exams.vue'),
        meta: { title: '考试管理', keepAlive: true },
      },
    ],
  },
  {
    path: '/',
    redirect: '/admin',
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'AdminNotFound',
    component: () => import('@/views/error/404.vue'),
    meta: { title: '页面不存在' },
  },
];
