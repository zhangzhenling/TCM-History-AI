<script setup lang="ts">
// 登录页：用户名 + 密码，提交后跳回 redirect 或首页。
import { reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { Form, FormItem, Input, InputPassword, Button, message } from 'ant-design-vue';

import { useUserStore } from '@tcm/stores';
import { useApi } from '@/composables/useApi';

const route = useRoute();
const router = useRouter();
const userStore = useUserStore();
const apis = useApi();

const form = reactive({ username: '', password: '' });
const loading = ref(false);

async function handleSubmit() {
  if (!form.username || !form.password) {
    message.warning('请输入用户名与密码');
    return;
  }
  loading.value = true;
  try {
    await userStore.login({ username: form.username, password: form.password }, apis);
    message.success(`欢迎回来，${userStore.nickname}`);
    const redirect = route.query.redirect;
    router.push(typeof redirect === 'string' ? redirect : '/app/home');
  } catch {
    // 错误提示已由 HTTP 拦截器统一处理。
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="login-card">
    <div class="login-brand">
      <span class="brand-mark">医</span>
      <h1 class="brand-title">中医发展史 AI 学习平台</h1>
      <p class="brand-subtitle">登录后开启沉浸式中医历史学习之旅</p>
    </div>
    <Form layout="vertical" @submit.prevent="handleSubmit">
      <FormItem label="用户名">
        <Input
          v-model:value="form.username"
          placeholder="请输入用户名"
          size="large"
          autocomplete="username"
        />
      </FormItem>
      <FormItem label="密码">
        <InputPassword
          v-model:value="form.password"
          placeholder="请输入密码"
          size="large"
          autocomplete="current-password"
          @press-enter="handleSubmit"
        />
      </FormItem>
      <Button type="primary" size="large" block :loading="loading" @click="handleSubmit"
        >登录</Button
      >
    </Form>
    <div class="login-footer">
      <span>还没有账号？</span>
      <RouterLink to="/register">立即注册</RouterLink>
    </div>
  </div>
</template>

<style scoped lang="less">
.login-card {
  width: 400px;
  max-width: 92vw;
  padding: 32px;
  background-color: var(--tcm-color-paper);
  border-radius: var(--tcm-radius-lg);
  box-shadow: var(--tcm-shadow-hover);
}

.login-brand {
  text-align: center;
  margin-bottom: 24px;
}

.brand-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  border-radius: 12px;
  background-color: var(--tcm-color-primary);
  color: #fff;
  font-family: serif;
  font-size: 28px;
  margin-bottom: 12px;
}

.brand-title {
  font-size: 18px;
  margin: 8px 0 4px;
}

.brand-subtitle {
  font-size: 12px;
  color: rgba(31, 26, 23, 0.55);
  margin: 0;
}

.login-footer {
  margin-top: 16px;
  text-align: center;
  font-size: 13px;
  color: rgba(31, 26, 23, 0.55);
}
</style>
