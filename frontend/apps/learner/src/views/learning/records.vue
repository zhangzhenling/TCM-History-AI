<script setup lang="ts">
import { h, onMounted, reactive, ref, computed } from 'vue';
import { Spin, Empty, Pagination, Table, Tag, message } from 'ant-design-vue';

import PageHeader from '@/components/PageHeader.vue';
import { useApi } from '@/composables/useApi';
import { useUserStore } from '@tcm/stores';
import type { LearningRecord, ListResponse } from '@tcm/api';

const apis = useApi();
const userStore = useUserStore();

const loading = ref(false);
const records = ref<LearningRecord[]>([]);
const total = ref(0);

const query = reactive({ page: 1, page_size: 10 });

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '课程 ID', dataIndex: 'course_id', key: 'course_id', width: 100 },
  { title: '课时 ID', dataIndex: 'lesson_id', key: 'lesson_id', width: 100 },
  {
    title: '学习时长',
    dataIndex: 'duration_seconds',
    key: 'duration_seconds',
    width: 120,
    customRender: ({ record }: { record: LearningRecord }) =>
      `${Math.floor(record.duration_seconds / 60)} 分 ${record.duration_seconds % 60} 秒`,
  },
  {
    title: '进度',
    dataIndex: 'position_percent',
    key: 'position_percent',
    width: 100,
    customRender: ({ record }: { record: LearningRecord }) =>
      `${(record.position_percent * 100).toFixed(0)}%`,
  },
  {
    title: '状态',
    dataIndex: 'is_completed',
    key: 'is_completed',
    width: 100,
    customRender: ({ record }: { record: LearningRecord }) =>
      record.is_completed
        ? h(Tag, { color: 'green' }, () => '已完成')
        : h(Tag, { color: 'blue' }, () => '学习中'),
  },
  { title: '学习时间', dataIndex: 'learned_at', key: 'learned_at', width: 180 },
];

const dataSource = computed(() => records.value.map((r) => ({ ...r, key: r.id })));

async function load() {
  loading.value = true;
  try {
    const userId = userStore.userId;
    if (!userId) {
      records.value = [];
      total.value = 0;
      return;
    }
    const res = await apis.learning.listLearningRecords(userId, {
      page: query.page,
      page_size: query.page_size,
    });
    if (Array.isArray(res)) {
      records.value = res;
      total.value = res.length;
    } else {
      records.value = (res as ListResponse<LearningRecord>).items ?? [];
      total.value = (res as ListResponse<LearningRecord>).total ?? 0;
    }
  } catch {
    message.error('加载学习记录失败，请重试');
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
</script>

<template>
  <div class="tcm-container">
    <PageHeader title="学习记录" subtitle="查看您的学习轨迹" />

    <Spin :spinning="loading">
      <Table
        v-if="records.length > 0"
        :columns="columns"
        :data-source="dataSource"
        :pagination="false"
        bordered
        size="middle"
      />
      <Empty v-else description="暂无学习记录" />
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
.pagination-wrap {
  display: flex;
  justify-content: center;
  margin-top: var(--tcm-spacing-xl);
}
</style>
