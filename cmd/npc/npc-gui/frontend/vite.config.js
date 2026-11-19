import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    // mygo dev 通过 mygo.json 的 devUrl 连接开发服务器。
    host: '127.0.0.1',
    port: 34115,
    strictPort: true
  },
  build: {
    outDir: 'dist'
  }
})
