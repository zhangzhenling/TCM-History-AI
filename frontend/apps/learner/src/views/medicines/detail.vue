<script setup lang="ts">
import { onMounted, ref, watch, computed } from 'vue';
import { useRoute } from 'vue-router';
import { Spin, Descriptions, DescriptionsItem, Tag, Empty } from 'ant-design-vue';

import PageHeader from '@/components/PageHeader.vue';
import { useApi } from '@/composables/useApi';
import type { Medicine } from '@tcm/api';

const props = defineProps<{ id?: string | number }>();
const route = useRoute();
const apis = useApi();

const loading = ref(false);
const medicine = ref<Medicine | null>(null);

const medicineId = () => props.id ?? (route.params.id as string);

const aliasList = computed(() => {
  if (!medicine.value?.alias_json) return [] as string[];
  const a = medicine.value.alias_json;
  return Array.isArray(a) ? a : [a];
});

async function load() {
  loading.value = true;
  try {
    medicine.value = await apis.history.getMedicine(medicineId());
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
      <template v-if="medicine">
        <PageHeader :title="medicine.name" :subtitle="medicine.pinyin || ''">
          <template #actions>
            <Tag v-if="medicine.nature || medicine.flavor" color="var(--tcm-color-celadon)" style="color: #fff">
              {{ medicine.nature }}{{ medicine.flavor }}
            </Tag>
            <Tag v-if="medicine.toxicity" color="orange">有毒</Tag>
          </template>
        </PageHeader>

        <Descriptions :column="2" bordered size="middle">
          <DescriptionsItem label="拼音">{{ medicine.pinyin || '—' }}</DescriptionsItem>
          <DescriptionsItem label="别名">
            <template v-if="aliasList.length">
              <Tag v-for="a in aliasList" :key="a" style="margin-bottom: 4px">{{ a }}</Tag>
            </template>
            <span v-else>—</span>
          </DescriptionsItem>
          <DescriptionsItem label="性味">{{ medicine.nature || '—' }}{{ medicine.flavor || '' }}</DescriptionsItem>
          <DescriptionsItem label="归经">{{ medicine.meridian || '—' }}</DescriptionsItem>
          <DescriptionsItem label="毒性">{{ medicine.toxicity || '无毒' }}</DescriptionsItem>
          <DescriptionsItem label="用量">{{ medicine.dosage || '—' }}</DescriptionsItem>
        </Descriptions>

        <section v-if="medicine.efficacy" class="block">
          <h2 class="section-title">功效</h2>
          <p class="section-text">{{ medicine.efficacy }}</p>
        </section>
      </template>
      <Empty v-else-if="!loading" description="未找到该药物" />
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
  border-left: 3px solid var(--tcm-color-celadon);
}

.section-text {
  margin: 0;
  line-height: 1.8;
  color: rgba(31, 26, 23, 0.85);
  font-size: 14px;
  white-space: pre-wrap;
}
</style>