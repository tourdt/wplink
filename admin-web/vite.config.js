import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  base: process.env.VITE_ADMIN_BASE || '/',
  plugins: [vue()],
  server: {
    port: 5173,
  },
})
