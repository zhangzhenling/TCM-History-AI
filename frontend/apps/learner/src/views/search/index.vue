<script setup lang="ts">
// 跨实体检索页：调用 History Service /search，按类型分组呈现结果。
import { computed, onMounted, ref } from 'vue';
import { Input, Spin, Empty, Tag, Button, message } from 'ant-design-vue';
import { SearchOutlined, FireOutlined, HistoryOutlined } from '@ant-design/icons-vue';
import { useRouter } from 'vue-router';

import PageHeader from '@/components/PageHeader.vue';
import { useApi } from '@/composables/useApi';
import { ENTITY_LABELS, type EntityType, truncate } from '@tcm/shared';
import type { SearchHit, SearchResponse } from '@tcm/api';

const apis = useApi();
const router = useRouter();

const HISTORY_KEY = 'tcm-search-history';
const MAX_HISTORY = 10;

const HOT_KEYWORDS = ['伤寒', '金匮要略', '麻黄', '张仲景', '千金方', '针灸', '本草纲目', '温病'];

const keyword = ref('');
const loading = ref(false);
const result = ref<SearchResponse | null>(null);
const searched = ref(false);
const error = ref(false);
const searchHistory = ref<string[]>([]);

onMounted(() => {
  loadHistory();
});

function loadHistory() {
  try {
    const saved = localStorage.getItem(HISTORY_KEY);
    if (saved) {
      searchHistory.value = JSON.parse(saved);
    }
  } catch {
    searchHistory.value = [];
  }
}

function saveHistory(kw: string) {
  const trimmed = kw.trim();
  if (!trimmed) return;
  const list = searchHistory.value.filter((k) => k !== trimmed);
  list.unshift(trimmed);
  searchHistory.value = list.slice(0, MAX_HISTORY);
  try {
    localStorage.setItem(HISTORY_KEY, JSON.stringify(searchHistory.value));
  } catch {
    // ignore storage errors
  }
}

function clearHistory() {
  searchHistory.value = [];
  try {
    localStorage.removeItem(HISTORY_KEY);
  } catch {
    // ignore
  }
}

function removeHistoryItem(kw: string) {
  searchHistory.value = searchHistory.value.filter((k) => k !== kw);
  try {
    localStorage.setItem(HISTORY_KEY, JSON.stringify(searchHistory.value));
  } catch {
    // ignore
  }
}

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
  const trimmed = keyword.value.trim();
  if (!trimmed) return;
  loading.value = true;
  error.value = false;
  searched.value = true;
  saveHistory(trimmed);
  try {
    result.value = await apis.history.search({
      q: trimmed,
      page: 1,
      page_size: 30,
    });
  } catch {
    error.value = true;
    message.error('搜索失败，请重试');
  } finally {
    loading.value = false;
  }
}

function useKeyword(kw: string) {
  keyword.value = kw;
  doSearch();
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
          : t === 'prescription'
            ? 'PrescriptionDetail'
            : t === 'medicine'
              ? 'MedicineDetail'
              : t === 'disease'
                ? 'DiseaseDetail'
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
        <template #button-content>
          <Button type="primary" @click="doSearch">搜索</Button>
        </template>
      </Input>
    </div>

    <!-- 搜索历史 -->
    <div v-if="searchHistory.length > 0" class="search-history">
      <div class="section-header">
        <HistoryOutlined />
        <span class="section-title">搜索历史</span>
        <Button type="link" size="small" @click="clearHistory">清空</Button>
      </div>
      <div class="tag-list">
        <Tag
          v-for="kw in searchHistory"
          :key="kw"
          class="history-tag"
          closable
          @close="removeHistoryItem(kw)"
          @click="useKeyword(kw)"
        >
          {{ kw }}
        </Tag>
      </div>
    </div>

    <!-- 热门搜索 -->
    <div class="hot-keywords">
      <div class="section-header">
        <FireOutlined style="color: #ff4d4f" />
        <span class="section-title">热门搜索</span>
      </div>
      <div class="tag-list">
        <Tag
          v-for="(kw, idx) in HOT_KEYWORDS"
          :key="kw"
          :color="idx < 3 ? 'volcano' : 'orange'"
          class="hot-tag"
          @click="useKeyword(kw)"
        >
          {{ idx < 3 ? '🔥' : '' }} {{ kw }}
        </Tag>
      </div>
    </div>

    <Spin :spinning="loading">
      <div v-if="error" class="state-wrap">
        <Empty description="搜索失败">
          <template #extra>
            <Button type="primary" @click="doSearch">重新搜索</Button>
          </template>
        </Empty>
      </div>

      <template v-else-if="result && result.total > 0">
        <div class="result-meta">
          共找到 <strong>{{ result.total }}</strong> 条结果
        </div>

        <div class="result-groups">
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
      </template>

      <Empty v-else-if="searched && !loading && !error" description="未找到相关结果" />
    </Spin>
  </div>
</template>

<style scoped lang="less">
.search-box {
  max-width: 640px;
  margin-bottom: var(--tcm-spacing-xl);
}

.search-history,
.hot-keywords {
  max-width: 640px;
  margin-bottom: var(--tcm-spacing-lg);
}

.section-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 10px;
  font-size: 13px;
  color: var(--tcm-color-text-secondary);

  .section-title {
    font-weight: 500;
  }
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.history-tag {
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    opacity: 0.8;
  }
}

.hot-tag {
  cursor: pointer;
  transition: all 0.2s;
  font-size: 13px;

  &:hover {
    transform: scale(1.05);
  }
}

.state-wrap {
  display: flex;
  justify-content: center;
  padding: var(--tcm-spacing-xl) 0;
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
  transition: box-shadow 0.2s;

  &:hover {
    box-shadow: 0 4px 12px rgba(31, 26, 23, 0.08);
  }
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
