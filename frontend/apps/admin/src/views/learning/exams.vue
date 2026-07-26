<script setup lang="ts">
// 考试管理列表：调用 apis.learning.listExams()，表格展示 + 新增/编辑（Modal 占位）。
import { computed, onMounted, reactive, ref } from 'vue';
import { Table, Pagination, Button, Modal, Tag } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import { formatDateTime, truncate } from '@tcm/shared';
import type { Exam } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const exams = ref<Exam[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });
const modalVisible = ref(false);

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '考试名称', dataIndex: 'title', key: 'title' },
  { title: '课程 ID', dataIndex: 'course_id', key: 'course_id', width: 100 },
  { title: '题目数', dataIndex: 'question_count', key: 'question_count', width: 90 },
  { title: '及格分', dataIndex: 'pass_score', key: 'pass_score', width: 90 },
  { title: '时长(分)', dataIndex: 'duration_minutes', key: 'duration_minutes', width: 100 },
  { title: '发布', dataIndex: 'is_published', key: 'is_published', width: 80 },
  { title: '描述', dataIndex: 'description', key: 'description' },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 100 },
];

const dataSource = computed(() =>
  exams.value.map((e) => ({
    ...e,
    description: truncate(e.description, 40),
    created_at: formatDateTime(e.created_at),
  })),
);

async function load() {
  loading.value = true;
  try {
    const res = await apis.learning.listExams({
      page: query.page,
      page_size: query.page_size,
    });
    exams.value = res.items ?? [];
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
    <h1 class="admin-page-title">考试管理</h1>

    <div class="table-card">
      <div class="toolbar">
        <Button type="primary" @click="openModal">新增考试</Button>
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
          <template v-if="column.dataIndex === 'is_published'">
            <Tag :color="text ? 'success' : 'default'">{{ text ? '已发布' : '未发布' }}</Tag>
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
      title="新增 / 编辑考试"
      @cancel="modalVisible = false"
      @ok="modalVisible = false"
    >
      <p class="modal-placeholder">表单占位：后续接入完整的考试新增 / 编辑表单。</p>
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
