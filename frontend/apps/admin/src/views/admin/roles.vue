<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { Table, Pagination, Button, Modal, Form, Input, Popconfirm, message } from 'ant-design-vue';
import type { FormInstance } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import type { Role, RoleRequest } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const roles = ref<Role[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });
const modalVisible = ref(false);
const modalLoading = ref(false);
const currentId = ref<number | null>(null);

const formRef = ref<FormInstance>();
const formState = reactive<RoleRequest>({ name: '', description: '' });

const columns: any[] = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '角色名称', dataIndex: 'name', key: 'name' },
  { title: '描述', dataIndex: 'description', key: 'description' },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 200 },
];

const dataSource = computed<any>(() =>
  roles.value.map((r) => ({
    ...r,
    description: r.description || '—',
    created_at: r.created_at?.slice(0, 19).replace('T', ' ') || '—',
  })),
);

async function load() {
  loading.value = true;
  try {
    const res = await apis.user.listRoles({ page: query.page, page_size: query.page_size });
    roles.value = res.items ?? [];
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
  formState.name = '';
  formState.description = '';
  modalVisible.value = true;
}

function openEdit(record: Role) {
  currentId.value = record.id;
  formState.name = record.name;
  formState.description = record.description;
  modalVisible.value = true;
}

async function handleSubmit() {
  modalLoading.value = true;
  try {
    if (currentId.value) {
      await apis.user.updateRole(currentId.value, {
        name: formState.name,
        description: formState.description,
      });
      message.success('更新成功');
    } else {
      await apis.user.createRole({ name: formState.name, description: formState.description });
      message.success('创建成功');
    }
    modalVisible.value = false;
    load();
  } finally {
    modalLoading.value = false;
  }
}

async function handleDelete(id: number) {
  await apis.user.deleteRole(id);
  message.success('删除成功');
  load();
}
</script>

<template>
  <div class="admin-page">
    <h1 class="admin-page-title">角色管理</h1>

    <div class="table-card">
      <div style="margin-bottom: 16px">
        <Button type="primary" @click="openCreate">新增角色</Button>
      </div>

      <Table
        :data-source="dataSource as any"
        :columns="columns"
        :loading="loading"
        :pagination="false"
        row-key="id"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'action'">
            <Button type="link" size="small" @click="openEdit(record as any)">编辑</Button>
            <Popconfirm title="确定删除此角色?" @confirm="handleDelete(record.id)">
              <Button type="link" danger size="small">删除</Button>
            </Popconfirm>
          </template>
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

    <Modal
      v-model:open="modalVisible"
      :title="currentId ? '编辑角色' : '新增角色'"
      :confirm-loading="modalLoading"
      @ok="handleSubmit"
    >
      <Form ref="formRef" :model="formState" layout="vertical">
        <Form.Item label="角色名称" required>
          <Input v-model:value="formState.name" placeholder="请输入角色名称" />
        </Form.Item>
        <Form.Item label="描述">
          <Input v-model:value="formState.description" placeholder="请输入角色描述" />
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
