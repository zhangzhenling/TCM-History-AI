<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { Table, Pagination, Button, Modal, Form, Input, InputNumber, Switch, Popconfirm, message } from 'ant-design-vue';
import type { FormInstance } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import type { MembershipPlan, MembershipPlanRequest } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const plans = ref<MembershipPlan[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });
const modalVisible = ref(false);
const modalLoading = ref(false);
const currentId = ref<number | null>(null);

const formRef = ref<FormInstance>();
const formState = reactive<MembershipPlanRequest>({
  name: '',
  description: '',
  price: 0,
  duration_days: 30,
  is_active: true,
});

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '方案名称', dataIndex: 'name', key: 'name' },
  { title: '描述', dataIndex: 'description', key: 'description' },
  { title: '价格', dataIndex: 'price', key: 'price', width: 100 },
  { title: '有效期(天)', dataIndex: 'duration_days', key: 'duration_days', width: 100 },
  { title: '状态', dataIndex: 'is_active', key: 'is_active', width: 100 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 160 },
];

const dataSource = computed<any>(() =>
  plans.value.map((p) => ({
    ...p,
    description: p.description || '—',
    price: `¥${(p.price / 100).toFixed(2)}`,
    is_active: p.is_active ? '启用' : '禁用',
  })),
);

async function load() {
  loading.value = true;
  try {
    const res = await apis.user.listMembershipPlans({ page: query.page, page_size: query.page_size });
    plans.value = res.items ?? [];
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
  formState.price = 0;
  formState.duration_days = 30;
  formState.is_active = true;
  modalVisible.value = true;
}

function openEdit(record: MembershipPlan) {
  currentId.value = record.id;
  formState.name = record.name;
  formState.description = record.description;
  formState.price = record.price;
  formState.duration_days = record.duration_days;
  formState.is_active = record.is_active;
  modalVisible.value = true;
}

async function handleSubmit() {
  modalLoading.value = true;
  try {
    if (currentId.value) {
      await apis.user.updateMembershipPlan(currentId.value, {
        name: formState.name,
        description: formState.description,
        price: formState.price,
        duration_days: formState.duration_days,
        is_active: formState.is_active,
      });
      message.success('更新成功');
    } else {
      await apis.user.createMembershipPlan({
        name: formState.name,
        description: formState.description,
        price: formState.price,
        duration_days: formState.duration_days,
        is_active: formState.is_active,
      });
      message.success('创建成功');
    }
    modalVisible.value = false;
    load();
  } finally {
    modalLoading.value = false;
  }
}

async function handleDelete(id: number) {
  await apis.user.deleteMembershipPlan(id);
  message.success('删除成功');
  load();
}
</script>

<template>
  <div class="admin-page">
    <h1 class="admin-page-title">会员方案管理</h1>

    <div class="table-card">
      <div style="margin-bottom: 16px;">
        <Button type="primary" @click="openCreate">新增方案</Button>
      </div>

      <Table :data-source="dataSource" :columns="columns" :loading="loading" :pagination="false" row-key="id" size="middle">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'action'">
            <Button type="link" size="small" @click="openEdit(record)">编辑</Button>
            <Popconfirm title="确定删除此方案?" @confirm="handleDelete(record.id)">
              <Button type="link" danger size="small">删除</Button>
            </Popconfirm>
          </template>
        </template>
      </Table>

      <div v-if="total > 0" class="pagination-wrap">
        <Pagination :current="query.page" :page-size="query.page_size" :total="total" show-size-changer :page-size-options="['10', '20', '50']" @change="onPageChange" />
      </div>
    </div>

    <Modal v-model:open="modalVisible" :title="currentId ? '编辑方案' : '新增方案'" :confirm-loading="modalLoading" @ok="handleSubmit">
      <Form ref="formRef" :model="formState" layout="vertical">
        <Form.Item label="方案名称" required>
          <Input v-model:value="formState.name" placeholder="请输入方案名称" />
        </Form.Item>
        <Form.Item label="描述">
          <Input v-model:value="formState.description" placeholder="请输入方案描述" />
        </Form.Item>
        <Form.Item label="价格(分)" required>
          <InputNumber v-model:value="formState.price" :min="0" style="width: 100%;" placeholder="价格单位：分" />
        </Form.Item>
        <Form.Item label="有效期(天)" required>
          <InputNumber v-model:value="formState.duration_days" :min="1" style="width: 100%;" />
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