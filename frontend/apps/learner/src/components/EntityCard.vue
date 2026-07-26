<script setup lang="ts">
// 实体卡片：人物/著作/学派/事件等通用的可点击卡片。
// 通过 type 区分图标与配色，点击跳转对应详情页。

import { computed } from 'vue';
import { useRouter } from 'vue-router';

import { ENTITY_LABELS, type EntityType, formatYearRange } from '@tcm/shared';

const props = withDefaults(
  defineProps<{
    type: EntityType;
    id: number | string;
    title: string;
    subtitle?: string;
    description?: string;
    tags?: string[];
    cover?: string;
    yearRange?: { start?: number; end?: number };
    clickable?: boolean;
  }>(),
  { clickable: true },
);

const emit = defineEmits<{
  (e: 'click', payload: { id: string | number; type: EntityType }): void;
}>();

const router = useRouter();

const routeName = computed(() => {
  switch (props.type) {
    case 'person':
      return 'PersonDetail';
    case 'book':
      return 'BookDetail';
    case 'school':
      return 'SchoolDetail';
    default:
      return null;
  }
});

const yearText = computed(() => {
  if (!props.yearRange) return '';
  return formatYearRange(props.yearRange.start, props.yearRange.end);
});

function handleClick() {
  if (!props.clickable) return;
  if (routeName.value) {
    router.push({ name: routeName.value, params: { id: props.id } });
    return;
  }
  emit('click', { id: props.id, type: props.type });
}
</script>

<template>
  <div
    class="entity-card tcm-card-shadow"
    :class="[`type-${type}`, { clickable }]"
    @click="handleClick"
  >
    <div v-if="cover" class="card-cover">
      <img :src="cover" :alt="title" loading="lazy" />
    </div>
    <div class="card-body">
      <div class="card-header">
        <span class="card-type-tag">{{ ENTITY_LABELS[type] }}</span>
        <span v-if="yearText" class="card-year">{{ yearText }}</span>
      </div>
      <h3 class="card-title">{{ title }}</h3>
      <p v-if="subtitle" class="card-subtitle">{{ subtitle }}</p>
      <p v-if="description" class="card-desc">{{ description }}</p>
      <div v-if="tags?.length" class="card-tags">
        <span v-for="t in tags" :key="t" class="card-tag">{{ t }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped lang="less">
.entity-card {
  background-color: var(--tcm-color-paper);
  border-radius: var(--tcm-radius-lg);
  overflow: hidden;
  border: 1px solid rgba(31, 26, 23, 0.06);
  height: 100%;
  display: flex;
  flex-direction: column;
}

.entity-card.clickable {
  cursor: pointer;
}

.card-cover {
  width: 100%;
  height: 140px;
  overflow: hidden;
  background-color: rgba(31, 26, 23, 0.04);

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.card-body {
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.card-type-tag {
  display: inline-block;
  padding: 1px 8px;
  font-size: 11px;
  border-radius: var(--tcm-radius-base);
  background-color: rgba(162, 58, 48, 0.1);
  color: var(--tcm-color-primary);
}

.type-book .card-type-tag {
  background-color: rgba(46, 74, 107, 0.1);
  color: var(--tcm-color-indigo);
}
.type-school .card-type-tag {
  background-color: rgba(92, 138, 106, 0.1);
  color: var(--tcm-color-celadon);
}
.type-event .card-type-tag {
  background-color: rgba(201, 162, 74, 0.15);
  color: var(--tcm-color-gold);
}

.card-year {
  font-size: 11px;
  color: rgba(31, 26, 23, 0.55);
}

.card-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--tcm-color-ink);
}

.card-subtitle {
  margin: 0;
  font-size: 12px;
  color: rgba(31, 26, 23, 0.55);
}

.card-desc {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: rgba(31, 26, 23, 0.75);
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.card-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: auto;
}

.card-tag {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 4px;
  background-color: rgba(31, 26, 23, 0.06);
  color: rgba(31, 26, 23, 0.7);
}
</style>
