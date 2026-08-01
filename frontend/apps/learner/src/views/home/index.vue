<script setup lang="ts">
// 首页：欢迎区 + 快捷入口 + AI 学习推荐 + 最近医家/典籍。
// P3 阶段未接入 Learning/AI Service，推荐以规则引擎生成。

import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { Spin, Empty, Card, Tag } from 'ant-design-vue';
import { BulbOutlined, RightOutlined } from '@ant-design/icons-vue';

import PageHeader from '@/components/PageHeader.vue';
import EntityCard from '@/components/EntityCard.vue';
import { useApi } from '@/composables/useApi';
import { useUserStore } from '@tcm/stores';
import { truncate } from '@tcm/shared';
import type { Person, Book } from '@tcm/api';
import { useViewport } from '@/composables/useViewport';

const router = useRouter();
const apis = useApi();
const userStore = useUserStore();
const { isMobile } = useViewport();

const loading = ref(false);
const persons = ref<Person[]>([]);
const books = ref<Book[]>([]);

// ---- AI 学习推荐（本地规则引擎）----
interface Recommendation {
  id: string;
  type: 'course' | 'entity' | 'search';
  title: string;
  reason: string;
  action: { name: string; params?: Record<string, unknown> };
  color: string;
  icon: string;
}

const recommendations = computed<Recommendation[]>(() => {
  const list: Recommendation[] = [];

  // 基于热门实体推荐
  if (persons.value[0]) {
    list.push({
      id: `p-${persons.value[0].id}`,
      type: 'entity',
      title: `探索 ${persons.value[0].name}`,
      reason: '热门医家',
      action: { name: 'PersonDetail', params: { id: persons.value[0].id } },
      color: '#a23a30',
      icon: '👤',
    });
  }
  if (books.value[0]) {
    list.push({
      id: `b-${books.value[0].id}`,
      type: 'entity',
      title: `研读《${books.value[0].title}》`,
      reason: '经典典籍',
      action: { name: 'BookDetail', params: { id: books.value[0].id } },
      color: '#2c4a6b',
      icon: '📖',
    });
  }

  // 固定推荐
  list.push({
    id: 'r-timeline',
    type: 'search',
    title: '浏览发展时间轴',
    reason: '推荐学习路径',
    action: { name: 'Timeline' },
    color: '#5c8a6a',
    icon: '⏳',
  });
  list.push({
    id: 'r-search',
    type: 'search',
    title: '搜索「伤寒论」',
    reason: '热门检索',
    action: { name: 'Search' },
    color: '#c9a24a',
    icon: '🔍',
  });

  return list.slice(0, isMobile.value ? 3 : 4);
});

function goRecommend(rec: Recommendation) {
  router.push({ name: rec.action.name, params: rec.action.params as Record<string, string | number> });
}

const quickEntries = [
  {
    name: 'Timeline',
    title: '发展时间轴',
    desc: '按朝代纵览中医千年演进',
    color: 'var(--tcm-color-celadon)',
  },
  {
    name: 'PersonList',
    title: '历代医家',
    desc: '走近张仲景、孙思邈等大家',
    color: 'var(--tcm-color-primary)',
  },
  {
    name: 'BookList',
    title: '中医典籍',
    desc: '素问、本草、伤寒论经典原文',
    color: 'var(--tcm-color-indigo)',
  },
  {
    name: 'SchoolList',
    title: '医学学派',
    desc: '伤寒、温病、汇通学派源流',
    color: 'var(--tcm-color-gold)',
  },
];

onMounted(async () => {
  loading.value = true;
  try {
    const [personsRes, booksRes] = await Promise.all([
      apis.history.listPersons({ page: 1, page_size: isMobile.value ? 3 : 4 }),
      apis.history.listBooks({ page: 1, page_size: isMobile.value ? 3 : 4 }),
    ]);
    persons.value = personsRes.items ?? [];
    books.value = booksRes.items ?? [];
  } finally {
    loading.value = false;
  }
});

function go(name: string) {
  router.push({ name });
}
</script>

<template>
  <div class="tcm-container home-page" :class="{ 'is-mobile': isMobile }">
    <PageHeader
      :title="`欢迎，${userStore.nickname || '学习者'}`"
      subtitle="中医发展史 AI 学习平台 · 首页"
    />

    <!-- 移动端：快捷入口 2x2 网格 -->
    <section class="quick-grid" :class="{ 'mobile-grid': isMobile }">
      <div
        v-for="entry in quickEntries"
        :key="entry.name"
        class="quick-entry tcm-card-shadow tcm-touch-target"
        :style="{ '--entry-color': entry.color }"
        @click="go(entry.name)"
      >
        <div class="quick-entry-accent" />
        <h3 class="quick-entry-title">{{ entry.title }}</h3>
        <p class="quick-entry-desc">{{ entry.desc }}</p>
        <span class="quick-entry-arrow">→</span>
      </div>
    </section>

    <!-- AI 学习推荐 -->
    <section v-if="recommendations.length" class="recommend-section">
      <div class="section-header">
        <h2 class="section-title">
          <BulbOutlined class="ai-icon" /> AI 学习推荐
        </h2>
        <span class="section-sub">为你精选的学习路径</span>
      </div>
      <div class="recommend-grid">
        <Card
          v-for="rec in recommendations"
          :key="rec.id"
          class="recommend-card tcm-card-shadow"
          :style="{ '--rec-color': rec.color }"
          hoverable
          @click="goRecommend(rec)"
        >
          <div class="rec-icon">{{ rec.icon }}</div>
          <div class="rec-content">
            <div class="rec-title">{{ rec.title }}</div>
            <div class="rec-reason">
              <Tag :color="rec.color" style="color: #fff">{{ rec.reason }}</Tag>
            </div>
          </div>
          <RightOutlined class="rec-arrow" />
        </Card>
      </div>
    </section>

    <Spin :spinning="loading">
      <section class="home-section">
        <div class="section-header">
          <h2 class="section-title">医家一览</h2>
          <a class="section-more" @click="go('PersonList')">查看全部 →</a>
        </div>
        <div v-if="persons.length" class="card-grid" :class="{ 'mobile-grid-1': isMobile }">
          <EntityCard
            v-for="p in persons"
            :id="p.id"
            :key="p.id"
            type="person"
            :title="p.name"
            :subtitle="p.title || p.alias_name || ''"
            :description="truncate(p.biography || p.achievements || '', isMobile ? 40 : 60)"
            :year-range="{ start: p.birth_year, end: p.death_year }"
          />
        </div>
        <Empty v-else description="暂无医家数据" />
      </section>

      <section class="home-section">
        <div class="section-header">
          <h2 class="section-title">典籍推荐</h2>
          <a class="section-more" @click="go('BookList')">查看全部 →</a>
        </div>
        <div v-if="books.length" class="card-grid" :class="{ 'mobile-grid-1': isMobile }">
          <EntityCard
            v-for="b in books"
            :id="b.id"
            :key="b.id"
            type="book"
            :title="b.title"
            :subtitle="b.category || ''"
            :description="truncate(b.summary || '', isMobile ? 40 : 60)"
            :year-range="{ start: b.published_year, end: b.published_year }"
          />
        </div>
        <Empty v-else description="暂无典籍数据" />
      </section>
    </Spin>
  </div>
</template>

<style scoped lang="less">
.quick-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--tcm-spacing-lg);
  margin-bottom: var(--tcm-spacing-xl);

  &.mobile-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: var(--tcm-spacing-base);
  }
}

@media (max-width: 1024px) {
  .quick-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .quick-grid {
    grid-template-columns: repeat(2, 1fr);
    margin-bottom: var(--tcm-spacing-lg);
  }
}

@media (max-width: 480px) {
  .quick-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 10px;
  }
}

.quick-entry {
  position: relative;
  padding: 18px 20px;
  border-radius: var(--tcm-radius-lg);
  background-color: var(--tcm-color-paper);
  border: 1px solid rgba(31, 26, 23, 0.06);
  cursor: pointer;
  overflow: hidden;
  min-height: 100px;
  transition:
    transform 0.2s ease,
    box-shadow 0.2s ease;

  &:active {
    transform: scale(0.98);
  }
}

.quick-entry:hover {
  transform: translateY(-2px);
}

@media (max-width: 768px) {
  .quick-entry {
    padding: 14px;
    min-height: 86px;
  }
}

.quick-entry-accent {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 4px;
  background-color: var(--entry-color);
}

.quick-entry-title {
  margin: 0 0 6px;
  font-size: 16px;
  font-weight: 600;
  padding-left: 8px;
}

@media (max-width: 768px) {
  .quick-entry-title {
    font-size: 14px;
    padding-left: 6px;
  }
}

.quick-entry-desc {
  margin: 0;
  font-size: 12px;
  color: var(--tcm-color-text-secondary);
  line-height: 1.5;
  padding-left: 8px;
}

@media (max-width: 768px) {
  .quick-entry-desc {
    font-size: 11px;
    line-height: 1.4;
    padding-left: 6px;
  }
}

.quick-entry-arrow {
  position: absolute;
  right: 12px;
  bottom: 10px;
  color: var(--entry-color);
  font-size: 16px;
}

@media (max-width: 480px) {
  .quick-entry-arrow {
    font-size: 14px;
    right: 10px;
    bottom: 8px;
  }
}

.home-section {
  margin-top: var(--tcm-spacing-xl);
}

/* AI 推荐区 */
.recommend-section {
  margin-bottom: var(--tcm-spacing-xl);
}

.recommend-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: var(--tcm-spacing-lg);
}

.recommend-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  cursor: pointer;
  transition: transform 0.2s;
  border-left: 4px solid var(--rec-color);

  &:active {
    transform: scale(0.98);
  }

  :deep(.ant-card-body) {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 0;
    width: 100%;
  }
}

.rec-icon {
  font-size: 28px;
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  background-color: var(--tcm-color-bg-secondary);
  flex-shrink: 0;
}

.rec-content {
  flex: 1;
  min-width: 0;
}

.rec-title {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.rec-reason {
  display: flex;
}

.rec-arrow {
  color: var(--tcm-color-text-tertiary);
  font-size: 14px;
  flex-shrink: 0;
}

.ai-icon {
  color: var(--tcm-color-gold);
  margin-right: 6px;
}

.section-sub {
  font-size: 13px;
  color: var(--tcm-color-text-tertiary);
}

@media (max-width: 768px) {
  .home-section {
    margin-top: var(--tcm-spacing-lg);
  }

  .recommend-section {
    margin-bottom: var(--tcm-spacing-lg);
  }

  .recommend-grid {
    grid-template-columns: 1fr;
    gap: var(--tcm-spacing-base);
  }

  .recommend-card {
    padding: 10px 12px;
  }

  .rec-icon {
    width: 36px;
    height: 36px;
    font-size: 22px;
  }

  .rec-title {
    font-size: 13px;
  }
}

.section-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: var(--tcm-spacing-base);
}

.section-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
}

@media (max-width: 768px) {
  .section-title {
    font-size: 17px;
  }
}

.section-more {
  font-size: 13px;
  cursor: pointer;
  padding: 6px 2px;
  min-height: 44px;
  min-width: 60px;
  display: inline-flex;
  align-items: center;
}

@media (max-width: 768px) {
  .section-more {
    font-size: 12px;
  }
}

.card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: var(--tcm-spacing-lg);
}

.card-grid.mobile-grid-1 {
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: var(--tcm-spacing-base);
}

@media (max-width: 480px) {
  .card-grid {
    grid-template-columns: 1fr;
    gap: var(--tcm-spacing-base);
  }

  .card-grid.mobile-grid-1 {
    grid-template-columns: 1fr;
  }
}
</style>
