<script setup lang="ts">
// 用户管理列表：调用 apis.user.list()，展示 ID/用户名/昵称/邮箱/状态/创建时间 + 分页。
import { computed, onMounted, reactive, ref } from 'vue';
import { Table, Pagination } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import { formatDateTime } from '@tcm/shared';
import type { UserListItem } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const users = ref<UserListItem[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '用户名', dataIndex: 'username', key: 'username' },
  { title: '昵称', dataIndex: 'nickname', key: 'nickname' },
  { title: '邮箱', dataIndex: 'email', key: 'email' },
  { title: '状态', dataIndex: 'status', key: 'status', width: 100 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
];

const dataSource = computed(() =>
  users.value.map((u) => ({
    ...u,
    nickname: u.nickname || '—',
    email: u.email || '—',
    created_at: formatDateTime(u.created_at),
  })),
);

async function load() {
  loading.value = true;
  try {
    const res = await apis.user.list({ page: query.page, page_size: query.page_size });
    users.value = res.items ?? [];
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
    <h1 class="admin-page-title">用户管理</h1>

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
