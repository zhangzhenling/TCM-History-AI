<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { Table, Pagination, Button, Modal, Form, Input, Select, Popconfirm, message } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import { truncate } from '@tcm/shared';
import type { Prescription, PrescriptionRequest } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const prescriptions = ref<Prescription[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });
const modalVisible = ref(false);
const modalLoading = ref(false);
const currentId = ref<number | null>(null);

const formRef = ref();
const formState = reactive<PrescriptionRequest>({
  name: '',
  pinyin: '',
  source_book_id: undefined,
  source_person_id: undefined,
  dynasty_id: undefined,
  composition: '',
  usage: '',
  indications: '',
  category: '',
});

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '方剂名', dataIndex: 'name', key: 'name' },
  { title: '拼音', dataIndex: 'pinyin', key: 'pinyin', width: 140 },
  { title: '来源著作', dataIndex: 'source_book_id', key: 'source_book_id', width: 100 },
  { title: '来源人物', dataIndex: 'source_person_id', key: 'source_person_id', width: 100 },
  { title: '朝代', dataIndex: 'dynasty_id', key: 'dynasty_id', width: 100 },
  { title: '分类', dataIndex: 'category', key: 'category', width: 120 },
  { title: '功效', dataIndex: 'indications', key: 'indications' },
  { title: '操作', dataIndex: 'action', key: 'action', width: 160, fixed: 'right' as const },
];

const dataSource = computed(() =>
  prescriptions.value.map((p) => ({
    ...p,
    pinyin: p.pinyin || '—',
    category: p.category || '—',
    indications: truncate(p.indications, 40),
  })),
);

const isEdit = computed(() => currentId.value !== null);
const modalTitle = computed(() => (isEdit.value ? '编辑方剂' : '新增方剂'));

async function load() {
  loading.value = true;
  try {
    const res = await apis.history.listPrescriptions({ page: query.page, page_size: query.page_size });
    prescriptions.value = res.items ?? [];
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
  formState.pinyin = '';
  formState.source_book_id = undefined;
  formState.source_person_id = undefined;
  formState.dynasty_id = undefined;
  formState.composition = '';
  formState.usage = '';
  formState.indications = '';
  formState.category = '';
  formRef.value?.clearValidate();
}

async function openAddModal() {
  currentId.value = null;
  resetForm();
  modalVisible.value = true;
}

async function openEditModal(id: number) {
  currentId.value = id;
  resetForm();
  modalLoading.value = true;
  modalVisible.value = true;
  try {
    const data = await apis.history.getPrescription(id);
    formState.name = data.name;
    formState.pinyin = data.pinyin;
    formState.source_book_id = data.source_book_id;
    formState.source_person_id = data.source_person_id;
    formState.dynasty_id = data.dynasty_id;
    formState.composition = data.composition;
    formState.usage = data.usage;
    formState.indications = data.indications;
    formState.category = data.category;
  } finally {
    modalLoading.value = false;
  }
}

async function handleOk() {
  try {
    await formRef.value.validate();
  } catch {
    return;
  }
  modalLoading.value = true;
  try {
    if (isEdit.value && currentId.value) {
      await apis.history.updatePrescription(currentId.value, formState);
      message.success('编辑成功');
    } else {
      await apis.history.createPrescription(formState);
      message.success('新增成功');
    }
    modalVisible.value = false;
    load();
  } finally {
    modalLoading.value = false;
  }
}

async function handleDelete(id: number) {
  try {
    await apis.history.deletePrescription(id);
    message.success('删除成功');
    load();
  } catch {
    message.error('删除失败');
  }
}
</script>

<template>
  <div class="admin-page">
    <h1 class="admin-page-title">方剂管理</h1>

    <div class="table-card">
      <div class="toolbar">
        <Button type="primary" @click="openAddModal">新增方剂</Button>
      </div>
      <Table
        :data-source="dataSource"
        :columns="columns"
        :loading="loading"
        :pagination="false"
        row-key="id"
        size="middle"
        :scroll="{ x: 1200 }"
      >
        <template #bodyCell="{ text, column, record }">
          <template v-if="column.dataIndex === 'action'">
            <Button type="link" size="small" @click="openEditModal((record as Prescription).id)">编辑</Button>
            <Popconfirm title="确定删除该方剂吗？" @confirm="handleDelete((record as Prescription).id)">
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
      :title="modalTitle"
      :confirm-loading="modalLoading"
      @cancel="modalVisible = false"
      @ok="handleOk"
      width="640px"
    >
      <Form ref="formRef" layout="vertical" :model="formState">
        <Form.Item label="方剂名称" name="name" :rules="[{ required: true, message: '请输入方剂名称' }]">
          <Input v-model:value="formState.name" placeholder="请输入方剂名称" />
        </Form.Item>
        <Form.Item label="拼音" name="pinyin">
          <Input v-model:value="formState.pinyin" placeholder="请输入拼音" />
        </Form.Item>
        <div class="form-row">
          <Form.Item label="来源著作ID" name="source_book_id">
            <InputNumber v-model:value="formState.source_book_id" placeholder="著作ID" :min="1" style="width: 100%" />
          </Form.Item>
          <Form.Item label="来源人物ID" name="source_person_id">
            <InputNumber v-model:value="formState.source_person_id" placeholder="人物ID" :min="1" style="width: 100%" />
          </Form.Item>
          <Form.Item label="朝代ID" name="dynasty_id">
            <InputNumber v-model:value="formState.dynasty_id" placeholder="朝代ID" :min="1" style="width: 100%" />
          </Form.Item>
        </div>
        <Form.Item label="分类" name="category">
          <Select v-model:value="formState.category" placeholder="请选择分类" allow-clear>
            <Select.Option value="解表剂">解表剂</Select.Option>
            <Select.Option value="清热剂">清热剂</Select.Option>
            <Select.Option value="泻下剂">泻下剂</Select.Option>
            <Select.Option value="和解剂">和解剂</Select.Option>
            <Select.Option value="温里剂">温里剂</Select.Option>
            <Select.Option value="补益剂">补益剂</Select.Option>
            <Select.Option value="固涩剂">固涩剂</Select.Option>
            <Select.Option value="安神剂">安神剂</Select.Option>
            <Select.Option value="开窍剂">开窍剂</Select.Option>
            <Select.Option value="理气剂">理气剂</Select.Option>
            <Select.Option value="理血剂">理血剂</Select.Option>
            <Select.Option value="治风剂">治风剂</Select.Option>
            <Select.Option value="治燥剂">治燥剂</Select.Option>
            <Select.Option value="祛湿剂">祛湿剂</Select.Option>
            <Select.Option value="祛痰剂">祛痰剂</Select.Option>
            <Select.Option value="消食剂">消食剂</Select.Option>
            <Select.Option value="驱虫剂">驱虫剂</Select.Option>
            <Select.Option value="涌吐剂">涌吐剂</Select.Option>
            <Select.Option value="其他">其他</Select.Option>
          </Select>
        </Form.Item>
        <Form.Item label="组成" name="composition">
          <Input.TextArea v-model:value="formState.composition" placeholder="请输入方剂组成" :rows="3" />
        </Form.Item>
        <Form.Item label="用法" name="usage">
          <Input.TextArea v-model:value="formState.usage" placeholder="请输入用法用量" :rows="2" />
        </Form.Item>
        <Form.Item label="主治" name="indications">
          <Input.TextArea v-model:value="formState.indications" placeholder="请输入主治功效" :rows="3" />
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

.form-row {
  display: flex;
  gap: var(--tcm-spacing-base);

  .ant-form-item {
    flex: 1;
  }
}
</style>
