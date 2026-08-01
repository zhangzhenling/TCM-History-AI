<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import {
  Table,
  Pagination,
  Button,
  Modal,
  Form,
  Input,
  Select,
  Popconfirm,
  message,
} from 'ant-design-vue';
import type { FormInstance } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import { truncate } from '@tcm/shared';
import type { Disease, DiseaseRequest } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const diseases = ref<Disease[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });
const modalVisible = ref(false);
const modalLoading = ref(false);
const currentId = ref<number | null>(null);

const formRef = ref<FormInstance>();
const formState = reactive<DiseaseRequest>({
  name: '',
  pinyin: '',
  category: '',
  description: '',
  symptoms: '',
  tcm_pathogenesis: '',
});

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '病名', dataIndex: 'name', key: 'name' },
  { title: '拼音', dataIndex: 'pinyin', key: 'pinyin', width: 140 },
  { title: '分类', dataIndex: 'category', key: 'category', width: 120 },
  { title: '描述', dataIndex: 'description', key: 'description' },
  { title: '症状', dataIndex: 'symptoms', key: 'symptoms' },
  { title: '病机', dataIndex: 'tcm_pathogenesis', key: 'tcm_pathogenesis' },
  { title: '操作', dataIndex: 'action', key: 'action', width: 200, fixed: 'right' as const },
];

const categoryOptions = [
  { value: '外感病', label: '外感病' },
  { value: '内伤病', label: '内伤病' },
  { value: '伤寒', label: '伤寒' },
  { value: '温病', label: '温病' },
  { value: '杂病', label: '杂病' },
  { value: '妇科病', label: '妇科病' },
  { value: '儿科病', label: '儿科病' },
  { value: '外科病', label: '外科病' },
  { value: '皮肤病', label: '皮肤病' },
  { value: '五官病', label: '五官病' },
  { value: '其他', label: '其他' },
];

const dataSource = computed(() =>
  diseases.value.map((d) => ({
    ...d,
    pinyin: d.pinyin || '—',
    category: d.category || '—',
    description: truncate(d.description, 40),
    symptoms: truncate(d.symptoms, 40),
    tcm_pathogenesis: truncate(d.tcm_pathogenesis, 40),
  })),
);

async function load() {
  loading.value = true;
  try {
    const res = await apis.history.listDiseases({ page: query.page, page_size: query.page_size });
    diseases.value = res.items ?? [];
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
  formState.category = '';
  formState.description = '';
  formState.symptoms = '';
  formState.tcm_pathogenesis = '';
  formRef.value?.clearValidate();
}

function openAddModal() {
  currentId.value = null;
  resetForm();
  modalVisible.value = true;
}

async function openEditModal(record: Disease) {
  currentId.value = record.id;
  resetForm();
  try {
    const detail = await apis.history.getDisease(record.id);
    formState.name = detail.name;
    formState.pinyin = detail.pinyin;
    formState.category = detail.category;
    formState.description = detail.description;
    formState.symptoms = detail.symptoms;
    formState.tcm_pathogenesis = detail.tcm_pathogenesis;
  } catch (e) {
    formState.name = record.name;
    formState.pinyin = record.pinyin;
    formState.category = record.category;
    formState.description = record.description;
    formState.symptoms = record.symptoms;
    formState.tcm_pathogenesis = record.tcm_pathogenesis;
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
      await apis.history.updateDisease(currentId.value, formState);
      message.success('更新成功');
    } else {
      await apis.history.createDisease(formState);
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
    await apis.history.deleteDisease(id);
    message.success('删除成功');
    load();
  } catch (e) {
    // error handled by interceptor
  }
}
</script>

<template>
  <div class="admin-page">
    <h1 class="admin-page-title">疾病管理</h1>

    <div class="table-card">
      <div class="toolbar">
        <Button type="primary" @click="openAddModal">新增疾病</Button>
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
            <Button type="link" size="small" @click="openEditModal(record as Disease)">编辑</Button>
            <Popconfirm title="确定删除该疾病吗？" @confirm="handleDelete((record as Disease).id)">
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
      :title="currentId ? '编辑疾病' : '新增疾病'"
      :confirm-loading="modalLoading"
      :width="640"
      @cancel="modalVisible = false"
      @ok="handleOk"
    >
      <Form ref="formRef" :model="formState" layout="vertical">
        <Form.Item label="病名" name="name" :rules="[{ required: true, message: '请输入病名' }]">
          <Input v-model:value="formState.name" placeholder="请输入病名" />
        </Form.Item>
        <div class="form-row">
          <Form.Item label="拼音" name="pinyin">
            <Input v-model:value="formState.pinyin" placeholder="请输入拼音" />
          </Form.Item>
          <Form.Item label="分类" name="category">
            <Select v-model:value="formState.category" placeholder="请选择分类" allow-clear>
              <Select.Option v-for="opt in categoryOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </Select.Option>
            </Select>
          </Form.Item>
        </div>
        <Form.Item label="描述" name="description">
          <Input.TextArea
            v-model:value="formState.description"
            placeholder="请输入疾病描述"
            :rows="3"
          />
        </Form.Item>
        <Form.Item label="症状" name="symptoms">
          <Input.TextArea
            v-model:value="formState.symptoms"
            placeholder="请输入症状表现"
            :rows="3"
          />
        </Form.Item>
        <Form.Item label="病机" name="tcm_pathogenesis">
          <Input.TextArea
            v-model:value="formState.tcm_pathogenesis"
            placeholder="请输入中医病机分析"
            :rows="3"
          />
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
