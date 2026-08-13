import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'node:path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },

  server: {
    proxy: {
      "/auth": "http://localhost:8080",
      "/preferences": "http://localhost:8080",
      "/recommendations": "http://localhost:8080",
      "/meal-logs": "http://localhost:8080",
      "/meals": "http://localhost:8080",
      "/custom-meals": "http://localhost:8080",
      "/catalog": "http://localhost:8080",
    }
  }
})
