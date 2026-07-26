<script setup lang="ts">
// 空白布局：登录、注册等无导航页使用。
</script>

<template>
  <div class="blank-layout">
    <RouterView />
  </div>
</template>

<style scoped lang="less">
.blank-layout {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--tcm-color-paper) 0%, #efe6d4 100%);
}
</style>
