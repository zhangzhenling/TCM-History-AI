<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { Spin, Empty, Pagination } from 'ant-design-vue';

import PageHeader from '@/components/PageHeader.vue';
import EntityCard from '@/components/EntityCard.vue';
import { useApi } from '@/composables/useApi';
import type { Dynasty } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const dynasties = ref<Dynasty[]>([]);
const total = ref(0);

const query = reactive({ page: 1, page_size: 12 });

async function load() {
  loading.value = true;
  try {
    const res = await apis.history.listDynasties({
      page: query.page,
      page_size: query.page_size,
    });
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
</script>

<template>
  <div class="tcm-container">
    <PageHeader title="朝代列表" subtitle="浏览中医史上的各个朝代" />

    <Spin :spinning="loading">
      <div v-if="dynasties.length" class="card-grid">
        <EntityCard
          v-for="d in dynasties"
          :id="d.id"
          :key="d.id"
          type="dynasty"
          :title="d.name"
          :description="d.description || ''"
          :year-range="{ start: d.start_year, end: d.end_year }"
          :tags="[`排序: ${d.sort_order}`]"
        />
      </div>
      <Empty v-else description="未找到朝代" />
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
