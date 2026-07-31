// 路由表：学习端采用静态路由 + 详情页动态 ID。
// 路由前缀 /app，登录页 /login 独立。

import type { RouteRecordRaw } from 'vue-router';

export const learnerRoutes: RouteRecordRaw[] = [
  {
    path: '/app',
    component: () => import('@/layouts/LearnLayout.vue'),
    redirect: '/app/home',
    meta: { requiresAuth: true },
    children: [
      {
        path: 'home',
        name: 'Home',
        component: () => import('@/views/home/index.vue'),
        meta: { title: '首页', keepAlive: true },
      },
      {
        path: 'timeline',
        name: 'Timeline',
        component: () => import('@/views/timeline/index.vue'),
        meta: { title: '发展时间轴', keepAlive: true },
      },
      {
        path: 'persons',
        name: 'PersonList',
        component: () => import('@/views/person/list.vue'),
        meta: { title: '历代医家', keepAlive: true },
      },
      {
        path: 'persons/:id',
        name: 'PersonDetail',
        component: () => import('@/views/person/detail.vue'),
        props: true,
        meta: { title: '医家详情' },
      },
      {
        path: 'books',
        name: 'BookList',
        component: () => import('@/views/book/list.vue'),
        meta: { title: '中医典籍', keepAlive: true },
      },
      {
        path: 'books/:id',
        name: 'BookDetail',
        component: () => import('@/views/book/detail.vue'),
        props: true,
        meta: { title: '典籍详情' },
      },
      {
        path: 'schools',
        name: 'SchoolList',
        component: () => import('@/views/school/list.vue'),
        meta: { title: '医学学派', keepAlive: true },
      },
      {
        path: 'schools/:id',
        name: 'SchoolDetail',
        component: () => import('@/views/school/detail.vue'),
        props: true,
        meta: { title: '学派详情' },
      },
      {
        path: 'search',
        name: 'Search',
        component: () => import('@/views/search/index.vue'),
        meta: { title: '检索' },
      },
      {
        path: 'graph',
        name: 'Graph',
        component: () => import('@/views/graph/index.vue'),
        meta: { title: '知识图谱' },
      },
      {
        path: 'knowledge',
        name: 'Knowledge',
        component: () => import('@/views/knowledge/index.vue'),
        meta: { title: '文献检索' },
      },
      {
        path: 'learning/courses',
        name: 'LearningCourses',
        component: () => import('@/views/learning/courses.vue'),
        meta: { title: '课程中心', keepAlive: true },
      },
      {
        path: 'learning/exams',
        name: 'LearningExams',
        component: () => import('@/views/learning/exams.vue'),
        meta: { title: '考试中心', keepAlive: true },
      },
      {
        path: 'learning/wrong-questions',
        name: 'LearningWrongQuestions',
        component: () => import('@/views/learning/wrong-questions.vue'),
        meta: { title: '错题本', keepAlive: true },
      },
      {
        path: 'learning/study-plans',
        name: 'LearningStudyPlans',
        component: () => import('@/views/learning/study-plans.vue'),
        meta: { title: '学习计划', keepAlive: true },
      },
    ],
  },
  {
    path: '/',
    component: () => import('@/layouts/BlankLayout.vue'),
    redirect: '/app/home',
    children: [
      {
        path: 'login',
        name: 'Login',
        component: () => import('@/views/auth/login.vue'),
        meta: { title: '登录' },
      },
      {
        path: 'register',
        name: 'Register',
        component: () => import('@/views/auth/register.vue'),
        meta: { title: '注册' },
      },
    ],
  },
  {
    path: '/home',
    redirect: '/app/home',
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/error/404.vue'),
    meta: { title: '页面不存在' },
  },
];
