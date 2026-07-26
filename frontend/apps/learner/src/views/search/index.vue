<script setup lang="ts">
// 跨实体检索页：调用 History Service /search，按类型分组呈现结果。
import { computed, ref } from 'vue';
import { Input, Spin, Empty, Tag } from 'ant-design-vue';
import { SearchOutlined } from '@ant-design/icons-vue';
import { useRouter } from 'vue-router';

import PageHeader from '@/components/PageHeader.vue';
import { useApi } from '@/composables/useApi';
import { ENTITY_LABELS, type EntityType, truncate } from '@tcm/shared';
import type { SearchHit, SearchResponse } from '@tcm/api';

const apis = useApi();
const router = useRouter();

const keyword = ref('');
const loading = ref(false);
const result = ref<SearchResponse | null>(null);
const searched = ref(false);

const grouped = computed<{ type: string; hits: SearchHit[] }[]>(() => {
  if (!result.value?.items?.length) return [];
  const map = new Map<string, SearchHit[]>();
  for (const h of result.value.items) {
    const arr = map.get(h.type) ?? [];
    arr.push(h);
    map.set(h.type, arr);
  }
  return [...map.entries()].map(([type, hits]) => ({ type, hits }));
});

async function doSearch() {
  if (!keyword.value.trim()) return;
  loading.value = true;
  searched.value = true;
  try {
    result.value = await apis.history.search({
      q: keyword.value.trim(),
      page: 1,
      page_size: 30,
    });
  } finally {
    loading.value = false;
  }
}

function hitTitle(h: SearchHit): string {
  return String(h.source?.name ?? h.source?.title ?? `#${h.id}`);
}

function hitDesc(h: SearchHit): string {
  const s = h.source ?? {};
  return truncate(String(s.biography ?? s.summary ?? s.description ?? s.efficacy ?? ''), 100);
}

function goDetail(h: SearchHit) {
  const t = h.type as EntityType;
  const routeName =
    t === 'person'
      ? 'PersonDetail'
      : t === 'book'
        ? 'BookDetail'
        : t === 'school'
          ? 'SchoolDetail'
          : null;
  if (routeName) router.push({ name: routeName, params: { id: h.id } });
}
</script>

<template>
  <div class="tcm-container">
    <PageHeader title="跨实体检索" subtitle="一次查询，遍历人物、典籍、方剂、药物等" />

    <div class="search-box">
      <Input
        v-model:value="keyword"
        placeholder="输入关键词，如「伤寒」「麻黄」「张仲景」"
        size="large"
        allow-clear
        @press-enter="doSearch"
      >
        <template #prefix><SearchOutlined /></template>
      </Input>
    </div>

    <Spin :spinning="loading">
      <div v-if="result && result.total > 0" class="result-meta">
        共找到 <strong>{{ result.total }}</strong> 条结果
      </div>

      <div v-if="grouped.length" class="result-groups">
        <section v-for="g in grouped" :key="g.type" class="result-group">
          <div class="group-header">
            <Tag color="var(--tcm-color-primary)" style="color: #fff">
              {{ ENTITY_LABELS[g.type as EntityType] ?? g.type }}
            </Tag>
            <span class="group-count">{{ g.hits.length }} 条</span>
          </div>
          <div class="hit-list">
            <div
              v-for="h in g.hits"
              :key="`${h.type}-${h.id}`"
              class="hit-item tcm-card-shadow"
              @click="goDetail(h)"
            >
              <div class="hit-title">{{ hitTitle(h) }}</div>
              <div v-if="hitDesc(h)" class="hit-desc">{{ hitDesc(h) }}</div>
            </div>
          </div>
        </section>
      </div>

      <Empty v-else-if="searched && !loading" description="未找到相关结果" />
    </Spin>
  </div>
</template>

<style scoped lang="less">
.search-box {
  max-width: 640px;
  margin-bottom: var(--tcm-spacing-xl);
}

.result-meta {
  font-size: 13px;
  color: rgba(31, 26, 23, 0.6);
  margin-bottom: var(--tcm-spacing-lg);
}

.result-groups {
  display: flex;
  flex-direction: column;
  gap: var(--tcm-spacing-xl);
}

.result-group {
  display: flex;
  flex-direction: column;
  gap: var(--tcm-spacing-base);
}

.group-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.group-count {
  font-size: 12px;
  color: rgba(31, 26, 23, 0.55);
}

.hit-list {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--tcm-spacing-base);
}

@media (max-width: 768px) {
  .hit-list {
    grid-template-columns: 1fr;
  }
}

.hit-item {
  padding: var(--tcm-spacing-base) var(--tcm-spacing-lg);
  border-radius: var(--tcm-radius-lg);
  background-color: var(--tcm-color-paper);
  border: 1px solid rgba(31, 26, 23, 0.06);
  cursor: pointer;
}

.hit-title {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 4px;
}

.hit-desc {
  font-size: 12px;
  color: rgba(31, 26, 23, 0.65);
  line-height: 1.6;
}
</style>
