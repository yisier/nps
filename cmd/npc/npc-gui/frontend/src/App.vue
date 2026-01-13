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
              <div v-if="client.error && client.running" class="info-row error-message">
                <span class="label">错误:</span>
                <span class="value">{{ client.error }}</span>
              </div>
            </div>

            <div class="card-footer">
              <label class="toggle-switch">
                <input
                  type="checkbox"
                  :checked="client.status !== 'stopped'"
                  @change="toggleClient(client)"
                />
                <span class="toggle-slider"></span>
                <span class="toggle-label">
                  {{ getStatusLabel(client.status) }}
                </span>
              </label>
              <div v-if="client.error && client.status !== 'stopped'" class="status-error">
                {{ client.error }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="activeView === 'logs'" class="view logs-view">
        <div class="logs-header">
          <div class="logs-controls">
            <label>选择客户端：</label>
            <select v-model="selectedClientId" class="client-select">
              <option value="">-- 全部客户端 --</option>
              <option v-for="client in clients" :key="`${client.addr}|${client.key}`" :value="`${client.addr}|${client.key}`">
                {{ client.name }} ({{ client.addr }})
              </option>
            </select>
            <button class="btn btn-secondary" @click="clearLogs">清空日志</button>
            <button v-if="!autoScroll" class="btn btn-secondary btn-scroll-to-bottom" @click="scrollToBottom">
              ↓ 回到底部
            </button>
          </div>
        </div>
        
        <div class="logs-container">
          <div class="log-content" ref="logContentRef" @scroll="onLogScroll">
            <div v-if="filteredLogs.length === 0" class="empty-logs">
              <p>暂无日志记录</p>
            </div>
            <div v-for="(log, index) in filteredLogs" :key="index" :class="['log-item', `log-${log.type}`]">
              <span class="log-timestamp">{{ log.timestamp }}</span>
              <span class="log-message">{{ log.message }}</span>
            </div>
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
import { ref, onMounted, computed, watch, nextTick } from 'vue'
// 直接导入 Wails 生成的 API 绑定
import * as AppAPI from '../wailsjs/go/main/App.js'

export default {
  name: 'App',
  setup() {
    const activeView = ref('clients')
    const clients = ref([])
    const commandInput = ref('')
    const message = ref(null)
    const selectedClientId = ref('')
    const allLogs = ref([])
    const logContentRef = ref(null)
    const autoScroll = ref(true)
    const toggleStates = ref({}) // 记录正在切换的客户端，防止快速重复切换
    const logCache = ref({}) // 缓存每个客户端的日志，格式: { clientId: lastSeenLogHash }
    let isLoadingLogs = false // 防止并发加载日志

    // 从直接导入获取 Wails API（使用 let 以便在浏览器中可替换为 mock）
    let GetShortcuts = AppAPI.GetShortcuts
    let AddShortcutFromBase64 = AppAPI.AddShortcutFromBase64
    let RemoveShortcut = AppAPI.RemoveShortcut
    let ToggleClient = AppAPI.ToggleClient
    let TestConnection = AppAPI.TestConnection
    let GetConnectionLogs = AppAPI.GetConnectionLogs
    let ClearConnectionLogs = AppAPI.ClearConnectionLogs

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
      GetConnectionLogs = async (clientId) => {
        console.log('mock GetConnectionLogs', clientId)
        return [
          { timestamp: '2024-01-09 10:30:15', message: 'Mock 日志消息', type: 'info', clientId: clientId }
        ]
      }
      ClearConnectionLogs = async (clientId) => {
        console.log('mock ClearConnectionLogs', clientId)
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
      const clientId = `${client.addr}|${client.key}`
      
      // 如果正在切换中，忽略这次点击
      if (toggleStates.value[clientId]) {
        console.log('Client is already toggling, ignoring this click')
        return
      }
      
      // 根据status判断切换状态
      const isCurrentlyRunning = client.status !== 'stopped'
      const newState = !isCurrentlyRunning
      console.log('Toggling client:', { name: client.name, currentStatus: client.status, newState })
      
      // 标记为正在切换中
      toggleStates.value[clientId] = true
      
      try {
        await ToggleClient(client.name, client.addr, client.key, client.tls, newState)
        console.log('ToggleClient succeeded')
        
        // 稍后重新加载状态，让后端返回最新的状态
        await new Promise(resolve => setTimeout(resolve, 500))
        await loadClients()
        
        showMessage(newState ? '已启动' : '已停止', 'success')
      } catch (error) {
        console.error('Toggle client error:', error)
        const errMsg = extractErrorMessage(error)
        showMessage(`${newState ? '启动' : '停止'}失败: ${errMsg}`, 'error')
        // 确保UI状态回滚到原来的状态
        await loadClients()
      } finally {
        // 清除切换标记
        delete toggleStates.value[clientId]
      }
    }

    const showMessage = (text, type = 'info') => {
      message.value = { text, type }
      setTimeout(() => {
        message.value = null
      }, 3000)
    }

    const getStatusLabel = (status) => {
      switch (status) {
        case 'connected':
          return '✓ 已连接'
        case 'connecting':
          return '⟳ 连接中'
        case 'stopped':
        default:
          return '⊘ 已停止'
      }
    }

    const loadLogs = async () => {
      // 防止并发加载
      if (isLoadingLogs) {
        console.debug('日志已在加载中，跳过本次请求')
        return
      }
      
      isLoadingLogs = true
      try {
        console.log('loadLogs called, selectedClientId=', selectedClientId.value)
        let newLogs = []
        
        if (selectedClientId.value) {
          console.log('加载特定客户端日志:', selectedClientId.value)
          const logs = await GetConnectionLogs(selectedClientId.value)
          console.log('GetConnectionLogs 返回:', logs ? logs.length + ' 条日志' : '0 条日志')
          newLogs = logs || []
        } else {
          // 获取所有客户端的日志
          console.log('加载所有客户端日志，总共', clients.value.length, '个客户端')
          let allClientLogs = []
          for (const client of clients.value) {
            const clientId = `${client.addr}|${client.key}`
            console.log('加载客户端日志:', clientId)
            const logs = await GetConnectionLogs(clientId)
            console.log('该客户端返回:', logs ? logs.length + ' 条日志' : '0 条日志')
            if (logs) {
              allClientLogs = allClientLogs.concat(logs)
            }
          }
          newLogs = allClientLogs
        }

        console.log('本次加载新日志数:', newLogs.length)

        // 创建当前日志的唯一标识集合（用于去重）
        const existingKeys = new Set()
        allLogs.value.forEach(log => {
          const logKey = `${log.timestamp}|${log.message}|${log.clientId}`
          existingKeys.add(logKey)
        })

        // 筛选出新增的日志
        const addedLogs = []
        newLogs.forEach(log => {
          const logKey = `${log.timestamp}|${log.message}|${log.clientId}`
          if (!existingKeys.has(logKey)) {
            addedLogs.push(log)
            existingKeys.add(logKey)
          }
        })

        console.log('新增日志数:', addedLogs.length)

        // 将新增日志添加到现有日志的末尾
        if (addedLogs.length > 0) {
          allLogs.value = allLogs.value.concat(addedLogs)
          
          // 定期进行完整排序，确保顺序正确（每10条新日志排一次）
          if (allLogs.value.length % 10 === 0) {
            allLogs.value.sort((a, b) => {
              // 先按客户端ID排序，再按时间戳排序，最后按消息内容排序
              if (a.clientId !== b.clientId) {
                return a.clientId.localeCompare(b.clientId)
              }
              if (a.timestamp !== b.timestamp) {
                return a.timestamp.localeCompare(b.timestamp)
              }
              return a.message.localeCompare(b.message)
            })
          }
        }
        
        // 限制日志数量，避免内存溢出（最多保留10000条）
        if (allLogs.value.length > 10000) {
          // 保留最新的10000条
          allLogs.value = allLogs.value.slice(allLogs.value.length - 10000)
        }
      } catch (error) {
        console.error('加载日志失败:', error)
      } finally {
        isLoadingLogs = false
      }
    }

    const filteredLogs = computed(() => {
      // 只在选择了特定客户端时过滤，否则显示所有日志
      if (selectedClientId.value) {
        // 使用缓存避免频繁创建新数组
        return allLogs.value.filter(log => log.clientId === selectedClientId.value)
      }
      return allLogs.value
    })

    const clearLogs = async () => {
      if (!confirm('确定要清空日志吗？')) return
      try {
        if (selectedClientId.value) {
          await ClearConnectionLogs(selectedClientId.value)
        } else {
          // 清空所有客户端的日志
          for (const client of clients.value) {
            const clientId = `${client.addr}|${client.key}`
            await ClearConnectionLogs(clientId)
          }
        }
        allLogs.value = []
        showMessage('日志已清空', 'success')
      } catch (error) {
        console.error('清空日志失败:', error)
        showMessage('清空日志失败', 'error')
      }
    }

    // 检查是否在底部
    const isAtBottom = () => {
      if (!logContentRef.value) return true
      const { scrollTop, scrollHeight, clientHeight } = logContentRef.value
      // 允许5px的误差
      return scrollHeight - scrollTop - clientHeight <= 5
    }

    // 滚动到底部
    const scrollToBottom = () => {
      nextTick(() => {
        if (logContentRef.value) {
          logContentRef.value.scrollTop = logContentRef.value.scrollHeight
          autoScroll.value = true
        }
      })
    }

    // 用户滚动时检测是否还在底部
    const onLogScroll = () => {
      if (!isAtBottom()) {
        // 用户已滚上去，禁用自动滚动
        autoScroll.value = false
      } else {
        // 用户在底部，启用自动滚动
        autoScroll.value = true
      }
    }

    // 监听日志内容变化，仅在用户在底部时自动滚动
    // 使用 immediate: false 和防抖逻辑避免频繁更新
    let scrollTimeout = null
    watch(filteredLogs, () => {
      // 清除之前的延时
      if (scrollTimeout) clearTimeout(scrollTimeout)
      
      // 延迟 50ms 后执行滚动，避免频繁触发
      scrollTimeout = setTimeout(() => {
        if (autoScroll.value) {
          scrollToBottom()
        }
      }, 50)
    })

    // 监听日志view激活，定期刷新日志
    let logRefreshInterval = null
    watch(activeView, (newView) => {
      // 清除旧的刷新间隔
      if (logRefreshInterval) {
        clearInterval(logRefreshInterval)
        logRefreshInterval = null
      }
      
      if (newView === 'logs') {
        loadLogs()
        // 设置日志刷新间隔为 3 秒，减少频率避免页面频繁闪烁
        logRefreshInterval = setInterval(() => {
          loadLogs()
        }, 3000)
      }
    })

    onMounted(() => {
      // 直接初始化，因为 API 是静态导入的
      initWails()
      
      // 每 2 秒自动刷新客户端状态，保持与服务器同步
      const refreshInterval = setInterval(() => {
        loadClients()
      }, 2000)

      // 如果初始视图是日志，则加载日志
      if (activeView.value === 'logs') {
        loadLogs()
        logRefreshInterval = setInterval(() => {
          loadLogs()
        }, 3000)
      }
      
      // Cleanup interval on unmount
      return () => {
        clearInterval(refreshInterval)
        if (logRefreshInterval) {
          clearInterval(logRefreshInterval)
        }
      }
    })

    return {
      activeView,
      clients,
      commandInput,
      message,
      selectedClientId,
      allLogs,
      logContentRef,
      autoScroll,
      filteredLogs,
      addConnection,
      removeClient,
      toggleClient,
      getStatusLabel,
      clearLogs,
      loadLogs,
      onLogScroll,
      scrollToBottom,
      isAtBottom,
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

.btn-secondary {
  background: #2d3e54;
  color: #e8eef7;
}

.btn-secondary:hover {
  background: #3a4d66;
  transform: translateY(-1px);
}

.btn-secondary:active {
  transform: translateY(0);
}

.btn-scroll-to-bottom {
  background: #f39c12;
  color: white;
}

.btn-scroll-to-bottom:hover {
  background: #e67e22;
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

.info-row.error-message {
  color: #e74c3c;
  background: rgba(231, 76, 60, 0.1);
  padding: 8px;
  border-radius: 4px;
  border-left: 3px solid #e74c3c;
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

.status-error {
  margin-top: 8px;
  padding: 8px;
  border-radius: 4px;
  background: rgba(231, 76, 60, 0.1);
  border-left: 3px solid #e74c3c;
  color: #e74c3c;
  font-size: 12px;
  line-height: 1.4;
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
  gap: 15px;
}

.logs-header {
  background: #0f1419;
  border: 1px solid #2d3e54;
  border-radius: 8px;
  padding: 15px;
}

.logs-controls {
  display: flex;
  align-items: center;
  gap: 12px;
}

.logs-controls label {
  font-size: 14px;
  color: #a8b5c8;
  font-weight: 500;
}

.client-select {
  padding: 8px 12px;
  background: #1a2332;
  border: 1px solid #2d3e54;
  border-radius: 6px;
  color: #e8eef7;
  font-size: 13px;
  cursor: pointer;
  flex: 1;
  min-width: 200px;
}

.client-select:hover {
  border-color: #3a4d66;
}

.client-select:focus {
  outline: none;
  border-color: #4a5d76;
  box-shadow: 0 0 0 2px rgba(74, 93, 118, 0.2);
}

.logs-container {
  flex: 1;
  background: #0f1419;
  border: 1px solid #2d3e54;
  border-radius: 8px;
  padding: 15px;
  display: flex;
  flex-direction: column;
  min-height: 300px;
}

.log-content {
  flex: 1;
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  color: #a8b5c8;
  overflow-y: auto;
  word-break: break-all;
  white-space: pre-wrap;
}

.empty-logs {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #5a6d7f;
  font-style: italic;
}

.log-item {
  padding: 6px 0;
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.log-timestamp {
  color: #5a6d7f;
  flex-shrink: 0;
  font-weight: 500;
}

.log-message {
  color: #a8b5c8;
  flex: 1;
}

.log-info .log-timestamp {
  color: #5a9fd4;
}

.log-info .log-message {
  color: #a8b5c8;
}

.log-success .log-timestamp {
  color: #2ecc71;
}

.log-success .log-message {
  color: #2ecc71;
}

.log-warning .log-timestamp {
  color: #f39c12;
}

.log-warning .log-message {
  color: #f39c12;
}

.log-error .log-timestamp {
  color: #e74c3c;
}

.log-error .log-message {
  color: #e74c3c;
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
