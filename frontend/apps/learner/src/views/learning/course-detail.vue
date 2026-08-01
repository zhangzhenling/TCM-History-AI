<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import {
  Spin,
  Descriptions,
  DescriptionsItem,
  Tag,
  Empty,
  List,
  Button,
  Progress,
  message,
  Modal,
} from 'ant-design-vue';

import PageHeader from '@/components/PageHeader.vue';
import { useApi } from '@/composables/useApi';
import { useUserStore } from '@tcm/stores';
import type { Course, Lesson, Enrollment } from '@tcm/api';

const props = defineProps<{ id?: string | number }>();
const route = useRoute();
const router = useRouter();
const apis = useApi();
const userStore = useUserStore();

const loading = ref(false);
const course = ref<Course | null>(null);
const lessons = ref<Lesson[]>([]);
const enrollment = ref<Enrollment | null>(null);

const courseId = () => props.id ?? (route.params.id as string);

const hasEnrolled = computed(() => !!enrollment.value);
const progressPercent = computed(
  () => Math.round((enrollment.value?.progress_percent ?? 0) * 100),
);

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
    const cid = courseId();
    if (!cid) {
      course.value = null;
      return;
    }
    course.value = await apis.learning.getCourse(cid);
    const [lessonRes, enrollRes] = await Promise.all([
      apis.learning.listLessonsByCourse(cid, { page: 1, page_size: 200 }),
      userStore.userId
        ? apis.learning.listEnrollments(userStore.userId, { page: 1, page_size: 200 })
        : null,
    ]);
    lessons.value = lessonRes.items ?? [];
    if (enrollRes && Array.isArray(enrollRes)) {
      enrollment.value =
        enrollRes.find((e) => e.course_id === Number(cid) && e.user_id === userStore.userId) ?? null;
    } else if (enrollRes) {
      enrollment.value =
        enrollRes.items?.find(
          (e) => e.course_id === Number(cid) && e.user_id === userStore.userId,
        ) ?? null;
    }
  } catch {
    message.error('加载课程详情失败，请重试');
  } finally {
    loading.value = false;
  }
}

async function handleEnroll() {
  if (!userStore.userId) {
    message.warning('请先登录');
    return;
  }
  try {
    const res = await apis.learning.enroll({
      user_id: userStore.userId,
      course_id: Number(courseId()),
    });
    enrollment.value = res;
    message.success('选课成功！');
  } catch {
    message.error('选课失败，请重试');
  }
}

function handleCancelEnroll() {
  Modal.confirm({
    title: '确认取消选课',
    content: '确定要取消选择这门课程吗？',
    okText: '确认取消',
    cancelText: '返回',
    okButtonProps: { danger: true },
    onOk: async () => {
      if (!enrollment.value) return;
      try {
        await apis.learning.unroll(enrollment.value.id);
        enrollment.value = null;
        message.success('已取消选课');
      } catch {
        message.error('取消选课失败，请重试');
      }
    },
  });
}

function goToLesson(lesson: Lesson) {
  if (!hasEnrolled.value && !lesson.is_free) {
    message.warning('请先选课');
    return;
  }
  router.push({ name: 'LessonDetail', params: { id: lesson.id } });
}

onMounted(load);
watch(() => props.id, load);
</script>

<template>
  <div class="course-detail-page">
    <Spin :spinning="loading">
      <template v-if="course">
        <PageHeader :title="course.title" :subtitle="course.description">
          <template #extra>
            <template v-if="hasEnrolled">
              <span class="progress-text">学习进度: {{ progressPercent }}%</span>
              <Progress
                :percent="progressPercent"
                :stroke-color="{ from: '#108ee9', to: '#87d068' }"
                class="header-progress"
              />
              <Button danger @click="handleCancelEnroll">取消选课</Button>
            </template>
            <Button v-else type="primary" @click="handleEnroll">立即选课</Button>
          </template>
        </PageHeader>

        <div class="detail-body">
          <section class="info-section">
            <h2 class="section-title">课程信息</h2>
            <Descriptions :column="2" bordered size="middle">
              <DescriptionsItem label="难度">
                <Tag :color="difficultyColors[course.difficulty] || 'default'">
                  {{ difficultyLabels[course.difficulty] || course.difficulty }}
                </Tag>
              </DescriptionsItem>
              <DescriptionsItem label="分类">{{ course.category || '—' }}</DescriptionsItem>
              <DescriptionsItem label="课时数">{{ course.lesson_count }}</DescriptionsItem>
              <DescriptionsItem label="总时长">{{ course.duration_minutes }} 分钟</DescriptionsItem>
              <DescriptionsItem label="状态">
                <Tag :color="course.is_published ? 'green' : 'default'">
                  {{ course.is_published ? '已发布' : '未发布' }}
                </Tag>
              </DescriptionsItem>
              <DescriptionsItem label="创建时间">{{ course.created_at?.slice(0, 10) }}</DescriptionsItem>
            </Descriptions>
          </section>

          <section class="info-section">
            <h2 class="section-title">课程目录</h2>
            <List
              v-if="lessons.length > 0"
              :data-source="lessons"
              item-layout="vertical"
              :pagination="false"
            >
              <template #renderItem="{ item }">
                <List.Item
                  class="lesson-item"
                  @click="goToLesson(item)"
                  :class="{ 'is-free': item.is_free }"
                >
                  <List.Item.Meta>
                    <template #title>
                      <span class="lesson-order">第 {{ item.sort_order }} 课时</span>
                      <Tag v-if="item.is_free" color="green">免费试看</Tag>
                      <Tag v-else color="blue">需选课</Tag>
                      <Tag
                        :color="item.content_type === 'video' ? 'orange' : item.content_type === 'audio' ? 'purple' : 'cyan'"
                      >
                        {{ item.content_type === 'video' ? '视频' : item.content_type === 'audio' ? '音频' : '图文' }}
                      </Tag>
                    </template>
                    <template #description>
                      <span>{{ item.duration_minutes }} 分钟</span>
                    </template>
                  </List.Item.Meta>
                </List.Item>
              </template>
            </List>
            <Empty v-else description="暂无课时" />
          </section>
        </div>
      </template>
      <Empty v-else-if="!loading" description="未找到该课程">
        <template #extra>
          <Button type="primary" @click="router.back()">返回</Button>
        </template>
      </Empty>
    </Spin>
  </div>
</template>

<style scoped lang="less">
.course-detail-page {
  max-width: 960px;
  margin: 0 auto;
}

.header-progress {
  width: 200px;
  margin-right: 16px;
  display: inline-block;
  vertical-align: middle;
}

.progress-text {
  margin-right: 8px;
  font-size: 13px;
  color: var(--tcm-color-text-secondary);
}

.detail-body {
  padding: var(--tcm-spacing-lg) 0;
}

.info-section {
  margin-bottom: var(--tcm-spacing-xl);
}

.section-title {
  margin: 0 0 var(--tcm-spacing-md);
  font-size: 18px;
  font-weight: 600;
  padding-left: 10px;
  border-left: 3px solid var(--tcm-color-indigo);
}

.lesson-item {
  cursor: pointer;
  transition: background-color 0.2s;
  padding: 12px 16px;
  border-radius: 8px;

  &:hover {
    background-color: var(--tcm-color-bg-secondary);
  }

  &.is-free {
    background-color: rgba(82, 196, 26, 0.04);
  }
}

.lesson-order {
  font-weight: 500;
  margin-right: 8px;
}
</style>
