<script setup lang="ts">
// 404 页面。
import { useRouter } from 'vue-router';
const router = useRouter();
</script>

<template>
  <div class="not-found">
    <div class="code">404</div>
    <p class="text">您访问的页面不存在或已被移除。</p>
    <button class="back" @click="router.push('/admin/dashboard')">返回仪表盘</button>
  </div>
</template>

<style scoped lang="less">
.not-found {
  min-height: 80vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 40px;
}

.code {
  font-size: 96px;
  font-weight: 700;
  color: var(--tcm-color-primary);
  line-height: 1;
}

.text {
  color: rgba(31, 42, 68, 0.55);
  margin: 0;
}

.back {
  margin-top: 12px;
  padding: 8px 20px;
  border: none;
  border-radius: var(--tcm-radius-base);
  background-color: var(--tcm-color-primary);
  color: #fff;
  cursor: pointer;
  font-size: 14px;
}
.back:hover {
  background-color: var(--tcm-color-primary-hover);
}
</style>
