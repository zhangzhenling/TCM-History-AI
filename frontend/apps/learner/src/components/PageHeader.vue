<script setup lang="ts">
// 通用页面头部：标题 + 可选副标题 + 右侧操作槽。

defineProps<{
  title: string;
  subtitle?: string;
}>();
</script>

<template>
  <div class="page-header">
    <div class="page-header-text">
      <h1 class="page-title">{{ title }}</h1>
      <p v-if="subtitle" class="page-subtitle">{{ subtitle }}</p>
    </div>
    <div class="page-header-actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<style scoped lang="less">
.page-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  margin-bottom: var(--tcm-spacing-lg);
  gap: var(--tcm-spacing-lg);
}

.page-title {
  font-size: var(--tcm-font-size-display);
  font-weight: 600;
  margin: 0;
  color: var(--tcm-color-ink);
}

.page-subtitle {
  margin: 4px 0 0;
  font-size: 13px;
  color: rgba(31, 26, 23, 0.55);
}

.page-header-actions {
  display: flex;
  gap: 8px;
}
</style>
