<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { Spin, Empty, Pagination, Card, Tag, Progress, message } from 'ant-design-vue';
import { useRouter } from 'vue-router';

import PageHeader from '@/components/PageHeader.vue';
import { useApi } from '@/composables/useApi';
import { useUserStore } from '@tcm/stores';
import type { StudyPlan, ListResponse } from '@tcm/api';

const apis = useApi();
const userStore = useUserStore();
const router = useRouter();

const loading = ref(false);
const error = ref(false);
const plans = ref<StudyPlan[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });

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

async function load() {
  loading.value = true;
  error.value = false;
  try {
    const userId = userStore.userId;
    if (!userId) {
      plans.value = [];
      total.value = 0;
      return;
    }
    const res = (await apis.learning.listStudyPlans(userId, {
      page: query.page,
      page_size: query.page_size,
    })) as ListResponse<StudyPlan> | StudyPlan[];
    if (Array.isArray(res)) {
      plans.value = res;
      total.value = res.length;
    } else {
      plans.value = res.items ?? [];
      total.value = res.total ?? 0;
    }
  } catch {
    error.value = true;
    message.error('加载学习计划失败，请重试');
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

function viewPlan(id: number) {
  router.push({ name: 'StudyPlanDetail', params: { id } });
}
</script>

<template>
  <div class="learning-page">
    <PageHeader title="学习计划" subtitle="制定学习目标，跟踪学习进度">
      <template #extra>
        <a-button type="primary" @click="load" :loading="loading">刷新</a-button>
      </template>
    </PageHeader>

    <Spin :spinning="loading">
      <div v-if="error" class="state-wrap">
        <Empty description="加载失败">
          <template #extra>
            <a-button type="primary" @click="load">重新加载</a-button>
          </template>
        </Empty>
      </div>
      <div v-else-if="plans.length > 0" class="plan-list">
        <Card
          v-for="plan in plans"
          :key="plan.id"
          class="plan-card"
          hoverable
          @click="viewPlan(plan.id)"
        >
          <div class="plan-info">
            <div class="plan-header">
              <h3 class="plan-title">{{ plan.title }}</h3>
              <Tag :color="statusColors[plan.status] || 'default'">
                {{ statusLabels[plan.status] || plan.status }}
              </Tag>
            </div>
            <div class="plan-progress">
              <Progress
                :percent="plan.progress_percent"
                :stroke-color="{ from: '#108ee9', to: '#87d068' }"
              />
            </div>
            <div class="plan-footer">
              <span v-if="plan.target_date" class="plan-target"
                >目标日期: {{ plan.target_date }}</span
              >
              <span class="plan-updated">更新于: {{ plan.updated_at?.slice(0, 10) || '—' }}</span>
            </div>
          </div>
        </Card>
      </div>
      <Empty v-else description="暂无学习计划" />
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

.plan-list {
  display: flex;
  flex-direction: column;
  gap: var(--tcm-spacing-md);
  margin-top: var(--tcm-spacing-lg);
}

.plan-card {
  border-radius: var(--tcm-radius-lg);

  .plan-info {
    .plan-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: var(--tcm-spacing-md);

      .plan-title {
        font-size: 16px;
        font-weight: 600;
        margin: 0;
      }
    }

    .plan-progress {
      margin-bottom: var(--tcm-spacing-md);
    }

    .plan-footer {
      display: flex;
      justify-content: space-between;
      font-size: 12px;
      color: var(--tcm-color-text-tertiary);
    }
  }
}

.state-wrap {
  display: flex;
  justify-content: center;
  padding: var(--tcm-spacing-xl) 0;
}

.pagination-wrap {
  display: flex;
  justify-content: center;
  margin-top: var(--tcm-spacing-xl);
}
</style>
