<script setup lang="ts">
// 首页：欢迎区 + 快捷入口 + 最近医家/典籍。
// P3 阶段未接入 Learning/AI Service，今日学习卡片以占位呈现。

import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { Spin, Empty } from 'ant-design-vue';

import PageHeader from '@/components/PageHeader.vue';
import EntityCard from '@/components/EntityCard.vue';
import { useApi } from '@/composables/useApi';
import { useUserStore } from '@tcm/stores';
import { truncate } from '@tcm/shared';
import type { Person, Book } from '@tcm/api';

const router = useRouter();
const apis = useApi();
const userStore = useUserStore();

const loading = ref(false);
const persons = ref<Person[]>([]);
const books = ref<Book[]>([]);

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
      apis.history.listPersons({ page: 1, page_size: 4 }),
      apis.history.listBooks({ page: 1, page_size: 4 }),
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
  <div class="tcm-container">
    <PageHeader
      :title="`欢迎，${userStore.nickname || '学习者'}`"
      subtitle="中医发展史 AI 学习平台 · 首页"
    />

    <section class="quick-grid">
      <div
        v-for="entry in quickEntries"
        :key="entry.name"
        class="quick-entry tcm-card-shadow"
        :style="{ '--entry-color': entry.color }"
        @click="go(entry.name)"
      >
        <div class="quick-entry-accent" />
        <h3 class="quick-entry-title">{{ entry.title }}</h3>
        <p class="quick-entry-desc">{{ entry.desc }}</p>
        <span class="quick-entry-arrow">→</span>
      </div>
    </section>

    <Spin :spinning="loading">
      <section class="home-section">
        <div class="section-header">
          <h2 class="section-title">医家一览</h2>
          <a class="section-more" @click="go('PersonList')">查看全部 →</a>
        </div>
        <div v-if="persons.length" class="card-grid">
          <EntityCard
            v-for="p in persons"
            :id="p.id"
            :key="p.id"
            type="person"
            :title="p.name"
            :subtitle="p.title || p.alias_name || ''"
            :description="truncate(p.biography || p.achievements || '', 60)"
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
        <div v-if="books.length" class="card-grid">
          <EntityCard
            v-for="b in books"
            :id="b.id"
            :key="b.id"
            type="book"
            :title="b.title"
            :subtitle="b.category || ''"
            :description="truncate(b.summary || '', 60)"
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
}

@media (max-width: 1024px) {
  .quick-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .quick-grid {
    grid-template-columns: 1fr;
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
  transition:
    transform 0.2s ease,
    box-shadow 0.2s ease;
}
.quick-entry:hover {
  transform: translateY(-2px);
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
}

.quick-entry-desc {
  margin: 0;
  font-size: 12px;
  color: rgba(31, 26, 23, 0.6);
  line-height: 1.6;
}

.quick-entry-arrow {
  position: absolute;
  right: 16px;
  bottom: 14px;
  color: var(--entry-color);
  font-size: 16px;
}

.home-section {
  margin-top: var(--tcm-spacing-xl);
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

.section-more {
  font-size: 13px;
  cursor: pointer;
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
</style>
