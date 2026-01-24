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
              placeholder="输入快捷启动命令"
              @keyup.enter="addConnection"
            />
            <button class="btn btn-primary" @click="addConnection">连接</button>
            <button class="btn btn-secondary" @click="showManualAddDialog">手工添加</button>
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
          <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:12px">
            <h3 style="margin:0">设置</h3>
            <div style="color:var(--text-secondary)">版本: {{ appVersion || '未知' }}</div>
          </div>

          <div style="display:flex;align-items:center;gap:12px;margin-bottom:12px">
            <label style="flex:1;color:var(--text-secondary)">主题</label>
            <select v-model="themeMode" class="theme-select">
              <option value="auto">跟随系统</option>
              <option value="light">亮色</option>
              <option value="dark">暗色</option>
            </select>
          </div>

          <div style="display:flex;align-items:center;gap:12px;margin-bottom:12px">
            <label style="flex:1;color:var(--text-secondary)">开机启动</label>
            <label class="toggle-switch">
              <input type="checkbox" v-model="startupEnabled" />
              <span class="toggle-slider"></span>
            </label>
          </div>

          <div style="display:flex;align-items:center;gap:12px;margin-bottom:12px">
            <label style="flex:1;color:var(--text-secondary)">记住客户端状态</label>
            <label class="toggle-switch">
              <input type="checkbox" v-model="rememberClientState" />
              <span class="toggle-slider"></span>
            </label>
          </div>

          <div style="display:flex;align-items:center;gap:12px;margin-bottom:18px">
            <label style="flex:1;color:var(--text-secondary)">日志目录</label>
            <div style="display:flex;gap:8px;align-items:center">
              <input v-model="logDir" type="text" style="padding:8px;border-radius:6px;border:1px solid var(--border-color);background:var(--bg-primary);color:var(--text-primary);min-width:320px" readonly />
              <button class="btn btn-secondary" @click="selectLogDirectory" style="white-space:nowrap">浏览...</button>
            </div>
          </div>

          <div style="display:flex;gap:12px;justify-content:flex-end">
            <button class="btn btn-secondary" @click="resetSettings">重置</button>
            <button class="btn btn-primary" @click="saveSettings">保存</button>
          </div>
        </div>
      </div>

      <div v-if="message" :class="['message', message.type]">
        {{ message.text }}
      </div>

      <!-- 手工添加客户端对话框 -->
      <div v-if="showManualDialog" class="modal-overlay" @click.self="closeManualAddDialog">
        <div class="modal-dialog">
          <div class="modal-header">
            <h3>手工添加客户端</h3>
            <button class="btn-close" @click="closeManualAddDialog">✕</button>
          </div>
          <div class="modal-body">
            <div class="form-group">
              <label>名称 </label>
              <input v-model="manualForm.name" type="text" class="form-input" placeholder="例如: test" />
            </div>
            <div class="form-group">
              <label>连接地址 <span class="required">*</span></label>
              <input v-model="manualForm.addr" type="text" class="form-input" placeholder="例如: 127.0.0.1:8024" />
            </div>
            <div class="form-group">
              <label>密钥 <span class="required">*</span></label>
              <input v-model="manualForm.key" type="text" class="form-input" placeholder="例如: 6237ed8d52" />
            </div>
            <div class="form-group">
              <label class="checkbox-label">
                <input type="checkbox" v-model="manualForm.tls" />
                <span>启用 TLS</span>
              </label>
            </div>
            <div v-if="manualFormError" class="form-error">
              {{ manualFormError }}
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" @click="closeManualAddDialog">取消</button>
            <button class="btn btn-primary" @click="submitManualAdd">确定</button>
          </div>
        </div>
      </div>

      <!-- 确认对话框 -->
      <div v-if="confirmState.show" class="modal-overlay" @click.self="confirmCancel">
        <div class="modal-dialog" style="max-width:360px">
          <div class="modal-header">
            <h3>确认</h3>
            <button class="btn-close" @click="confirmCancel">✕</button>
          </div>
          <div class="modal-body">
            <div style="white-space:pre-wrap">{{ confirmState.text }}</div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" @click="confirmCancel">取消</button>
            <button class="btn btn-primary" @click="confirmOk">确定</button>
          </div>
        </div>
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
    let hasRestoredClientStates = false // 标记是否已恢复过客户端状态（只在首次加载时恢复一次）

    // Settings
    const startupEnabled = ref(true)
    const rememberClientState = ref(true)
    const logDir = ref('')
    const themeMode = ref('auto') // 'auto', 'light', 'dark'
    const appVersion = ref('')

    // Manual add dialog
    const showManualDialog = ref(false)
    const manualForm = ref({
      name: '',
      addr: '',
      key: '',
      tls: false
    })
    const manualFormError = ref('')

    // Theme
    const isDarkTheme = ref(true)

    const confirmState = ref({
      show: false,
      text: '',
      resolve: null
    })

    const SETTINGS_KEY = 'npc_settings'
    const CLIENT_STATES_KEY = 'npc_client_states'

    // 检测系统主题
    const detectSystemTheme = () => {
      if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
        return 'dark'
      }
      return 'light'
    }

    // 应用主题（auto 时交给 CSS 的 prefers-color-scheme 处理）
    const applyTheme = (theme) => {
      if (theme === 'auto') {
        document.documentElement.removeAttribute('data-theme')
        if (window.matchMedia) {
          isDarkTheme.value = window.matchMedia('(prefers-color-scheme: dark)').matches
        }
        return
      }
      isDarkTheme.value = theme === 'dark'
      document.documentElement.setAttribute('data-theme', theme)
    }

    // 根据主题模式应用主题
    const applyThemeMode = (mode) => {
      if (mode === 'auto') {
        applyTheme('auto')
      } else {
        applyTheme(mode)
      }
    }

    // 初始化主题
    const initTheme = () => {
      applyThemeMode(themeMode.value)
    }

    const detectDefaultLogDir = async () => {
      try {
        // 优先从后端获取默认路径
        if (typeof GetDefaultLogDir === 'function') {
          const defaultPath = await GetDefaultLogDir()
          if (defaultPath) {
            return defaultPath
          }
        }
      } catch (e) {
        console.warn('GetDefaultLogDir failed, using fallback', e)
      }

      // Fallback: 基于平台猜测路径
      try {
        const platform = navigator.platform || navigator.userAgent || ''
        if (/Win/i.test(platform)) {
          // Windows: 使用 AppData\Roaming\npc\logs
          return 'C:\\Users\\' + (process.env.USERNAME || 'User') + '\\AppData\\Roaming\\npc\\logs'
        }
        if (/Mac/i.test(platform)) return '~/Library/Application Support/npc/logs'
        if (/Linux/i.test(platform)) return '~/.config/npc/logs'
      } catch (e) {
        // fallback
      }
      return ''
    }

    const loadAppVersion = async () => {
      try {
        if (typeof GetAppVersion === 'function') {
          appVersion.value = await GetAppVersion()
        } else {
          appVersion.value = ''
        }
      } catch (e) {
        console.warn('GetAppVersion failed', e)
      }
    }

    const loadSettings = async () => {
      try {
        if (typeof GetGuiSettings === 'function') {
          const s = await GetGuiSettings()
          startupEnabled.value = typeof s.startupEnabled === 'boolean' ? s.startupEnabled : true
          rememberClientState.value = typeof s.rememberClientState === 'boolean' ? s.rememberClientState : true
          logDir.value = typeof s.logDir === 'string' && s.logDir ? s.logDir : await detectDefaultLogDir()
          themeMode.value = typeof s.themeMode === 'string' && ['auto', 'light', 'dark'].includes(s.themeMode) ? s.themeMode : 'auto'
          return
        }
      } catch (e) {
        console.warn('GetGuiSettings failed, fallback to localStorage', e)
      }

      // fallback: localStorage or defaults
      try {
        const raw = localStorage.getItem(SETTINGS_KEY)
        if (raw) {
          const s = JSON.parse(raw)
          startupEnabled.value = typeof s.startupEnabled === 'boolean' ? s.startupEnabled : true
          rememberClientState.value = typeof s.rememberClientState === 'boolean' ? s.rememberClientState : true
          logDir.value = typeof s.logDir === 'string' && s.logDir ? s.logDir : await detectDefaultLogDir()
          themeMode.value = typeof s.themeMode === 'string' && ['auto', 'light', 'dark'].includes(s.themeMode) ? s.themeMode : 'auto'
        } else {
          // defaults
          startupEnabled.value = true
          rememberClientState.value = true
          logDir.value = await detectDefaultLogDir()
          themeMode.value = 'auto'
        }
      } catch (e) {
        startupEnabled.value = true
        rememberClientState.value = true
        logDir.value = await detectDefaultLogDir()
        themeMode.value = 'auto'
      }
    }

    const resetSettings = async () => {
      // 重置到默认值
      startupEnabled.value = true
      rememberClientState.value = true
      logDir.value = await detectDefaultLogDir()
      themeMode.value = 'auto'
      showMessage('已重置为默认值', 'success')
    }

    const saveSettings = async () => {
      try {
        const s = {
          startupEnabled: !!startupEnabled.value,
          rememberClientState: !!rememberClientState.value,
          logDir: logDir.value,
          themeMode: themeMode.value
        }

        // 优先使用后端绑定保存
        if (typeof SaveGuiSettings === 'function') {
          await SaveGuiSettings(s)
        } else {
          localStorage.setItem(SETTINGS_KEY, JSON.stringify(s))
        }

        // 保存 client 状态（如果开启）
        if (rememberClientState.value) {
          const map = {}
          clients.value.forEach(c => {
            const id = `${c.addr}|${c.key}`
            map[id] = c.status || 'stopped'
          })
          if (typeof SaveClientStates === 'function') {
            await SaveClientStates(map)
          } else {
            localStorage.setItem(CLIENT_STATES_KEY, JSON.stringify(map))
          }
        }

        showMessage('设置已保存', 'success')
      } catch (e) {
        console.error('保存设置失败', e)
        showMessage('保存设置失败', 'error')
      }
    }

    const selectLogDirectory = async () => {
      try {
        console.log('selectLogDirectory 被调用')
        console.log('SelectDirectory 类型:', typeof SelectDirectory)

        if (typeof SelectDirectory === 'function') {
          console.log('准备调用 SelectDirectory')
          const selectedPath = await SelectDirectory()
          console.log('选择的路径:', selectedPath)

          if (selectedPath && selectedPath.trim() !== '') {
            logDir.value = selectedPath
            showMessage('目录已选择', 'success')
          } else {
            console.log('用户取消了选择或返回空路径')
            // 用户取消了选择，不显示错误消息
          }
        } else {
          console.warn('SelectDirectory 不是函数')
          showMessage('目录选择功能不可用', 'error')
        }
      } catch (e) {
        console.error('选择目录失败:', e)
        showMessage('选择目录失败: ' + e.message, 'error')
      }
    }

    // 从直接导入获取 Wails API（使用 let 以便在浏览器中可替换为 mock）
    let GetShortcuts = AppAPI.GetShortcuts
    let AddShortcut = AppAPI.AddShortcut
    let AddShortcutFromBase64 = AppAPI.AddShortcutFromBase64
    let RemoveShortcut = AppAPI.RemoveShortcut
    let ToggleClient = AppAPI.ToggleClient
    let GetConnectionLogs = AppAPI.GetConnectionLogs
    let ClearConnectionLogs = AppAPI.ClearConnectionLogs

    // 在普通浏览器里运行时 Wails API 可能不存在，提供简单 mock 方便调试 UI
    // 同时尝试绑定新的设置 & clientStates API
    let GetGuiSettings = AppAPI.GetGuiSettings
    let SaveGuiSettings = AppAPI.SaveGuiSettings
    let GetClientStates = AppAPI.GetClientStates
    let SaveClientStates = AppAPI.SaveClientStates
    let SelectDirectory = AppAPI.SelectDirectory
    let GetDefaultLogDir = AppAPI.GetDefaultLogDir
    let GetAppVersion = AppAPI.GetAppVersion

    if (!AppAPI || typeof AppAPI.GetShortcuts !== 'function') {
      console.warn('Wails App API not available — using mock implementations for browser debugging')
      GetShortcuts = async () => {
        return [
          { name: 'MyServer', addr: '127.0.0.1:8024', key: 'alefa114df', tls: false, running: false },
        ]
      }
      AddShortcut = async (jsonStr) => {
        console.log('mock AddShortcut', jsonStr)
        return
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

      GetGuiSettings = async () => ({ startupEnabled: true, rememberClientState: true, logDir: '' })
      SaveGuiSettings = async (s) => { console.log('mock SaveGuiSettings', s); return }
      GetClientStates = async () => { return {} }
      SaveClientStates = async (m) => { console.log('mock SaveClientStates', m); return }
      SelectDirectory = async () => { console.log('mock SelectDirectory'); return '/mock/selected/path' }
      GetDefaultLogDir = async () => { console.log('mock GetDefaultLogDir'); return 'C:\\Users\\User\\AppData\\Roaming\\npc\\logs' }
      GetAppVersion = async () => { console.log('mock GetAppVersion'); return 'dev' }
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

        // 只在首次加载时恢复客户端状态，避免后续刷新时重复恢复
        if (!hasRestoredClientStates && rememberClientState.value) {
          hasRestoredClientStates = true
          console.log('首次加载，尝试恢复客户端状态...')

          try {
            let map = null
            if (typeof GetClientStates === 'function') {
              try {
                map = await GetClientStates()
              } catch (e) {
                console.warn('GetClientStates failed, fallback to localStorage', e)
              }
            }
            if (!map) {
              const raw = localStorage.getItem(CLIENT_STATES_KEY)
              if (raw) {
                map = JSON.parse(raw)
              }
            }

            if (map) {
              for (const c of clients.value) {
                const id = `${c.addr}|${c.key}`
                if (map[id] === 'connected' && c.status !== 'connected') {
                  console.log('恢复客户端连接:', c.name)
                  try {
                    await ToggleClient(c.name, c.addr, c.key, c.tls, true)
                    await new Promise(r => setTimeout(r, 300))
                  } catch (e) {
                    console.warn('恢复客户端状态失败', id, e)
                  }
                }
              }
              // 刷新一次客户端列表以获取最新状态
              const refreshed = await GetShortcuts()
              clients.value = refreshed || clients.value
            }
          } catch (e) {
            console.warn('恢复客户端状态过程发生错误', e)
          }
        }
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
      if (!input) {
        showMessage('请输入快捷启动命令', 'error')
        return
      }

      try {
        // Try to parse as Base64 first
        if (input.length > 10 && !input.includes('|')) {
          await AddShortcutFromBase64(input)
        } else {
          // Try direct key connection
          showMessage('快捷启动命令格式错误', 'error')
          return
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

    const showManualAddDialog = () => {
      manualForm.value = {
        name: '',
        addr: '',
        key: '',
        tls: false
      }
      manualFormError.value = ''
      showManualDialog.value = true
    }

    const closeManualAddDialog = () => {
      showManualDialog.value = false
      manualFormError.value = ''
    }

    const submitManualAdd = async () => {
      // 清除之前的错误
      manualFormError.value = ''

      // 验证必填字段
      const { name, addr, key, tls } = manualForm.value

      if (!addr || !addr.trim()) {
        manualFormError.value = '请输入连接地址'
        return
      }

      if (!key || !key.trim()) {
        manualFormError.value = '请输入密钥'
        return
      }

      // 检查是否已存在相同的客户端
      const clientId = `${addr.trim()}|${key.trim()}`
      const existingClient = clients.value.find(c => `${c.addr}|${c.key}` === clientId)
      if (existingClient) {
        manualFormError.value = '该客户端已存在，不能重复添加'
        return
      }

      try {
        // 构造 ShortClient 对象
        const shortClient = {
          name: name.trim(),
          addr: addr.trim(),
          key: key.trim(),
          tls: tls
        }

        // 调用 AddShortcut API，传递 JSON 字符串
        await AddShortcut(JSON.stringify(shortClient))

        closeManualAddDialog()
        await loadClients()
        showMessage('客户端已添加', 'success')
      } catch (error) {
        console.error('Manual add error:', error)
        const errMsg = extractErrorMessage(error)
        manualFormError.value = `添加失败: ${errMsg}`
      }
    }

    const removeClient = async (client) => {
      const confirmed = await confirmDialog(`确定要删除 "${client.name}" 吗？`)
      if (!confirmed) return

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

        // 如果启用了记住客户端状态，保存当前状态到本地和后端
        try {
          if (rememberClientState.value) {
            const map = {}
            clients.value.forEach(c => {
              const id = `${c.addr}|${c.key}`
              map[id] = c.status || 'stopped'
            })
            // 保存到后端
            if (typeof SaveClientStates === 'function') {
              try {
                await SaveClientStates(map)
              } catch (err) {
                console.warn('保存客户端状态到后端失败，fallback to localStorage', err)
              }
            }
            // 同时保存到 localStorage 作为备份
            localStorage.setItem(CLIENT_STATES_KEY, JSON.stringify(map))
          }
        } catch (e) {
          console.warn('保存客户端状态失败', e)
        }

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
      const confirmed = await confirmDialog('确定要清空日志吗？')
      if (!confirmed) return
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

    const confirmDialog = (text) => {
      return new Promise((resolve) => {
        confirmState.value = { show: true, text, resolve }
      })
    }

    const closeConfirmDialog = (confirmed) => {
      if (confirmState.value.resolve) {
        confirmState.value.resolve(confirmed)
      }
      confirmState.value = { show: false, text: '', resolve: null }
    }

    const confirmOk = () => closeConfirmDialog(true)
    const confirmCancel = () => closeConfirmDialog(false)

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

    // 监听主题模式变化
    watch(themeMode, (newMode) => {
      applyThemeMode(newMode)
    })

    onMounted(async () => {
      // 先加载本地设置
      await loadSettings()

      // 加载设置后再初始化主题
      initTheme()

      // 监听系统主题变化（仅在 auto 模式下生效）
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
      const handleThemeChange = (e) => {
        if (themeMode.value === 'auto') {
          isDarkTheme.value = e.matches
          document.documentElement.removeAttribute('data-theme')
        }
      }
      if (mediaQuery.addEventListener) {
        mediaQuery.addEventListener('change', handleThemeChange)
      } else if (mediaQuery.addListener) {
        mediaQuery.addListener(handleThemeChange)
      }

      // 初始化 Wails
      initWails()

      // 加载版本号（如果后端已绑定）
      await loadAppVersion()

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
        if (mediaQuery.removeEventListener) {
          mediaQuery.removeEventListener('change', handleThemeChange)
        } else if (mediaQuery.removeListener) {
          mediaQuery.removeListener(handleThemeChange)
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
      // settings
      startupEnabled,
      rememberClientState,
      logDir,
      themeMode,
      appVersion,
      loadSettings,
      resetSettings,
      saveSettings,
      selectLogDirectory,
      // manual add
      showManualDialog,
      manualForm,
      manualFormError,
      showManualAddDialog,
      closeManualAddDialog,
      submitManualAdd,
      confirmState,
      confirmOk,
      confirmCancel,
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

<style>
/* CSS Variables for Theme */
:root {
  /* Dark Theme (Default) */
  --bg-primary: #1a2332;
  --bg-secondary: #0f1419;
  --bg-tertiary: #2d3e54;
  --text-primary: #e8eef7;
  --text-secondary: #a8b5c8;
  --text-tertiary: #5a6d7f;
  --border-color: #2d3e54;
  --accent-color: #2b8fe8;
  --accent-hover: #2079d4;
  --success-color: #2ecc71;
  --error-color: #e74c3c;
  --warning-color: #f39c12;
}

/* Light Theme (explicit) */
[data-theme="light"] {
  --bg-primary: #f5f7fa;
  --bg-secondary: #ffffff;
  --bg-tertiary: #e4e7eb;
  --text-primary: #1a202c;
  --text-secondary: #4a5568;
  --text-tertiary: #718096;
  --border-color: #cbd5e0;
  --accent-color: #3182ce;
  --accent-hover: #2c5aa0;
  --success-color: #38a169;
  --error-color: #e53e3e;
  --warning-color: #dd6b20;
}

/* Light Theme (auto via prefers-color-scheme) */
@media (prefers-color-scheme: light) {
  :root:not([data-theme]) {
    --bg-primary: #f5f7fa;
    --bg-secondary: #ffffff;
    --bg-tertiary: #e4e7eb;
    --text-primary: #1a202c;
    --text-secondary: #4a5568;
    --text-tertiary: #718096;
    --border-color: #cbd5e0;
    --accent-color: #3182ce;
    --accent-hover: #2c5aa0;
    --success-color: #38a169;
    --error-color: #e53e3e;
    --warning-color: #dd6b20;
  }
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

.container {
  display: flex;
  height: 100vh;
  background: var(--bg-primary);
  color: var(--text-primary);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell,
    sans-serif;
}

/* Sidebar */
.sidebar {
  width: 180px;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border-color);
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
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 6px;
  font-size: 14px;
  transition: all 0.2s ease;
  text-align: left;
}

.sidebar-btn:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.sidebar-btn.active {
  background: var(--accent-color);
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
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  border-radius: 6px;
  font-size: 14px;
  transition: border-color 0.2s ease;
}

.command-input:focus {
  outline: none;
  border-color: var(--accent-color);
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
  background: var(--accent-color);
  color: white;
}

.btn-primary:hover {
  background: var(--accent-hover);
  transform: translateY(-1px);
}

.btn-primary:active {
  transform: translateY(0);
}

.btn-secondary {
  background: var(--border-color);
  color: var(--text-primary);
}

.btn-secondary:hover {
  background: var(--bg-tertiary);
  transform: translateY(-1px);
}

.btn-secondary:active {
  transform: translateY(0);
}

.btn-scroll-to-bottom {
  background: var(--warning-color);
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
  color: var(--text-secondary);
}

.client-card {
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  transition: all 0.2s ease;
}

.client-card:hover {
  border-color: var(--accent-color);
  box-shadow: 0 4px 12px rgba(43, 143, 232, 0.1);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 15px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.btn-close {
  background: transparent;
  border: none;
  color: var(--text-secondary);
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
  color: var(--error-color);
  background: rgba(231, 76, 60, 0.1);
  padding: 8px;
  border-radius: 4px;
  border-left: 3px solid var(--error-color);
}

.label {
  color: var(--text-secondary);
  min-width: 50px;
}

.value {
  color: var(--text-primary);
  word-break: break-all;
  flex: 1;
}

.value.code {
  font-family: 'Monaco', 'Courier New', monospace;
  background: var(--bg-secondary);
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 12px;
}

.card-footer {
  padding: 12px 15px;
  background: var(--bg-secondary);
  border-top: 1px solid var(--border-color);
}

.status-error {
  margin-top: 8px;
  padding: 8px;
  border-radius: 4px;
  background: rgba(231, 76, 60, 0.1);
  border-left: 3px solid var(--error-color);
  color: var(--error-color);
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
  background: var(--border-color);
  border-radius: 12px;
  position: relative;
  transition: background 0.3s ease;
}

.toggle-switch input:checked + .toggle-slider {
  background: var(--accent-color);
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
  color: var(--text-secondary);
}

/* Logs View */
.logs-view {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.logs-header {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
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
  color: var(--text-secondary);
  font-weight: 500;
}

.client-select {
  padding: 8px 12px;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 13px;
  cursor: pointer;
  flex: 1;
  min-width: 200px;
}

.client-select:hover {
  border-color: var(--bg-tertiary);
}

.client-select:focus {
  outline: none;
  border-color: #4a5d76;
  box-shadow: 0 0 0 2px rgba(74, 93, 118, 0.2);
}

.logs-container {
  flex: 1;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
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
  color: var(--text-secondary);
  overflow-y: auto;
  word-break: break-all;
  white-space: pre-wrap;
}

.empty-logs {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--text-tertiary);
  font-style: italic;
}

.log-item {
  padding: 6px 0;
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.log-timestamp {
  color: var(--text-tertiary);
  flex-shrink: 0;
  font-weight: 500;
}

.log-message {
  color: var(--text-secondary);
  flex: 1;
}

.log-info .log-timestamp {
  color: #5a9fd4;
}

.log-info .log-message {
  color: var(--text-secondary);
}

.log-success .log-timestamp {
  color: var(--success-color);
}

.log-success .log-message {
  color: var(--success-color);
}

.log-warning .log-timestamp {
  color: var(--warning-color);
}

.log-warning .log-message {
  color: var(--warning-color);
}

.log-error .log-timestamp {
  color: var(--error-color);
}

.log-error .log-message {
  color: var(--error-color);
}

/* Settings View */
.settings-view {
  display: flex;
  flex-direction: column;
}

.settings-container {
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 20px;
}

.theme-select {
  padding: 8px 12px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 14px;
  cursor: pointer;
  min-width: 120px;
}

.theme-select:hover {
  border-color: var(--bg-tertiary);
}

.theme-select:focus {
  outline: none;
  border-color: var(--accent-color);
  box-shadow: 0 0 0 2px rgba(43, 143, 232, 0.1);
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
  background: var(--success-color);
  color: white;
}

.message.error {
  background: var(--error-color);
  color: white;
}

.message.info {
  background: var(--accent-color);
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

/* Modal */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
  animation: fadeIn 0.2s ease;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.modal-dialog {
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  width: 90%;
  max-width: 500px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  animation: slideUp 0.3s ease;
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid var(--border-color);
}

.modal-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
}

.modal-body {
  padding: 20px;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 15px 20px;
  border-top: 1px solid var(--border-color);
  background: var(--bg-secondary);
}

.form-group {
  margin-bottom: 16px;
}

.form-group:last-child {
  margin-bottom: 0;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  font-size: 14px;
  color: var(--text-secondary);
  font-weight: 500;
}

.required {
  color: var(--error-color);
  margin-left: 2px;
}

.form-input {
  width: 100%;
  padding: 10px 12px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 14px;
  transition: border-color 0.2s ease;
}

.form-input:focus {
  outline: none;
  border-color: var(--accent-color);
  box-shadow: 0 0 0 2px rgba(43, 143, 232, 0.1);
}

.form-input::placeholder {
  color: var(--text-tertiary);
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  user-select: none;
}

.checkbox-label input[type="checkbox"] {
  width: 18px;
  height: 18px;
  cursor: pointer;
}

.checkbox-label span {
  font-size: 14px;
  color: var(--text-secondary);
}

.form-error {
  margin-top: 12px;
  padding: 10px 12px;
  background: rgba(231, 76, 60, 0.1);
  border: 1px solid var(--error-color);
  border-radius: 6px;
  color: var(--error-color);
  font-size: 13px;
  line-height: 1.4;
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
  background: var(--border-color);
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: var(--bg-tertiary);
}
</style>
