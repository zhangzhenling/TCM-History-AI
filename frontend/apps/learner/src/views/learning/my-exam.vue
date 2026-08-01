<script setup lang="ts">
import { h, computed, onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { Spin, Descriptions, DescriptionsItem, Tag, Empty, Table, Button, message } from 'ant-design-vue';

import PageHeader from '@/components/PageHeader.vue';
import { useApi } from '@/composables/useApi';
import { useUserStore } from '@tcm/stores';
import type { Exam, Question, ExamAttempt } from '@tcm/api';

const props = defineProps<{ id?: string | number }>();
const route = useRoute();
const apis = useApi();
const userStore = useUserStore();

const loading = ref(false);
const exam = ref<Exam | null>(null);
const questions = ref<Question[]>([]);
const attempts = ref<ExamAttempt[]>([]);
const attemptsTotal = ref(0);
const attemptsQuery = ref({ page: 1, page_size: 10 });

const examId = () => props.id ?? (route.params.id as string);

const questionColumns = [
  { title: '#', dataIndex: 'id', key: 'id', width: 60 },
  {
    title: '题型',
    dataIndex: 'type',
    key: 'type',
    width: 120,
    customRender: ({ record }: { record: Question }) => {
      const map: Record<string, string> = {
        single_choice: '单选题',
        multiple_choice: '多选题',
        true_false: '判断题',
        fill_blank: '填空题',
        essay: '问答题',
      };
      return map[record.type] || record.type;
    },
  },
  { title: '题目', dataIndex: 'content', key: 'content' },
  { title: '分值', dataIndex: 'score', key: 'score', width: 80 },
  { title: '难度', dataIndex: 'difficulty', key: 'difficulty', width: 80 },
];

const attemptColumns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  {
    title: '得分',
    key: 'score',
    width: 120,
    customRender: ({ record }: { record: ExamAttempt }) =>
      `${record.score} / ${record.total_score}`,
  },
  {
    title: '结果',
    dataIndex: 'is_passed',
    key: 'is_passed',
    width: 100,
    customRender: ({ record }: { record: ExamAttempt }) =>
      record.is_passed
        ? h(Tag, { color: 'green' }, () => '及格')
        : h(Tag, { color: 'red' }, () => '不及格'),
  },
  {
    title: '用时',
    key: 'duration',
    width: 120,
    customRender: ({ record }: { record: ExamAttempt }) =>
      `${Math.floor(record.duration_seconds / 60)} 分 ${record.duration_seconds % 60} 秒`,
  },
  { title: '开始时间', dataIndex: 'started_at', key: 'started_at', width: 180 },
  {
    title: '提交时间',
    dataIndex: 'submitted_at',
    key: 'submitted_at',
    width: 180,
    customRender: ({ record }: { record: ExamAttempt }) =>
      record.submitted_at || '进行中',
  },
];

const attemptDataSource = computed(() =>
  attempts.value.map((a) => ({ ...a, key: a.id })),
);

async function loadAttempts() {
  const userId = userStore.userId;
  if (!userId) return;
  try {
    const res = await apis.learning.listExamAttempts(userId, examId(), {
      page: attemptsQuery.value.page,
      page_size: attemptsQuery.value.page_size,
    });
    attempts.value = res.items ?? [];
    attemptsTotal.value = res.total ?? 0;
  } catch {
    attempts.value = [];
    attemptsTotal.value = 0;
  }
}

async function load() {
  loading.value = true;
  try {
    exam.value = await apis.learning.getExam(examId());
    if (exam.value) {
      questions.value = await apis.learning.listQuestionsByExam(exam.value.id);
    }
    await loadAttempts();
  } finally {
    loading.value = false;
  }
}

onMounted(load);
watch(() => props.id, load);

function onAttemptsPageChange(p: number, ps: number) {
  attemptsQuery.value.page = p;
  attemptsQuery.value.page_size = ps;
  loadAttempts();
}

function startExam() {
  message.info('考试功能待实现');
}
</script>

<template>
  <div class="tcm-container">
    <Spin :spinning="loading">
      <template v-if="exam">
        <PageHeader :title="exam.title" :subtitle="exam.description || ''">
          <template #actions>
            <Tag v-if="exam.is_published" color="green">已发布</Tag>
            <Tag v-else color="default">未发布</Tag>
            <Button type="primary" :disabled="!exam.is_published" @click="startExam">
              开始考试
            </Button>
          </template>
        </PageHeader>

        <Descriptions :column="3" bordered size="middle">
          <DescriptionsItem label="题目数量">{{ exam.question_count }} 题</DescriptionsItem>
          <DescriptionsItem label="及格分数">{{ exam.pass_score }} 分</DescriptionsItem>
          <DescriptionsItem label="考试时长">{{ exam.duration_minutes }} 分钟</DescriptionsItem>
        </Descriptions>

        <section class="block">
          <h2 class="section-title">题目列表</h2>
          <Table
            v-if="questions.length > 0"
            :columns="questionColumns"
            :data-source="questions.map((q) => ({ ...q, key: q.id }))"
            :pagination="false"
            bordered
            size="small"
          />
          <Empty v-else description="暂无题目" />
        </section>

        <section class="block">
          <h2 class="section-title">考试记录</h2>
          <Table
            v-if="attempts.length > 0"
            :columns="attemptColumns"
            :data-source="attemptDataSource"
            :pagination="false"
            bordered
            size="middle"
          />
          <Empty v-else description="暂无考试记录" />

          <div v-if="attemptsTotal > 0" class="pagination-wrap">
            <Pagination
              :current="attemptsQuery.page"
              :page-size="attemptsQuery.page_size"
              :total="attemptsTotal"
              @change="onAttemptsPageChange"
            />
          </div>
        </section>
      </template>
      <Empty v-else-if="!loading" description="未找到该考试" />
    </Spin>
  </div>
</template>

<style scoped lang="less">
.block {
  margin-top: var(--tcm-spacing-xl);
}

.section-title {
  margin: 0 0 12px;
  font-size: 18px;
  font-weight: 600;
  padding-left: 10px;
  border-left: 3px solid var(--tcm-color-primary);
}

.pagination-wrap {
  display: flex;
  justify-content: center;
  margin-top: var(--tcm-spacing-lg);
}
</style>