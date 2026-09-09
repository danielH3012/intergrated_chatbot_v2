import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      // Proxy all API + WebSocket calls to the Go backend on :3000
      '/api': {
        target: 'http://localhost:3000',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://localhost:3000',
        ws: true,
        changeOrigin: true,
      },
      '/message': {
        target: 'http://localhost:3000',
        changeOrigin: true,
      },
      '/messages': {
        target: 'http://localhost:3000',
        changeOrigin: true,
      },
      '/transcribe': {
        target: 'http://localhost:3000',
        changeOrigin: true,
      },
      '/tts': {
        target: 'http://localhost:3000',
        changeOrigin: true,
      },
      '/uploads': {
        target: 'http://localhost:3000',
        changeOrigin: true,
      },
    },
  },
})
