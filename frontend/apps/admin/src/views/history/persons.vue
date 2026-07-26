<script setup lang="ts">
// 人物管理列表：调用 apis.history.listPersons()，表格展示 + 新增/编辑（Modal 占位）。
import { computed, onMounted, reactive, ref } from 'vue';
import { Table, Pagination, Button, Modal } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import type { Person } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const persons = ref<Person[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });
const modalVisible = ref(false);

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '姓名', dataIndex: 'name', key: 'name' },
  { title: '字', dataIndex: 'courtesy_name', key: 'courtesy_name', width: 100 },
  { title: '朝代', dataIndex: 'dynasty_id', key: 'dynasty_id', width: 100 },
  { title: '生卒年', dataIndex: 'year_range', key: 'year_range', width: 160 },
  { title: '称号', dataIndex: 'title', key: 'title' },
  { title: '操作', dataIndex: 'action', key: 'action', width: 100 },
];

const dataSource = computed(() =>
  persons.value.map((p) => ({
    ...p,
    courtesy_name: p.courtesy_name || '—',
    title: p.title || '—',
    year_range: `${p.birth_year} - ${p.death_year}`,
  })),
);

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
    <h1 class="admin-page-title">人物管理</h1>

    <div class="table-card">
      <div class="toolbar">
        <Button type="primary" @click="openModal">新增人物</Button>
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
      title="新增 / 编辑人物"
      @cancel="modalVisible = false"
      @ok="modalVisible = false"
    >
      <p class="modal-placeholder">表单占位：后续接入完整的人物新增 / 编辑表单。</p>
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
