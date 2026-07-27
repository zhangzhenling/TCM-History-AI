<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { Table, Pagination, Button, Modal, Form, Input, InputNumber, Select, Popconfirm, message } from 'ant-design-vue';
import type { FormInstance } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import { truncate } from '@tcm/shared';
import type { School, SchoolRequest, Dynasty, Person } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const schools = ref<School[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });
const modalVisible = ref(false);
const modalLoading = ref(false);
const currentId = ref<number | null>(null);

const dynasties = ref<Dynasty[]>([]);
const persons = ref<Person[]>([]);

const formRef = ref<FormInstance>();
const formState = reactive<SchoolRequest>({
  name: '',
  dynasty_id: undefined,
  founder_person_id: undefined,
  summary: '',
  established_year: undefined,
});

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '学派名', dataIndex: 'name', key: 'name' },
  { title: '朝代', dataIndex: 'dynasty_name', key: 'dynasty_name', width: 120 },
  { title: '创始人', dataIndex: 'founder_name', key: 'founder_name', width: 120 },
  { title: '创立年份', dataIndex: 'established_year', key: 'established_year', width: 120 },
  { title: '简介', dataIndex: 'summary', key: 'summary' },
  { title: '操作', dataIndex: 'action', key: 'action', width: 160 },
];

const dynastyMap = computed(() => {
  const map: Record<number, string> = {};
  dynasties.value.forEach((d) => {
    map[d.id] = d.name;
  });
  return map;
});

const personMap = computed(() => {
  const map: Record<number, string> = {};
  persons.value.forEach((p) => {
    map[p.id] = p.name;
  });
  return map;
});

const dataSource = computed(() =>
  schools.value.map((s) => ({
    ...s,
    dynasty_name: dynastyMap.value[s.dynasty_id] || s.dynasty_id || '—',
    founder_name: personMap.value[s.founder_person_id] || s.founder_person_id || '—',
    summary: truncate(s.summary, 40) || '—',
    established_year: s.established_year || '—',
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
    const res = await apis.history.listSchools({ page: query.page, page_size: query.page_size });
    schools.value = res.items ?? [];
    total.value = res.total ?? 0;
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  load();
  loadDynasties();
  loadPersons();
});

function onPageChange(p: number, ps: number) {
  query.page = p;
  query.page_size = ps;
  load();
}

function resetForm() {
  formState.name = '';
  formState.dynasty_id = undefined;
  formState.founder_person_id = undefined;
  formState.summary = '';
  formState.established_year = undefined;
  formRef.value?.clearValidate();
}

function openAddModal() {
  currentId.value = null;
  resetForm();
  modalVisible.value = true;
}

async function openEditModal(record: School) {
  currentId.value = record.id;
  resetForm();
  try {
    const detail = await apis.history.getSchool(record.id);
    formState.name = detail.name;
    formState.dynasty_id = detail.dynasty_id;
    formState.founder_person_id = detail.founder_person_id;
    formState.summary = detail.summary;
    formState.established_year = detail.established_year;
  } catch (e) {
    formState.name = record.name;
    formState.dynasty_id = record.dynasty_id;
    formState.founder_person_id = record.founder_person_id;
    formState.summary = record.summary;
    formState.established_year = record.established_year;
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
      await apis.history.updateSchool(currentId.value, formState);
      message.success('更新成功');
    } else {
      await apis.history.createSchool(formState);
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
    await apis.history.deleteSchool(id);
    message.success('删除成功');
    load();
  } catch (e) {
    // error handled by interceptor
  }
}
</script>

<template>
  <div class="admin-page">
    <h1 class="admin-page-title">学派管理</h1>

    <div class="table-card">
      <div class="toolbar">
        <Button type="primary" @click="openAddModal">新增学派</Button>
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
            <Button type="link" size="small" @click="openEditModal(record as School)">编辑</Button>
            <Popconfirm title="确定删除该学派吗？" @confirm="handleDelete((record as School).id)">
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
      :title="currentId ? '编辑学派' : '新增学派'"
      :confirm-loading="modalLoading"
      @cancel="modalVisible = false"
      @ok="handleOk"
    >
      <Form ref="formRef" :model="formState" layout="vertical">
        <Form.Item label="学派名称" name="name" :rules="[{ required: true, message: '请输入学派名称' }]">
          <Input v-model:value="formState.name" placeholder="请输入学派名称" />
        </Form.Item>
        <Form.Item label="朝代" name="dynasty_id">
          <Select v-model:value="formState.dynasty_id" placeholder="请选择朝代" allow-clear>
            <Select.Option v-for="d in dynasties" :key="d.id" :value="d.id">
              {{ d.name }}
            </Select.Option>
          </Select>
        </Form.Item>
        <Form.Item label="创始人" name="founder_person_id">
          <Select v-model:value="formState.founder_person_id" placeholder="请选择创始人" allow-clear show-search option-filter-prop="children">
            <Select.Option v-for="p in persons" :key="p.id" :value="p.id">
              {{ p.name }}
            </Select.Option>
          </Select>
        </Form.Item>
        <Form.Item label="创立年份" name="established_year">
          <InputNumber v-model:value="formState.established_year" placeholder="请输入创立年份" style="width: 100%" />
        </Form.Item>
        <Form.Item label="简介" name="summary">
          <Input.TextArea v-model:value="formState.summary" placeholder="请输入简介" :rows="4" />
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
