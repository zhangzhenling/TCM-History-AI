<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { Table, Pagination, Button, Modal, Form, Input, Switch, Popconfirm, message, Alert } from 'ant-design-vue';
import type { FormInstance } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import { formatDateTime } from '@tcm/shared';
import type { ApiKey, ApiKeyCreateResponse } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const apiKeys = ref<ApiKey[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });
const modalVisible = ref(false);
const modalLoading = ref(false);
const currentId = ref<number | null>(null);

const newKey = ref<string | null>(null);
const formRef = ref<FormInstance>();
const formState = reactive({ name: '', is_active: true });

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: 'Key 前缀', dataIndex: 'key_prefix', key: 'key_prefix' },
  { title: '状态', dataIndex: 'is_active', key: 'is_active', width: 80 },
  { title: '最后使用', dataIndex: 'last_used_at', key: 'last_used_at', width: 180 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 240 },
];

const dataSource = computed(() =>
  apiKeys.value.map((k) => ({
    ...k,
    is_active: k.is_active ? '启用' : '禁用',
    last_used_at: k.last_used_at ? formatDateTime(k.last_used_at) : '—',
    created_at: formatDateTime(k.created_at),
  })),
) as unknown as Record<string, unknown>[];

async function load() {
  loading.value = true;
  try {
    const res = await apis.user.listApiKeys({ page: query.page, page_size: query.page_size });
    apiKeys.value = res.items ?? [];
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

function openCreate() {
  currentId.value = null;
  newKey.value = null;
  formState.name = '';
  formState.is_active = true;
  modalVisible.value = true;
}

function openEdit(record: ApiKey) {
  currentId.value = record.id;
  newKey.value = null;
  formState.name = record.name;
  formState.is_active = record.is_active;
  modalVisible.value = true;
}

async function handleSubmit() {
  modalLoading.value = true;
  try {
    if (currentId.value) {
      await apis.user.updateApiKey(currentId.value, { name: formState.name, is_active: formState.is_active });
      message.success('更新成功');
    } else {
      const res = await apis.user.createApiKey({ name: formState.name, is_active: formState.is_active });
      newKey.value = (res as unknown as ApiKeyCreateResponse).key ?? null;
      message.success('创建成功');
    }
    if (!newKey.value) {
      modalVisible.value = false;
    }
    load();
  } finally {
    modalLoading.value = false;
  }
}

async function handleDelete(id: number) {
  await apis.user.deleteApiKey(id);
  message.success('删除成功');
  load();
}

async function handleRegenerate(id: number) {
  const res = await apis.user.regenerateApiKey(id);
  newKey.value = (res as unknown as ApiKeyCreateResponse).key ?? null;
  if (newKey.value) {
    message.success('重新生成成功');
  }
  load();
}

function closeModal() {
  modalVisible.value = false;
  newKey.value = null;
}
</script>

<template>
  <div class="admin-page">
    <h1 class="admin-page-title">API Key 管理</h1>

    <div class="table-card">
      <div style="margin-bottom: 16px;">
        <Button type="primary" @click="openCreate">创建 API Key</Button>
      </div>

      <Table :data-source="dataSource" :columns="columns" :loading="loading" :pagination="false" row-key="id" size="middle">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'action'">
            <Button type="link" size="small" @click="openEdit(record)">编辑</Button>
            <Button type="link" size="small" @click="handleRegenerate(record.id)">重新生成</Button>
            <Popconfirm title="确定删除此 API Key?" @confirm="handleDelete(record.id)">
              <Button type="link" danger size="small">删除</Button>
            </Popconfirm>
          </template>
        </template>
      </Table>

      <div v-if="total > 0" class="pagination-wrap">
        <Pagination :current="query.page" :page-size="query.page_size" :total="total" show-size-changer :page-size-options="['10', '20', '50']" @change="onPageChange" />
      </div>
    </div>

    <Modal v-model:open="modalVisible" :title="currentId ? '编辑 API Key' : '创建 API Key'" :confirm-loading="modalLoading" @ok="handleSubmit" @cancel="closeModal">
      <Alert v-if="newKey" type="success" show-icon style="margin-bottom: 16px;">
        <template #message>
          <div>API Key 已生成，请立即复制保存，关闭后将无法再次查看：</div>
          <code style="word-break: break-all; font-size: 12px; margin-top: 8px; display: block;">{{ newKey }}</code>
        </template>
      </Alert>

      <Form v-if="!newKey" ref="formRef" :model="formState" layout="vertical">
        <Form.Item label="名称" required>
          <Input v-model:value="formState.name" placeholder="请输入 API Key 名称" />
        </Form.Item>
        <Form.Item label="启用">
          <Switch v-model:checked="formState.is_active" />
        </Form.Item>
      </Form>
    </Modal>
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