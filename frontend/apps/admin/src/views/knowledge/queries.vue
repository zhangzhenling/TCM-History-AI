<script setup lang="ts">
// 检索日志（Embedding 任务监控）：调用 apis.knowledge.listTasks()，展示任务队列与状态。
import { computed, onMounted, reactive, ref } from 'vue';
import { Table, Pagination, Tag } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import { formatDateTime } from '@tcm/shared';
import type { EmbeddingTask } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const tasks = ref<EmbeddingTask[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10, status: '' as string });

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '文档 ID', dataIndex: 'document_id', key: 'document_id', width: 100 },
  { title: '分片 ID', dataIndex: 'chunk_id', key: 'chunk_id', width: 100 },
  { title: '任务类型', dataIndex: 'task_type', key: 'task_type', width: 120 },
  { title: '阶段', dataIndex: 'stage', key: 'stage', width: 120 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 110 },
  { title: '进度', dataIndex: 'progress', key: 'progress', width: 90 },
  { title: '模型', dataIndex: 'model', key: 'model', width: 160 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
];

const statusColorMap: Record<string, string> = {
  pending: 'default',
  running: 'processing',
  success: 'success',
  failed: 'error',
};

const dataSource = computed(() =>
  tasks.value.map((t) => ({
    ...t,
    task_type: t.task_type || '—',
    stage: t.stage || '—',
    model: t.model || '—',
    progress: `${Math.round((t.progress ?? 0) * 100)}%`,
    created_at: formatDateTime(t.created_at),
  })),
);

async function load() {
  loading.value = true;
  try {
    const res = await apis.knowledge.listTasks({
      page: query.page,
      page_size: query.page_size,
      status: query.status || undefined,
    });
    tasks.value = res.items ?? [];
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
</script>

<template>
  <div class="admin-page">
    <h1 class="admin-page-title">检索日志</h1>

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
