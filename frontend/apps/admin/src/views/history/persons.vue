<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { Table, Pagination, Button, Modal, Form, Input, InputNumber, Select, Popconfirm, message } from 'ant-design-vue';
import type { FormInstance } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import type { Person, PersonRequest, Dynasty } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const persons = ref<Person[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });
const modalVisible = ref(false);
const modalLoading = ref(false);
const currentId = ref<number | null>(null);

const dynasties = ref<Dynasty[]>([]);

const formRef = ref<FormInstance>();
const formState = reactive<PersonRequest>({
  name: '',
  courtesy_name: '',
  alias_name: '',
  dynasty_id: undefined,
  birth_year: undefined,
  death_year: undefined,
  gender: '',
  title: '',
  biography: '',
  achievements: '',
  portrait_url: '',
});

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '姓名', dataIndex: 'name', key: 'name' },
  { title: '字', dataIndex: 'courtesy_name', key: 'courtesy_name', width: 100 },
  { title: '朝代', dataIndex: 'dynasty_name', key: 'dynasty_name', width: 120 },
  { title: '生卒年', dataIndex: 'year_range', key: 'year_range', width: 160 },
  { title: '称号', dataIndex: 'title', key: 'title' },
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
  persons.value.map((p) => ({
    ...p,
    courtesy_name: p.courtesy_name || '—',
    title: p.title || '—',
    dynasty_name: dynastyMap.value[p.dynasty_id] || p.dynasty_id || '—',
    year_range: `${p.birth_year || '—'} - ${p.death_year || '—'}`,
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
    const res = await apis.history.listPersons({ page: query.page, page_size: query.page_size });
    persons.value = res.items ?? [];
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
  formState.name = '';
  formState.courtesy_name = '';
  formState.alias_name = '';
  formState.dynasty_id = undefined;
  formState.birth_year = undefined;
  formState.death_year = undefined;
  formState.gender = '';
  formState.title = '';
  formState.biography = '';
  formState.achievements = '';
  formState.portrait_url = '';
  formRef.value?.clearValidate();
}

function openAddModal() {
  currentId.value = null;
  resetForm();
  modalVisible.value = true;
}

async function openEditModal(record: Person) {
  currentId.value = record.id;
  resetForm();
  try {
    const detail = await apis.history.getPerson(record.id);
    formState.name = detail.name;
    formState.courtesy_name = detail.courtesy_name;
    formState.alias_name = detail.alias_name;
    formState.dynasty_id = detail.dynasty_id;
    formState.birth_year = detail.birth_year;
    formState.death_year = detail.death_year;
    formState.gender = detail.gender;
    formState.title = detail.title;
    formState.biography = detail.biography;
    formState.achievements = detail.achievements;
    formState.portrait_url = detail.portrait_url;
  } catch (e) {
    formState.name = record.name;
    formState.courtesy_name = record.courtesy_name;
    formState.alias_name = record.alias_name;
    formState.dynasty_id = record.dynasty_id;
    formState.birth_year = record.birth_year;
    formState.death_year = record.death_year;
    formState.gender = record.gender;
    formState.title = record.title;
    formState.biography = record.biography;
    formState.achievements = record.achievements;
    formState.portrait_url = record.portrait_url;
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
      await apis.history.updatePerson(currentId.value, formState);
      message.success('更新成功');
    } else {
      await apis.history.createPerson(formState);
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
    await apis.history.deletePerson(id);
    message.success('删除成功');
    load();
  } catch (e) {
    // error handled by interceptor
  }
}
</script>

<template>
  <div class="admin-page">
    <h1 class="admin-page-title">人物管理</h1>

    <div class="table-card">
      <div class="toolbar">
        <Button type="primary" @click="openAddModal">新增人物</Button>
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
            <Button type="link" size="small" @click="openEditModal(record as Person)">编辑</Button>
            <Popconfirm title="确定删除该人物吗？" @confirm="handleDelete((record as Person).id)">
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
      :title="currentId ? '编辑人物' : '新增人物'"
      :confirm-loading="modalLoading"
      @cancel="modalVisible = false"
      @ok="handleOk"
      :width="600"
    >
      <Form ref="formRef" :model="formState" layout="vertical">
        <Form.Item label="姓名" name="name" :rules="[{ required: true, message: '请输入姓名' }]">
          <Input v-model:value="formState.name" placeholder="请输入姓名" />
        </Form.Item>
        <div class="form-row">
          <Form.Item label="字" name="courtesy_name">
            <Input v-model:value="formState.courtesy_name" placeholder="请输入字" />
          </Form.Item>
          <Form.Item label="号" name="alias_name">
            <Input v-model:value="formState.alias_name" placeholder="请输入号" />
          </Form.Item>
        </div>
        <Form.Item label="朝代" name="dynasty_id">
          <Select v-model:value="formState.dynasty_id" placeholder="请选择朝代" allow-clear>
            <Select.Option v-for="d in dynasties" :key="d.id" :value="d.id">
              {{ d.name }}
            </Select.Option>
          </Select>
        </Form.Item>
        <div class="form-row">
          <Form.Item label="生年" name="birth_year">
            <InputNumber v-model:value="formState.birth_year" placeholder="请输入生年" style="width: 100%" />
          </Form.Item>
          <Form.Item label="卒年" name="death_year">
            <InputNumber v-model:value="formState.death_year" placeholder="请输入卒年" style="width: 100%" />
          </Form.Item>
        </div>
        <div class="form-row">
          <Form.Item label="性别" name="gender">
            <Select v-model:value="formState.gender" placeholder="请选择性别" allow-clear>
              <Select.Option value="男">男</Select.Option>
              <Select.Option value="女">女</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item label="称号" name="title">
            <Input v-model:value="formState.title" placeholder="请输入称号" />
          </Form.Item>
        </div>
        <Form.Item label="生平" name="biography">
          <Input.TextArea v-model:value="formState.biography" placeholder="请输入生平" :rows="3" />
        </Form.Item>
        <Form.Item label="成就" name="achievements">
          <Input.TextArea v-model:value="formState.achievements" placeholder="请输入成就" :rows="3" />
        </Form.Item>
        <Form.Item label="头像URL" name="portrait_url">
          <Input v-model:value="formState.portrait_url" placeholder="请输入头像URL" />
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
