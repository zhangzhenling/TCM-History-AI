// useApi 组合式函数：从应用 provide 中获取全局 Apis 实例。
// 避免每个组件重复创建 Axios。

import { inject } from 'vue';
import type { Apis } from '@tcm/api';

export function useApi(): Apis {
  const apis = inject<Apis>('apis');
  if (!apis) {
    throw new Error('Apis not provided. Did you forget app.provide("apis", apis) in main.ts?');
  }
  return apis;
}
