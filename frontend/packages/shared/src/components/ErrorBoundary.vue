<script setup lang="ts">
import { ref, onErrorCaptured } from 'vue';
import { Alert, Button } from 'ant-design-vue';

const collapsed = ref(false);
const errorMessage = ref('');
const errorStack = ref('');

onErrorCaptured((err: unknown) => {
  if (err instanceof Error) {
    errorMessage.value = err.message;
    errorStack.value = err.stack ?? '';
  } else {
    errorMessage.value = String(err);
  }
  collapsed.value = false;
  return false;
});

function retry() {
  errorMessage.value = '';
  errorStack.value = '';
}

function toggleCollapse() {
  collapsed.value = !collapsed.value;
}
</script>

<template>
  <div class="error-boundary">
    <template v-if="errorMessage">
      <Alert
        type="error"
        show-icon
        message="出错了！"
        :description="
          collapsed
            ? errorMessage
            : '应用遇到了一个意外错误，请尝试重试或刷新页面。'
        "
      >
        <template #actions>
          <Button size="small" type="primary" @click="retry">重试</Button>
          <Button size="small" @click="toggleCollapse">
            {{ collapsed ? '收起详情' : '查看详情' }}
          </Button>
        </template>
      </Alert>
      <pre
        v-if="!collapsed && errorStack"
        class="error-boundary__stack"
      ><code>{{ errorStack }}</code></pre>
    </template>
    <slot v-else />
  </div>
</template>

<style scoped>
.error-boundary {
  width: 100%;
}

.error-boundary__stack {
  margin: 12px 0 0;
  padding: 12px;
  background: var(--tcm-color-paper);
  border: 1px solid var(--tcm-color-ink);
  border-radius: var(--tcm-radius-base);
  overflow-x: auto;
  font-size: 12px;
  line-height: 1.5;
}
</style>