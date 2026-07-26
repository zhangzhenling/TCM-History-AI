<script setup lang="ts">
// 统计卡片：仪表盘关键指标占位。title/value/suffix 文本，icon 走具名插槽。

defineProps<{
  title: string;
  value: number | string;
  suffix?: string;
}>();
</script>

<template>
  <div class="stat-card">
    <div class="stat-icon">
      <slot name="icon" />
    </div>
    <div class="stat-body">
      <p class="stat-title">{{ title }}</p>
      <p class="stat-value">
        {{ value }}<span v-if="suffix" class="stat-suffix">{{ suffix }}</span>
      </p>
    </div>
  </div>
</template>

<style scoped lang="less">
.stat-card {
  display: flex;
  align-items: center;
  gap: var(--tcm-spacing-lg);
  padding: 20px 24px;
  background-color: #fff;
  border-radius: var(--tcm-radius-lg);
  box-shadow: var(--tcm-shadow-card);
}

.stat-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 10px;
  background-color: rgba(44, 74, 107, 0.1);
  color: var(--tcm-color-primary);
  font-size: 24px;
  flex-shrink: 0;
}

.stat-body {
  flex: 1;
  min-width: 0;
}

.stat-title {
  margin: 0;
  font-size: 13px;
  color: rgba(31, 42, 68, 0.55);
}

.stat-value {
  margin: 4px 0 0;
  font-size: 26px;
  font-weight: 600;
  line-height: 1.2;
  color: var(--tcm-color-ink);
}

.stat-suffix {
  margin-left: 4px;
  font-size: 13px;
  font-weight: 400;
  color: rgba(31, 42, 68, 0.55);
}
</style>
