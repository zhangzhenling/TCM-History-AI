<script setup lang="ts">
// 发展时间轴页：朝代选择条 + 自研 SVG 时间轴 + 事件详情侧滑。
// P5 阶段接入自研 TimelineAxis 组件，支持横向时间轴/纵向列表双视图切换，
// 点击时间轴节点打开 EventDrawer 侧滑查看事件完整信息。

import { computed, onMounted, ref, watch } from 'vue';
import { Spin, Empty, Tag } from 'ant-design-vue';
import { UnorderedListOutlined, NodeIndexOutlined } from '@ant-design/icons-vue';

import PageHeader from '@/components/PageHeader.vue';
import TimelineAxis from '@/components/TimelineAxis.vue';
import EventDrawer from '@/components/EventDrawer.vue';
import { useApi } from '@/composables/useApi';
import { formatYear } from '@tcm/shared';
import type { Dynasty, HistoryEvent } from '@tcm/api';

const apis = useApi();

const dynasties = ref<Dynasty[]>([]);
const events = ref<HistoryEvent[]>([]);
const loadingDynasties = ref(false);
const loadingEvents = ref(false);
const activeDynastyId = ref<number | null>(null);

// 视图模式：axis=自研 SVG 时间轴，list=纵向列表（辅助视图）。
type ViewMode = 'axis' | 'list';
const viewMode = ref<ViewMode>('axis');

// 侧滑详情状态。
const drawerOpen = ref(false);
const selectedEvent = ref<HistoryEvent | null>(null);

const activeDynasty = computed<Dynasty | undefined>(() =>
  dynasties.value.find((d) => d.id === activeDynastyId.value),
);

onMounted(async () => {
  loadingDynasties.value = true;
  try {
    const res = await apis.history.listDynasties({ page: 1, page_size: 50 });
    dynasties.value = res.items ?? [];
    if (dynasties.value.length && activeDynastyId.value === null) {
      activeDynastyId.value = dynasties.value[0].id;
    }
  } finally {
    loadingDynasties.value = false;
  }
});

watch(
  activeDynastyId,
  async (id) => {
    if (id === null) return;
    loadingEvents.value = true;
    try {
      const res = await apis.history.listEvents({ dynasty_id: id, page: 1, page_size: 50 });
      events.value = res.items ?? [];
    } finally {
      loadingEvents.value = false;
    }
  },
  { immediate: true },
);

function eventTypeColor(t: string): string {
  switch (t) {
    case '出版':
    case '著作':
      return 'var(--tcm-color-indigo)';
    case '战乱':
      return 'var(--tcm-color-primary)';
    case '学术':
    case '出生':
      return 'var(--tcm-color-celadon)';
    case '学派':
    case '制度':
      return 'var(--tcm-color-gold)';
    case '逝世':
      return 'var(--tcm-color-ink)';
    default:
      return 'rgba(31,26,23,0.4)';
  }
}

function onAxisSelect(event: HistoryEvent) {
  selectedEvent.value = event;
  drawerOpen.value = true;
}

function onListItemClick(event: HistoryEvent) {
  selectedEvent.value = event;
  drawerOpen.value = true;
}
</script>

<template>
  <div class="tcm-container">
    <PageHeader title="发展时间轴" subtitle="按朝代纵览中医千年演进">
      <template #actions>
        <div class="view-switch" role="tablist">
          <button
            class="switch-btn"
            :class="{ active: viewMode === 'axis' }"
            role="tab"
            :aria-selected="viewMode === 'axis'"
            @click="viewMode = 'axis'"
          >
            <NodeIndexOutlined />
            <span>时间轴</span>
          </button>
          <button
            class="switch-btn"
            :class="{ active: viewMode === 'list' }"
            role="tab"
            :aria-selected="viewMode === 'list'"
            @click="viewMode = 'list'"
          >
            <UnorderedListOutlined />
            <span>列表</span>
          </button>
        </div>
      </template>
    </PageHeader>

    <Spin :spinning="loadingDynasties">
      <div class="dynasty-bar">
        <button
          v-for="d in dynasties"
          :key="d.id"
          class="dynasty-chip"
          :class="{ active: activeDynastyId === d.id }"
          @click="activeDynastyId = d.id"
        >
          {{ d.name }}
        </button>
      </div>
    </Spin>

    <div v-if="activeDynasty" class="dynasty-banner">
      <h2 class="dynasty-name">{{ activeDynasty.name }}</h2>
      <span class="dynasty-range"
        >{{ formatYear(activeDynasty.start_year) }} — {{ formatYear(activeDynasty.end_year) }}</span
      >
      <p v-if="activeDynasty.description" class="dynasty-desc">{{ activeDynasty.description }}</p>
    </div>

    <Spin :spinning="loadingEvents">
      <!-- 时间轴视图 -->
      <TimelineAxis
        v-if="viewMode === 'axis' && events.length"
        :events="events"
        :dynasty="activeDynasty"
        @select="onAxisSelect"
      />

      <!-- 列表视图（纵向时间线，辅助视图）-->
      <div v-if="viewMode === 'list' && events.length" class="timeline">
        <div v-for="(evt, idx) in events" :key="evt.id" class="timeline-item clickable" @click="onListItemClick(evt)">
          <div class="timeline-dot" :style="{ borderColor: eventTypeColor(evt.event_type) }" />
          <div v-if="idx !== events.length - 1" class="timeline-line" />
          <div class="timeline-content">
            <div class="timeline-head">
              <span class="timeline-year">{{ formatYear(evt.occurred_year) }}</span>
              <Tag :color="eventTypeColor(evt.event_type)" style="color: #fff">{{
                evt.event_type
              }}</Tag>
            </div>
            <h3 class="timeline-title">{{ evt.title }}</h3>
            <p v-if="evt.description" class="timeline-desc">{{ evt.description }}</p>
            <p v-if="evt.impact" class="timeline-impact"><strong>影响：</strong>{{ evt.impact }}</p>
            <p v-if="evt.location" class="timeline-location">📍 {{ evt.location }}</p>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <Empty v-if="!loadingEvents && !events.length" description="该朝代暂无事件" />
    </Spin>

    <!-- 事件详情侧滑 -->
    <EventDrawer v-model:open="drawerOpen" :event="selectedEvent" :dynasty="activeDynasty ?? null" />
  </div>
</template>

<style scoped lang="less">
.view-switch {
  display: inline-flex;
  background-color: rgba(31, 26, 23, 0.05);
  border-radius: var(--tcm-radius-pill);
  padding: 2px;
}

.switch-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 12px;
  border: none;
  background-color: transparent;
  color: rgba(31, 26, 23, 0.65);
  cursor: pointer;
  font-size: 13px;
  border-radius: var(--tcm-radius-pill);
  transition: all 0.15s ease;
}
.switch-btn:hover {
  color: var(--tcm-color-ink);
}
.switch-btn.active {
  background-color: var(--tcm-color-paper);
  color: var(--tcm-color-primary);
  box-shadow: var(--tcm-shadow-card);
}

.dynasty-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: var(--tcm-spacing-lg);
}

.dynasty-chip {
  padding: 6px 14px;
  border: 1px solid rgba(31, 26, 23, 0.15);
  border-radius: 999px;
  background-color: var(--tcm-color-paper);
  cursor: pointer;
  font-size: 13px;
  color: var(--tcm-color-ink);
  transition: all 0.15s ease;
}
.dynasty-chip:hover {
  border-color: var(--tcm-color-primary);
}
.dynasty-chip.active {
  background-color: var(--tcm-color-primary);
  border-color: var(--tcm-color-primary);
  color: #fff;
}

.dynasty-banner {
  background-color: var(--tcm-color-paper);
  border-radius: var(--tcm-radius-lg);
  padding: var(--tcm-spacing-lg) var(--tcm-spacing-xl);
  margin-bottom: var(--tcm-spacing-xl);
  border-left: 4px solid var(--tcm-color-primary);
}

.dynasty-name {
  margin: 0;
  font-size: 22px;
  font-weight: 600;
  display: inline-block;
  margin-right: 12px;
}

.dynasty-range {
  font-size: 13px;
  color: rgba(31, 26, 23, 0.55);
}

.dynasty-desc {
  margin: 8px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: rgba(31, 26, 23, 0.75);
}

.timeline {
  position: relative;
  padding-left: 8px;
}

.timeline-item {
  position: relative;
  padding-left: 28px;
  padding-bottom: var(--tcm-spacing-xl);
}

.timeline-item.clickable {
  cursor: pointer;
}
.timeline-item.clickable:hover .timeline-content {
  border-color: rgba(162, 58, 48, 0.3);
  box-shadow: var(--tcm-shadow-hover);
}

.timeline-dot {
  position: absolute;
  left: 0;
  top: 6px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background-color: var(--tcm-color-paper);
  border: 2px solid var(--tcm-color-primary);
  z-index: 1;
}

.timeline-line {
  position: absolute;
  left: 5px;
  top: 18px;
  bottom: 0;
  width: 2px;
  background-color: rgba(31, 26, 23, 0.1);
}

.timeline-content {
  background-color: var(--tcm-color-paper);
  border-radius: var(--tcm-radius-lg);
  padding: var(--tcm-spacing-lg);
  border: 1px solid rgba(31, 26, 23, 0.06);
  transition: all 0.15s ease;
}

.timeline-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}

.timeline-year {
  font-size: 13px;
  font-weight: 600;
  color: var(--tcm-color-primary);
}

.timeline-title {
  margin: 0 0 6px;
  font-size: 16px;
}

.timeline-desc {
  margin: 0 0 6px;
  font-size: 13px;
  line-height: 1.7;
  color: rgba(31, 26, 23, 0.75);
}

.timeline-impact,
.timeline-location {
  margin: 4px 0 0;
  font-size: 12px;
  color: rgba(31, 26, 23, 0.65);
}
</style>
