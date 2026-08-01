<script setup lang="ts">
import { Skeleton } from 'ant-design-vue';

withDefaults(
  defineProps<{
    loading?: boolean;
    avatar?: boolean;
    avatarSize?: number;
    title?: boolean;
    titleWidth?: string;
    paragraphs?: number;
    rounded?: boolean;
  }>(),
  {
    loading: true,
    avatar: false,
    avatarSize: 48,
    title: true,
    titleWidth: '60%',
    paragraphs: 2,
    rounded: true,
  },
);
</script>

<template>
  <div class="skeleton-card" :class="{ 'is-rounded': rounded }">
    <Skeleton
      :active="loading"
      :avatar="{ size: avatarSize }"
      :title="title ? { width: titleWidth } : false"
      :paragraph="{ rows: paragraphs }"
    >
      <template #avatar v-if="avatar">
        <div class="skeleton-card__avatar-placeholder" />
      </template>
      <slot />
    </Skeleton>
  </div>
</template>

<style scoped>
.skeleton-card {
  background: var(--tcm-color-bg-container);
  padding: 16px;
  min-height: 120px;

  &.is-rounded {
    border-radius: var(--tcm-radius-lg);
  }

  &__avatar-placeholder {
    width: 48px;
    height: 48px;
    border-radius: 50%;
    background: var(--tcm-color-bg-secondary);
  }
}
</style>
