<script setup lang="ts">
// Prompt 模板列表：调用 apis.ai.listPrompts()，表格展示 + 新增/编辑（Modal 占位）。
import { computed, onMounted, reactive, ref } from 'vue';
import { Table, Pagination, Button, Modal, Tag } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import { formatDateTime, truncate } from '@tcm/shared';
import type { PromptTemplate } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const prompts = ref<PromptTemplate[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });
const modalVisible = ref(false);

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
  { title: '操作', dataIndex: 'action', key: 'action', width: 100 },
];

const sceneColorMap: Record<string, string> = {
  chat: 'blue',
  agent: 'purple',
  reasoning: 'cyan',
  summarize: 'gold',
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

function openModal() {
  modalVisible.value = true;
}
</script>

<template>
  <div class="admin-page">
    <h1 class="admin-page-title">Prompt 模板</h1>

    <div class="table-card">
      <div class="toolbar">
        <Button type="primary" @click="openModal">新增模板</Button>
      </div>
      <Table
        :data-source="dataSource"
        :columns="columns"
        :loading="loading"
        :pagination="false"
        row-key="id"
        size="middle"
      >
        <template #bodyCell="{ text, column }">
          <template v-if="column.dataIndex === 'scene'">
            <Tag :color="sceneColorMap[text] || 'default'">{{ text || '—' }}</Tag>
          </template>
          <template v-else-if="column.dataIndex === 'is_active'">
            <Tag :color="text ? 'success' : 'default'">{{ text ? '启用' : '停用' }}</Tag>
          </template>
          <template v-else-if="column.dataIndex === 'action'">
            <Button type="link" size="small" @click="openModal">编辑</Button>
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
      title="新增 / 编辑 Prompt 模板"
      @cancel="modalVisible = false"
      @ok="modalVisible = false"
    >
      <p class="modal-placeholder">表单占位：后续接入完整的 Prompt 模板新增 / 编辑表单。</p>
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

.modal-placeholder {
  color: rgba(31, 42, 68, 0.55);
  margin: 0;
}
</style>
