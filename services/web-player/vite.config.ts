import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const apiBase = env.VITE_API_BASE_URL || 'http://localhost:3000'

  return {
    plugins: [vue()],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    server: {
      port: 5174,
      host: '0.0.0.0',
      proxy: {
        '/api': { target: apiBase, changeOrigin: true },
        '/health': { target: apiBase, changeOrigin: true },
        '/stream': { target: apiBase, changeOrigin: true },
      },
    },
    build: {
      outDir: 'dist',
      rollupOptions: {
        output: {
          manualChunks: {
            'vendor-vue': ['vue', 'vue-router', 'pinia'],
            'vendor-video': ['hls.js', 'video.js'],
          },
        },
      },
    },
  }
})
