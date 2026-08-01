<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { Spin, Empty, Pagination, Card, Tag, message } from 'ant-design-vue';

import PageHeader from '@/components/PageHeader.vue';
import { useApi } from '@/composables/useApi';
import type { Course } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const courses = ref<Course[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 12 });

const difficultyColors: Record<string, string> = {
  beginner: 'green',
  intermediate: 'blue',
  advanced: 'red',
};

const difficultyLabels: Record<string, string> = {
  beginner: '入门',
  intermediate: '进阶',
  advanced: '高级',
};

async function load() {
  loading.value = true;
  try {
    const res = await apis.learning.listCourses({ page: query.page, page_size: query.page_size });
    courses.value = res.items ?? [];
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

function goToCourse(id: number) {
  // Navigate to course detail (placeholder for future implementation)
  message.info(`课程详情页待实现 (ID: ${id})`);
}
</script>

<template>
  <div class="learning-page">
    <PageHeader title="课程中心" subtitle="系统学习中医理论知识" />

    <Spin :spinning="loading">
      <div v-if="courses.length > 0" class="course-grid">
        <Card
          v-for="course in courses"
          :key="course.id"
          class="course-card"
          hoverable
          @click="goToCourse(course.id)"
        >
          <div class="course-cover">
            <img v-if="course.cover_url" :src="course.cover_url" :alt="course.title" />
            <div v-else class="course-cover-placeholder">{{ course.title.charAt(0) }}</div>
          </div>
          <div class="course-info">
            <h3 class="course-title">{{ course.title }}</h3>
            <p class="course-desc">{{ course.description || '暂无描述' }}</p>
            <div class="course-meta">
              <Tag :color="difficultyColors[course.difficulty] || 'default'">
                {{ difficultyLabels[course.difficulty] || course.difficulty }}
              </Tag>
              <span class="course-lessons">{{ course.lesson_count }} 课时</span>
              <span class="course-duration">{{ course.duration_minutes }} 分钟</span>
            </div>
          </div>
        </Card>
      </div>
      <Empty v-else description="暂无课程" />
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
  max-width: 1200px;
  margin: 0 auto;
}

.course-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: var(--tcm-spacing-lg);
  margin-top: var(--tcm-spacing-lg);
}

.course-card {
  border-radius: var(--tcm-radius-lg);
  overflow: hidden;

  .course-cover {
    height: 160px;
    overflow: hidden;
    background-color: var(--tcm-color-bg-secondary);

    img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }

    .course-cover-placeholder {
      display: flex;
      align-items: center;
      justify-content: center;
      height: 100%;
      font-size: 48px;
      font-weight: 700;
      color: var(--tcm-color-primary);
      background: linear-gradient(
        135deg,
        var(--tcm-color-primary-light),
        var(--tcm-color-primary-bg)
      );
    }
  }

  .course-info {
    padding: var(--tcm-spacing-md);

    .course-title {
      font-size: 16px;
      font-weight: 600;
      margin-bottom: 8px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .course-desc {
      color: var(--tcm-color-text-secondary);
      font-size: 13px;
      margin-bottom: 12px;
      display: -webkit-box;
      -webkit-line-clamp: 2;
      -webkit-box-orient: vertical;
      overflow: hidden;
    }

    .course-meta {
      display: flex;
      align-items: center;
      gap: 12px;
      font-size: 12px;
      color: var(--tcm-color-text-tertiary);
    }
  }
}

.pagination-wrap {
  display: flex;
  justify-content: center;
  margin-top: var(--tcm-spacing-xl);
}
</style>
