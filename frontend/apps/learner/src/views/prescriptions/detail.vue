<script setup lang="ts">
import { onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { Spin, Descriptions, DescriptionsItem, Tag, Empty } from 'ant-design-vue';

import PageHeader from '@/components/PageHeader.vue';
import { useApi } from '@/composables/useApi';
import type { Prescription } from '@tcm/api';

const props = defineProps<{ id?: string | number }>();
const route = useRoute();
const apis = useApi();

const loading = ref(false);
const prescription = ref<Prescription | null>(null);

const prescriptionId = () => props.id ?? (route.params.id as string);

async function load() {
  loading.value = true;
  try {
    prescription.value = await apis.history.getPrescription(prescriptionId());
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
      <template v-if="prescription">
        <PageHeader :title="prescription.name" :subtitle="prescription.pinyin || ''">
          <template #actions>
            <Tag v-if="prescription.category" color="var(--tcm-color-indigo)" style="color: #fff">
              {{ prescription.category }}
            </Tag>
          </template>
        </PageHeader>

        <Descriptions :column="2" bordered size="middle">
          <DescriptionsItem label="拼音">{{ prescription.pinyin || '—' }}</DescriptionsItem>
          <DescriptionsItem label="分类">{{ prescription.category || '—' }}</DescriptionsItem>
          <DescriptionsItem label="来源著作">ID: {{ prescription.source_book_id || '—' }}</DescriptionsItem>
          <DescriptionsItem label="来源人物">ID: {{ prescription.source_person_id || '—' }}</DescriptionsItem>
        </Descriptions>

        <section v-if="prescription.composition" class="block">
          <h2 class="section-title">处方组成</h2>
          <p class="section-text">{{ prescription.composition }}</p>
        </section>

        <section v-if="prescription.usage" class="block">
          <h2 class="section-title">用法用量</h2>
          <p class="section-text">{{ prescription.usage }}</p>
        </section>

        <section v-if="prescription.indications" class="block">
          <h2 class="section-title">主治功效</h2>
          <p class="section-text">{{ prescription.indications }}</p>
        </section>
      </template>
      <Empty v-else-if="!loading" description="未找到该方剂" />
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
  border-left: 3px solid var(--tcm-color-indigo);
}

.section-text {
  margin: 0;
  line-height: 1.8;
  color: rgba(31, 26, 23, 0.85);
  font-size: 14px;
  white-space: pre-wrap;
}
</style>