<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { Spin, Empty, Pagination, Card, Button, Tag, message } from 'ant-design-vue';

import PageHeader from '@/components/PageHeader.vue';
import { useApi } from '@/composables/useApi';
import type { WrongQuestion } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const wrongQuestions = ref<WrongQuestion[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });

async function load() {
  loading.value = true;
  try {
    // Get current user ID from store or context
    const userId = 1; // TODO: get from auth store
    const res = await apis.learning.listWrongQuestions(userId, {
      page: query.page,
      page_size: query.page_size,
    });
    wrongQuestions.value = res.items ?? [];
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

async function markMastered(id: number) {
  try {
    await apis.learning.markWrongQuestionMastered(id);
    message.success('已标记为掌握');
    load();
  } catch {
    message.error('操作失败');
  }
}
</script>

<template>
  <div class="learning-page">
    <PageHeader title="错题本" subtitle="查漏补缺，巩固知识点" />

    <Spin :spinning="loading">
      <div v-if="wrongQuestions.length > 0" class="wrong-list">
        <Card v-for="item in wrongQuestions" :key="item.id" class="wrong-card">
          <div class="wrong-info">
            <div class="wrong-main">
              <div class="wrong-header">
                <Tag :color="item.is_mastered ? 'green' : 'red'">
                  {{ item.is_mastered ? '已掌握' : '未掌握' }}
                </Tag>
                <span class="wrong-count">错 {{ item.wrong_count }} 次</span>
                <span class="wrong-date">最近: {{ item.last_wrong_at?.slice(0, 10) || '—' }}</span>
              </div>
              <p class="wrong-question">题目 ID: {{ item.question_id }}</p>
            </div>
            <div v-if="!item.is_mastered" class="wrong-action">
              <Button type="primary" size="small" @click="markMastered(item.id)">标记已掌握</Button>
            </div>
          </div>
        </Card>
      </div>
      <Empty v-else description="暂无错题记录" />
    </Spin>

    <div v-if="total > 0" class="pagination-wrap">
      <Pagination
        :current="query.page"
        :page-size="query.page_size"
        :total="total"
        show-size-changer
        :page-size-options="['10', '20', '50']"
        @change="onPageChange"
      />
    </div>
  </div>
</template>

<style scoped lang="less">
.learning-page {
  max-width: 900px;
  margin: 0 auto;
}

.wrong-list {
  display: flex;
  flex-direction: column;
  gap: var(--tcm-spacing-md);
  margin-top: var(--tcm-spacing-lg);
}

.wrong-card {
  border-radius: var(--tcm-radius-lg);

  .wrong-info {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--tcm-spacing-md);

    .wrong-main {
      flex: 1;

      .wrong-header {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-bottom: 8px;

        .wrong-count {
          font-size: 12px;
          color: var(--tcm-color-text-tertiary);
        }

        .wrong-date {
          font-size: 12px;
          color: var(--tcm-color-text-tertiary);
        }
      }

      .wrong-question {
        font-size: 14px;
        color: var(--tcm-color-text-secondary);
      }
    }

    .wrong-action {
      flex-shrink: 0;
    }
  }
}

.pagination-wrap {
  display: flex;
  justify-content: center;
  margin-top: var(--tcm-spacing-xl);
}
</style>
