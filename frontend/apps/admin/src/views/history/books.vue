<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import {
  Table,
  Pagination,
  Button,
  Modal,
  Form,
  Input,
  InputNumber,
  Select,
  Switch,
  Popconfirm,
  message,
} from 'ant-design-vue';
import type { FormInstance } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import { truncate } from '@tcm/shared';
import type { Book, BookRequest, Dynasty } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const books = ref<Book[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });
const modalVisible = ref(false);
const modalLoading = ref(false);
const currentId = ref<number | null>(null);

const dynasties = ref<Dynasty[]>([]);

const formRef = ref<FormInstance>();
const formState = reactive<BookRequest>({
  title: '',
  dynasty_id: undefined,
  published_year: undefined,
  category: '',
  summary: '',
  volume_count: undefined,
  is_extant: true,
  file_url: '',
});

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '书名', dataIndex: 'title', key: 'title' },
  { title: '朝代', dataIndex: 'dynasty_name', key: 'dynasty_name', width: 120 },
  { title: '成书年', dataIndex: 'published_year', key: 'published_year', width: 100 },
  { title: '分类', dataIndex: 'category', key: 'category', width: 120 },
  { title: '卷数', dataIndex: 'volume_count', key: 'volume_count', width: 80 },
  { title: '存世', dataIndex: 'is_extant_text', key: 'is_extant_text', width: 80 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 160 },
];

const dynastyMap = computed(() => {
  const map: Record<number, string> = {};
  dynasties.value.forEach((d) => {
    map[d.id] = d.name;
  });
  return map;
});

const dataSource = computed(() =>
  books.value.map((b) => ({
    ...b,
    category: b.category || '—',
    dynasty_name: dynastyMap.value[b.dynasty_id] || b.dynasty_id || '—',
    is_extant_text: b.is_extant ? '是' : '否',
    summary: truncate(b.summary, 40) || '—',
  })),
);

async function loadDynasties() {
  try {
    const res = await apis.history.listDynasties({ page: 1, page_size: 100 });
    dynasties.value = res.items ?? [];
  } catch (e) {
    // ignore
  }
}

async function load() {
  loading.value = true;
  try {
    const res = await apis.history.listBooks({ page: query.page, page_size: query.page_size });
    books.value = res.items ?? [];
    total.value = res.total ?? 0;
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  load();
  loadDynasties();
});

function onPageChange(p: number, ps: number) {
  query.page = p;
  query.page_size = ps;
  load();
}

function resetForm() {
  formState.title = '';
  formState.dynasty_id = undefined;
  formState.published_year = undefined;
  formState.category = '';
  formState.summary = '';
  formState.volume_count = undefined;
  formState.is_extant = true;
  formState.file_url = '';
  formRef.value?.clearValidate();
}

function openAddModal() {
  currentId.value = null;
  resetForm();
  modalVisible.value = true;
}

async function openEditModal(record: Book) {
  currentId.value = record.id;
  resetForm();
  try {
    const detail = await apis.history.getBook(record.id);
    formState.title = detail.title;
    formState.dynasty_id = detail.dynasty_id;
    formState.published_year = detail.published_year;
    formState.category = detail.category;
    formState.summary = detail.summary;
    formState.volume_count = detail.volume_count;
    formState.is_extant = detail.is_extant;
    formState.file_url = detail.file_url;
  } catch (e) {
    formState.title = record.title;
    formState.dynasty_id = record.dynasty_id;
    formState.published_year = record.published_year;
    formState.category = record.category;
    formState.summary = record.summary;
    formState.volume_count = record.volume_count;
    formState.is_extant = record.is_extant;
    formState.file_url = record.file_url;
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
      await apis.history.updateBook(currentId.value, formState);
      message.success('更新成功');
    } else {
      await apis.history.createBook(formState);
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
    await apis.history.deleteBook(id);
    message.success('删除成功');
    load();
  } catch (e) {
    // error handled by interceptor
  }
}
</script>

<template>
  <div class="admin-page">
    <h1 class="admin-page-title">著作管理</h1>

    <div class="table-card">
      <div class="toolbar">
        <Button type="primary" @click="openAddModal">新增著作</Button>
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
            <Button type="link" size="small" @click="openEditModal(record as Book)">编辑</Button>
            <Popconfirm title="确定删除该著作吗？" @confirm="handleDelete((record as Book).id)">
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
      :title="currentId ? '编辑著作' : '新增著作'"
      :confirm-loading="modalLoading"
      :width="600"
      @cancel="modalVisible = false"
      @ok="handleOk"
    >
      <Form ref="formRef" :model="formState" layout="vertical">
        <Form.Item label="书名" name="title" :rules="[{ required: true, message: '请输入书名' }]">
          <Input v-model:value="formState.title" placeholder="请输入书名" />
        </Form.Item>
        <div class="form-row">
          <Form.Item label="朝代" name="dynasty_id">
            <Select v-model:value="formState.dynasty_id" placeholder="请选择朝代" allow-clear>
              <Select.Option v-for="d in dynasties" :key="d.id" :value="d.id">
                {{ d.name }}
              </Select.Option>
            </Select>
          </Form.Item>
          <Form.Item label="成书年" name="published_year">
            <InputNumber
              v-model:value="formState.published_year"
              placeholder="请输入成书年"
              style="width: 100%"
            />
          </Form.Item>
        </div>
        <div class="form-row">
          <Form.Item label="分类" name="category">
            <Input v-model:value="formState.category" placeholder="请输入分类" />
          </Form.Item>
          <Form.Item label="卷数" name="volume_count">
            <InputNumber
              v-model:value="formState.volume_count"
              placeholder="请输入卷数"
              style="width: 100%"
              :min="0"
            />
          </Form.Item>
        </div>
        <Form.Item label="是否存世" name="is_extant" value-prop-name="checked">
          <Switch
            v-model:checked="formState.is_extant"
            checked-children="是"
            un-checked-children="否"
          />
        </Form.Item>
        <Form.Item label="简介" name="summary">
          <Input.TextArea v-model:value="formState.summary" placeholder="请输入简介" :rows="4" />
        </Form.Item>
        <Form.Item label="文件URL" name="file_url">
          <Input v-model:value="formState.file_url" placeholder="请输入文件URL" />
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
