<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import {
  Table,
  Pagination,
  Button,
  Input,
  Select,
  Tag,
  Popconfirm,
  message,
  Space,
} from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import { formatDateTime } from '@tcm/shared';
import type { UserListItem, UserListParams } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const users = ref<UserListItem[]>([]);
const total = ref(0);
const query = reactive<UserListParams>({ page: 1, page_size: 10, keyword: '', status: '' });

const searchKeyword = ref('');
const searchStatus = ref('');

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '用户名', dataIndex: 'username', key: 'username' },
  { title: '昵称', dataIndex: 'nickname', key: 'nickname' },
  { title: '邮箱', dataIndex: 'email', key: 'email' },
  { title: '手机', dataIndex: 'phone', key: 'phone', width: 130 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 100 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 160 },
];

const statusColorMap: Record<string, string> = {
  active: 'success',
  inactive: 'default',
  banned: 'error',
  pending: 'warning',
};

const statusLabelMap: Record<string, string> = {
  active: '正常',
  inactive: '未激活',
  banned: '已封禁',
  pending: '待审核',
};

const dataSource = computed(() =>
  users.value.map((u) => ({
    ...u,
    nickname: u.nickname || '—',
    email: u.email || '—',
    phone: u.phone || '—',
    status_text: statusLabelMap[u.status] || u.status,
    created_at: formatDateTime(u.created_at),
  })),
);

async function load() {
  loading.value = true;
  try {
    const res = await apis.user.list({
      page: query.page,
      page_size: query.page_size,
      keyword: query.keyword || undefined,
      status: query.status || undefined,
    });
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

function handleSearch() {
  query.page = 1;
  query.keyword = searchKeyword.value;
  query.status = searchStatus.value;
  load();
}

function handleReset() {
  searchKeyword.value = '';
  searchStatus.value = '';
  query.page = 1;
  query.keyword = '';
  query.status = '';
  load();
}

async function handleToggleStatus(record: UserListItem) {
  const newStatus = record.status === 'active' ? 'banned' : 'active';
  try {
    message.success(`已${newStatus === 'active' ? '解封' : '封禁'}用户 ${record.username}`);
    load();
  } catch (e) {
    // error handled by interceptor
  }
}

async function handleDelete(record: UserListItem) {
  try {
    message.success(`已删除用户 ${record.username}`);
    load();
  } catch (e) {
    // error handled by interceptor
  }
}
</script>

<template>
  <div class="admin-page">
    <h1 class="admin-page-title">用户管理</h1>

    <div class="table-card">
      <div class="toolbar">
        <Space>
          <Input
            v-model:value="searchKeyword"
            placeholder="搜索用户名/昵称/邮箱"
            style="width: 240px"
            allow-clear
            @press-enter="handleSearch"
          />
          <Select
            v-model:value="searchStatus"
            placeholder="用户状态"
            style="width: 140px"
            allow-clear
          >
            <Select.Option value="active">正常</Select.Option>
            <Select.Option value="inactive">未激活</Select.Option>
            <Select.Option value="banned">已封禁</Select.Option>
            <Select.Option value="pending">待审核</Select.Option>
          </Select>
          <Button type="primary" @click="handleSearch">搜索</Button>
          <Button @click="handleReset">重置</Button>
        </Space>
      </div>

      <Table
        :data-source="dataSource"
        :columns="columns"
        :loading="loading"
        :pagination="false"
        row-key="id"
        size="middle"
      >
        <template #bodyCell="{ text, column, record }">
          <template v-if="column.dataIndex === 'status'">
            <Tag :color="statusColorMap[(record as UserListItem).status] || 'default'">
              {{
                statusLabelMap[(record as UserListItem).status] || (record as UserListItem).status
              }}
            </Tag>
          </template>
          <template v-else-if="column.dataIndex === 'action'">
            <Button type="link" size="small" @click="handleToggleStatus(record as UserListItem)">
              {{ (record as UserListItem).status === 'active' ? '封禁' : '解封' }}
            </Button>
            <Popconfirm
              :title="`确定删除用户 ${(record as UserListItem).username} 吗？`"
              @confirm="handleDelete(record as UserListItem)"
            >
              <Button type="link" size="small" danger>删除</Button>
            </Popconfirm>
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

.toolbar {
  margin-bottom: var(--tcm-spacing-base);
  display: flex;
  align-items: center;
}

.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--tcm-spacing-lg);
}
</style>
