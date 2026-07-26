<script setup lang="ts">
// 管理员登录页：用户名 + 密码，复用 user store，登录成功跳转 /admin/dashboard。
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
    router.push(typeof redirect === 'string' ? redirect : '/admin/dashboard');
  } catch {
    // 错误提示已由 HTTP 拦截器统一处理。
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-brand">
        <span class="brand-mark">管</span>
        <h1 class="brand-title">管理后台登录</h1>
        <p class="brand-subtitle">中医发展史 AI 管理后台</p>
      </div>
      <Form layout="vertical" @submit.prevent="handleSubmit">
        <FormItem label="用户名">
          <Input
            v-model:value="form.username"
            placeholder="请输入管理员账号"
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
    </div>
  </div>
</template>

<style scoped lang="less">
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1d3958 0%, #2c4a6b 100%);
}

.login-card {
  width: 400px;
  max-width: 92vw;
  padding: 32px;
  background-color: #fff;
  border-radius: var(--tcm-radius-lg);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
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
  font-size: 20px;
  margin: 8px 0 4px;
}

.brand-subtitle {
  font-size: 12px;
  color: rgba(31, 42, 68, 0.55);
  margin: 0;
}
</style>
