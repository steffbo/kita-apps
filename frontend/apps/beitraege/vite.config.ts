import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import { fileURLToPath, URL } from 'node:url';

export default defineConfig({
  base: '/beitraege/',
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 5175,
    proxy: {
      '/api/fees': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
    },
  },
});
