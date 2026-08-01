<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { Table, Pagination, Button, Modal, Tag, Popconfirm, message, Descriptions } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import { formatDateTime } from '@tcm/shared';
import type { Conversation, Message } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const conversations = ref<Conversation[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });

const detailVisible = ref(false);
const detailLoading = ref(false);
const currentConv = ref<Conversation | null>(null);
const messages = ref<Message[]>([]);

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '标题', dataIndex: 'title', key: 'title' },
  { title: '用户 ID', dataIndex: 'user_id', key: 'user_id', width: 100 },
  { title: '模式', dataIndex: 'mode', key: 'mode', width: 100 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 100 },
  { title: '消息数', dataIndex: 'message_count', key: 'message_count', width: 90 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
  { title: '更新时间', dataIndex: 'updated_at', key: 'updated_at', width: 180 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 160 },
];

const statusColorMap: Record<string, string> = {
  active: 'success',
  archived: 'default',
};

const modeLabelMap: Record<string, string> = {
  chat: '对话',
  agent: 'Agent',
  reasoning: '推理',
};

const dataSource = computed(() =>
  conversations.value.map((c) => ({
    ...c,
    title: c.title || '—',
    created_at: formatDateTime(c.created_at),
    updated_at: formatDateTime(c.updated_at),
  })),
);

async function load() {
  loading.value = true;
  try {
    const res = await apis.ai.listConversations({ page: query.page, page_size: query.page_size });
    conversations.value = res.items ?? [];
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

async function openDetail(record: Conversation) {
  detailLoading.value = true;
  detailVisible.value = true;
  currentConv.value = null;
  messages.value = [];
  try {
    currentConv.value = await apis.ai.getConversation(record.id);
    const msgRes = await apis.ai.listMessages(record.id, { page: 1, page_size: 50 });
    messages.value = msgRes.items ?? [];
  } finally {
    detailLoading.value = false;
  }
}

async function handleDelete(id: number) {
  try {
    await apis.ai.deleteConversation(id);
    message.success('删除成功');
    load();
  } catch (e) {
    // error handled by interceptor
  }
}

function roleColor(role: string): string {
  const map: Record<string, string> = {
    user: 'blue',
    assistant: 'green',
    system: 'orange',
    tool: 'purple',
  };
  return map[role] || 'default';
}
</script>

<template>
  <div class="admin-page">
    <h1 class="admin-page-title">对话记录</h1>

    <div class="table-card">
      <Table
        :data-source="dataSource"
        :columns="columns"
        :loading="loading"
        :pagination="false"
        row-key="id"
        size="middle"
      >
        <template #bodyCell="{ text, column, record }">
          <template v-if="column.dataIndex === 'mode'">
            <Tag :color="modeLabelMap[text] ? 'blue' : 'default'">{{ modeLabelMap[text] || text || '—' }}</Tag>
          </template>
          <template v-else-if="column.dataIndex === 'status'">
            <Tag :color="statusColorMap[text] || 'default'">{{ text || '—' }}</Tag>
          </template>
          <template v-else-if="column.dataIndex === 'action'">
            <Button type="link" size="small" @click="openDetail(record as Conversation)">查看</Button>
            <Popconfirm title="确定删除该对话吗？" @confirm="handleDelete((record as Conversation).id)">
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
      :open="detailVisible"
      title="对话详情"
      :footer="null"
      width="720px"
      @cancel="detailVisible = false"
    >
      <div v-if="detailLoading" class="detail-loading">加载中...</div>
      <div v-else-if="currentConv" class="detail-content">
        <Descriptions :column="2" size="small" bordered>
          <Descriptions.Item label="ID">{{ currentConv.id }}</Descriptions.Item>
          <Descriptions.Item label="用户 ID">{{ currentConv.user_id }}</Descriptions.Item>
          <Descriptions.Item label="标题">{{ currentConv.title || '—' }}</Descriptions.Item>
          <Descriptions.Item label="模式">{{ modeLabelMap[currentConv.mode] || currentConv.mode }}</Descriptions.Item>
          <Descriptions.Item label="状态">
            <Tag :color="statusColorMap[currentConv.status] || 'default'">{{ currentConv.status }}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="消息数">{{ currentConv.message_count }}</Descriptions.Item>
          <Descriptions.Item label="创建时间" :span="2">{{ formatDateTime(currentConv.created_at) }}</Descriptions.Item>
          <Descriptions.Item label="更新时间" :span="2">{{ formatDateTime(currentConv.updated_at) }}</Descriptions.Item>
        </Descriptions>

        <h4 class="messages-title">消息列表（最近 50 条）</h4>
        <div class="messages-list">
          <div v-for="msg in messages" :key="msg.id" class="message-item" :class="msg.role">
            <div class="message-header">
              <Tag :color="roleColor(msg.role)">{{ msg.role }}</Tag>
              <span class="message-time">{{ formatDateTime(msg.created_at) }}</span>
              <span v-if="msg.model_name" class="message-model">{{ msg.model_name }}</span>
            </div>
            <div class="message-body">{{ msg.content }}</div>
          </div>
          <div v-if="messages.length === 0" class="messages-empty">暂无消息</div>
        </div>
      </div>
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

.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--tcm-spacing-lg);
}

.detail-loading {
  color: rgba(31, 42, 68, 0.55);
  text-align: center;
  padding: var(--tcm-spacing-lg) 0;
}

.detail-content {
  display: flex;
  flex-direction: column;
  gap: var(--tcm-spacing-lg);
}

.messages-title {
  margin: 0;
  font-size: 14px;
  color: rgba(31, 42, 68, 0.85);
}

.messages-list {
  max-height: 400px;
  overflow-y: auto;
  border: 1px solid #f0f0f0;
  border-radius: var(--tcm-radius-base);
  padding: var(--tcm-spacing-base);
}

.message-item {
  padding: var(--tcm-spacing-base);
  border-bottom: 1px solid #f5f5f5;

  &:last-child {
    border-bottom: none;
  }

  &.user {
    background-color: #f6f8ff;
    border-radius: var(--tcm-radius-base);
    margin-bottom: var(--tcm-spacing-small);
  }

  &.assistant {
    background-color: #f0fff4;
    border-radius: var(--tcm-radius-base);
    margin-bottom: var(--tcm-spacing-small);
  }
}

.message-header {
  display: flex;
  align-items: center;
  gap: var(--tcm-spacing-base);
  margin-bottom: 4px;
  font-size: 12px;
  color: rgba(31, 42, 68, 0.55);
}

.message-body {
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 14px;
  line-height: 1.6;
}

.messages-empty {
  text-align: center;
  color: rgba(31, 42, 68, 0.35);
  padding: var(--tcm-spacing-lg) 0;
}
</style>