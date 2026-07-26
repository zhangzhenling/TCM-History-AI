<script setup lang="ts">
// 学派列表页：分页浏览所有学派。
import { onMounted, reactive, ref } from 'vue';
import { Spin, Empty, Pagination } from 'ant-design-vue';

import PageHeader from '@/components/PageHeader.vue';
import EntityCard from '@/components/EntityCard.vue';
import { useApi } from '@/composables/useApi';
import { truncate } from '@tcm/shared';
import type { School } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const schools = ref<School[]>([]);
const total = ref(0);

const query = reactive({ page: 1, page_size: 12 });

async function load() {
  loading.value = true;
  try {
    const res = await apis.history.listSchools({ page: query.page, page_size: query.page_size });
    schools.value = res.items ?? [];
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
    <PageHeader title="医学学派" subtitle="伤寒、温病、汇通学派源流" />

    <Spin :spinning="loading">
      <div v-if="schools.length" class="card-grid">
        <EntityCard
          v-for="s in schools"
          :id="s.id"
          :key="s.id"
          type="school"
          :title="s.name"
          :description="truncate(s.summary || '', 80)"
          :year-range="{ start: s.established_year, end: s.established_year }"
        />
      </div>
      <Empty v-else description="暂无学派数据" />
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
