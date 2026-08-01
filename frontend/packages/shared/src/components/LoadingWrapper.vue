<script setup lang="ts">
import { computed } from 'vue';
import { Spin, Empty } from 'ant-design-vue';
import ErrorBoundary from './ErrorBoundary.vue';

const props = withDefaults(
  defineProps<{
    loading?: boolean;
    error?: boolean;
    empty?: boolean;
    emptyText?: string;
  }>(),
  {
    loading: false,
    error: false,
    empty: false,
    emptyText: '暂无数据',
  },
);

const showLoading = computed(() => props.loading);
const showError = computed(() => props.error);
const showEmpty = computed(
  () => !props.loading && !props.error && props.empty,
);
const showContent = computed(
  () => !props.loading && !props.error && !props.empty,
);
</script>

<template>
  <div class="loading-wrapper">
    <Spin v-if="showLoading" tip="加载中...">
      <div class="loading-wrapper__placeholder" />
    </Spin>

    <ErrorBoundary v-else-if="showError">
      <template #default>
        <slot />
      </template>
    </ErrorBoundary>

    <Empty v-else-if="showEmpty" :description="emptyText">
      <template #image>
        <slot name="empty">
          <span class="loading-wrapper__empty-icon">📭</span>
        </slot>
      </template>
    </Empty>

    <slot v-else />
  </div>
</template>

<style scoped>
.loading-wrapper {
  width: 100%;
  min-height: 120px;
}

.loading-wrapper__placeholder {
  min-height: 120px;
}

.loading-wrapper__empty-icon {
  font-size: 48px;
  line-height: 1;
}
</style>