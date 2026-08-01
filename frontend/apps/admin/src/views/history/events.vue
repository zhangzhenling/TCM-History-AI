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
  Popconfirm,
  message,
} from 'ant-design-vue';
import type { FormInstance } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import { truncate } from '@tcm/shared';
import type { HistoryEvent, EventRequest, Dynasty } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const events = ref<HistoryEvent[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });
const modalVisible = ref(false);
const modalLoading = ref(false);
const currentId = ref<number | null>(null);

const dynasties = ref<Dynasty[]>([]);

const eventTypeOptions = [
  { label: '医学事件', value: 'medical' },
  { label: '政治事件', value: 'political' },
  { label: '文化事件', value: 'cultural' },
  { label: '战争事件', value: 'war' },
  { label: '其他', value: 'other' },
];

const formRef = ref<FormInstance>();
const formState = reactive<EventRequest>({
  title: '',
  dynasty_id: undefined,
  occurred_year: undefined,
  event_type: '',
  description: '',
  impact: '',
  location: '',
});

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '事件标题', dataIndex: 'title', key: 'title' },
  { title: '朝代', dataIndex: 'dynasty_name', key: 'dynasty_name', width: 120 },
  { title: '发生年份', dataIndex: 'occurred_year', key: 'occurred_year', width: 120 },
  { title: '类型', dataIndex: 'event_type_label', key: 'event_type_label', width: 120 },
  { title: '描述', dataIndex: 'description', key: 'description' },
  { title: '地点', dataIndex: 'location', key: 'location', width: 120 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 160 },
];

const dynastyMap = computed(() => {
  const map: Record<number, string> = {};
  dynasties.value.forEach((d) => {
    map[d.id] = d.name;
  });
  return map;
});

const eventTypeMap = computed(() => {
  const map: Record<string, string> = {};
  eventTypeOptions.forEach((o) => {
    map[o.value] = o.label;
  });
  return map;
});

const dataSource = computed(() =>
  events.value.map((e) => ({
    ...e,
    dynasty_name: dynastyMap.value[e.dynasty_id] || e.dynasty_id || '—',
    event_type_label: eventTypeMap.value[e.event_type] || e.event_type || '—',
    description: truncate(e.description, 40) || '—',
    location: e.location || '—',
    occurred_year: e.occurred_year || '—',
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
    const res = await apis.history.listEvents({ page: query.page, page_size: query.page_size });
    events.value = res.items ?? [];
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
  formState.occurred_year = undefined;
  formState.event_type = '';
  formState.description = '';
  formState.impact = '';
  formState.location = '';
  formRef.value?.clearValidate();
}

function openAddModal() {
  currentId.value = null;
  resetForm();
  modalVisible.value = true;
}

async function openEditModal(record: HistoryEvent) {
  currentId.value = record.id;
  resetForm();
  try {
    const detail = await apis.history.getEvent(record.id);
    formState.title = detail.title;
    formState.dynasty_id = detail.dynasty_id;
    formState.occurred_year = detail.occurred_year;
    formState.event_type = detail.event_type;
    formState.description = detail.description;
    formState.impact = detail.impact;
    formState.location = detail.location;
  } catch (e) {
    formState.title = record.title;
    formState.dynasty_id = record.dynasty_id;
    formState.occurred_year = record.occurred_year;
    formState.event_type = record.event_type;
    formState.description = record.description;
    formState.impact = record.impact;
    formState.location = record.location;
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
      await apis.history.updateEvent(currentId.value, formState);
      message.success('更新成功');
    } else {
      await apis.history.createEvent(formState);
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
    await apis.history.deleteEvent(id);
    message.success('删除成功');
    load();
  } catch (e) {
    // error handled by interceptor
  }
}
</script>

<template>
  <div class="admin-page">
    <h1 class="admin-page-title">事件管理</h1>

    <div class="table-card">
      <div class="toolbar">
        <Button type="primary" @click="openAddModal">新增事件</Button>
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
            <Button type="link" size="small" @click="openEditModal(record as HistoryEvent)"
              >编辑</Button
            >
            <Popconfirm
              title="确定删除该事件吗？"
              @confirm="handleDelete((record as HistoryEvent).id)"
            >
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
      :title="currentId ? '编辑事件' : '新增事件'"
      :confirm-loading="modalLoading"
      :width="600"
      @cancel="modalVisible = false"
      @ok="handleOk"
    >
      <Form ref="formRef" :model="formState" layout="vertical">
        <Form.Item
          label="事件标题"
          name="title"
          :rules="[{ required: true, message: '请输入事件标题' }]"
        >
          <Input v-model:value="formState.title" placeholder="请输入事件标题" />
        </Form.Item>
        <Form.Item label="朝代" name="dynasty_id">
          <Select v-model:value="formState.dynasty_id" placeholder="请选择朝代" allow-clear>
            <Select.Option v-for="d in dynasties" :key="d.id" :value="d.id">
              {{ d.name }}
            </Select.Option>
          </Select>
        </Form.Item>
        <Form.Item label="发生年份" name="occurred_year">
          <InputNumber
            v-model:value="formState.occurred_year"
            placeholder="请输入发生年份"
            style="width: 100%"
          />
        </Form.Item>
        <Form.Item
          label="事件类型"
          name="event_type"
          :rules="[{ required: true, message: '请选择事件类型' }]"
        >
          <Select v-model:value="formState.event_type" placeholder="请选择事件类型">
            <Select.Option v-for="o in eventTypeOptions" :key="o.value" :value="o.value">
              {{ o.label }}
            </Select.Option>
          </Select>
        </Form.Item>
        <Form.Item label="地点" name="location">
          <Input v-model:value="formState.location" placeholder="请输入地点" />
        </Form.Item>
        <Form.Item label="描述" name="description">
          <Input.TextArea
            v-model:value="formState.description"
            placeholder="请输入描述"
            :rows="3"
          />
        </Form.Item>
        <Form.Item label="影响" name="impact">
          <Input.TextArea v-model:value="formState.impact" placeholder="请输入影响" :rows="3" />
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
