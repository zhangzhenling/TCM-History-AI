<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { Spin, Empty, Pagination } from 'ant-design-vue';

import PageHeader from '@/components/PageHeader.vue';
import EntityCard from '@/components/EntityCard.vue';
import { useApi } from '@/composables/useApi';
import { truncate } from '@tcm/shared';
import type { Medicine } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const medicines = ref<Medicine[]>([]);
const total = ref(0);

const query = reactive({ page: 1, page_size: 12 });

async function load() {
  loading.value = true;
  try {
    const res = await apis.history.listMedicines({
      page: query.page,
      page_size: query.page_size,
    });
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
</script>

<template>
  <div class="tcm-container">
    <PageHeader title="药物列表" subtitle="浏览本草药物" />

    <Spin :spinning="loading">
      <div v-if="medicines.length" class="card-grid">
        <EntityCard
          v-for="m in medicines"
          :id="m.id"
          :key="m.id"
          type="medicine"
          :title="m.name"
          :subtitle="m.pinyin || ''"
          :description="truncate(m.efficacy || '', 80)"
          :tags="[m.nature || '', m.flavor || ''].filter(Boolean)"
        />
      </div>
      <Empty v-else description="未找到药物" />
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