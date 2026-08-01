<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref, watch, nextTick } from 'vue';
import type { GraphEdgeView, GraphNodeView } from '@tcm/api';

const props = defineProps<{
  nodes: GraphNodeView[];
  edges: GraphEdgeView[];
  height?: number;
}>();

const emit = defineEmits<{
  (e: 'node-click', node: GraphNodeView): void;
}>();

const height = computed(() => props.height ?? 400);
const svgRef = ref<SVGSVGElement | null>(null);
const containerWidth = ref(600);
const containerHeight = ref(400);

const LABEL_COLOR: Record<string, string> = {
  Person: '#a23a30',
  Classic: '#2c4a6b',
  School: '#5c8a6a',
  Prescription: '#c9a24a',
  Medicine: '#5c8a6a',
  Disease: '#a23a30',
  Dynasty: '#c9a24a',
  HistoricalEvent: '#2c4a6b',
};

interface SimNode {
  uid: string;
  name: string;
  label: string;
  x: number;
  y: number;
  vx: number;
  vy: number;
  r: number;
}

interface SimEdge {
  source: string;
  target: string;
  type: string;
}

const simNodes = ref<SimNode[]>([]);
const simEdges = ref<SimEdge[]>([]);
const hoveredNode = ref<string | null>(null);
const selectedUid = ref<string | null>(null);
const draggingUid = ref<string | null>(null);

function initLayout() {
  const cx = containerWidth.value / 2;
  const cy = containerHeight.value / 2;
  const radius = Math.min(containerWidth.value, containerHeight.value) / 2 - 60;
  const n = props.nodes.length;

  simNodes.value = props.nodes.map((node, i) => {
    const angle = (i / Math.max(n, 1)) * Math.PI * 2 - Math.PI / 2;
    return {
      uid: node.uid,
      name: node.name,
      label: node.label,
      x: cx + Math.cos(angle) * radius * (n > 1 ? 1 : 0),
      y: cy + Math.sin(angle) * radius * (n > 1 ? 1 : 0),
      vx: 0,
      vy: 0,
      r: node.label === 'Person' || node.label === 'Classic' ? 22 : 18,
    };
  });

  simEdges.value = props.edges.map((e) => ({
    source: e.source_uid,
    target: e.target_uid,
    type: e.type,
  }));
}

let rafId: number | null = null;
let frameCount = 0;

function simulate() {
  const nodes = simNodes.value;
  if (nodes.length < 2) return;

  const cx = containerWidth.value / 2;
  const cy = containerHeight.value / 2;

  // Repulsion
  for (let i = 0; i < nodes.length; i++) {
    for (let j = i + 1; j < nodes.length; j++) {
      const dx = nodes[i].x - nodes[j].x;
      const dy = nodes[i].y - nodes[j].y;
      const dist = Math.sqrt(dx * dx + dy * dy) || 1;
      const force = 800 / (dist * dist);
      const fx = (dx / dist) * force;
      const fy = (dy / dist) * force;
      nodes[i].vx += fx;
      nodes[i].vy += fy;
      nodes[j].vx -= fx;
      nodes[j].vy -= fy;
    }
  }

  // Attraction (edges)
  for (const edge of simEdges.value) {
    const s = nodes.find((n) => n.uid === edge.source);
    const t = nodes.find((n) => n.uid === edge.target);
    if (!s || !t) continue;
    const dx = t.x - s.x;
    const dy = t.y - s.y;
    const dist = Math.sqrt(dx * dx + dy * dy) || 1;
    const force = (dist - 100) * 0.02;
    const fx = (dx / dist) * force;
    const fy = (dy / dist) * force;
    s.vx += fx;
    s.vy += fy;
    t.vx -= fx;
    t.vy -= fy;
  }

  // Center gravity
  for (const node of nodes) {
    node.vx += (cx - node.x) * 0.005;
    node.vy += (cy - node.y) * 0.005;
  }

  // Apply velocities + damping
  for (const node of nodes) {
    if (draggingUid.value === node.uid) {
      node.vx = 0;
      node.vy = 0;
      continue;
    }
    node.vx *= 0.85;
    node.vy *= 0.85;
    node.x += node.vx;
    node.y += node.vy;

    // Boundary
    const pad = node.r + 5;
    node.x = Math.max(pad, Math.min(containerWidth.value - pad, node.x));
    node.y = Math.max(pad, Math.min(containerHeight.value - pad, node.y));
  }

  frameCount++;
  if (frameCount > 300) return;
  rafId = requestAnimationFrame(simulate);
}

function startSimulation() {
  stopSimulation();
  frameCount = 0;
  rafId = requestAnimationFrame(simulate);
}

function stopSimulation() {
  if (rafId !== null) {
    cancelAnimationFrame(rafId);
    rafId = null;
  }
}

function updateSize() {
  if (svgRef.value) {
    const rect = svgRef.value.getBoundingClientRect();
    containerWidth.value = rect.width || 600;
    containerHeight.value = rect.height || 400;
  }
}

function onSvgMouseDown(e: MouseEvent) {
  const target = e.target as SVGElement;
  const nodeUid = target.closest('[data-node-uid]')?.getAttribute('data-node-uid');
  if (nodeUid) {
    draggingUid.value = nodeUid;
    selectedUid.value = nodeUid;
  }
}

function onSvgMouseMove(e: MouseEvent) {
  if (!draggingUid.value || !svgRef.value) return;
  const rect = svgRef.value.getBoundingClientRect();
  const x = e.clientX - rect.left;
  const y = e.clientY - rect.top;
  const node = simNodes.value.find((n) => n.uid === draggingUid.value);
  if (node) {
    node.x = x;
    node.y = y;
    node.vx = 0;
    node.vy = 0;
  }
}

function onSvgMouseUp() {
  if (draggingUid.value && selectedUid.value) {
    const original = props.nodes.find((n) => n.uid === selectedUid.value);
    if (original) {
      emit('node-click', original);
    }
  }
  draggingUid.value = null;
  setTimeout(startSimulation, 50);
}

function onNodeClick(_e: MouseEvent, uid: string) {
  // 如果是点击（非拖拽结束），触发节点选择
  if (draggingUid.value === null || draggingUid.value !== uid) {
    const original = props.nodes.find((n) => n.uid === uid);
    if (original) {
      selectedUid.value = uid;
      emit('node-click', original);
    }
  }
}

const nodeColor = (label: string): string => LABEL_COLOR[label] || '#888';

watch(
  () => props.nodes,
  () => {
    nextTick(() => {
      updateSize();
      initLayout();
      startSimulation();
    });
  },
  { immediate: true },
);

onMounted(() => {
  nextTick(() => {
    updateSize();
    initLayout();
    startSimulation();
  });
  window.addEventListener('resize', updateSize);
});

onBeforeUnmount(() => {
  stopSimulation();
  window.removeEventListener('resize', updateSize);
});

const edgePaths = computed(() => {
  return simEdges.value
    .map((edge) => {
      const s = simNodes.value.find((n) => n.uid === edge.source);
      const t = simNodes.value.find((n) => n.uid === edge.target);
      if (!s || !t) return null;
      const midX = (s.x + t.x) / 2;
      const midY = (s.y + t.y) / 2 - 20;
      return {
        key: `${edge.source}-${edge.target}`,
        d: `M ${s.x} ${s.y} Q ${midX} ${midY} ${t.x} ${t.y}`,
        type: edge.type,
      };
    })
    .filter((p): p is NonNullable<typeof p> => p !== null);
});

const showEmpty = computed(() => props.nodes.length === 0);
</script>

<template>
  <div class="force-graph" :style="{ height: height + 'px' }">
    <svg
      v-if="!showEmpty"
      ref="svgRef"
      class="graph-svg"
      @mousedown="onSvgMouseDown"
      @mousemove="onSvgMouseMove"
      @mouseup="onSvgMouseUp"
      @mouseleave="onSvgMouseUp"
    >
      <defs>
        <marker
          v-for="color in Object.values(LABEL_COLOR)"
          :key="color"
          :id="'arrow-' + color.replace('#', '')"
          viewBox="0 0 10 10"
          refX="9"
          refY="5"
          markerWidth="6"
          markerHeight="6"
          orient="auto-start-reverse"
        >
          <path d="M 0 0 L 10 5 L 0 10 z" :fill="color" />
        </marker>
      </defs>

      <!-- Edges -->
      <path
        v-for="ep in edgePaths"
        :key="ep.key"
        :d="ep.d"
        :data-type="ep.type"
        fill="none"
        stroke="rgba(31, 26, 23, 0.2)"
        stroke-width="1.5"
      />

      <!-- Nodes -->
      <g
        v-for="node in simNodes"
        :key="node.uid"
        class="node-group"
        :data-node-uid="node.uid"
        :style="{ cursor: 'pointer' }"
        @click.stop="onNodeClick($event, node.uid)"
      >
        <circle
          :cx="node.x"
          :cy="node.y"
          :r="node.r"
          :fill="nodeColor(node.label)"
          :stroke="selectedUid === node.uid ? '#fff' : 'transparent'"
          :stroke-width="selectedUid === node.uid ? 3 : 0"
          :opacity="hoveredNode && hoveredNode !== node.uid ? 0.4 : 0.9"
          class="node-circle"
        />
        <text
          :x="node.x"
          :y="node.y + node.r + 14"
          text-anchor="middle"
          class="node-label"
        >
          {{ node.name }}
        </text>
      </g>
    </svg>

    <div v-else class="graph-empty">
      暂无图谱数据
    </div>
  </div>
</template>

<style scoped lang="less">
.force-graph {
  width: 100%;
  position: relative;
  background: var(--tcm-color-paper);
  border-radius: var(--tcm-radius-lg);
  overflow: hidden;
  border: 1px solid rgba(31, 26, 23, 0.08);
}

.graph-svg {
  width: 100%;
  height: 100%;
  display: block;
  user-select: none;
}

.node-circle {
  transition: stroke-width 0.2s, opacity 0.2s;
}

.node-group:hover .node-circle {
  stroke: var(--tcm-color-ink);
  stroke-width: 2;
}

.node-label {
  font-size: 11px;
  fill: var(--tcm-color-ink);
  pointer-events: none;
  font-weight: 500;
}

.graph-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--tcm-color-text-tertiary);
  font-size: 14px;
}

@media (max-width: 768px) {
  .node-label {
    font-size: 10px;
  }
}
</style>
