<script setup lang="ts">
// 文献管理列表：调用 apis.knowledge.listDocuments()，表格展示。
import { computed, onMounted, reactive, ref } from 'vue';
import { Table, Pagination } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import { formatDateTime } from '@tcm/shared';
import type { Document } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const documents = ref<Document[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '编号', dataIndex: 'classic_code', key: 'classic_code', width: 120 },
  { title: '标题', dataIndex: 'title', key: 'title' },
  { title: '朝代', dataIndex: 'dynasty', key: 'dynasty', width: 100 },
  { title: '作者', dataIndex: 'author', key: 'author', width: 140 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 100 },
  { title: '分片数', dataIndex: 'chunk_count', key: 'chunk_count', width: 90 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
];

const dataSource = computed(() =>
  documents.value.map((d) => ({
    ...d,
    dynasty: d.dynasty || '—',
    author: d.author || '—',
    status: d.status || '—',
    created_at: formatDateTime(d.created_at),
  })),
);

async function load() {
  loading.value = true;
  try {
    const res = await apis.knowledge.listDocuments({
      page: query.page,
      page_size: query.page_size,
    });
    documents.value = res.items ?? [];
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
    <h1 class="admin-page-title">文献管理</h1>

    <div class="table-card">
      <Table
        :data-source="dataSource"
        :columns="columns"
        :loading="loading"
        :pagination="false"
        row-key="id"
        size="middle"
      />
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
