import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import { fileURLToPath, URL } from 'node:url';

export default defineConfig({
  base: '/portal/',
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    host: process.env.HOST ?? '127.0.0.1',
    port: 5176,
    proxy: {
      '/api/portal': {
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
    },
  },
});
