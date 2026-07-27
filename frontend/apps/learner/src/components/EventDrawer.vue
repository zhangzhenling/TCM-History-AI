<script setup lang="ts">
// 事件详情侧滑面板：基于 ant-design-vue Drawer，呈现选中事件完整信息。
// 通过 v-model:open 双向绑定开关，event 为 null 时面板内容占位。

import { computed } from 'vue';
import { Drawer, Tag, Empty } from 'ant-design-vue';

import { formatYear } from '@tcm/shared';
import type { Dynasty, HistoryEvent } from '@tcm/api';

const props = defineProps<{
  open: boolean;
  event: HistoryEvent | null;
  dynasty?: Dynasty | null;
}>();

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void;
}>();

const innerOpen = computed({
  get: () => props.open,
  set: (v: boolean) => emit('update:open', v),
});

// 事件类型 → 配色，与时间轴节点保持一致。
const EVENT_TYPE_COLOR: Record<string, string> = {
  出生: '#5C8A6A',
  逝世: '#1F1A17',
  著作: '#2C4A6B',
  出版: '#2C4A6B',
  学派: '#C9A24A',
  学术: '#5C8A6A',
  战乱: '#A23A30',
  制度: '#C9A24A',
};

function eventTypeColor(t: string): string {
  return EVENT_TYPE_COLOR[t] ?? '#A23A30';
}
</script>

<template>
  <Drawer v-model:open="innerOpen" title="事件详情" placement="right" :width="420" class="event-drawer">
    <div v-if="event" class="event-detail">
      <div class="detail-type-row">
        <Tag :color="eventTypeColor(event.event_type)" style="color: #fff">
          {{ event.event_type || '事件' }}
        </Tag>
        <span class="detail-year">{{ formatYear(event.occurred_year) }}</span>
      </div>

      <h2 class="detail-title">{{ event.title }}</h2>

      <div v-if="dynasty" class="detail-meta">
        <span class="meta-label">所属朝代</span>
        <span class="meta-value">{{ dynasty.name }}</span>
        <span class="meta-range">
          （{{ formatYear(dynasty.start_year) }} — {{ formatYear(dynasty.end_year) }}）
        </span>
      </div>

      <section v-if="event.description" class="detail-section">
        <h3 class="section-title">事件描述</h3>
        <p class="section-text">{{ event.description }}</p>
      </section>

      <section v-if="event.impact" class="detail-section">
        <h3 class="section-title">历史影响</h3>
        <p class="section-text">{{ event.impact }}</p>
      </section>

      <section v-if="event.location" class="detail-section">
        <h3 class="section-title">相关地点</h3>
        <p class="section-text">📍 {{ event.location }}</p>
      </section>
    </div>

    <Empty v-else description="未选择事件" class="detail-empty" />
  </Drawer>
</template>

<style scoped lang="less">
.event-detail {
  display: flex;
  flex-direction: column;
  gap: var(--tcm-spacing-lg);
}

.detail-type-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.detail-year {
  font-size: 13px;
  font-weight: 600;
  color: var(--tcm-color-primary);
}

.detail-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--tcm-color-ink);
  line-height: 1.4;
}

.detail-meta {
  font-size: 13px;
  color: rgba(31, 26, 23, 0.75);
  background-color: rgba(162, 58, 48, 0.05);
  border-left: 3px solid var(--tcm-color-primary);
  padding: 8px 12px;
  border-radius: 0 var(--tcm-radius-base) var(--tcm-radius-base) 0;
}

.meta-label {
  color: rgba(31, 26, 23, 0.55);
  margin-right: 6px;
}

.meta-value {
  font-weight: 600;
  color: var(--tcm-color-ink);
}

.meta-range {
  color: rgba(31, 26, 23, 0.5);
  font-size: 12px;
}

.detail-section {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.section-title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--tcm-color-primary);
  position: relative;
  padding-left: 10px;
}
.section-title::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 14px;
  background-color: var(--tcm-color-primary);
  border-radius: 2px;
}

.section-text {
  margin: 0;
  font-size: 13px;
  line-height: 1.8;
  color: rgba(31, 26, 23, 0.78);
}

.detail-empty {
  margin-top: 80px;
}
</style>
