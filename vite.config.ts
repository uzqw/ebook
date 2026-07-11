import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    proxy: {
      '/api': {
        target: `http://${process.env.POCKETBASE_HOST || '127.0.0.1'}:${process.env.POCKETBASE_PORT || '8090'}`,
        changeOrigin: true,
      },
      '/pb': {
        target: `http://${process.env.POCKETBASE_HOST || '127.0.0.1'}:${process.env.POCKETBASE_PORT || '8090'}`,
        changeOrigin: true,
        rewrite: path => path.replace(/^\/pb/, ''),
      },
    },
  },
})
