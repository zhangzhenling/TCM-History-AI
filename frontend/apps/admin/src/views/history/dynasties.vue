<script setup lang="ts">
// 朝代管理列表：调用 apis.history.listDynasties()，表格展示 + 新增/编辑（Modal 占位）。
import { computed, onMounted, reactive, ref } from 'vue';
import { Table, Pagination, Button, Modal } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import { truncate } from '@tcm/shared';
import type { Dynasty } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const dynasties = ref<Dynasty[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });
const modalVisible = ref(false);

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '朝代名', dataIndex: 'name', key: 'name' },
  { title: '起止年份', dataIndex: 'year_range', key: 'year_range', width: 160 },
  { title: '排序', dataIndex: 'sort_order', key: 'sort_order', width: 80 },
  { title: '描述', dataIndex: 'description', key: 'description' },
  { title: '操作', dataIndex: 'action', key: 'action', width: 100 },
];

const dataSource = computed(() =>
  dynasties.value.map((d) => ({
    ...d,
    year_range: `${d.start_year} - ${d.end_year}`,
    description: truncate(d.description, 40),
  })),
);

async function load() {
  loading.value = true;
  try {
    const res = await apis.history.listDynasties({ page: query.page, page_size: query.page_size });
    dynasties.value = res.items ?? [];
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
    <h1 class="admin-page-title">朝代管理</h1>

    <div class="table-card">
      <div class="toolbar">
        <Button type="primary" @click="openModal">新增朝代</Button>
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
          <template v-if="column.dataIndex === 'action'">
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
      title="新增 / 编辑朝代"
      @cancel="modalVisible = false"
      @ok="modalVisible = false"
    >
      <p class="modal-placeholder">表单占位：后续接入完整的朝代新增 / 编辑表单。</p>
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
