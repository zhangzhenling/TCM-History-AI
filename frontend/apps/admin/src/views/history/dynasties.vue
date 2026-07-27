<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { Table, Pagination, Button, Modal, Form, Input, InputNumber, Popconfirm, message } from 'ant-design-vue';
import type { FormInstance } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import { truncate } from '@tcm/shared';
import type { Dynasty, DynastyRequest } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const dynasties = ref<Dynasty[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });
const modalVisible = ref(false);
const modalLoading = ref(false);
const currentId = ref<number | null>(null);

const formRef = ref<FormInstance>();
const formState = reactive<DynastyRequest>({
  name: '',
  start_year: undefined,
  end_year: undefined,
  sort_order: undefined,
  description: '',
});

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '朝代名', dataIndex: 'name', key: 'name' },
  { title: '起止年份', dataIndex: 'year_range', key: 'year_range', width: 160 },
  { title: '排序', dataIndex: 'sort_order', key: 'sort_order', width: 80 },
  { title: '描述', dataIndex: 'description', key: 'description' },
  { title: '操作', dataIndex: 'action', key: 'action', width: 160 },
];

const dataSource = computed(() =>
  dynasties.value.map((d) => ({
    ...d,
    year_range: `${d.start_year} - ${d.end_year}`,
    description: truncate(d.description, 40) || '—',
  })),
);

async function load() {
  loading.value = true;
  try {
    const res = await apis.history.listDynasties({ page: query.page, page_size: query.page_size });
    dynasties.value = res.items ?? [];
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

function resetForm() {
  formState.name = '';
  formState.start_year = undefined;
  formState.end_year = undefined;
  formState.sort_order = undefined;
  formState.description = '';
  formRef.value?.clearValidate();
}

function openAddModal() {
  currentId.value = null;
  resetForm();
  modalVisible.value = true;
}

async function openEditModal(record: Dynasty) {
  currentId.value = record.id;
  resetForm();
  try {
    const detail = await apis.history.getDynasty(record.id);
    formState.name = detail.name;
    formState.start_year = detail.start_year;
    formState.end_year = detail.end_year;
    formState.sort_order = detail.sort_order;
    formState.description = detail.description;
  } catch (e) {
    formState.name = record.name;
    formState.start_year = record.start_year;
    formState.end_year = record.end_year;
    formState.sort_order = record.sort_order;
    formState.description = record.description;
  }
  modalVisible.value = true;
}

async function handleOk() {
  try {
    await formRef.value?.validate();
  } catch (e) {
    return;
  }

  modalLoading.value = true;
  try {
    if (currentId.value) {
      await apis.history.updateDynasty(currentId.value, formState);
      message.success('更新成功');
    } else {
      await apis.history.createDynasty(formState);
      message.success('创建成功');
    }
    modalVisible.value = false;
    load();
  } finally {
    modalLoading.value = false;
  }
}

async function handleDelete(id: number) {
  try {
    await apis.history.deleteDynasty(id);
    message.success('删除成功');
    load();
  } catch (e) {
    // error handled by interceptor
  }
}
</script>

<template>
  <div class="admin-page">
    <h1 class="admin-page-title">朝代管理</h1>

    <div class="table-card">
      <div class="toolbar">
        <Button type="primary" @click="openAddModal">新增朝代</Button>
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
          <template v-if="column.dataIndex === 'action'">
            <Button type="link" size="small" @click="openEditModal(record as Dynasty)">编辑</Button>
            <Popconfirm title="确定删除该朝代吗？" @confirm="handleDelete((record as Dynasty).id)">
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

    <Modal
      :open="modalVisible"
      :title="currentId ? '编辑朝代' : '新增朝代'"
      :confirm-loading="modalLoading"
      @cancel="modalVisible = false"
      @ok="handleOk"
    >
      <Form ref="formRef" :model="formState" layout="vertical">
        <Form.Item label="朝代名称" name="name" :rules="[{ required: true, message: '请输入朝代名称' }]">
          <Input v-model:value="formState.name" placeholder="请输入朝代名称" />
        </Form.Item>
        <Form.Item label="起始年份" name="start_year">
          <InputNumber v-model:value="formState.start_year" placeholder="请输入起始年份" style="width: 100%" />
        </Form.Item>
        <Form.Item label="结束年份" name="end_year">
          <InputNumber v-model:value="formState.end_year" placeholder="请输入结束年份" style="width: 100%" />
        </Form.Item>
        <Form.Item label="排序" name="sort_order">
          <InputNumber v-model:value="formState.sort_order" placeholder="请输入排序值" style="width: 100%" />
        </Form.Item>
        <Form.Item label="描述" name="description">
          <Input.TextArea v-model:value="formState.description" placeholder="请输入描述" :rows="4" />
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

.toolbar {
  margin-bottom: var(--tcm-spacing-base);
}

.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--tcm-spacing-lg);
}
</style>
