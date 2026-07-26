<script setup lang="ts">
// 对话记录列表：调用 apis.ai.listConversations()，表格展示 + 详情抽屉占位。
import { computed, onMounted, reactive, ref } from 'vue';
import { Table, Pagination, Button, Tag } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import { formatDateTime } from '@tcm/shared';
import type { Conversation } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const conversations = ref<Conversation[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });
const detailVisible = ref(false);

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '标题', dataIndex: 'title', key: 'title' },
  { title: '用户 ID', dataIndex: 'user_id', key: 'user_id', width: 100 },
  { title: '模式', dataIndex: 'mode', key: 'mode', width: 100 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 100 },
  { title: '消息数', dataIndex: 'message_count', key: 'message_count', width: 90 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
  { title: '更新时间', dataIndex: 'updated_at', key: 'updated_at', width: 180 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 100 },
];

const statusColorMap: Record<string, string> = {
  active: 'success',
  archived: 'default',
};

const dataSource = computed(() =>
  conversations.value.map((c) => ({
    ...c,
    title: c.title || '—',
    created_at: formatDateTime(c.created_at),
    updated_at: formatDateTime(c.updated_at),
  })),
);

async function load() {
  loading.value = true;
  try {
    const res = await apis.ai.listConversations({ page: query.page, page_size: query.page_size });
    conversations.value = res.items ?? [];
    total.value = res.total ?? 0;
  } finally {
    loading.value = false;
  }
}

onMounted(load);

function onPageChange(p: number, ps: number) {
  query.page = p;
  query.page_size = ps;
  load();
}

function openDetail() {
  detailVisible.value = true;
}
</script>

<template>
  <div class="admin-page">
    <h1 class="admin-page-title">对话记录</h1>

    <div class="table-card">
      <Table
        :data-source="dataSource"
        :columns="columns"
        :loading="loading"
        :pagination="false"
        row-key="id"
        size="middle"
      >
        <template #bodyCell="{ text, column }">
          <template v-if="column.dataIndex === 'status'">
            <Tag :color="statusColorMap[text] || 'default'">{{ text || '—' }}</Tag>
          </template>
          <template v-else-if="column.dataIndex === 'action'">
            <Button type="link" size="small" @click="openDetail">查看</Button>
          </template>
          <template v-else>{{ text }}</template>
        </template>
      </Table>
      <div v-if="total > 0" class="pagination-wrap">
        <Pagination
          :current="query.page"
          :page-size="query.page_size"
          :total="total"
          show-size-changer
          :page-size-options="['10', '20', '50']"
          @change="onPageChange"
        />
      </div>
    </div>
  </div>
</template>

<style scoped lang="less">
.table-card {
  background-color: #fff;
  border-radius: var(--tcm-radius-lg);
  padding: var(--tcm-spacing-lg);
  box-shadow: var(--tcm-shadow-card);
}

.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--tcm-spacing-lg);
}
</style>
