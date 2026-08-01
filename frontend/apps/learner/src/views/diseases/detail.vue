<script setup lang="ts">
import { onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { Spin, Descriptions, DescriptionsItem, Tag, Empty } from 'ant-design-vue';

import PageHeader from '@/components/PageHeader.vue';
import { useApi } from '@/composables/useApi';
import type { Disease } from '@tcm/api';

const props = defineProps<{ id?: string | number }>();
const route = useRoute();
const apis = useApi();

const loading = ref(false);
const disease = ref<Disease | null>(null);

const diseaseId = () => props.id ?? (route.params.id as string);

async function load() {
  loading.value = true;
  try {
    disease.value = await apis.history.getDisease(diseaseId());
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
      <template v-if="disease">
        <PageHeader :title="disease.name" :subtitle="disease.pinyin || ''">
          <template #actions>
            <Tag v-if="disease.category" color="var(--tcm-color-primary)" style="color: #fff">
              {{ disease.category }}
            </Tag>
          </template>
        </PageHeader>

        <Descriptions :column="2" bordered size="middle">
          <DescriptionsItem label="拼音">{{ disease.pinyin || '—' }}</DescriptionsItem>
          <DescriptionsItem label="分类">{{ disease.category || '—' }}</DescriptionsItem>
        </Descriptions>

        <section v-if="disease.description" class="block">
          <h2 class="section-title">疾病概述</h2>
          <p class="section-text">{{ disease.description }}</p>
        </section>

        <section v-if="disease.symptoms" class="block">
          <h2 class="section-title">症状</h2>
          <p class="section-text">{{ disease.symptoms }}</p>
        </section>

        <section v-if="disease.tcm_pathogenesis" class="block">
          <h2 class="section-title">中医病机</h2>
          <p class="section-text">{{ disease.tcm_pathogenesis }}</p>
        </section>
      </template>
      <Empty v-else-if="!loading" description="未找到该疾病" />
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