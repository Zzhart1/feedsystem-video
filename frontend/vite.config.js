import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// 前端 dev server：/api/xxx 代理到后端 127.0.0.1:8080，并去掉 /api 前缀
export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
})
