<script setup lang="ts">
// 仪表盘：4 个 StatCard（用户/文献/对话/考试总数）+ 最近注册用户表格占位。
// 使用 useApi() 调用对应 API，loading 用 Spin，空数据用 Empty。
import { computed, onMounted, reactive, ref } from 'vue';
import { Spin, Empty, Table } from 'ant-design-vue';
import {
  UserOutlined,
  FileTextOutlined,
  MessageOutlined,
  FormOutlined,
} from '@ant-design/icons-vue';

import StatCard from '@/components/StatCard.vue';
import { useApi } from '@/composables/useApi';
import { formatDateTime } from '@tcm/shared';
import type { UserListItem } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const stats = reactive({
  users: null as number | null,
  documents: null as number | null,
  conversations: null as number | null,
  exams: null as number | null,
});
const recentUsers = ref<UserListItem[]>([]);

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '用户名', dataIndex: 'username', key: 'username' },
  { title: '昵称', dataIndex: 'nickname', key: 'nickname' },
  { title: '邮箱', dataIndex: 'email', key: 'email' },
  { title: '状态', dataIndex: 'status', key: 'status', width: 100 },
  { title: '注册时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
];

const dataSource = computed(() =>
  recentUsers.value.map((u) => ({
    ...u,
    nickname: u.nickname || '—',
    email: u.email || '—',
    created_at: formatDateTime(u.created_at),
  })),
);

onMounted(async () => {
  loading.value = true;
  const [usersR, docsR, convsR, examsR] = await Promise.allSettled([
    apis.user.list({ page: 1, page_size: 5 }),
    apis.knowledge.listDocuments({ page: 1, page_size: 1 }),
    apis.ai.listConversations({ page: 1, page_size: 1 }),
    apis.learning.listExams({ page: 1, page_size: 1 }),
  ]);
  if (usersR.status === 'fulfilled') {
    stats.users = usersR.value.total;
    recentUsers.value = usersR.value.items ?? [];
  }
  if (docsR.status === 'fulfilled') {
    stats.documents = docsR.value.total;
  }
  if (convsR.status === 'fulfilled') {
    stats.conversations = convsR.value.total;
  }
  if (examsR.status === 'fulfilled') {
    stats.exams = examsR.value.total;
  }
  loading.value = false;
});
</script>

<template>
  <div class="admin-page">
    <h1 class="admin-page-title">仪表盘</h1>

    <Spin :spinning="loading">
      <div class="stat-grid">
        <StatCard title="用户总数" :value="stats.users ?? '—'">
          <template #icon><UserOutlined /></template>
        </StatCard>
        <StatCard title="文献总数" :value="stats.documents ?? '—'">
          <template #icon><FileTextOutlined /></template>
        </StatCard>
        <StatCard title="对话总数" :value="stats.conversations ?? '—'">
          <template #icon><MessageOutlined /></template>
        </StatCard>
        <StatCard title="考试总数" :value="stats.exams ?? '—'">
          <template #icon><FormOutlined /></template>
        </StatCard>
      </div>

      <section class="recent-section">
        <h2 class="section-title">最近注册用户</h2>
        <Table
          v-if="recentUsers.length"
          :data-source="dataSource"
          :columns="columns"
          :pagination="false"
          row-key="id"
          size="middle"
        />
        <Empty v-else description="暂无用户数据" />
      </section>
    </Spin>
  </div>
</template>

<style scoped lang="less">
.stat-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--tcm-spacing-lg);
  margin-bottom: var(--tcm-spacing-xl);
}

@media (max-width: 1024px) {
  .stat-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

.recent-section {
  background-color: #fff;
  border-radius: var(--tcm-radius-lg);
  padding: var(--tcm-spacing-lg);
  box-shadow: var(--tcm-shadow-card);
}

.section-title {
  margin: 0 0 var(--tcm-spacing-base);
  font-size: 16px;
  font-weight: 600;
}
</style>
