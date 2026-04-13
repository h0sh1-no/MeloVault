import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:5000',
        changeOrigin: true
      },
      '/song': {
        target: 'http://localhost:5000',
        changeOrigin: true
      },
      '/playlist': {
        target: 'http://localhost:5000',
        changeOrigin: true,
        bypass(req) {
          const path = req.url.split('?')[0]
          if (path !== '/playlist' && path !== '/Playlist') return req.url
        }
      },
      '/album': {
        target: 'http://localhost:5000',
        changeOrigin: true,
        bypass(req) {
          const path = req.url.split('?')[0]
          if (path !== '/album' && path !== '/Album') return req.url
        }
      },
      '/download': {
        target: 'http://localhost:5000',
        changeOrigin: true
      },
      '/health': {
        target: 'http://localhost:5000',
        changeOrigin: true
      }
    }
  }
})
