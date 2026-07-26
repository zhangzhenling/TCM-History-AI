<script setup lang="ts">
// 典籍列表页：分页 + 朝代筛选 + 分类筛选。
import { onMounted, reactive, ref } from 'vue';
import { Spin, Empty, Pagination, Select, SelectOption } from 'ant-design-vue';

import PageHeader from '@/components/PageHeader.vue';
import EntityCard from '@/components/EntityCard.vue';
import { useApi } from '@/composables/useApi';
import { truncate } from '@tcm/shared';
import type { Dynasty, Book } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const books = ref<Book[]>([]);
const dynasties = ref<Dynasty[]>([]);
const total = ref(0);

const categories = ['经典', '方书', '本草', '医案'];

const query = reactive({
  page: 1,
  page_size: 12,
  dynasty_id: undefined as number | undefined,
  category: undefined as string | undefined,
});

async function load() {
  loading.value = true;
  try {
    const res = await apis.history.listBooks({
      page: query.page,
      page_size: query.page_size,
      dynasty_id: query.dynasty_id,
      category: query.category,
    });
    books.value = res.items ?? [];
    total.value = res.total ?? 0;
  } finally {
    loading.value = false;
  }
}

onMounted(async () => {
  const [d] = await Promise.all([apis.history.listDynasties({ page: 1, page_size: 100 }), load()]);
  dynasties.value = d.items ?? [];
});

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
    <PageHeader title="中医典籍" subtitle="素问、本草、伤寒论等经典原文" />

    <div class="filter-bar">
      <Select
        v-model:value="query.dynasty_id"
        placeholder="朝代"
        allow-clear
        style="width: 160px"
        @change="onFilterChange"
      >
        <SelectOption v-for="d in dynasties" :key="d.id" :value="d.id">{{ d.name }}</SelectOption>
      </Select>
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
      <div v-if="books.length" class="card-grid">
        <EntityCard
          v-for="b in books"
          :id="b.id"
          :key="b.id"
          type="book"
          :title="b.title"
          :subtitle="b.category"
          :description="truncate(b.summary || '', 80)"
          :year-range="{ start: b.published_year, end: b.published_year }"
          :tags="b.is_extant ? ['存世'] : ['已佚']"
        />
      </div>
      <Empty v-else description="未找到匹配的典籍" />
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
