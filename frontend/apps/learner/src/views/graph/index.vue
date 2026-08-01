<script setup lang="ts">
// 知识图谱浏览页：节点检索 + 子图关系展示。
// 调用 Graph Service：searchNodes 检索节点，getSubgraph 获取中心节点的关联子图。
import { computed, ref } from 'vue';
import { Input, Spin, Empty, Tag, Select, SelectOption, Card } from 'ant-design-vue';
import { SearchOutlined } from '@ant-design/icons-vue';

import PageHeader from '@/components/PageHeader.vue';
import { useApi } from '@/composables/useApi';
import { truncate } from '@tcm/shared';
import {
  GRAPH_NODE_LABELS,
  type GraphEdgeView,
  type GraphNodeLabel,
  type GraphNodeView,
  type GraphSearchResponse,
  type GraphSubgraph,
} from '@tcm/api';

const apis = useApi();

// ---- 节点类型中文标签映射 ----
const LABEL_TEXT: Record<GraphNodeLabel, string> = {
  Person: '人物',
  Classic: '经典',
  School: '学派',
  Prescription: '方剂',
  Medicine: '药物',
  Disease: '疾病',
  Dynasty: '朝代',
  HistoricalEvent: '历史事件',
};

// ---- 关系类型中文标签映射 ----
const EDGE_TEXT: Record<string, string> = {
  AUTHORED: '著作',
  DISCIPLED: '师承',
  INFLUENCED: '影响',
  BELONGS_TO: '属于',
  OCCURRED_IN: '发生于',
  CITED: '引用',
  PROPOSED: '提出',
  OPPOSED: '反对',
  INHERITED: '继承',
};

// 不同节点类型对应不同配色，便于在列表/子图中区分。
const LABEL_COLOR: Record<GraphNodeLabel, string> = {
  Person: 'var(--tcm-color-primary)',
  Classic: 'var(--tcm-color-indigo)',
  School: 'var(--tcm-color-celadon)',
  Prescription: 'var(--tcm-color-gold)',
  Medicine: 'var(--tcm-color-celadon)',
  Disease: 'var(--tcm-color-primary)',
  Dynasty: 'var(--tcm-color-gold)',
  HistoricalEvent: 'var(--tcm-color-indigo)',
};

// ---- 搜索表单 ----
const keyword = ref('');
const labelFilter = ref<GraphNodeLabel | undefined>(undefined);
const loading = ref(false);
const result = ref<GraphSearchResponse | null>(null);
const searched = ref(false);

// ---- 子图 ----
const selectedNode = ref<GraphNodeView | null>(null);
const subgraph = ref<GraphSubgraph | null>(null);
const subgraphLoading = ref(false);

const nodes = computed<GraphNodeView[]>(() => result.value?.items ?? []);

// 子图中除中心节点外的关联节点。
const relatedNodes = computed<GraphNodeView[]>(() => {
  if (!subgraph.value || !selectedNode.value) return [];
  const centerUid = selectedNode.value.uid;
  return subgraph.value.nodes.filter((n) => n.uid !== centerUid);
});

// 为每条边附加两端节点的名称，便于在 UI 上呈现「A → 关系 → B」。
const edgeWithNames = computed<{ edge: GraphEdgeView; sourceName: string; targetName: string }[]>(
  () => {
    if (!subgraph.value) return [];
    const nameByUid = new Map<string, string>();
    for (const n of subgraph.value.nodes) nameByUid.set(n.uid, n.name);
    return subgraph.value.edges.map((edge) => ({
      edge,
      sourceName: nameByUid.get(edge.source_uid) ?? edge.source_uid,
      targetName: nameByUid.get(edge.target_uid) ?? edge.target_uid,
    }));
  },
);

async function doSearch() {
  if (!keyword.value.trim()) return;
  loading.value = true;
  searched.value = true;
  // 切换搜索时清空已选节点与子图。
  selectedNode.value = null;
  subgraph.value = null;
  try {
    result.value = await apis.graph.searchNodes({
      keyword: keyword.value.trim(),
      label: labelFilter.value,
      limit: 30,
    });
  } finally {
    loading.value = false;
  }
}

// 点击节点：加载以该节点为中心、深度 1 的子图。
async function selectNode(node: GraphNodeView) {
  selectedNode.value = node;
  subgraph.value = null;
  subgraphLoading.value = true;
  try {
    subgraph.value = await apis.graph.getSubgraph({
      center_uid: node.uid,
      depth: 1,
      limit: 20,
    });
  } finally {
    subgraphLoading.value = false;
  }
}

// 节点 properties 是任意结构的 Record，这里抽取常见字段做摘要展示。
function nodeSummary(node: GraphNodeView): string {
  const p = node.properties ?? {};
  const candidates = [p.biography, p.summary, p.description, p.efficacy, p.composition, p.note];
  for (const c of candidates) {
    if (typeof c === 'string' && c.trim()) return truncate(c, 80);
  }
  return '';
}

// properties 中除常见摘要字段外的其它键值对，用于在卡片底部以小字展示。
function nodeExtraProps(node: GraphNodeView): { key: string; value: string }[] {
  const p = node.properties ?? {};
  const skip = new Set(['biography', 'summary', 'description', 'efficacy', 'composition', 'note']);
  return Object.entries(p)
    .filter(([k, v]) => !skip.has(k) && v !== null && v !== undefined && v !== '')
    .slice(0, 4)
    .map(([k, v]) => ({ key: k, value: String(v) }));
}

function clearSelected() {
  selectedNode.value = null;
  subgraph.value = null;
}
</script>

<template>
  <div class="tcm-container">
    <PageHeader title="知识图谱" subtitle="检索人物、典籍、方剂等节点，并探索其关联关系" />

    <div class="search-box">
      <Input
        v-model:value="keyword"
        placeholder="输入关键词，如「张仲景」「伤寒论」「麻黄汤」"
        size="large"
        allow-clear
        @press-enter="doSearch"
      >
        <template #prefix><SearchOutlined /></template>
      </Input>
      <Select
        v-model:value="labelFilter"
        placeholder="节点类型（全部）"
        allow-clear
        style="width: 180px"
        @change="doSearch"
      >
        <SelectOption v-for="l in GRAPH_NODE_LABELS" :key="l" :value="l">
          {{ LABEL_TEXT[l] }}（{{ l }}）
        </SelectOption>
      </Select>
    </div>

    <div class="graph-layout">
      <!-- 左：节点搜索结果列表 -->
      <section class="node-panel">
        <Spin :spinning="loading">
          <div v-if="result && result.total > 0" class="result-meta">
            共找到 <strong>{{ result.total }}</strong> 个节点
          </div>

          <div v-if="nodes.length" class="node-list">
            <div
              v-for="n in nodes"
              :key="n.uid"
              class="node-item tcm-card-shadow"
              :class="{ active: selectedNode?.uid === n.uid }"
              @click="selectNode(n)"
            >
              <div class="node-item-head">
                <Tag :color="LABEL_COLOR[n.label]" style="color: #fff">
                  {{ LABEL_TEXT[n.label] }}
                </Tag>
                <span class="node-name">{{ n.name }}</span>
              </div>
              <div v-if="nodeSummary(n)" class="node-summary">{{ nodeSummary(n) }}</div>
              <div v-if="nodeExtraProps(n).length" class="node-extra">
                <span v-for="kv in nodeExtraProps(n)" :key="kv.key" class="extra-kv">
                  <span class="extra-key">{{ kv.key }}</span>
                  <span class="extra-val">{{ kv.value }}</span>
                </span>
              </div>
              <div class="node-uid" :title="n.uid">{{ n.uid }}</div>
            </div>
          </div>

          <Empty v-else-if="searched && !loading" description="未找到匹配的节点" />
        </Spin>
      </section>

      <!-- 右：选中节点的子图关系 -->
      <section class="subgraph-panel">
        <Spin :spinning="subgraphLoading">
          <template v-if="selectedNode">
            <div class="subgraph-head">
              <div class="subgraph-title">
                <Tag :color="LABEL_COLOR[selectedNode.label]" style="color: #fff">
                  {{ LABEL_TEXT[selectedNode.label] }}
                </Tag>
                <span class="center-name">{{ selectedNode.name }}</span>
              </div>
              <a class="subgraph-close" @click="clearSelected">收起 ×</a>
            </div>

            <div v-if="nodeSummary(selectedNode)" class="center-summary">
              {{ nodeSummary(selectedNode) }}
            </div>

            <Card v-if="subgraph" size="small" class="subgraph-card">
              <template #title>
                关联节点 <span class="count">({{ relatedNodes.length }})</span>
              </template>
              <div v-if="relatedNodes.length" class="related-list">
                <div
                  v-for="rn in relatedNodes"
                  :key="rn.uid"
                  class="related-item"
                  @click="selectNode(rn)"
                >
                  <Tag :color="LABEL_COLOR[rn.label]" style="color: #fff">
                    {{ LABEL_TEXT[rn.label] }}
                  </Tag>
                  <span class="related-name">{{ rn.name }}</span>
                </div>
              </div>
              <Empty v-else description="无关联节点" :image="Empty.PRESENTED_IMAGE_SIMPLE" />
            </Card>

            <Card v-if="subgraph" size="small" class="subgraph-card">
              <template #title>
                关系 <span class="count">({{ edgeWithNames.length }})</span>
              </template>
              <div v-if="edgeWithNames.length" class="edge-list">
                <div v-for="item in edgeWithNames" :key="item.edge.uid" class="edge-item">
                  <span class="edge-node">{{ item.sourceName }}</span>
                  <Tag color="var(--tcm-color-gold)" style="color: #fff">
                    {{ EDGE_TEXT[item.edge.type] ?? item.edge.type }}
                  </Tag>
                  <span class="edge-arrow">→</span>
                  <span class="edge-node">{{ item.targetName }}</span>
                </div>
              </div>
              <Empty v-else description="无关系" :image="Empty.PRESENTED_IMAGE_SIMPLE" />
            </Card>
          </template>

          <Empty
            v-else
            description="点击左侧节点查看关联关系"
            :image="Empty.PRESENTED_IMAGE_SIMPLE"
          />
        </Spin>
      </section>
    </div>
  </div>
</template>

<style scoped lang="less">
.search-box {
  display: flex;
  gap: var(--tcm-spacing-base);
  margin-bottom: var(--tcm-spacing-xl);
  max-width: 800px;
}

.graph-layout {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--tcm-spacing-xl);
  align-items: start;
}

@media (max-width: 1024px) {
  .graph-layout {
    grid-template-columns: 1fr;
  }
}

.result-meta {
  font-size: 13px;
  color: rgba(31, 26, 23, 0.6);
  margin-bottom: var(--tcm-spacing-base);
}

.node-list {
  display: flex;
  flex-direction: column;
  gap: var(--tcm-spacing-base);
}

.node-item {
  padding: var(--tcm-spacing-base) var(--tcm-spacing-lg);
  border-radius: var(--tcm-radius-lg);
  background-color: var(--tcm-color-paper);
  border: 1px solid rgba(31, 26, 23, 0.06);
  cursor: pointer;
}

.node-item.active {
  border-color: var(--tcm-color-primary);
  box-shadow: 0 0 0 2px rgba(162, 58, 48, 0.15);
}

.node-item-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.node-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--tcm-color-ink);
}

.node-summary {
  font-size: 12px;
  color: rgba(31, 26, 23, 0.65);
  line-height: 1.6;
  margin-bottom: 4px;
}

.node-extra {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 4px;
}

.extra-kv {
  font-size: 11px;
  display: inline-flex;
  gap: 4px;
}

.extra-key {
  color: rgba(31, 26, 23, 0.45);
}

.extra-val {
  color: rgba(31, 26, 23, 0.75);
}

.node-uid {
  font-size: 11px;
  color: rgba(31, 26, 23, 0.4);
  font-family: monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.subgraph-panel {
  position: sticky;
  top: 80px;
}

.subgraph-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--tcm-spacing-base);
}

.subgraph-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.center-name {
  font-size: 18px;
  font-weight: 600;
  color: var(--tcm-color-ink);
}

.subgraph-close {
  font-size: 13px;
  cursor: pointer;
}

.center-summary {
  font-size: 13px;
  line-height: 1.7;
  color: rgba(31, 26, 23, 0.75);
  padding: var(--tcm-spacing-base) var(--tcm-spacing-lg);
  background-color: var(--tcm-color-paper);
  border-radius: var(--tcm-radius-lg);
  border-left: 3px solid var(--tcm-color-primary);
  margin-bottom: var(--tcm-spacing-base);
}

.subgraph-card {
  margin-bottom: var(--tcm-spacing-base);
}

.count {
  font-size: 12px;
  color: rgba(31, 26, 23, 0.5);
  font-weight: normal;
}

.related-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.related-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: var(--tcm-radius-base);
  cursor: pointer;
  transition: background-color 0.15s ease;
}

.related-item:hover {
  background-color: rgba(162, 58, 48, 0.06);
}

.related-name {
  font-size: 13px;
  color: var(--tcm-color-ink);
}

.edge-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.edge-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  flex-wrap: wrap;
}

.edge-node {
  color: var(--tcm-color-ink);
  font-weight: 500;
}

.edge-arrow {
  color: rgba(31, 26, 23, 0.4);
}
</style>
