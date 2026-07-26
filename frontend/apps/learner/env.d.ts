/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue';
  const component: DefineComponent<Record<string, never>, Record<string, never>, unknown>;
  export default component;
}

interface ImportMetaEnv {
  readonly VITE_API_BASE?: string;
  readonly VITE_GATEWAY_TARGET?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
