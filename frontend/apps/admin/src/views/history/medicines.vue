<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { Table, Pagination, Button, Modal } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import { truncate } from '@tcm/shared';
import type { Medicine } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const medicines = ref<Medicine[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });
const detailVisible = ref(false);
const detailLoading = ref(false);
const currentMedicine = ref<Medicine | null>(null);

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '药名', dataIndex: 'name', key: 'name' },
  { title: '拼音', dataIndex: 'pinyin', key: 'pinyin', width: 140 },
  { title: '性味', dataIndex: 'nature_flavor', key: 'nature_flavor', width: 120 },
  { title: '归经', dataIndex: 'meridian', key: 'meridian', width: 120 },
  { title: '功效', dataIndex: 'efficacy', key: 'efficacy' },
  { title: '用量', dataIndex: 'dosage', key: 'dosage', width: 120 },
  { title: '毒性', dataIndex: 'toxicity', key: 'toxicity', width: 100 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 100, fixed: 'right' as const },
];

const dataSource = computed(() =>
  medicines.value.map((m) => ({
    ...m,
    pinyin: m.pinyin || '—',
    nature_flavor: `${m.nature || ''}${m.flavor || ''}` || '—',
    meridian: m.meridian || '—',
    efficacy: truncate(m.efficacy, 40),
    dosage: m.dosage || '—',
    toxicity: m.toxicity || '—',
  })),
);

async function load() {
  loading.value = true;
  try {
    const res = await apis.history.listMedicines({ page: query.page, page_size: query.page_size });
    medicines.value = res.items ?? [];
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
  currentMedicine.value = null;
  try {
    currentMedicine.value = await apis.history.getMedicine(id);
  } finally {
    detailLoading.value = false;
  }
}
</script>

<template>
  <div class="admin-page">
    <h1 class="admin-page-title">药物管理</h1>

    <div class="table-card">
      <div class="toolbar">
        <span class="table-hint">药物数据仅供查看，暂不支持编辑删除</span>
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
      title="药物详情"
      :confirm-loading="detailLoading"
      @cancel="detailVisible = false"
      @ok="detailVisible = false"
      width="640px"
    >
      <div v-if="detailLoading" class="detail-loading">加载中...</div>
      <div v-else-if="currentMedicine" class="detail-content">
        <div class="detail-item">
          <span class="detail-label">ID：</span>
          <span class="detail-value">{{ currentMedicine.id }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">药名：</span>
          <span class="detail-value">{{ currentMedicine.name }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">拼音：</span>
          <span class="detail-value">{{ currentMedicine.pinyin || '—' }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">别名：</span>
          <span class="detail-value">
            <template v-if="Array.isArray(currentMedicine.alias_json)">
              {{ currentMedicine.alias_json.join('、') || '—' }}
            </template>
            <template v-else>
              {{ currentMedicine.alias_json || '—' }}
            </template>
          </span>
        </div>
        <div class="detail-row">
          <div class="detail-item">
            <span class="detail-label">性味：</span>
            <span class="detail-value">{{ currentMedicine.nature }}{{ currentMedicine.flavor }}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">归经：</span>
            <span class="detail-value">{{ currentMedicine.meridian || '—' }}</span>
          </div>
        </div>
        <div class="detail-row">
          <div class="detail-item">
            <span class="detail-label">用量：</span>
            <span class="detail-value">{{ currentMedicine.dosage || '—' }}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">毒性：</span>
            <span class="detail-value">{{ currentMedicine.toxicity || '—' }}</span>
          </div>
        </div>
        <div class="detail-item">
          <span class="detail-label">功效：</span>
          <span class="detail-value">{{ currentMedicine.efficacy || '—' }}</span>
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
