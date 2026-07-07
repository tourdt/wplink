import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  base: process.env.VITE_ADMIN_BASE || '/',
  plugins: [vue()],
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          const packageName = nodeModulePackageName(id)
          if (!packageName) return undefined
          if (['vue', 'vue-router', 'pinia'].includes(packageName)) {
            return 'vue-vendor'
          }
          if (packageName === 'element-plus' || packageName.startsWith('@element-plus/')) {
            return 'element-plus'
          }
          return 'vendor'
        },
      },
    },
  },
  server: {
    port: 5173,
  },
})

function nodeModulePackageName(id) {
  const normalized = id.split('\\').join('/')
  const marker = '/node_modules/'
  const index = normalized.lastIndexOf(marker)
  if (index === -1) return ''
  const parts = normalized.slice(index + marker.length).split('/')
  if (parts[0]?.startsWith('@')) {
    return `${parts[0]}/${parts[1] || ''}`.replace(/\/$/, '')
  }
  return parts[0] || ''
}
