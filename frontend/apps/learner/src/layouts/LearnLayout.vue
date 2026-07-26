<script setup lang="ts">
// 学习端主布局：顶部导航 + 主区 + 简洁页脚。
// 不引入 Vben Admin 的多标签布局，C 端追求沉浸感。

import { computed } from 'vue';
import { useRoute, useRouter, RouterView } from 'vue-router';
import { Dropdown, Menu, MenuItem, Avatar } from 'ant-design-vue';

import { useUserStore } from '@tcm/stores';
import { shortId } from '@tcm/shared';

const route = useRoute();
const router = useRouter();
const userStore = useUserStore();

const navItems = [
  { name: 'Home', label: '首页', path: '/app/home' },
  { name: 'Timeline', label: '时间轴', path: '/app/timeline' },
  { name: 'PersonList', label: '医家', path: '/app/persons' },
  { name: 'BookList', label: '典籍', path: '/app/books' },
  { name: 'SchoolList', label: '学派', path: '/app/schools' },
  { name: 'Search', label: '检索', path: '/app/search' },
];

const activeName = computed(() => route.name as string | undefined);

function handleUserMenu(info: { key: string | number }) {
  if (info.key === 'logout') {
    userStore.logout();
    router.push({ name: 'Login' });
  } else if (info.key === 'profile') {
    // 占位：后续接入个人中心页。
  }
}
</script>

<template>
  <div class="learn-layout">
    <header class="learn-header">
      <div class="learn-header-inner">
        <RouterLink to="/app/home" class="brand">
          <span class="brand-mark">医</span>
          <span class="brand-text">中医发展史 AI</span>
        </RouterLink>
        <nav class="learn-nav">
          <RouterLink
            v-for="item in navItems"
            :key="item.name"
            :to="item.path"
            class="nav-item"
            :class="{ active: activeName === item.name }"
          >
            {{ item.label }}
          </RouterLink>
        </nav>
        <div class="learn-user">
          <Dropdown>
            <span class="user-trigger">
              <Avatar :size="32" style="background-color: var(--tcm-color-primary)">
                {{ userStore.nickname?.charAt(0) || '客' }}
              </Avatar>
              <span class="user-name">{{ userStore.nickname || '游客' }}</span>
            </span>
            <template #overlay>
              <Menu @click="handleUserMenu">
                <MenuItem key="profile">个人中心</MenuItem>
                <MenuItem key="logout">退出登录</MenuItem>
              </Menu>
            </template>
          </Dropdown>
        </div>
      </div>
    </header>

    <main class="learn-main">
      <RouterView v-slot="{ Component }">
        <KeepAlive :include="['Home', 'Timeline', 'PersonList', 'BookList', 'SchoolList']">
          <component :is="Component" />
        </KeepAlive>
      </RouterView>
    </main>

    <footer class="learn-footer">
      <span>TCM-History-AI · {{ shortId('20260725') }}</span>
      <span class="footer-sep">·</span>
      <span>仅用于学习演示</span>
    </footer>
  </div>
</template>

<style scoped lang="less">
.learn-layout {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.learn-header {
  position: sticky;
  top: 0;
  z-index: 100;
  background-color: var(--tcm-color-paper);
  border-bottom: 1px solid rgba(31, 26, 23, 0.08);
  backdrop-filter: blur(8px);
}

.learn-header-inner {
  max-width: 1280px;
  margin: 0 auto;
  height: 60px;
  padding: 0 var(--tcm-spacing-xl);
  display: flex;
  align-items: center;
  gap: var(--tcm-spacing-lg);
}

.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: var(--tcm-color-ink);
}

.brand-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 6px;
  background-color: var(--tcm-color-primary);
  color: #fff;
  font-family: serif;
  font-size: 18px;
}

.brand-text {
  font-size: 16px;
}

.learn-nav {
  display: flex;
  gap: 4px;
  margin-left: 24px;
  flex: 1;
}

.nav-item {
  padding: 6px 14px;
  border-radius: var(--tcm-radius-base);
  color: var(--tcm-color-ink);
  transition: background-color 0.15s ease;
}
.nav-item:hover {
  background-color: rgba(162, 58, 48, 0.08);
}
.nav-item.active {
  background-color: var(--tcm-color-primary);
  color: #fff;
}

.learn-user {
  margin-left: auto;
}

.user-trigger {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.user-name {
  font-size: 13px;
}

.learn-main {
  flex: 1;
  width: 100%;
}

.learn-footer {
  text-align: center;
  padding: 16px;
  font-size: 12px;
  color: rgba(31, 26, 23, 0.55);
}

.footer-sep {
  margin: 0 8px;
}
</style>
