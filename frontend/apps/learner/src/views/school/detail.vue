<script setup lang="ts">
// 学派详情页：学派名、起源朝代、创立者、概述。
import { onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { Spin, Descriptions, DescriptionsItem, Empty, Button } from 'ant-design-vue';

import PageHeader from '@/components/PageHeader.vue';
import { useApi } from '@/composables/useApi';
import { formatYear } from '@tcm/shared';
import type { Dynasty, Person, School } from '@tcm/api';

const props = defineProps<{ id?: string | number }>();
const route = useRoute();
const router = useRouter();
const apis = useApi();

const loading = ref(false);
const school = ref<School | null>(null);
const dynasty = ref<Dynasty | null>(null);
const founder = ref<Person | null>(null);

const schoolId = () => props.id ?? (route.params.id as string);

async function load() {
  loading.value = true;
  try {
    school.value = await apis.history.getSchool(schoolId());
    const s = school.value;
    if (!s) return;
    const tasks: Promise<unknown>[] = [];
    if (s.dynasty_id)
      tasks.push(apis.history.getDynasty(s.dynasty_id).then((d) => (dynasty.value = d)));
    if (s.founder_person_id)
      tasks.push(apis.history.getPerson(s.founder_person_id).then((p) => (founder.value = p)));
    await Promise.all(tasks);
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
      <template v-if="school">
        <PageHeader :title="school.name" subtitle="医学学派" />

        <Descriptions :column="2" bordered size="middle">
          <DescriptionsItem label="起源朝代">{{ dynasty?.name || '—' }}</DescriptionsItem>
          <DescriptionsItem label="创立年份">{{
            formatYear(school.established_year)
          }}</DescriptionsItem>
          <DescriptionsItem label="创立者">
            <a
              v-if="founder"
              @click="router.push({ name: 'PersonDetail', params: { id: founder.id } })"
            >
              {{ founder.name }}
            </a>
            <span v-else>—</span>
          </DescriptionsItem>
        </Descriptions>

        <section v-if="school.summary" class="block">
          <h2 class="section-title">学派概述</h2>
          <p class="section-text">{{ school.summary }}</p>
        </section>

        <section v-if="founder" class="block">
          <h2 class="section-title">创立者</h2>
          <div
            class="founder-card tcm-card-shadow"
            @click="router.push({ name: 'PersonDetail', params: { id: founder.id } })"
          >
            <div class="founder-avatar">{{ founder.name.charAt(0) }}</div>
            <div>
              <div class="founder-name">{{ founder.name }}</div>
              <div class="founder-desc">
                {{ founder.title || founder.alias_name || '学派创始人' }}
              </div>
            </div>
            <Button type="link" size="small">查看详情 →</Button>
          </div>
        </section>
      </template>
      <Empty v-else-if="!loading" description="未找到该学派" />
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
  border-left: 3px solid var(--tcm-color-celadon);
}

.section-text {
  margin: 0;
  line-height: 1.8;
  color: rgba(31, 26, 23, 0.85);
  font-size: 14px;
  white-space: pre-wrap;
}

.founder-card {
  display: flex;
  align-items: center;
  gap: var(--tcm-spacing-lg);
  padding: var(--tcm-spacing-lg);
  border-radius: var(--tcm-radius-lg);
  background-color: var(--tcm-color-paper);
  border: 1px solid rgba(31, 26, 23, 0.06);
  cursor: pointer;
}

.founder-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background-color: var(--tcm-color-celadon);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: serif;
  font-size: 22px;
}

.founder-name {
  font-size: 16px;
  font-weight: 600;
}

.founder-desc {
  font-size: 12px;
  color: rgba(31, 26, 23, 0.55);
}
</style>
