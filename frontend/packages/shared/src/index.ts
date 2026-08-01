// 共享包入口：聚合所有可被两端复用的设计 Token、常量、工具。
export * from './design-tokens';
export * from './constants/error-codes';
export * from './constants/entity';
export * from './utils/format';

// 组件
export { default as ErrorBoundary } from './components/ErrorBoundary.vue';
export { default as LoadingWrapper } from './components/LoadingWrapper.vue';
