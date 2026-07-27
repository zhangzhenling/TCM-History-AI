<script setup lang="ts">
// 自研 SVG 水平时间轴：按事件年份分布节点，支持鼠标拖拽平移与滚轮缩放。
// 不引入第三方图表库，配色取自 @tcm/shared design-tokens（中医风格）。
// 节点颜色按事件类型区分，hover 显示 tooltip，点击 emit select 交由父组件打开侧滑。

import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { designTokens, formatYear } from '@tcm/shared';
import type { Dynasty, HistoryEvent } from '@tcm/api';

const props = defineProps<{
  events: HistoryEvent[];
  dynasty?: Dynasty;
}>();

const emit = defineEmits<{
  (e: 'select', event: HistoryEvent): void;
}>();

// ---- 几何常量 ----
const HEIGHT = 220;
const AXIS_Y = 120;
const PADDING_X = 48;
const MIN_SCALE = 0.5;
const MAX_SCALE = 3;
const MIN_NODE_GAP = 64;

// ---- 视图状态 ----
const containerRef = ref<HTMLDivElement | null>(null);
const containerWidth = ref(800);
const scale = ref(1);
const offset = ref(0);

// ---- 事件类型 → 配色（hex 取自 design-tokens，可直接用于 SVG fill）----
const EVENT_TYPE_COLOR: Record<string, string> = {
  出生: designTokens.color.celadon,
  逝世: designTokens.color.ink,
  著作: designTokens.color.indigo,
  出版: designTokens.color.indigo,
  学派: designTokens.color.gold,
  学术: designTokens.color.celadon,
  战乱: designTokens.color.primary,
  制度: designTokens.color.gold,
  其他: designTokens.color.ink,
};

function eventTypeColor(t: string): string {
  return EVENT_TYPE_COLOR[t] ?? designTokens.color.primary;
}

// ---- 时间范围：优先取朝代区间并兼容事件年份，否则取事件年份极值 ----
const range = computed<{ min: number; max: number }>(() => {
  const years = props.events.map((e) => e.occurred_year);
  if (props.dynasty) {
    const min = Math.min(props.dynasty.start_year, ...years);
    const max = Math.max(props.dynasty.end_year, ...years);
    return { min, max: max === min ? max + 1 : max };
  }
  if (years.length === 0) return { min: 0, max: 1 };
  const min = Math.min(...years);
  const max = Math.max(...years);
  return { min, max: max === min ? max + 1 : max };
});

const yearSpan = computed(() => Math.max(1, range.value.max - range.value.min));

// 基准像素/年（不含缩放因子），保证最小可用宽度避免退化。
const baseScale = computed(() => {
  const usable = Math.max(120, containerWidth.value - PADDING_X * 2);
  return usable / yearSpan.value;
});

// 当前有效像素/年。
const pxPerYear = computed(() => baseScale.value * scale.value);

function yearToX(year: number): number {
  return PADDING_X + (year - range.value.min) * pxPerYear.value + offset.value;
}

function xToYear(x: number): number {
  return range.value.min + (x - PADDING_X - offset.value) / pxPerYear.value;
}

// ---- 排序后的事件节点（按年份升序），相近节点上下交错避免重叠 ----
interface PositionedNode {
  event: HistoryEvent;
  x: number;
  side: 1 | -1; // 1=轴下方，-1=轴上方
}

const nodes = computed<PositionedNode[]>(() => {
  const sorted = [...props.events].sort((a, b) => a.occurred_year - b.occurred_year);
  const result: PositionedNode[] = [];
  let lastX = -Infinity;
  let side: 1 | -1 = -1;
  for (const event of sorted) {
    const x = yearToX(event.occurred_year);
    if (x - lastX < MIN_NODE_GAP) {
      side = (side === -1 ? 1 : -1) as 1 | -1;
    } else {
      side = -1;
    }
    result.push({ event, x, side });
    lastX = x;
  }
  return result;
});

// ---- 主轴刻度：约 8 等分 ----
interface Tick {
  x: number;
  label: string;
}

const ticks = computed<Tick[]>(() => {
  const count = 8;
  const step = yearSpan.value / count;
  const arr: Tick[] = [];
  for (let i = 0; i <= count; i++) {
    const year = Math.round(range.value.min + step * i);
    arr.push({ x: yearToX(year), label: formatYear(year) });
  }
  return arr;
});

// ---- Tooltip 浮层 ----
interface TooltipState {
  visible: boolean;
  x: number;
  y: number;
  title: string;
  year: number;
}
const tooltip = ref<TooltipState>({
  visible: false,
  x: 0,
  y: 0,
  title: '',
  year: 0,
});

function onNodeEnter(event: HistoryEvent, e: MouseEvent) {
  const rect = containerRef.value?.getBoundingClientRect();
  if (!rect) return;
  tooltip.value = {
    visible: true,
    x: e.clientX - rect.left,
    y: e.clientY - rect.top,
    title: event.title,
    year: event.occurred_year,
  };
}

function onNodeMove(e: MouseEvent) {
  if (!tooltip.value.visible) return;
  const rect = containerRef.value?.getBoundingClientRect();
  if (!rect) return;
  tooltip.value.x = e.clientX - rect.left;
  tooltip.value.y = e.clientY - rect.top;
}

function onNodeLeave() {
  tooltip.value.visible = false;
}

function onNodeClick(event: HistoryEvent) {
  emit('select', event);
}

// ---- 拖拽平移（仅在背景区域按下，避免与节点点击冲突）----
const dragging = ref(false);
let dragStartX = 0;
let dragStartOffset = 0;

function onBgMouseDown(e: MouseEvent) {
  if (e.button !== 0) return;
  dragging.value = true;
  dragStartX = e.clientX;
  dragStartOffset = offset.value;
  window.addEventListener('mousemove', onWindowMouseMove);
  window.addEventListener('mouseup', onWindowMouseUp);
}

function onWindowMouseMove(e: MouseEvent) {
  if (!dragging.value) return;
  offset.value = dragStartOffset + (e.clientX - dragStartX);
}

function onWindowMouseUp() {
  dragging.value = false;
  window.removeEventListener('mousemove', onWindowMouseMove);
  window.removeEventListener('mouseup', onWindowMouseUp);
}

// ---- 滚轮缩放（围绕鼠标位置保持年份不动）----
function onWheel(e: WheelEvent) {
  const rect = containerRef.value?.getBoundingClientRect();
  if (!rect) return;
  const mouseX = e.clientX - rect.left;
  const yearAtMouse = xToYear(mouseX);
  const factor = e.deltaY < 0 ? 1.1 : 1 / 1.1;
  const next = Math.min(MAX_SCALE, Math.max(MIN_SCALE, scale.value * factor));
  scale.value = next;
  offset.value = mouseX - PADDING_X - (yearAtMouse - range.value.min) * pxPerYear.value;
}

// ---- 缩放控件（围绕容器中心）----
function zoomBy(factor: number) {
  const centerX = containerWidth.value / 2;
  const yearAtCenter = xToYear(centerX);
  const next = Math.min(MAX_SCALE, Math.max(MIN_SCALE, scale.value * factor));
  scale.value = next;
  offset.value = centerX - PADDING_X - (yearAtCenter - range.value.min) * pxPerYear.value;
}

function resetView() {
  scale.value = 1;
  offset.value = 0;
}

// ---- 响应式：ResizeObserver 跟踪容器宽度 ----
let observer: ResizeObserver | null = null;

function updateWidth() {
  if (containerRef.value) {
    containerWidth.value = containerRef.value.clientWidth;
  }
}

onMounted(() => {
  updateWidth();
  if (containerRef.value && typeof ResizeObserver !== 'undefined') {
    observer = new ResizeObserver(() => updateWidth());
    observer.observe(containerRef.value);
  }
});

onUnmounted(() => {
  observer?.disconnect();
  window.removeEventListener('mousemove', onWindowMouseMove);
  window.removeEventListener('mouseup', onWindowMouseUp);
});

// 朝代/事件集合切换时重置视图，避免缩放偏移错乱。
watch(
  () => [props.dynasty?.id, props.events.length] as const,
  () => {
    scale.value = 1;
    offset.value = 0;
  },
);
</script>

<template>
  <div ref="containerRef" class="timeline-axis" :class="{ dragging }">
    <svg :width="containerWidth" :height="HEIGHT" class="axis-svg" @wheel.prevent="onWheel">
      <!-- 背景拖拽区（位于最底层，节点渲染在其上以优先响应点击）-->
      <rect
        :x="0"
        :y="0"
        :width="containerWidth"
        :height="HEIGHT"
        fill="transparent"
        @mousedown="onBgMouseDown"
      />

      <!-- 朝代区间底色 -->
      <rect
        v-if="dynasty"
        :x="yearToX(dynasty.start_year)"
        :y="AXIS_Y - 28"
        :width="Math.max(0, yearToX(dynasty.end_year) - yearToX(dynasty.start_year))"
        :height="56"
        :fill="designTokens.color.primary"
        fill-opacity="0.06"
        rx="6"
      />

      <!-- 主轴 -->
      <line
        :x1="PADDING_X"
        :y1="AXIS_Y"
        :x2="containerWidth - PADDING_X"
        :y2="AXIS_Y"
        :stroke="designTokens.color.ink"
        stroke-opacity="0.25"
        stroke-width="2"
      />

      <!-- 刻度 -->
      <g v-for="(tick, i) in ticks" :key="i">
        <line
          :x1="tick.x"
          :y1="AXIS_Y - 4"
          :x2="tick.x"
          :y2="AXIS_Y + 4"
          :stroke="designTokens.color.ink"
          stroke-opacity="0.3"
        />
        <text :x="tick.x" :y="AXIS_Y + 20" text-anchor="middle" class="tick-label">
          {{ tick.label }}
        </text>
      </g>

      <!-- 事件节点 -->
      <g
        v-for="node in nodes"
        :key="node.event.id"
        class="node-group"
        @click="onNodeClick(node.event)"
        @mouseenter="onNodeEnter(node.event, $event)"
        @mousemove="onNodeMove($event)"
        @mouseleave="onNodeLeave"
      >
        <line
          :x1="node.x"
          :y1="AXIS_Y"
          :x2="node.x"
          :y2="AXIS_Y + node.side * 36"
          :stroke="eventTypeColor(node.event.event_type)"
          stroke-opacity="0.5"
          stroke-width="1.5"
        />
        <circle
          :cx="node.x"
          :cy="AXIS_Y"
          :r="7"
          :fill="eventTypeColor(node.event.event_type)"
          stroke="#fff"
          stroke-width="2"
        />
        <text
          :x="node.x"
          :y="AXIS_Y + node.side * 36 + (node.side === -1 ? -8 : 14)"
          text-anchor="middle"
          class="node-year"
        >
          {{ node.event.occurred_year }}
        </text>
      </g>
    </svg>

    <!-- Tooltip 浮层 -->
    <div v-show="tooltip.visible" class="tooltip" :style="{ left: tooltip.x + 'px', top: tooltip.y + 'px' }">
      <div class="tooltip-title">{{ tooltip.title }}</div>
      <div class="tooltip-year">{{ formatYear(tooltip.year) }}</div>
    </div>

    <!-- 缩放控件 -->
    <div class="controls">
      <button class="ctrl-btn" title="放大" @click="zoomBy(1.2)">＋</button>
      <button class="ctrl-btn" title="缩小" @click="zoomBy(1 / 1.2)">－</button>
      <button class="ctrl-btn" title="重置视图" @click="resetView">↺</button>
      <span class="scale-label">{{ Math.round(scale * 100) }}%</span>
    </div>

    <!-- 操作提示 -->
    <div class="hint">拖拽平移 · 滚轮缩放 · 点击节点查看详情</div>
  </div>
</template>

<style scoped lang="less">
.timeline-axis {
  position: relative;
  background-color: var(--tcm-color-paper);
  border: 1px solid rgba(31, 26, 23, 0.06);
  border-radius: var(--tcm-radius-lg);
  padding: 8px 0 4px;
  user-select: none;
}

.axis-svg {
  display: block;
  cursor: grab;
}

.timeline-axis.dragging .axis-svg {
  cursor: grabbing;
}

.tick-label {
  font-size: 11px;
  fill: rgba(31, 26, 23, 0.5);
}

.node-group {
  cursor: pointer;
}

.node-year {
  font-size: 11px;
  font-weight: 600;
  fill: var(--tcm-color-ink);
}

.tooltip {
  position: absolute;
  transform: translate(-50%, calc(-100% - 12px));
  background-color: var(--tcm-color-ink);
  color: #fff;
  padding: 6px 10px;
  border-radius: var(--tcm-radius-base);
  pointer-events: none;
  white-space: nowrap;
  box-shadow: var(--tcm-shadow-hover);
  z-index: 2;
}

.tooltip-title {
  font-size: 13px;
  font-weight: 600;
}

.tooltip-year {
  font-size: 11px;
  opacity: 0.75;
  margin-top: 2px;
}

.controls {
  position: absolute;
  top: 10px;
  right: 12px;
  display: flex;
  align-items: center;
  gap: 4px;
  background-color: rgba(247, 242, 232, 0.85);
  border-radius: var(--tcm-radius-pill);
  padding: 2px 6px;
  border: 1px solid rgba(31, 26, 23, 0.1);
}

.ctrl-btn {
  width: 24px;
  height: 24px;
  border: none;
  background-color: transparent;
  color: var(--tcm-color-ink);
  cursor: pointer;
  border-radius: 50%;
  font-size: 14px;
  line-height: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: background-color 0.15s ease;
}
.ctrl-btn:hover {
  background-color: rgba(31, 26, 23, 0.08);
}

.scale-label {
  font-size: 11px;
  color: rgba(31, 26, 23, 0.6);
  margin-left: 2px;
  min-width: 34px;
  text-align: center;
}

.hint {
  text-align: center;
  font-size: 11px;
  color: rgba(31, 26, 23, 0.45);
  padding: 0 12px 6px;
}
</style>
