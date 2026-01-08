<template>
  <div class="container">
    <div class="sidebar">
      <div class="sidebar-content">
        <button
          class="sidebar-btn"
          :class="{ active: activeView === 'clients' }"
          @click="activeView = 'clients'"
        >
          🔗 客户端
        </button>
        <button
          class="sidebar-btn"
          :class="{ active: activeView === 'logs' }"
          @click="activeView = 'logs'"
        >
          📋 连接日志
        </button>
        <button
          class="sidebar-btn"
          :class="{ active: activeView === 'settings' }"
          @click="activeView = 'settings'"
        >
          ⚙️ 设置
        </button>
      </div>
    </div>

    <div class="main-content">
      <div v-if="activeView === 'clients'" class="view clients-view">
        <div class="header">
          <div class="input-group">
            <input
              v-model="commandInput"
              type="text"
              class="command-input"
              placeholder="输入秘钥或粘贴快捷命令的 Base64 文本"
              @keyup.enter="addConnection"
            />
            <button class="btn btn-primary" @click="addConnection">连接</button>
          </div>
        </div>

        <div class="clients-grid">
          <div v-if="clients.length === 0" class="empty-state">
            <p>暂无客户端，粘贴 Base64 格式的快捷命令并点击连接即可添加</p>
          </div>

          <div v-for="(client, index) in clients" :key="index" class="client-card">
            <div class="card-header">
              <h3 class="card-title">{{ client.name }}</h3>
              <button class="btn-close" @click="removeClient(client)">✕</button>
            </div>

            <div class="card-content">
              <div class="info-row">
                <span class="label">地址:</span>
                <span class="value">{{ client.addr }}</span>
              </div>
              <div class="info-row">
                <span class="label">密钥:</span>
                <span class="value code">{{ client.key }}</span>
              </div>
              <div class="info-row">
                <span class="label">TLS:</span>
                <span class="value">{{ client.tls ? '是' : '否' }}</span>
              </div>
            </div>

            <div class="card-footer">
              <label class="toggle-switch">
                <input
                  type="checkbox"
                  :checked="client.running"
                  @change="toggleClient(client)"
                />
                <span class="toggle-slider"></span>
                <span class="toggle-label">{{ client.running ? '运行中' : '已停止' }}</span>
              </label>
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="activeView === 'logs'" class="view logs-view">
        <div class="logs-container">
          <div class="log-content">
            <p>日志功能开发中...</p>
          </div>
        </div>
      </div>

      <div v-else-if="activeView === 'settings'" class="view settings-view">
        <div class="settings-container">
          <p>设置功能开发中...</p>
        </div>
      </div>

      <div v-if="message" :class="['message', message.type]">
        {{ message.text }}
      </div>
    </div>
  </div>
</template>

<script>
import { ref, onMounted } from 'vue'
// 直接导入 Wails 生成的 API 绑定
import * as AppAPI from '../wailsjs/go/main/App.js'

export default {
  name: 'App',
  setup() {
    const activeView = ref('clients')
    const clients = ref([])
    const commandInput = ref('')
    const message = ref(null)

    // 从直接导入获取 Wails API（使用 let 以便在浏览器中可替换为 mock）
    let GetShortcuts = AppAPI.GetShortcuts
    let AddShortcutFromBase64 = AppAPI.AddShortcutFromBase64
    let RemoveShortcut = AppAPI.RemoveShortcut
    let ToggleClient = AppAPI.ToggleClient
    let TestConnection = AppAPI.TestConnection

    // 在普通浏览器里运行时 Wails API 可能不存在，提供简单 mock 方便调试 UI
    if (!AppAPI || typeof AppAPI.GetShortcuts !== 'function') {
      console.warn('Wails App API not available — using mock implementations for browser debugging')
      GetShortcuts = async () => {
        return [
          { name: 'MyServer', addr: '127.0.0.1:8024', key: 'alefa114df', tls: false, running: false },
        ]
      }
      AddShortcutFromBase64 = async (b64) => {
        console.log('mock AddShortcutFromBase64', b64)
        return
      }
      RemoveShortcut = async (name, addr, key) => {
        console.log('mock RemoveShortcut', name, addr, key)
        return
      }
      ToggleClient = async (name, addr, key, tls, newState) => {
        console.log('mock ToggleClient', name, newState)
        return
      }
      TestConnection = async (input) => {
        console.log('mock TestConnection', input)
        return
      }
    }

    const initWails = async () => {
      try {
        console.log('Wails API loaded successfully')
        await loadClients()
      } catch (error) {
        console.error('Failed to initialize Wails:', error)
        // Fallback: show empty state
        clients.value = []
      }
    }

    const loadClients = async () => {
      try {
        if (!GetShortcuts) {
          clients.value = []
          return
        }
        const result = await GetShortcuts()
        clients.value = result || []
      } catch (error) {
        console.error('加载客户端失败:', error)
        const errMsg = extractErrorMessage(error)
        showMessage('加载客户端失败: ' + errMsg, 'error')
      }
    }

    const extractErrorMessage = (error) => {
      console.error('Error object:', error, 'Type:', typeof error)
      
      if (!error) return '未知错误'
      
      // Handle string errors
      if (typeof error === 'string') {
        const trimmed = error.trim()
        if (!trimmed || trimmed === 'undefined' || trimmed === 'null') return '未知错误'
        return trimmed
      }
      
      // Handle error objects with message property
      if (error.message) {
        const msg = String(error.message).trim()
        if (!msg || msg === 'undefined' || msg === 'null') return '未知错误'
        return msg
      }
      
      // Handle custom error property
      if (error.error && typeof error.error === 'string') {
        const msg = String(error.error).trim()
        if (!msg || msg === 'undefined' || msg === 'null') return '未知错误'
        return msg
      }
      
      // Handle Wails error structure
      if (error.errorMessage && typeof error.errorMessage === 'string') {
        const msg = String(error.errorMessage).trim()
        if (!msg || msg === 'undefined' || msg === 'null') return '未知错误'
        return msg
      }
      
      // Try toString
      if (error.toString && typeof error.toString === 'function') {
        const s = error.toString()
        if (s && s !== '[object Object]' && s !== 'undefined' && s !== 'null') {
          return s
        }
      }
      
      // Last resort: stringify
      try {
        const json = JSON.stringify(error)
        if (json && json !== '{}') return json
      } catch (e) {
        // ignore
      }
      
      return '未知错误'
    }

    const addConnection = async () => {
      const input = commandInput.value.trim()
      if (!input) return

      try {
        // Try to parse as Base64 first
        if (input.length > 10 && !input.includes('|')) {
          await AddShortcutFromBase64(input)
        } else {
          // Try direct key connection
          await TestConnection(input)
        }

        commandInput.value = ''
        await loadClients()
        showMessage('连接已添加', 'success')
      } catch (error) {
        console.error('Add connection error:', error)
        const errMsg = extractErrorMessage(error)
        showMessage(`错误: ${errMsg}`, 'error')
      }
    }

    const removeClient = async (client) => {
      if (!confirm(`确定要删除 "${client.name}" 吗？`)) return

      try {
        await RemoveShortcut(client.name, client.addr, client.key)
        await loadClients()
        showMessage('已删除', 'success')
      } catch (error) {
        console.error('Remove client error:', error)
        const errMsg = extractErrorMessage(error)
        showMessage(`删除失败: ${errMsg}`, 'error')
      }
    }

    const toggleClient = async (client) => {
      const newState = !client.running
      console.log('Toggling client:', { name: client.name, newState })
      
      try {
        await ToggleClient(client.name, client.addr, client.key, client.tls, newState)
        console.log('ToggleClient succeeded')
        // Only update state after successful call
        client.running = newState
        showMessage(newState ? '已启动' : '已停止', 'success')
        // Reload to get server-side state
        setTimeout(() => {
          loadClients()
        }, 500)
      } catch (error) {
        console.error('Toggle client error:', error)
        const errMsg = extractErrorMessage(error)
        showMessage(`${newState ? '启动' : '停止'}失败: ${errMsg}`, 'error')
      }
    }

    const showMessage = (text, type = 'info') => {
      message.value = { text, type }
      setTimeout(() => {
        message.value = null
      }, 3000)
    }

    onMounted(() => {
      // 直接初始化，因为 API 是静态导入的
      initWails()
      
      // 每 2 秒自动刷新客户端状态，保持与服务器同步
      const refreshInterval = setInterval(() => {
        loadClients()
      }, 2000)
      
      // Cleanup interval on unmount
      return () => {
        clearInterval(refreshInterval)
      }
    })

    return {
      activeView,
      clients,
      commandInput,
      message,
      addConnection,
      removeClient,
      toggleClient,
    }
  },
}
</script>

<style scoped>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

.container {
  display: flex;
  height: 100vh;
  background: #1a2332;
  color: #e8eef7;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell,
    sans-serif;
}

/* Sidebar */
.sidebar {
  width: 180px;
  background: #0f1419;
  border-right: 1px solid #2d3e54;
  padding: 20px 0;
  display: flex;
  flex-direction: column;
}

.sidebar-content {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 0 10px;
}

.sidebar-btn {
  padding: 12px 15px;
  background: transparent;
  border: none;
  color: #a8b5c8;
  cursor: pointer;
  border-radius: 6px;
  font-size: 14px;
  transition: all 0.2s ease;
  text-align: left;
}

.sidebar-btn:hover {
  background: #2d3e54;
  color: #e8eef7;
}

.sidebar-btn.active {
  background: #2b8fe8;
  color: white;
  font-weight: 500;
}

/* Main Content */
.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  position: relative;
}

.view {
  flex: 1;
  padding: 20px;
  overflow: auto;
}

/* Clients View */
.clients-view {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.header {
  display: flex;
  gap: 10px;
}

.input-group {
  display: flex;
  gap: 10px;
  flex: 1;
}

.command-input {
  flex: 1;
  padding: 10px 15px;
  background: #1a2332;
  border: 1px solid #2d3e54;
  color: #e8eef7;
  border-radius: 6px;
  font-size: 14px;
  transition: border-color 0.2s ease;
}

.command-input:focus {
  outline: none;
  border-color: #2b8fe8;
  box-shadow: 0 0 0 2px rgba(43, 143, 232, 0.1);
}

.btn {
  padding: 10px 20px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.2s ease;
}

.btn-primary {
  background: #2b8fe8;
  color: white;
}

.btn-primary:hover {
  background: #2079d4;
  transform: translateY(-1px);
}

.btn-primary:active {
  transform: translateY(0);
}

/* Clients Grid */
.clients-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 15px;
}

.empty-state {
  grid-column: 1 / -1;
  padding: 40px 20px;
  text-align: center;
  color: #a8b5c8;
}

.client-card {
  background: #1a2332;
  border: 1px solid #2d3e54;
  border-radius: 8px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  transition: all 0.2s ease;
}

.client-card:hover {
  border-color: #2b8fe8;
  box-shadow: 0 4px 12px rgba(43, 143, 232, 0.1);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 15px;
  background: #0f1419;
  border-bottom: 1px solid #2d3e54;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: #e8eef7;
}

.btn-close {
  background: transparent;
  border: none;
  color: #a8b5c8;
  cursor: pointer;
  font-size: 18px;
  padding: 0;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: all 0.2s ease;
}

.btn-close:hover {
  color: #ff6b6b;
  background: rgba(255, 107, 107, 0.1);
}

.card-content {
  padding: 15px;
  flex: 1;
}

.info-row {
  display: flex;
  gap: 10px;
  margin-bottom: 8px;
  font-size: 13px;
}

.info-row:last-child {
  margin-bottom: 0;
}

.label {
  color: #a8b5c8;
  min-width: 50px;
}

.value {
  color: #e8eef7;
  word-break: break-all;
  flex: 1;
}

.value.code {
  font-family: 'Monaco', 'Courier New', monospace;
  background: #0f1419;
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 12px;
}

.card-footer {
  padding: 12px 15px;
  background: #0f1419;
  border-top: 1px solid #2d3e54;
}

/* Toggle Switch */
.toggle-switch {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  user-select: none;
}

.toggle-switch input {
  display: none;
}

.toggle-slider {
  width: 44px;
  height: 24px;
  background: #2d3e54;
  border-radius: 12px;
  position: relative;
  transition: background 0.3s ease;
}

.toggle-switch input:checked + .toggle-slider {
  background: #2b8fe8;
}

.toggle-slider::after {
  content: '';
  position: absolute;
  width: 20px;
  height: 20px;
  background: white;
  border-radius: 50%;
  top: 2px;
  left: 2px;
  transition: left 0.3s ease;
}

.toggle-switch input:checked + .toggle-slider::after {
  left: 22px;
}

.toggle-label {
  font-size: 13px;
  color: #a8b5c8;
}

/* Logs View */
.logs-view {
  display: flex;
  flex-direction: column;
}

.logs-container {
  flex: 1;
  background: #1a2332;
  border: 1px solid #2d3e54;
  border-radius: 8px;
  padding: 15px;
}

.log-content {
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  color: #a8b5c8;
}

/* Settings View */
.settings-view {
  display: flex;
  flex-direction: column;
}

.settings-container {
  background: #1a2332;
  border: 1px solid #2d3e54;
  border-radius: 8px;
  padding: 20px;
}

/* Message */
.message {
  position: fixed;
  bottom: 20px;
  right: 20px;
  padding: 12px 20px;
  border-radius: 6px;
  font-size: 14px;
  animation: slideIn 0.2s ease;
  z-index: 1000;
}

.message.success {
  background: #2ecc71;
  color: white;
}

.message.error {
  background: #e74c3c;
  color: white;
}

.message.info {
  background: #2b8fe8;
  color: white;
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* Scrollbar */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: transparent;
}

::-webkit-scrollbar-thumb {
  background: #2d3e54;
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: #3a4d66;
}
</style>
