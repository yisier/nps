import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: 'dist',
    rollupOptions: {
      external: ['/wails/go/main/App.js'],
      output: {
        globals: {
          '/wails/go/main/App.js': 'window'
        }
      }
    }
  }
})
