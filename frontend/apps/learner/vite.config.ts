import { fileURLToPath, URL } from 'node:url';

import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import Components from 'unplugin-vue-components/vite';
import { AntDesignVueResolver } from 'unplugin-vue-components/resolvers';

// 开发期：前端 5173，Gateway 8080。生产期：前端经 Nginx 反代到 Gateway。
const gatewayTarget = process.env.VITE_GATEWAY_TARGET || 'http://localhost:8080';

export default defineConfig({
  plugins: [
    vue(),
    Components({
      resolvers: [
        AntDesignVueResolver({ importStyle: false, resolveIcons: true }),
      ],
      dts: false,
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  css: {
    preprocessorOptions: {
      less: {
        javascriptEnabled: true,
      },
    },
  },
  server: {
    host: '0.0.0.0',
    port: 5173,
    proxy: {
      // 所有 /api 与 /health 请求转发到 Gateway，避开浏览器 CORS。
      '/api': { target: gatewayTarget, changeOrigin: true },
      '/health': { target: gatewayTarget, changeOrigin: true },
    },
  },
  build: {
    target: 'es2020',
    sourcemap: false,
    chunkSizeWarningLimit: 1200,
    rollupOptions: {
      output: {
        manualChunks: {
          vue: ['vue', 'vue-router', 'pinia'],
          antdv: ['ant-design-vue'],
        },
      },
    },
  },
});
