<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { Table, Pagination, Button, Modal } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import { truncate } from '@tcm/shared';
import type { Disease } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const diseases = ref<Disease[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });
const detailVisible = ref(false);
const detailLoading = ref(false);
const currentDisease = ref<Disease | null>(null);

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '病名', dataIndex: 'name', key: 'name' },
  { title: '拼音', dataIndex: 'pinyin', key: 'pinyin', width: 140 },
  { title: '分类', dataIndex: 'category', key: 'category', width: 120 },
  { title: '描述', dataIndex: 'description', key: 'description' },
  { title: '症状', dataIndex: 'symptoms', key: 'symptoms' },
  { title: '病机', dataIndex: 'tcm_pathogenesis', key: 'tcm_pathogenesis' },
  { title: '操作', dataIndex: 'action', key: 'action', width: 100, fixed: 'right' as const },
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

async function openDetail(id: number) {
  detailLoading.value = true;
  detailVisible.value = true;
  currentDisease.value = null;
  try {
    currentDisease.value = await apis.history.getDisease(id);
  } finally {
    detailLoading.value = false;
  }
}
</script>

<template>
  <div class="admin-page">
    <h1 class="admin-page-title">疾病管理</h1>

    <div class="table-card">
      <div class="toolbar">
        <span class="table-hint">疾病数据仅供查看，暂不支持编辑删除</span>
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
            <Button type="link" size="small" @click="openDetail(record.id)">详情</Button>
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
      title="疾病详情"
      :confirm-loading="detailLoading"
      @cancel="detailVisible = false"
      @ok="detailVisible = false"
      width="640px"
    >
      <div v-if="detailLoading" class="detail-loading">加载中...</div>
      <div v-else-if="currentDisease" class="detail-content">
        <div class="detail-item">
          <span class="detail-label">ID：</span>
          <span class="detail-value">{{ currentDisease.id }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">病名：</span>
          <span class="detail-value">{{ currentDisease.name }}</span>
        </div>
        <div class="detail-row">
          <div class="detail-item">
            <span class="detail-label">拼音：</span>
            <span class="detail-value">{{ currentDisease.pinyin || '—' }}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">分类：</span>
            <span class="detail-value">{{ currentDisease.category || '—' }}</span>
          </div>
        </div>
        <div class="detail-item">
          <span class="detail-label">描述：</span>
          <span class="detail-value">{{ currentDisease.description || '—' }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">症状：</span>
          <span class="detail-value">{{ currentDisease.symptoms || '—' }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">病机：</span>
          <span class="detail-value">{{ currentDisease.tcm_pathogenesis || '—' }}</span>
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

.toolbar {
  margin-bottom: var(--tcm-spacing-base);
  display: flex;
  align-items: center;
}

.table-hint {
  color: rgba(31, 42, 68, 0.55);
  font-size: 14px;
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
  gap: var(--tcm-spacing-base);
}

.detail-row {
  display: flex;
  gap: var(--tcm-spacing-lg);
}

.detail-item {
  display: flex;
  flex: 1;
  line-height: 1.6;
}

.detail-label {
  color: rgba(31, 42, 68, 0.55);
  min-width: 70px;
  flex-shrink: 0;
}

.detail-value {
  color: #1f2a44;
  word-break: break-all;
}
</style>
