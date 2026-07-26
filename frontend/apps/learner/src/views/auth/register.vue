<script setup lang="ts">
// 注册页：用户名 + 密码 + 邮箱（可选）+ 手机号（可选）。
import { reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { Form, FormItem, Input, InputPassword, Button, message } from 'ant-design-vue';

import { useUserStore } from '@tcm/stores';
import { useApi } from '@/composables/useApi';

const router = useRouter();
const userStore = useUserStore();
const apis = useApi();

const form = reactive({
  username: '',
  password: '',
  confirm: '',
  email: '',
  phone: '',
});
const loading = ref(false);

async function handleSubmit() {
  if (!form.username || !form.password) {
    message.warning('请填写用户名与密码');
    return;
  }
  if (form.password !== form.confirm) {
    message.warning('两次输入的密码不一致');
    return;
  }
  loading.value = true;
  try {
    await userStore.register(
      {
        username: form.username,
        password: form.password,
        email: form.email || undefined,
        phone: form.phone || undefined,
      },
      apis,
    );
    message.success('注册成功，欢迎加入');
    router.push('/app/home');
  } catch {
    // 错误提示已由 HTTP 拦截器统一处理。
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="register-card">
    <div class="register-brand">
      <span class="brand-mark">医</span>
      <h1 class="brand-title">创建账户</h1>
      <p class="brand-subtitle">填写信息即可开始学习</p>
    </div>
    <Form layout="vertical" @submit.prevent="handleSubmit">
      <FormItem label="用户名">
        <Input
          v-model:value="form.username"
          placeholder="3-64 位字符"
          size="large"
          autocomplete="username"
        />
      </FormItem>
      <FormItem label="密码">
        <InputPassword
          v-model:value="form.password"
          placeholder="至少 8 位"
          size="large"
          autocomplete="new-password"
        />
      </FormItem>
      <FormItem label="确认密码">
        <InputPassword
          v-model:value="form.confirm"
          placeholder="再次输入密码"
          size="large"
          autocomplete="new-password"
        />
      </FormItem>
      <FormItem label="邮箱（可选）">
        <Input
          v-model:value="form.email"
          placeholder="用于找回密码"
          size="large"
          autocomplete="email"
        />
      </FormItem>
      <FormItem label="手机号（可选）">
        <Input
          v-model:value="form.phone"
          placeholder="E.164 格式"
          size="large"
          autocomplete="tel"
        />
      </FormItem>
      <Button type="primary" size="large" block :loading="loading" @click="handleSubmit"
        >注册</Button
      >
    </Form>
    <div class="register-footer">
      <span>已有账号？</span>
      <RouterLink to="/login">返回登录</RouterLink>
    </div>
  </div>
</template>

<style scoped lang="less">
.register-card {
  width: 420px;
  max-width: 92vw;
  padding: 32px;
  background-color: var(--tcm-color-paper);
  border-radius: var(--tcm-radius-lg);
  box-shadow: var(--tcm-shadow-hover);
}

.register-brand {
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

.register-footer {
  margin-top: 16px;
  text-align: center;
  font-size: 13px;
  color: rgba(31, 26, 23, 0.55);
}
</style>
