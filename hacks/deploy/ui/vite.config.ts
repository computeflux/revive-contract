import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    host: '0.0.0.0',
    allowedHosts: ['dev-workflow.computeflux.ai', 'xiaobai.asyou.me'],
    // 开发模式代理：前端同源请求 /api → 后端 api-server
    // （生产模式由 api-server 内嵌托管 UI，无需此配置）
    proxy: {
      '/api': {
        target: process.env.VITE_API_PROXY_TARGET || 'http://127.0.0.1:8000',
        changeOrigin: true,
      },
    },
  },
})
