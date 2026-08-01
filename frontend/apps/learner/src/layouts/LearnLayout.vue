<script setup lang="ts">
// 学习端主布局：桌面端顶部导航，移动端顶部简版 + 底部 Tabbar。
// 不引入 Vben Admin 的多标签布局，C 端追求沉浸感。

import { computed } from 'vue';
import { useRoute, useRouter, RouterView, RouterLink } from 'vue-router';
import { Dropdown, Menu, MenuItem, Avatar } from 'ant-design-vue';
import {
  HomeOutlined,
  ClockCircleOutlined,
  SearchOutlined,
  UserOutlined,
  ReadOutlined,
} from '@ant-design/icons-vue';

import { useUserStore } from '@tcm/stores';
import { shortId } from '@tcm/shared';
import { useViewport } from '@/composables/useViewport';

const route = useRoute();
const router = useRouter();
const userStore = useUserStore();
const { isMobile, isDesktop } = useViewport();

// 桌面端顶部导航（全量）
const desktopNavItems = [
  { name: 'Home', label: '首页', path: '/app/home' },
  { name: 'Timeline', label: '时间轴', path: '/app/timeline' },
  { name: 'PersonList', label: '医家', path: '/app/persons' },
  { name: 'BookList', label: '典籍', path: '/app/books' },
  { name: 'SchoolList', label: '学派', path: '/app/schools' },
  { name: 'Search', label: '检索', path: '/app/search' },
  { name: 'Graph', label: '图谱', path: '/app/graph' },
  { name: 'Knowledge', label: '文献', path: '/app/knowledge' },
];

// 移动端底部 Tabbar（5 个以内核心入口）
const mobileTabbarItems = [
  { name: 'Home', label: '首页', path: '/app/home', icon: HomeOutlined },
  { name: 'Timeline', label: '时间轴', path: '/app/timeline', icon: ClockCircleOutlined },
  { name: 'Search', label: '检索', path: '/app/search', icon: SearchOutlined },
  { name: 'LearningCourses', label: '课程', path: '/app/learning/courses', icon: ReadOutlined },
  { name: 'UserMenu', label: '我的', path: '/app/learning/records', icon: UserOutlined, isUser: true },
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
  <div class="learn-layout" :class="{ 'is-mobile': isMobile }">
    <!-- 顶部导航 -->
    <header class="learn-header">
      <div class="learn-header-inner">
        <RouterLink to="/app/home" class="brand">
          <span class="brand-mark">医</span>
          <span v-if="!isMobile" class="brand-text">中医发展史 AI</span>
          <span v-else class="brand-text brand-text-mobile">中医AI</span>
        </RouterLink>

        <!-- 桌面端顶部导航 -->
        <nav v-if="isDesktop" class="learn-nav">
          <RouterLink
            v-for="item in desktopNavItems"
            :key="item.name"
            :to="item.path"
            class="nav-item"
            :class="{ active: activeName === item.name }"
          >
            {{ item.label }}
          </RouterLink>
        </nav>

        <!-- 用户区 -->
        <div class="learn-user">
          <Dropdown>
            <span class="user-trigger">
              <Avatar :size="isMobile ? 30 : 32" style="background-color: var(--tcm-color-primary)">
                {{ userStore.nickname?.charAt(0) || '客' }}
              </Avatar>
              <span v-if="!isMobile" class="user-name">{{ userStore.nickname || '游客' }}</span>
            </span>
            <template #overlay>
              <Menu @click="handleUserMenu">
                <MenuItem key="profile">
                  <span><UserOutlined /> 个人中心</span>
                </MenuItem>
                <MenuItem key="logout">
                  <span>退出登录</span>
                </MenuItem>
              </Menu>
            </template>
          </Dropdown>
        </div>
      </div>
    </header>

    <!-- 主内容区 -->
    <main class="learn-main" :class="{ 'has-tabbar': isMobile }">
      <RouterView v-slot="{ Component }">
        <KeepAlive
          :include="[
            'Home',
            'Timeline',
            'PersonList',
            'BookList',
            'SchoolList',
            'Graph',
            'Knowledge',
            'LearningCourses',
            'LearningStudyPlans',
            'LearningExams',
            'LearningRecords',
          ]"
        >
          <component :is="Component" />
        </KeepAlive>
      </RouterView>
    </main>

    <!-- 移动端底部 Tabbar -->
    <nav v-if="isMobile" class="learn-tabbar" role="tablist">
      <RouterLink
        v-for="item in mobileTabbarItems"
        :key="item.name"
        :to="item.path"
        class="tab-item"
        :class="{ active: activeName === item.name || (item.isUser && activeName?.startsWith('Learning')) }"
      >
        <component :is="item.icon" class="tab-icon" />
        <span class="tab-label">{{ item.label }}</span>
      </RouterLink>
    </nav>

    <!-- 仅桌面端显示页脚 -->
    <footer v-if="isDesktop" class="learn-footer">
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
  width: 100%;

  &.is-mobile {
    padding-bottom: 0;
  }
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
  padding-top: var(--sat, 0);
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
  flex-shrink: 0;
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
  flex-shrink: 0;
}

.brand-text {
  font-size: 16px;
  white-space: nowrap;
}

.brand-text-mobile {
  font-size: 15px;
}

.learn-nav {
  display: flex;
  gap: 4px;
  margin-left: 24px;
  flex: 1;
  flex-wrap: nowrap;
}

.nav-item {
  padding: 6px 14px;
  border-radius: var(--tcm-radius-base);
  color: var(--tcm-color-ink);
  transition: background-color 0.15s ease;
  white-space: nowrap;
  font-size: 14px;
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
  flex-shrink: 0;
}

.user-trigger {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: var(--tcm-radius-base);
  transition: background-color 0.15s;

  &:hover {
    background-color: var(--tcm-color-bg-secondary);
  }
}

.user-name {
  font-size: 13px;
}

.learn-main {
  flex: 1;
  width: 100%;

  &.has-tabbar {
    /* 为底部 Tabbar 留出空间 */
    padding-bottom: calc(var(--tcm-mobile-tabbar-height) + var(--sab, 0px));
  }
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

/* ========== 移动端底部 Tabbar ========== */
.learn-tabbar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 99;
  display: flex;
  justify-content: space-around;
  align-items: stretch;
  height: calc(var(--tcm-mobile-tabbar-height) + var(--sab, 0px));
  padding-bottom: var(--sab, 0px);
  background-color: var(--tcm-color-paper);
  border-top: 1px solid rgba(31, 26, 23, 0.08);
  backdrop-filter: blur(12px);
  box-shadow: 0 -2px 12px rgba(0, 0, 0, 0.04);
}

.tab-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  padding: 6px 2px;
  color: var(--tcm-color-text-secondary);
  transition: color 0.15s ease;
  font-size: 11px;
  min-height: var(--tcm-mobile-tabbar-height);
  text-decoration: none;

  &.active {
    color: var(--tcm-color-primary);
  }
}

.tab-icon {
  font-size: 22px;
  line-height: 1;
}

.tab-label {
  line-height: 1.2;
  font-weight: 500;
}

/* ========== 移动端响应式调整 ========== */
@media (max-width: 768px) {
  .learn-header-inner {
    height: calc(var(--tcm-mobile-header-height) + var(--sat, 0px));
    padding: 0 var(--tcm-spacing-lg);
    padding-top: var(--sat, 0px);
    gap: var(--tcm-spacing-base);
  }
}

@media (max-width: 480px) {
  .learn-header-inner {
    padding: 0 var(--tcm-spacing-base);
    padding-top: var(--sat, 0px);
  }
}
</style>
