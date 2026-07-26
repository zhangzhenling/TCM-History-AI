<script setup lang="ts">
// 典籍详情页：书名、朝代、卷数、内容提要、是否存世。
import { onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { Spin, Descriptions, DescriptionsItem, Tag, Empty } from 'ant-design-vue';

import PageHeader from '@/components/PageHeader.vue';
import { useApi } from '@/composables/useApi';
import { formatYear } from '@tcm/shared';
import type { Book, Dynasty } from '@tcm/api';

const props = defineProps<{ id?: string | number }>();
const route = useRoute();
const apis = useApi();

const loading = ref(false);
const book = ref<Book | null>(null);
const dynasty = ref<Dynasty | null>(null);

const bookId = () => props.id ?? (route.params.id as string);

async function load() {
  loading.value = true;
  try {
    book.value = await apis.history.getBook(bookId());
    if (book.value.dynasty_id) {
      try {
        dynasty.value = await apis.history.getDynasty(book.value.dynasty_id);
      } catch {
        dynasty.value = null;
      }
    }
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
      <template v-if="book">
        <PageHeader :title="book.title" :subtitle="book.category || ''">
          <template #actions>
            <Tag v-if="book.is_extant" color="green">存世</Tag>
            <Tag v-else color="red">已佚</Tag>
          </template>
        </PageHeader>

        <Descriptions :column="2" bordered size="middle">
          <DescriptionsItem label="成书朝代">{{ dynasty?.name || '—' }}</DescriptionsItem>
          <DescriptionsItem label="成书年份">{{
            formatYear(book.published_year)
          }}</DescriptionsItem>
          <DescriptionsItem label="卷数">{{ book.volume_count || '—' }}</DescriptionsItem>
          <DescriptionsItem label="分类">{{ book.category || '—' }}</DescriptionsItem>
        </Descriptions>

        <section v-if="book.summary" class="block">
          <h2 class="section-title">内容提要</h2>
          <p class="section-text">{{ book.summary }}</p>
        </section>

        <section v-if="book.file_url" class="block">
          <h2 class="section-title">原文</h2>
          <a :href="book.file_url" target="_blank" rel="noopener">下载 / 在线阅读 →</a>
        </section>
      </template>
      <Empty v-else-if="!loading" description="未找到该典籍" />
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
