<script setup lang="ts">
// 管理后台主布局：左侧 Sider（Logo + 分组菜单）+ 顶部 Header（折叠按钮 + 面包屑 + 用户下拉）+ 内容区（RouterView + KeepAlive）。
// 不引入 Vben Admin 完整框架，仅用 ant-design-vue 的 Layout/Menu/Breadcrumb 自实现 ProLayout 风格。

import { computed, ref } from 'vue';
import { useRoute, useRouter, RouterView, RouterLink } from 'vue-router';
import { Layout, Menu, Breadcrumb, Dropdown, Avatar } from 'ant-design-vue';
import {
  DashboardOutlined,
  TeamOutlined,
  HistoryOutlined,
  DatabaseOutlined,
  RobotOutlined,
  ReadOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  LogoutOutlined,
  UserOutlined,
} from '@ant-design/icons-vue';

import { useUserStore } from '@tcm/stores';

const route = useRoute();
const router = useRouter();
const userStore = useUserStore();

const collapsed = ref(false);

// 当前选中菜单项 = 当前路径。
const selectedKeys = computed<string[]>(() => [route.path]);

// 初始展开的子菜单（按当前路径所在分组）。
const defaultOpenKeys = (() => {
  const seg = route.path.split('/')[2];
  return ['history', 'knowledge', 'ai', 'learning'].includes(seg) ? [seg] : [];
})();

// 面包屑：管理后台 + 命中路由标题。
const breadcrumbs = computed(() => {
  const items: { label: string; path?: string }[] = [
    { label: '管理后台', path: '/admin/dashboard' },
  ];
  for (const r of route.matched) {
    if (r.meta?.title) {
      items.push({ label: String(r.meta.title), path: r.path });
    }
  }
  return items;
});

function onMenuClick(info: { key: string | number }): void {
  router.push(String(info.key));
}

function handleUserMenu(info: { key: string | number }): void {
  if (String(info.key) === 'logout') {
    userStore.logout();
    router.push('/admin/login');
  }
}
</script>

<template>
  <Layout class="admin-layout">
    <Layout.Sider
      :collapsed="collapsed"
      collapsible
      :trigger="null"
      :width="220"
      class="admin-sider"
    >
      <div class="admin-logo">
        <span class="logo-mark">管</span>
        <span v-if="!collapsed" class="logo-text">中医管理后台</span>
      </div>
      <Menu
        mode="inline"
        theme="dark"
        :selected-keys="selectedKeys"
        :default-open-keys="defaultOpenKeys"
        @click="onMenuClick"
      >
        <Menu.Item key="/admin/dashboard">
          <DashboardOutlined />
          <span>仪表盘</span>
        </Menu.Item>
        <Menu.Item key="/admin/users">
          <TeamOutlined />
          <span>用户管理</span>
        </Menu.Item>
        <Menu.SubMenu key="history">
          <template #title>
            <HistoryOutlined />
            <span>历史数据</span>
          </template>
          <Menu.Item key="/admin/history/dynasties">朝代</Menu.Item>
          <Menu.Item key="/admin/history/persons">人物</Menu.Item>
          <Menu.Item key="/admin/history/books">著作</Menu.Item>
          <Menu.Item key="/admin/history/schools">学派</Menu.Item>
          <Menu.Item key="/admin/history/events">事件</Menu.Item>
          <Menu.Item key="/admin/history/prescriptions">方剂</Menu.Item>
          <Menu.Item key="/admin/history/medicines">药物</Menu.Item>
          <Menu.Item key="/admin/history/diseases">疾病</Menu.Item>
        </Menu.SubMenu>
        <Menu.SubMenu key="knowledge">
          <template #title>
            <DatabaseOutlined />
            <span>知识库</span>
          </template>
          <Menu.Item key="/admin/knowledge/documents">文献</Menu.Item>
          <Menu.Item key="/admin/knowledge/queries">检索日志</Menu.Item>
        </Menu.SubMenu>
        <Menu.SubMenu key="ai">
          <template #title>
            <RobotOutlined />
            <span>AI</span>
          </template>
          <Menu.Item key="/admin/ai/conversations">对话</Menu.Item>
          <Menu.Item key="/admin/ai/prompts">Prompt 模板</Menu.Item>
        </Menu.SubMenu>
        <Menu.SubMenu key="learning">
          <template #title>
            <ReadOutlined />
            <span>学习</span>
          </template>
          <Menu.Item key="/admin/learning/courses">课程</Menu.Item>
          <Menu.Item key="/admin/learning/exams">考试</Menu.Item>
        </Menu.SubMenu>
      </Menu>
    </Layout.Sider>

    <Layout>
      <Layout.Header class="admin-header">
        <span class="collapse-trigger" @click="collapsed = !collapsed">
          <MenuUnfoldOutlined v-if="collapsed" />
          <MenuFoldOutlined v-else />
        </span>
        <Breadcrumb class="admin-breadcrumb">
          <Breadcrumb.Item v-for="(item, i) in breadcrumbs" :key="i">
            <RouterLink v-if="item.path && i < breadcrumbs.length - 1" :to="item.path">
              {{ item.label }}
            </RouterLink>
            <span v-else>{{ item.label }}</span>
          </Breadcrumb.Item>
        </Breadcrumb>
        <div class="admin-user">
          <Dropdown>
            <span class="user-trigger">
              <Avatar :size="32" style="background-color: var(--tcm-color-primary)">
                <template #icon><UserOutlined /></template>
              </Avatar>
              <span class="user-name">{{
                userStore.nickname || userStore.username || '管理员'
              }}</span>
            </span>
            <template #overlay>
              <Menu @click="handleUserMenu">
                <Menu.Item key="logout">
                  <LogoutOutlined />
                  <span>退出登录</span>
                </Menu.Item>
              </Menu>
            </template>
          </Dropdown>
        </div>
      </Layout.Header>

      <Layout.Content class="admin-content">
        <RouterView v-slot="{ Component }">
          <KeepAlive :max="12">
            <component :is="Component" />
          </KeepAlive>
        </RouterView>
      </Layout.Content>
    </Layout>
  </Layout>
</template>

<style scoped lang="less">
.admin-layout {
  min-height: 100vh;
}

.admin-sider {
  position: sticky;
  top: 0;
  height: 100vh;
  overflow: auto;
}

.admin-logo {
  height: 56px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 20px;
  color: #fff;
  font-weight: 600;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.logo-mark {
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

.logo-text {
  font-size: 15px;
  white-space: nowrap;
}

.admin-header {
  display: flex;
  align-items: center;
  gap: var(--tcm-spacing-lg);
  padding: 0 var(--tcm-spacing-xl);
  background-color: #fff;
  border-bottom: 1px solid rgba(31, 42, 68, 0.08);
  height: 56px;
}

.collapse-trigger {
  font-size: 18px;
  cursor: pointer;
  color: var(--tcm-color-ink);
}

.admin-breadcrumb {
  flex: 1;
}

.admin-user {
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

.admin-content {
  background-color: var(--tcm-color-paper);
}
</style>
