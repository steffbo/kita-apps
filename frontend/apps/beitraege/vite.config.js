var _a;
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
        host: (_a = process.env.HOST) !== null && _a !== void 0 ? _a : '127.0.0.1',
        port: 5175,
        proxy: {
            '/api/fees': {
                target: 'http://localhost:8081',
                changeOrigin: true,
            },
        },
    },
});
