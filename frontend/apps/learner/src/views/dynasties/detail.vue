<script setup lang="ts">
import { onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { Spin, Descriptions, DescriptionsItem, Tag, Empty } from 'ant-design-vue';

import PageHeader from '@/components/PageHeader.vue';
import { useApi } from '@/composables/useApi';
import { formatYearRange } from '@tcm/shared';
import type { Dynasty } from '@tcm/api';

const props = defineProps<{ id?: string | number }>();
const route = useRoute();
const apis = useApi();

const loading = ref(false);
const dynasty = ref<Dynasty | null>(null);

const dynastyId = () => props.id ?? (route.params.id as string);

async function load() {
  loading.value = true;
  try {
    dynasty.value = await apis.history.getDynasty(dynastyId());
  } finally {
    loading.value = false;
  }
}

onMounted(load);
watch(() => props.id, load);
</script>

<template>
  <div class="tcm-container">
    <Spin :spinning="loading">
      <template v-if="dynasty">
        <PageHeader :title="dynasty.name" :subtitle="formatYearRange(dynasty.start_year, dynasty.end_year)">
          <template #actions>
            <Tag color="var(--tcm-color-primary)" style="color: #fff">第 {{ dynasty.sort_order }} 朝</Tag>
          </template>
        </PageHeader>

        <Descriptions :column="2" bordered size="middle">
          <DescriptionsItem label="起止年份">{{ formatYearRange(dynasty.start_year, dynasty.end_year) }}</DescriptionsItem>
          <DescriptionsItem label="排序">{{ dynasty.sort_order }}</DescriptionsItem>
        </Descriptions>

        <section v-if="dynasty.description" class="block">
          <h2 class="section-title">朝代概述</h2>
          <p class="section-text">{{ dynasty.description }}</p>
        </section>
      </template>
      <Empty v-else-if="!loading" description="未找到该朝代" />
    </Spin>
  </div>
</template>

<style scoped lang="less">
.block {
  margin-top: var(--tcm-spacing-xl);
}

.section-title {
  margin: 0 0 8px;
  font-size: 18px;
  font-weight: 600;
  padding-left: 10px;
  border-left: 3px solid var(--tcm-color-primary);
}

.section-text {
  margin: 0;
  line-height: 1.8;
  color: rgba(31, 26, 23, 0.85);
  font-size: 14px;
  white-space: pre-wrap;
}
</style>