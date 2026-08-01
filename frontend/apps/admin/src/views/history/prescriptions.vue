<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { Table, Pagination, Button, Modal, Form, Input, Select, Popconfirm, message } from 'ant-design-vue';
import type { FormInstance } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import { truncate } from '@tcm/shared';
import type { Prescription, PrescriptionRequest, Dynasty, Book, Person } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const prescriptions = ref<Prescription[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });
const modalVisible = ref(false);
const modalLoading = ref(false);
const currentId = ref<number | null>(null);

const dynasties = ref<Dynasty[]>([]);
const books = ref<Book[]>([]);
const persons = ref<Person[]>([]);

const formRef = ref<FormInstance>();
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
  { title: '朝代', dataIndex: 'dynasty_name', key: 'dynasty_name', width: 120 },
  { title: '分类', dataIndex: 'category', key: 'category', width: 120 },
  { title: '功效', dataIndex: 'indications', key: 'indications' },
  { title: '操作', dataIndex: 'action', key: 'action', width: 160, fixed: 'right' as const },
];

const dynastyMap = computed(() => {
  const map: Record<number, string> = {};
  dynasties.value.forEach((d) => { map[d.id] = d.name; });
  return map;
});

const bookMap = computed(() => {
  const map: Record<number, string> = {};
  books.value.forEach((b) => { map[b.id] = b.title; });
  return map;
});

const personMap = computed(() => {
  const map: Record<number, string> = {};
  persons.value.forEach((p) => { map[p.id] = p.name; });
  return map;
});

const dataSource = computed(() =>
  prescriptions.value.map((p) => ({
    ...p,
    pinyin: p.pinyin || '—',
    category: p.category || '—',
    dynasty_name: dynastyMap.value[p.dynasty_id] || p.dynasty_id || '—',
    source_book_name: bookMap.value[p.source_book_id] || p.source_book_id || '—',
    source_person_name: personMap.value[p.source_person_id] || p.source_person_id || '—',
    indications: truncate(p.indications, 40),
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

async function loadBooks() {
  try {
    const res = await apis.history.listBooks({ page: 1, page_size: 100 });
    books.value = res.items ?? [];
  } catch (e) {
    // ignore
  }
}

async function loadPersons() {
  try {
    const res = await apis.history.listPersons({ page: 1, page_size: 100 });
    persons.value = res.items ?? [];
  } catch (e) {
    // ignore
  }
}

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

onMounted(() => {
  load();
  loadDynasties();
  loadBooks();
  loadPersons();
});

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

function openAddModal() {
  currentId.value = null;
  resetForm();
  modalVisible.value = true;
}

async function openEditModal(record: Prescription) {
  currentId.value = record.id;
  resetForm();
  try {
    const detail = await apis.history.getPrescription(record.id);
    formState.name = detail.name;
    formState.pinyin = detail.pinyin;
    formState.source_book_id = detail.source_book_id;
    formState.source_person_id = detail.source_person_id;
    formState.dynasty_id = detail.dynasty_id;
    formState.composition = detail.composition;
    formState.usage = detail.usage;
    formState.indications = detail.indications;
    formState.category = detail.category;
  } catch (e) {
    formState.name = record.name;
    formState.pinyin = record.pinyin;
    formState.source_book_id = record.source_book_id;
    formState.source_person_id = record.source_person_id;
    formState.dynasty_id = record.dynasty_id;
    formState.composition = record.composition;
    formState.usage = record.usage;
    formState.indications = record.indications;
    formState.category = record.category;
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
      await apis.history.updatePrescription(currentId.value, formState);
      message.success('更新成功');
    } else {
      await apis.history.createPrescription(formState);
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
    await apis.history.deletePrescription(id);
    message.success('删除成功');
    load();
  } catch (e) {
    // error handled by interceptor
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
            <Button type="link" size="small" @click="openEditModal(record as Prescription)">编辑</Button>
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
      :title="currentId ? '编辑方剂' : '新增方剂'"
      :confirm-loading="modalLoading"
      @cancel="modalVisible = false"
      @ok="handleOk"
      :width="680"
    >
      <Form ref="formRef" :model="formState" layout="vertical">
        <Form.Item label="方剂名称" name="name" :rules="[{ required: true, message: '请输入方剂名称' }]">
          <Input v-model:value="formState.name" placeholder="请输入方剂名称" />
        </Form.Item>
        <div class="form-row">
          <Form.Item label="拼音" name="pinyin">
            <Input v-model:value="formState.pinyin" placeholder="请输入拼音" />
          </Form.Item>
          <Form.Item label="朝代" name="dynasty_id">
            <Select v-model:value="formState.dynasty_id" placeholder="请选择朝代" allow-clear show-search option-filter-prop="children">
              <Select.Option v-for="d in dynasties" :key="d.id" :value="d.id">
                {{ d.name }}
              </Select.Option>
            </Select>
          </Form.Item>
        </div>
        <Form.Item label="来源著作" name="source_book_id">
          <Select v-model:value="formState.source_book_id" placeholder="请选择来源著作" allow-clear show-search option-filter-prop="children">
            <Select.Option v-for="b in books" :key="b.id" :value="b.id">
              {{ b.title }}
            </Select.Option>
          </Select>
        </Form.Item>
        <Form.Item label="来源人物" name="source_person_id">
          <Select v-model:value="formState.source_person_id" placeholder="请选择来源人物" allow-clear show-search option-filter-prop="children">
            <Select.Option v-for="p in persons" :key="p.id" :value="p.id">
              {{ p.name }}
            </Select.Option>
          </Select>
        </Form.Item>
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