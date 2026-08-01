<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { Spin, Empty, Pagination, Card, Button, Tag, message } from 'ant-design-vue';

import PageHeader from '@/components/PageHeader.vue';
import { useApi } from '@/composables/useApi';
import type { Exam } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const exams = ref<Exam[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 12 });

async function load() {
  loading.value = true;
  try {
    const res = await apis.learning.listExams({ page: query.page, page_size: query.page_size });
    exams.value = res.items ?? [];
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

function startExam(id: number) {
  message.info(`考试功能待实现 (ID: ${id})`);
}
</script>

<template>
  <div class="learning-page">
    <PageHeader title="考试中心" subtitle="检验学习成果" />

    <Spin :spinning="loading">
      <div v-if="exams.length > 0" class="exam-list">
        <Card v-for="exam in exams" :key="exam.id" class="exam-card">
          <div class="exam-info">
            <div class="exam-main">
              <h3 class="exam-title">{{ exam.title }}</h3>
              <p class="exam-desc">{{ exam.description || '暂无描述' }}</p>
              <div class="exam-meta">
                <Tag color="blue">{{ exam.question_count }} 题</Tag>
                <span>及格线: {{ exam.pass_score }} 分</span>
                <span>限时: {{ exam.duration_minutes }} 分钟</span>
              </div>
            </div>
            <div class="exam-action">
              <Button type="primary" :disabled="!exam.is_published" @click="startExam(exam.id)">
                {{ exam.is_published ? '开始考试' : '未发布' }}
              </Button>
            </div>
          </div>
        </Card>
      </div>
      <Empty v-else description="暂无考试" />
    </Spin>

    <div v-if="total > 0" class="pagination-wrap">
      <Pagination
        :current="query.page"
        :page-size="query.page_size"
        :total="total"
        show-size-changer
        :page-size-options="['12', '24', '48']"
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

.exam-list {
  display: flex;
  flex-direction: column;
  gap: var(--tcm-spacing-md);
  margin-top: var(--tcm-spacing-lg);
}

.exam-card {
  border-radius: var(--tcm-radius-lg);

  .exam-info {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--tcm-spacing-lg);

    .exam-main {
      flex: 1;

      .exam-title {
        font-size: 16px;
        font-weight: 600;
        margin-bottom: 6px;
      }

      .exam-desc {
        color: var(--tcm-color-text-secondary);
        font-size: 13px;
        margin-bottom: 10px;
      }

      .exam-meta {
        display: flex;
        align-items: center;
        gap: 16px;
        font-size: 12px;
        color: var(--tcm-color-text-tertiary);
      }
    }

    .exam-action {
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
