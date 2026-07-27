<script setup lang="ts">
// 文献检索页：RAG 语义检索 + 文献列表/详情浏览。
// 调用 Knowledge Service：
//   - retrieve() 对向量库做 top-k 召回；
//   - listDocuments()/getDocument()/listChunksByDocument() 浏览文献及其切片。
import { computed, onMounted, reactive, ref } from 'vue';
import {
  Input,
  InputNumber,
  Spin,
  Empty,
  Tag,
  Card,
  Pagination,
  Tabs,
  TabPane,
  Descriptions,
  DescriptionsItem,
  Button,
} from 'ant-design-vue';
import { SearchOutlined } from '@ant-design/icons-vue';

import PageHeader from '@/components/PageHeader.vue';
import { useApi } from '@/composables/useApi';
import { truncate } from '@tcm/shared';
import type {
  Document,
  DocumentChunk,
  ListResponse,
  RetrievedChunk,
  RetrieveResponse,
} from '@tcm/api';

const apis = useApi();

// ============ Tab 切换 ============
const activeTab = ref<'retrieve' | 'documents'>('retrieve');

// ============ RAG 检索 ============
const query = ref('');
const topk = ref(5);
const retrieving = ref(false);
const retrieveResult = ref<RetrieveResponse | null>(null);
const retrieved = ref(false);

const chunks = computed<RetrievedChunk[]>(() => retrieveResult.value?.chunks ?? []);

// 相似度分数格式化为百分比，并按高低映射颜色（>=0.8 高，>=0.5 中，否则低）。
function scoreText(score: number): string {
  return `${(score * 100).toFixed(1)}%`;
}
function scoreColor(score: number): string {
  if (score >= 0.8) return 'green';
  if (score >= 0.5) return 'gold';
  return 'default';
}

// 内容类型中文标签映射。
const CONTENT_TYPE_TEXT: Record<string, string> = {
  original: '原文',
  translation: '译文',
  annotation: '注解',
  commentary: '评注',
  formula: '方剂',
  herb: '药物',
};

async function doRetrieve() {
  if (!query.value.trim()) return;
  retrieving.value = true;
  retrieved.value = true;
  try {
    retrieveResult.value = await apis.knowledge.retrieve({
      query: query.value.trim(),
      topk: topk.value,
    });
  } finally {
    retrieving.value = false;
  }
}

// ============ 文献列表 ============
const docLoading = ref(false);
const documents = ref<Document[]>([]);
const docTotal = ref(0);
const docQuery = reactive({ page: 1, page_size: 12 });

// ============ 文献详情 ============
const selectedDoc = ref<Document | null>(null);
const docDetailLoading = ref(false);
const chunksOfDoc = ref<DocumentChunk[]>([]);
const chunksLoading = ref(false);

const hasDocList = computed(() => documents.value.length > 0);

async function loadDocuments() {
  docLoading.value = true;
  try {
    const res: ListResponse<Document> = await apis.knowledge.listDocuments({
      page: docQuery.page,
      page_size: docQuery.page_size,
    });
    documents.value = res.items ?? [];
    docTotal.value = res.total ?? 0;
  } finally {
    docLoading.value = false;
  }
}

function onDocPageChange(p: number, ps: number) {
  docQuery.page = p;
  docQuery.page_size = ps;
  loadDocuments();
}

// 点击文献：加载详情与切片。
async function openDocument(doc: Document) {
  selectedDoc.value = doc;
  chunksOfDoc.value = [];
  docDetailLoading.value = true;
  chunksLoading.value = true;
  try {
    // 详情与切片可并行加载；详情失败时回退为列表项中的快照。
    const [detail, chunkRes] = await Promise.allSettled([
      apis.knowledge.getDocument(doc.id),
      apis.knowledge.listChunksByDocument(doc.id, { page: 1, page_size: 50 }),
    ]);
    if (detail.status === 'fulfilled') selectedDoc.value = detail.value;
    if (chunkRes.status === 'fulfilled') chunksOfDoc.value = chunkRes.value.items ?? [];
  } finally {
    docDetailLoading.value = false;
    chunksLoading.value = false;
  }
}

function closeDocument() {
  selectedDoc.value = null;
  chunksOfDoc.value = [];
}

// 文献状态标签颜色映射。
function statusColor(status: string): string {
  switch (status) {
    case 'ready':
      return 'green';
    case 'processing':
      return 'gold';
    case 'failed':
      return 'red';
    default:
      return 'default';
  }
}

function statusText(status: string): string {
  switch (status) {
    case 'ready':
      return '就绪';
    case 'processing':
      return '处理中';
    case 'failed':
      return '失败';
    case 'pending':
      return '待处理';
    default:
      return status || '—';
  }
}

onMounted(() => {
  // 默认进入文献 Tab 时再懒加载列表，避免检索 Tab 下的多余请求。
  loadDocuments();
});
</script>

<template>
  <div class="tcm-container">
    <PageHeader title="文献检索" subtitle="基于 RAG 的语义检索与中医典籍文献浏览" />

    <Tabs v-model:activeKey="activeTab">
      <!-- ============ Tab 1：RAG 检索 ============ -->
      <TabPane key="retrieve" tab="语义检索">
        <div class="retrieve-box">
          <Input
            v-model:value="query"
            placeholder="输入检索问题，如「麻黄汤主治什么」「伤寒论如何论述太阳病」"
            size="large"
            allow-clear
            @press-enter="doRetrieve"
          >
            <template #prefix><SearchOutlined /></template>
          </Input>
          <div class="retrieve-controls">
            <span class="control-label">召回数量</span>
            <InputNumber v-model:value="topk" :min="1" :max="20" style="width: 90px" />
            <Button type="primary" :loading="retrieving" @click="doRetrieve">检索</Button>
          </div>
        </div>

        <Spin :spinning="retrieving">
          <div v-if="retrieveResult" class="retrieve-meta">
            共召回 <strong>{{ retrieveResult.total }}</strong> 条 · 耗时
            {{ retrieveResult.latency_ms }}ms
          </div>

          <div v-if="chunks.length" class="chunk-list">
            <Card v-for="(c, idx) in chunks" :key="c.chunk_id" size="small" class="chunk-card">
              <template #title>
                <div class="chunk-head">
                  <span class="chunk-rank">#{{ idx + 1 }}</span>
                  <Tag color="var(--tcm-color-indigo)" style="color: #fff">
                    {{ c.classic_code || '—' }}
                  </Tag>
                  <Tag v-if="c.content_type" :color="CONTENT_TYPE_TEXT[c.content_type] ? 'blue' : 'default'">
                    {{ CONTENT_TYPE_TEXT[c.content_type] ?? c.content_type }}
                  </Tag>
                  <span v-if="c.volume" class="chunk-meta">{{ c.volume }}</span>
                  <span v-if="c.clause_no" class="chunk-meta">第 {{ c.clause_no }} 条</span>
                  <Tag :color="scoreColor(c.score)" class="chunk-score">
                    相似度 {{ scoreText(c.score) }}
                  </Tag>
                </div>
              </template>
              <div class="chunk-content">{{ c.content }}</div>
              <div v-if="c.text_original && c.text_original !== c.content" class="chunk-original">
                <span class="chunk-label">原文：</span>{{ truncate(c.text_original, 200) }}
              </div>
              <div v-if="c.text_translation" class="chunk-translation">
                <span class="chunk-label">译文：</span>{{ truncate(c.text_translation, 200) }}
              </div>
            </Card>
          </div>

          <Empty v-else-if="retrieved && !retrieving" description="未检索到相关片段" />
        </Spin>
      </TabPane>

      <!-- ============ Tab 2：文献列表 ============ -->
      <TabPane key="documents" tab="文献列表">
        <Spin :spinning="docDetailLoading">
          <template v-if="selectedDoc">
            <!-- 文献详情视图 -->
            <div class="doc-detail">
              <div class="doc-detail-head">
                <a class="back-link" @click="closeDocument">← 返回列表</a>
              </div>
              <PageHeader :title="selectedDoc.title" :subtitle="selectedDoc.classic_code || ''">
                <template #actions>
                  <Tag :color="statusColor(selectedDoc.status)">
                    {{ statusText(selectedDoc.status) }}
                  </Tag>
                </template>
              </PageHeader>

              <Descriptions :column="2" bordered size="small">
                <DescriptionsItem label="朝代">{{ selectedDoc.dynasty || '—' }}</DescriptionsItem>
                <DescriptionsItem label="学派">{{ selectedDoc.school || '—' }}</DescriptionsItem>
                <DescriptionsItem label="作者">{{ selectedDoc.author || '—' }}</DescriptionsItem>
                <DescriptionsItem label="版本">{{ selectedDoc.version || '—' }}</DescriptionsItem>
                <DescriptionsItem label="分片数">{{ selectedDoc.chunk_count || 0 }}</DescriptionsItem>
                <DescriptionsItem label="卷数">{{ selectedDoc.volume_count || 0 }}</DescriptionsItem>
              </Descriptions>

              <section class="chunk-section">
                <h2 class="section-title">文献切片（{{ chunksOfDoc.length }}）</h2>
                <Spin :spinning="chunksLoading">
                  <div v-if="chunksOfDoc.length" class="doc-chunk-list">
                    <div
                      v-for="ch in chunksOfDoc"
                      :key="ch.id"
                      class="doc-chunk-item tcm-card-shadow"
                    >
                      <div class="doc-chunk-head">
                        <span class="doc-chunk-index">#{{ ch.chunk_index }}</span>
                        <Tag v-if="ch.content_type">
                          {{ CONTENT_TYPE_TEXT[ch.content_type] ?? ch.content_type }}
                        </Tag>
                        <span v-if="ch.volume" class="doc-chunk-meta">{{ ch.volume }}</span>
                        <span v-if="ch.clause_no" class="doc-chunk-meta">第 {{ ch.clause_no }} 条</span>
                      </div>
                      <div class="doc-chunk-content">{{ ch.content }}</div>
                      <div v-if="ch.text_original && ch.text_original !== ch.content" class="doc-chunk-original">
                        <span class="chunk-label">原文：</span>{{ truncate(ch.text_original, 150) }}
                      </div>
                      <div v-if="ch.text_translation" class="doc-chunk-translation">
                        <span class="chunk-label">译文：</span>{{ truncate(ch.text_translation, 150) }}
                      </div>
                    </div>
                  </div>
                  <Empty v-else-if="!chunksLoading" description="暂无切片数据" />
                </Spin>
              </section>
            </div>
          </template>

          <template v-else>
            <!-- 文献列表视图 -->
            <div v-if="docTotal > 0" class="result-meta">
              共 <strong>{{ docTotal }}</strong> 篇文献
            </div>
            <Spin :spinning="docLoading">
              <div v-if="hasDocList" class="doc-grid">
                <div
                  v-for="d in documents"
                  :key="d.id"
                  class="doc-item tcm-card-shadow"
                  @click="openDocument(d)"
                >
                  <div class="doc-item-head">
                    <Tag color="var(--tcm-color-indigo)" style="color: #fff">
                      {{ d.classic_code || '—' }}
                    </Tag>
                    <Tag :color="statusColor(d.status)">{{ statusText(d.status) }}</Tag>
                  </div>
                  <div class="doc-title">{{ d.title }}</div>
                  <div class="doc-meta">
                    <span v-if="d.dynasty">{{ d.dynasty }}</span>
                    <span v-if="d.author" class="meta-sep">·</span>
                    <span v-if="d.author">{{ d.author }}</span>
                  </div>
                  <div v-if="d.school" class="doc-meta">
                    <span class="meta-label">学派：</span>{{ d.school }}
                  </div>
                  <div class="doc-meta">
                    <span class="meta-label">分片：</span>{{ d.chunk_count || 0 }}
                  </div>
                </div>
              </div>
              <Empty v-else description="暂无文献数据" />
            </Spin>

            <div v-if="docTotal > 0" class="pagination-wrap">
              <Pagination
                :current="docQuery.page"
                :page-size="docQuery.page_size"
                :total="docTotal"
                show-size-changer
                :page-size-options="['12', '24', '48']"
                @change="onDocPageChange"
              />
            </div>
          </template>
        </Spin>
      </TabPane>
    </Tabs>
  </div>
</template>

<style scoped lang="less">
.retrieve-box {
  max-width: 800px;
  margin-bottom: var(--tcm-spacing-xl);
}

.retrieve-controls {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: var(--tcm-spacing-base);
}

.control-label {
  font-size: 13px;
  color: rgba(31, 26, 23, 0.65);
}

.retrieve-meta,
.result-meta {
  font-size: 13px;
  color: rgba(31, 26, 23, 0.6);
  margin-bottom: var(--tcm-spacing-base);
}

.chunk-list {
  display: flex;
  flex-direction: column;
  gap: var(--tcm-spacing-base);
}

.chunk-card {
  background-color: var(--tcm-color-paper);
}

.chunk-head {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.chunk-rank {
  font-size: 13px;
  font-weight: 600;
  color: var(--tcm-color-primary);
}

.chunk-meta {
  font-size: 12px;
  color: rgba(31, 26, 23, 0.55);
}

.chunk-score {
  margin-left: auto;
}

.chunk-content {
  font-size: 14px;
  line-height: 1.8;
  color: var(--tcm-color-ink);
  white-space: pre-wrap;
}

.chunk-original,
.chunk-translation,
.doc-chunk-original,
.doc-chunk-translation {
  margin-top: 8px;
  font-size: 13px;
  line-height: 1.7;
  color: rgba(31, 26, 23, 0.7);
}

.chunk-label {
  font-weight: 600;
  color: var(--tcm-color-primary);
}

.doc-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--tcm-spacing-lg);
}

@media (max-width: 1024px) {
  .doc-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .doc-grid {
    grid-template-columns: 1fr;
  }
}

.doc-item {
  padding: var(--tcm-spacing-base) var(--tcm-spacing-lg);
  border-radius: var(--tcm-radius-lg);
  background-color: var(--tcm-color-paper);
  border: 1px solid rgba(31, 26, 23, 0.06);
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.doc-item-head {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.doc-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--tcm-color-ink);
}

.doc-meta {
  font-size: 12px;
  color: rgba(31, 26, 23, 0.65);
}

.meta-label {
  color: rgba(31, 26, 23, 0.45);
}

.meta-sep {
  margin: 0 4px;
  color: rgba(31, 26, 23, 0.3);
}

.pagination-wrap {
  display: flex;
  justify-content: center;
  margin-top: var(--tcm-spacing-xl);
}

.doc-detail {
  display: flex;
  flex-direction: column;
  gap: var(--tcm-spacing-lg);
}

.back-link {
  font-size: 13px;
  cursor: pointer;
}

.chunk-section {
  margin-top: var(--tcm-spacing-base);
}

.section-title {
  margin: 0 0 var(--tcm-spacing-base);
  font-size: 18px;
  font-weight: 600;
  padding-left: 10px;
  border-left: 3px solid var(--tcm-color-primary);
}

.doc-chunk-list {
  display: flex;
  flex-direction: column;
  gap: var(--tcm-spacing-base);
}

.doc-chunk-item {
  padding: var(--tcm-spacing-base) var(--tcm-spacing-lg);
  border-radius: var(--tcm-radius-lg);
  background-color: var(--tcm-color-paper);
  border: 1px solid rgba(31, 26, 23, 0.06);
}

.doc-chunk-head {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 6px;
}

.doc-chunk-index {
  font-size: 13px;
  font-weight: 600;
  color: var(--tcm-color-primary);
}

.doc-chunk-meta {
  font-size: 12px;
  color: rgba(31, 26, 23, 0.55);
}

.doc-chunk-content {
  font-size: 14px;
  line-height: 1.8;
  color: var(--tcm-color-ink);
  white-space: pre-wrap;
}
</style>
