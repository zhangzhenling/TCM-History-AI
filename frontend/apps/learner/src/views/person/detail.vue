<script setup lang="ts">
// 人物详情页：头图、生平、成就、朝代信息。
import { onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { Spin, Descriptions, DescriptionsItem, Tag, Empty } from 'ant-design-vue';

import PageHeader from '@/components/PageHeader.vue';
import { useApi } from '@/composables/useApi';
import { formatYear, formatYearRange } from '@tcm/shared';
import type { Dynasty, Person } from '@tcm/api';

const props = defineProps<{ id?: string | number }>();
const route = useRoute();
const apis = useApi();

const loading = ref(false);
const person = ref<Person | null>(null);
const dynasty = ref<Dynasty | null>(null);

const personId = () => props.id ?? (route.params.id as string);

async function load() {
  loading.value = true;
  try {
    const p = await apis.history.getPerson(personId());
    person.value = p;
    if (p.dynasty_id) {
      try {
        dynasty.value = await apis.history.getDynasty(p.dynasty_id);
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
      <template v-if="person">
        <PageHeader :title="person.name" :subtitle="person.alias_name || person.title || ''" />
        <div class="person-layout">
          <aside class="person-side">
            <div class="portrait">
              <img v-if="person.portrait_url" :src="person.portrait_url" :alt="person.name" />
              <div v-else class="portrait-placeholder">{{ person.name.charAt(0) }}</div>
            </div>
            <Descriptions :column="1" size="small" bordered>
              <DescriptionsItem label="字">{{ person.courtesy_name || '—' }}</DescriptionsItem>
              <DescriptionsItem label="号">{{ person.alias_name || '—' }}</DescriptionsItem>
              <DescriptionsItem label="朝代">{{ dynasty?.name || '—' }}</DescriptionsItem>
              <DescriptionsItem label="生卒">{{
                formatYearRange(person.birth_year, person.death_year)
              }}</DescriptionsItem>
              <DescriptionsItem label="性别">{{ person.gender || '—' }}</DescriptionsItem>
              <DescriptionsItem label="官职/称谓">{{ person.title || '—' }}</DescriptionsItem>
            </Descriptions>
          </aside>

          <article class="person-main">
            <section v-if="person.biography">
              <h2 class="section-title">生平传记</h2>
              <p class="section-text">{{ person.biography }}</p>
            </section>
            <section v-if="person.achievements">
              <h2 class="section-title">主要成就</h2>
              <p class="section-text">{{ person.achievements }}</p>
            </section>
            <section v-if="dynasty">
              <h2 class="section-title">所处时代</h2>
              <p class="section-text">
                <Tag color="var(--tcm-color-primary)" style="color: #fff">{{ dynasty.name }}</Tag>
                <span style="margin-left: 8px"
                  >{{ formatYear(dynasty.start_year) }} — {{ formatYear(dynasty.end_year) }}</span
                >
              </p>
              <p v-if="dynasty.description" class="section-text">{{ dynasty.description }}</p>
            </section>
          </article>
        </div>
      </template>
      <Empty v-else-if="!loading" description="未找到该医家" />
    </Spin>
  </div>
</template>

<style scoped lang="less">
.person-layout {
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: var(--tcm-spacing-xl);
}

@media (max-width: 1024px) {
  .person-layout {
    grid-template-columns: 1fr;
  }
}

.person-side {
  display: flex;
  flex-direction: column;
  gap: var(--tcm-spacing-base);
}

.portrait {
  width: 100%;
  height: 280px;
  border-radius: var(--tcm-radius-lg);
  overflow: hidden;
  background-color: rgba(31, 26, 23, 0.06);
}

.portrait img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.portrait-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: serif;
  font-size: 96px;
  color: var(--tcm-color-primary);
  background-color: rgba(162, 58, 48, 0.08);
}

.person-main {
  display: flex;
  flex-direction: column;
  gap: var(--tcm-spacing-lg);
}

.section-title {
  margin: 0 0 8px;
  font-size: 18px;
  font-weight: 600;
  padding-left: 10px;
  border-left: 3px solid var(--tcm-color-primary);
}

.section-text {
  margin: 0 0 8px;
  line-height: 1.8;
  color: rgba(31, 26, 23, 0.85);
  font-size: 14px;
  white-space: pre-wrap;
}
</style>
