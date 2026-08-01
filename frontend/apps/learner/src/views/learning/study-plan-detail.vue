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
import type { StudyPlan } from '@tcm/api';

const props = defineProps<{ id?: string | number }>();
const route = useRoute();
const router = useRouter();
const apis = useApi();
const userStore = useUserStore();

const loading = ref(false);
const plan = ref<StudyPlan | null>(null);

const planId = () => props.id ?? (route.params.id as string);

const statusColors: Record<string, string> = {
  active: 'blue',
  completed: 'green',
  archived: 'default',
};

const statusLabels: Record<string, string> = {
  active: '进行中',
  completed: '已完成',
  archived: '已归档',
};

const courseList = computed<{ id: number; title: string }[]>(() => {
  if (!plan.value?.courses_json) return [];
  try {
    const data = typeof plan.value.courses_json === 'string'
      ? JSON.parse(plan.value.courses_json)
      : plan.value.courses_json;
    if (Array.isArray(data)) return data;
    return [];
  } catch {
    return [];
  }
});

async function load() {
  loading.value = true;
  try {
    const id = planId();
    if (!id) {
      plan.value = null;
      return;
    }
    plan.value = await apis.learning.getStudyPlan(id);
  } catch {
    message.error('加载学习计划详情失败，请重试');
  } finally {
    loading.value = false;
  }
}

async function handleDelete() {
  Modal.confirm({
    title: '确认删除',
    content: '确定要删除这个学习计划吗？此操作不可恢复。',
    okText: '确认删除',
    cancelText: '返回',
    okButtonProps: { danger: true },
    onOk: async () => {
      try {
        await apis.learning.deleteStudyPlan(Number(planId()));
        message.success('学习计划已删除');
        router.back();
      } catch {
        message.error('删除失败，请重试');
      }
    },
  });
}

async function handleUpdateStatus(status: string) {
  if (!plan.value) return;
  try {
    await apis.learning.updateStudyPlan(plan.value.id, {
      user_id: userStore.userId!,
      title: plan.value.title,
      status,
    });
    message.success('状态更新成功');
    await load();
  } catch {
    message.error('更新失败，请重试');
  }
}

onMounted(load);
watch(() => props.id, load);
</script>

<template>
  <div class="study-plan-detail-page">
    <Spin :spinning="loading">
      <template v-if="plan">
        <PageHeader :title="plan.title" subtitle="学习计划详情">
          <template #extra>
            <Tag :color="statusColors[plan.status] || 'default'">
              {{ statusLabels[plan.status] || plan.status }}
            </Tag>
            <Button
              v-if="plan.status === 'active'"
              @click="handleUpdateStatus('completed')"
              style="margin-left: 8px"
            >
              标记完成
            </Button>
            <Button
              v-if="plan.status === 'active'"
              danger
              @click="handleDelete"
              style="margin-left: 8px"
            >
              删除
            </Button>
          </template>
        </PageHeader>

        <div class="detail-body">
          <section class="info-section">
            <h2 class="section-title">计划概览</h2>
            <Descriptions :column="2" bordered size="middle">
              <DescriptionsItem label="创建时间">{{ plan.created_at?.slice(0, 10) }}</DescriptionsItem>
              <DescriptionsItem label="更新时间">{{ plan.updated_at?.slice(0, 10) }}</DescriptionsItem>
              <DescriptionsItem label="目标日期">{{ plan.target_date || '—' }}</DescriptionsItem>
              <DescriptionsItem label="进度">
                <Progress
                  :percent="plan.progress_percent"
                  :stroke-color="{ from: '#108ee9', to: '#87d068' }"
                  style="width: 160px"
                />
              </DescriptionsItem>
            </Descriptions>
          </section>

          <section class="info-section">
            <h2 class="section-title">计划课程</h2>
            <List
              v-if="courseList.length > 0"
              :data-source="courseList"
              item-layout="vertical"
            >
              <template #renderItem="{ item }">
                <List.Item class="course-item">
                  <List.Item.Meta>
                    <template #title>
                      <span>{{ item.title }}</span>
                    </template>
                  </List.Item.Meta>
                </List.Item>
              </template>
            </List>
            <Empty v-else description="暂无关联课程" />
          </section>
        </div>
      </template>
      <Empty v-else-if="!loading" description="未找到该学习计划">
        <template #extra>
          <Button type="primary" @click="router.back()">返回</Button>
        </template>
      </Empty>
    </Spin>
  </div>
</template>

<style scoped lang="less">
.study-plan-detail-page {
  max-width: 960px;
  margin: 0 auto;
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

.course-item {
  padding: 12px 16px;
  border-radius: 8px;
  transition: background-color 0.2s;

  &:hover {
    background-color: var(--tcm-color-bg-secondary);
  }
}
</style>
