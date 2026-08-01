<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { Spin, Empty, Pagination, Select, SelectOption } from 'ant-design-vue';

import PageHeader from '@/components/PageHeader.vue';
import EntityCard from '@/components/EntityCard.vue';
import { useApi } from '@/composables/useApi';
import { truncate } from '@tcm/shared';
import type { Prescription } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const prescriptions = ref<Prescription[]>([]);
const total = ref(0);

const categories = ['解表剂', '清热剂', '补益剂', '理血剂', '祛湿剂', '其他'];

const query = reactive({
  page: 1,
  page_size: 12,
  category: undefined as string | undefined,
});

async function load() {
  loading.value = true;
  try {
    const res = await apis.history.listPrescriptions({
      page: query.page,
      page_size: query.page_size,
      category: query.category,
    });
    prescriptions.value = res.items ?? [];
    total.value = res.total ?? 0;
  } finally {
    loading.value = false;
  }
}

onMounted(load);

function onFilterChange() {
  query.page = 1;
  load();
}

function onPageChange(p: number, ps: number) {
  query.page = p;
  query.page_size = ps;
  load();
}
</script>

<template>
  <div class="tcm-container">
    <PageHeader title="方剂列表" subtitle="浏览中医经典方剂" />

    <div class="filter-bar">
      <Select
        v-model:value="query.category"
        placeholder="分类"
        allow-clear
        style="width: 160px"
        @change="onFilterChange"
      >
        <SelectOption v-for="c in categories" :key="c" :value="c">{{ c }}</SelectOption>
      </Select>
    </div>

    <Spin :spinning="loading">
      <div v-if="prescriptions.length" class="card-grid">
        <EntityCard
          v-for="p in prescriptions"
          :id="p.id"
          :key="p.id"
          type="prescription"
          :title="p.name"
          :subtitle="p.pinyin || ''"
          :description="truncate(p.composition || p.indications || '', 80)"
          :tags="p.category ? [p.category] : []"
        />
      </div>
      <Empty v-else description="未找到方剂" />
    </Spin>

    <div v-if="total > 0" class="pagination-wrap">
      <Pagination
        :current="query.page"
        :page-size="query.page_size"
        :total="total"
        show-size-changer
        :page-size-options="['12', '24', '48']"
        @change="onPageChange"
      />
    </div>
  </div>
</template>

<style scoped lang="less">
.filter-bar {
  display: flex;
  gap: 12px;
  margin-bottom: var(--tcm-spacing-lg);
  flex-wrap: wrap;
}

.card-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--tcm-spacing-lg);
}

@media (max-width: 1024px) {
  .card-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .card-grid {
    grid-template-columns: 1fr;
  }
}

.pagination-wrap {
  display: flex;
  justify-content: center;
  margin-top: var(--tcm-spacing-xl);
}
</style>
