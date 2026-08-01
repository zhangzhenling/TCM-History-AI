<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { Table, Pagination, Button, Modal, Form, Input, InputNumber, Select, Switch, Tag, Popconfirm, message } from 'ant-design-vue';
import type { FormInstance } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import { formatDateTime, truncate } from '@tcm/shared';
import type { PromptTemplate, PromptTemplateRequest } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const prompts = ref<PromptTemplate[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });
const modalVisible = ref(false);
const modalLoading = ref(false);
const currentId = ref<number | null>(null);

const formRef = ref<FormInstance>();
const formState = reactive<PromptTemplateRequest>({
  name: '',
  scene: 'chat',
  system_prompt: '',
  template: '',
  variables_json: {},
  model: '',
  temperature: 0.7,
  max_tokens: 2048,
  top_p: 0.9,
  is_active: true,
  version: 1,
});

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '场景', dataIndex: 'scene', key: 'scene', width: 110 },
  { title: '模型', dataIndex: 'model', key: 'model', width: 140 },
  { title: '温度', dataIndex: 'temperature', key: 'temperature', width: 80 },
  { title: '版本', dataIndex: 'version', key: 'version', width: 80 },
  { title: '启用', dataIndex: 'is_active', key: 'is_active', width: 80 },
  { title: 'System Prompt', dataIndex: 'system_prompt', key: 'system_prompt' },
  { title: '更新时间', dataIndex: 'updated_at', key: 'updated_at', width: 180 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 200 },
];

const sceneColorMap: Record<string, string> = {
  chat: 'blue',
  agent: 'purple',
  reasoning: 'cyan',
  summarize: 'gold',
};

const sceneLabelMap: Record<string, string> = {
  chat: '对话',
  agent: 'Agent',
  reasoning: '推理',
  summarize: '摘要',
};

const dataSource = computed(() =>
  prompts.value.map((p) => ({
    ...p,
    model: p.model || '—',
    system_prompt: truncate(p.system_prompt, 60),
    updated_at: formatDateTime(p.updated_at),
  })),
);

async function load() {
  loading.value = true;
  try {
    const res = await apis.ai.listPrompts({ page: query.page, page_size: query.page_size });
    prompts.value = res.items ?? [];
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
  formState.scene = 'chat';
  formState.system_prompt = '';
  formState.template = '';
  formState.variables_json = {};
  formState.model = '';
  formState.temperature = 0.7;
  formState.max_tokens = 2048;
  formState.top_p = 0.9;
  formState.is_active = true;
  formState.version = 1;
  formRef.value?.clearValidate();
}

function openAddModal() {
  currentId.value = null;
  resetForm();
  modalVisible.value = true;
}

async function openEditModal(record: PromptTemplate) {
  currentId.value = record.id;
  resetForm();
  try {
    const detail = await apis.ai.getPrompt(record.id);
    formState.name = detail.name;
    formState.scene = detail.scene;
    formState.system_prompt = detail.system_prompt;
    formState.template = detail.template;
    formState.variables_json = detail.variables_json ?? {};
    formState.model = detail.model;
    formState.temperature = detail.temperature;
    formState.max_tokens = detail.max_tokens;
    formState.top_p = detail.top_p;
    formState.is_active = detail.is_active;
    formState.version = detail.version;
  } catch (e) {
    formState.name = record.name;
    formState.scene = record.scene;
    formState.system_prompt = record.system_prompt;
    formState.template = record.template;
    formState.variables_json = record.variables_json ?? {};
    formState.model = record.model;
    formState.temperature = record.temperature;
    formState.max_tokens = record.max_tokens;
    formState.top_p = record.top_p;
    formState.is_active = record.is_active;
    formState.version = record.version;
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
      await apis.ai.updatePrompt(currentId.value, formState);
      message.success('更新成功');
    } else {
      await apis.ai.createPrompt(formState);
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
    await apis.ai.deletePrompt(id);
    message.success('删除成功');
    load();
  } catch (e) {
    // error handled by interceptor
  }
}

async function handleActivate(id: number) {
  try {
    await apis.ai.activatePrompt(id);
    message.success('已激活');
    load();
  } catch (e) {
    // error handled by interceptor
  }
}
</script>

<template>
  <div class="admin-page">
    <h1 class="admin-page-title">Prompt 模板</h1>

    <div class="table-card">
      <div class="toolbar">
        <Button type="primary" @click="openAddModal">新增模板</Button>
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
          <template v-if="column.dataIndex === 'scene'">
            <Tag :color="sceneColorMap[text] || 'default'">{{ sceneLabelMap[text] || text || '—' }}</Tag>
          </template>
          <template v-else-if="column.dataIndex === 'is_active'">
            <Tag :color="text ? 'success' : 'default'">{{ text ? '启用' : '停用' }}</Tag>
          </template>
          <template v-else-if="column.dataIndex === 'action'">
            <Button type="link" size="small" @click="openEditModal(record as PromptTemplate)">编辑</Button>
            <Button
              v-if="!(record as PromptTemplate).is_active"
              type="link"
              size="small"
              @click="handleActivate((record as PromptTemplate).id)"
              >激活</Button
            >
            <Popconfirm title="确定删除该模板吗？" @confirm="handleDelete((record as PromptTemplate).id)">
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
      :title="currentId ? '编辑 Prompt 模板' : '新增 Prompt 模板'"
      :confirm-loading="modalLoading"
      @cancel="modalVisible = false"
      @ok="handleOk"
      :width="680"
    >
      <Form ref="formRef" :model="formState" layout="vertical">
        <div class="form-row">
          <Form.Item label="模板名称" name="name" :rules="[{ required: true, message: '请输入模板名称' }]">
            <Input v-model:value="formState.name" placeholder="请输入模板名称" />
          </Form.Item>
          <Form.Item label="场景" name="scene" :rules="[{ required: true, message: '请选择场景' }]">
            <Select v-model:value="formState.scene">
              <Select.Option value="chat">对话</Select.Option>
              <Select.Option value="agent">Agent</Select.Option>
              <Select.Option value="reasoning">推理</Select.Option>
              <Select.Option value="summarize">摘要</Select.Option>
            </Select>
          </Form.Item>
        </div>
        <Form.Item label="System Prompt" name="system_prompt" :rules="[{ required: true, message: '请输入 System Prompt' }]">
          <Input.TextArea v-model:value="formState.system_prompt" placeholder="请输入系统提示词" :rows="4" />
        </Form.Item>
        <Form.Item label="模板内容" name="template">
          <Input.TextArea v-model:value="formState.template" placeholder="请输入模板内容，使用 {{变量名}} 作为占位符" :rows="3" />
        </Form.Item>
        <div class="form-row">
          <Form.Item label="模型" name="model">
            <Input v-model:value="formState.model" placeholder="如 gpt-4o" />
          </Form.Item>
          <Form.Item label="温度" name="temperature">
            <InputNumber v-model:value="formState.temperature" :min="0" :max="2" :step="0.1" style="width: 100%" />
          </Form.Item>
        </div>
        <div class="form-row">
          <Form.Item label="最大 Token" name="max_tokens">
            <InputNumber v-model:value="formState.max_tokens" :min="1" :step="512" style="width: 100%" />
          </Form.Item>
          <Form.Item label="Top P" name="top_p">
            <InputNumber v-model:value="formState.top_p" :min="0" :max="1" :step="0.1" style="width: 100%" />
          </Form.Item>
        </div>
        <div class="form-row">
          <Form.Item label="版本" name="version">
            <InputNumber v-model:value="formState.version" :min="1" style="width: 100%" />
          </Form.Item>
          <Form.Item label="启用" name="is_active" value-prop-name="checked">
            <Switch v-model:checked="formState.is_active" checked-children="启用" un-checked-children="停用" />
          </Form.Item>
        </div>
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