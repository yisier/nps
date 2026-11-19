<template>
  <div class="app">
    <aside class="sidebar">
      <div class="sidebar-brand">
        <div class="brand-mark">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M4.5 16.5c-1.5 1.26-2 5-2 5s3.74-.5 5-2c.71-.84.7-2.13-.09-2.91a2.18 2.18 0 0 0-2.91-.09z"></path>
            <path d="m12 15-3-3a22 22 0 0 1 2-3.95A12.88 12.88 0 0 1 22 2c0 2.72-.78 7.5-6 11a22.35 22.35 0 0 1-4 2z"></path>
            <path d="M9 12H4s.55-3.03 2-4c1.62-1.08 5 0 5 0"></path>
            <path d="M12 15v5s3.03-.55 4-2c1.08-1.62 0-5 0-5"></path>
          </svg>
        </div>
        <div class="brand-text">
          <span class="brand-title">NPS Client</span>
          <span class="brand-sub">内网穿透客户端</span>
        </div>
      </div>

      <nav class="sidebar-nav">
        <button class="nav-item" :class="{ active: activeView === 'clients' }" @click="activeView = 'clients'">
          <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="4 17 10 11 4 5"></polyline>
            <line x1="12" y1="19" x2="20" y2="19"></line>
          </svg>
          <span>终端</span>
        </button>
        <button class="nav-item" :class="{ active: activeView === 'logs' }" @click="activeView = 'logs'">
          <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
            <polyline points="14 2 14 8 20 8"></polyline>
            <line x1="8" y1="13" x2="16" y2="13"></line>
            <line x1="8" y1="17" x2="16" y2="17"></line>
          </svg>
          <span>日志</span>
        </button>
        <button class="nav-item" :class="{ active: activeView === 'settings' }" @click="activeView = 'settings'">
          <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="4" y1="21" x2="4" y2="14"></line>
            <line x1="4" y1="10" x2="4" y2="3"></line>
            <line x1="12" y1="21" x2="12" y2="12"></line>
            <line x1="12" y1="8" x2="12" y2="3"></line>
            <line x1="20" y1="21" x2="20" y2="16"></line>
            <line x1="20" y1="12" x2="20" y2="3"></line>
            <line x1="1" y1="14" x2="7" y2="14"></line>
            <line x1="9" y1="8" x2="15" y2="8"></line>
            <line x1="17" y1="16" x2="23" y2="16"></line>
          </svg>
          <span>设置</span>
        </button>
      </nav>

      <div class="sidebar-footer">
        <span class="version-tag">v{{ appVersion || 'dev' }}</span>
      </div>
    </aside>

    <main class="main">
      <div v-if="activeView === 'clients'" class="view clients-view">
        <div class="page-header">
          <div>
            <h1 class="page-title">终端</h1>
            <p class="page-desc">管理你的内网穿透客户端连接</p>
          </div>
        </div>

        <div v-if="updateInfo && updateInfo.updateAvailable && !updateBannerDismissed" class="update-banner">
          <div class="update-banner-icon">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
              <polyline points="7 10 12 15 17 10"></polyline>
              <line x1="12" y1="15" x2="12" y2="3"></line>
            </svg>
          </div>
          <div class="update-banner-body">
            <div class="update-banner-title">发现新版本 v{{ updateInfo.latestVersion }}</div>
            <div class="update-banner-desc">当前版本 v{{ updateInfo.currentVersion }}，点击立即更新即可热升级</div>
          </div>
          <div class="update-banner-actions">
            <button class="btn btn-default btn-sm" :disabled="updating" @click="downloadUpdate">
              {{ updating ? '更新中...' : '立即更新' }}
            </button>
            <button class="btn btn-ghost btn-icon" title="忽略" @click="updateBannerDismissed = true">
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <line x1="18" y1="6" x2="6" y2="18"></line>
                <line x1="6" y1="6" x2="18" y2="18"></line>
              </svg>
            </button>
          </div>
        </div>

        <div class="toolbar">
          <input
            v-model="commandInput"
            type="text"
            class="input toolbar-input"
            placeholder="粘贴 Base64 快捷启动命令"
            @keyup.enter="addConnection"
          />
          <button class="btn btn-default" @click="addConnection">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round">
              <line x1="12" y1="5" x2="12" y2="19"></line>
              <line x1="5" y1="12" x2="19" y2="12"></line>
            </svg>
            连接
          </button>
          <button class="btn btn-outline" @click="showManualAddDialog">手工添加</button>
        </div>

        <div class="clients-grid">
          <div v-if="clients.length === 0" class="empty-state">
            <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round">
              <path d="M21 8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16Z"></path>
              <polyline points="3.29 7 12 12 20.71 7"></polyline>
              <line x1="12" y1="22" x2="12" y2="12"></line>
            </svg>
            <p class="empty-title">暂无客户端</p>
            <p class="empty-desc">粘贴 Base64 格式的快捷命令并点击连接即可添加</p>
          </div>

          <div v-for="(client, index) in clients" :key="index" class="card client-card">
            <div class="card-header">
              <div class="card-title-group">
                <h3 class="card-title">{{ client.name }}</h3>
                <span class="badge" :class="statusBadgeClass(client.status)">
                  <span class="badge-dot"></span>
                  {{ getStatusLabel(client.status) }}
                </span>
              </div>
              <button class="btn btn-ghost btn-icon" title="删除" @click="removeClient(client)">
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <line x1="18" y1="6" x2="6" y2="18"></line>
                  <line x1="6" y1="6" x2="18" y2="18"></line>
                </svg>
              </button>
            </div>

            <div class="card-body">
              <div class="info-row">
                <span class="label">地址</span>
                <span class="value mono">{{ client.addr }}</span>
              </div>
              <div class="info-row">
                <span class="label">密钥</span>
                <span class="value mono">{{ client.key }}</span>
              </div>
              <div class="info-row">
                <span class="label">TLS</span>
                <span class="value">{{ client.tls ? '已启用' : '未启用' }}</span>
              </div>
              <div v-if="client.error && client.running" class="info-row error-row">
                <span class="label">错误</span>
                <span class="value error-text">{{ client.error }}</span>
              </div>
            </div>

            <div class="card-footer">
              <span class="switch">
                <input
                  type="checkbox"
                  :checked="client.status !== 'stopped'"
                  @change="toggleClient(client)"
                />
                <span class="switch-thumb"></span>
              </span>
              <div v-if="client.error && client.status !== 'stopped'" class="status-error">
                {{ client.error }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="activeView === 'logs'" class="view logs-view">
        <div class="page-header">
          <div>
            <h1 class="page-title">日志</h1>
            <p class="page-desc">查看各客户端连接日志</p>
          </div>
        </div>

        <div class="card logs-panel">
          <div class="logs-toolbar">
            <select v-model="selectedClientId" class="select logs-select">
              <option value="">全部客户端</option>
              <option v-for="client in clients" :key="`${client.addr}|${client.key}`" :value="`${client.addr}|${client.key}`">
                {{ client.name }} ({{ client.addr }})
              </option>
            </select>
            <div class="logs-actions">
              <button v-if="!autoScroll" class="btn btn-outline" @click="scrollToBottom">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
                  <line x1="12" y1="5" x2="12" y2="19"></line>
                  <polyline points="19 12 12 19 5 12"></polyline>
                </svg>
                回到底部
              </button>
              <button class="btn btn-outline" @click="clearLogs">清空日志</button>
            </div>
          </div>

          <div class="logs-container">
            <div class="log-content" ref="logContentRef" @scroll="onLogScroll">
              <div v-if="filteredLogs.length === 0" class="empty-logs">
                <p>暂无日志记录</p>
              </div>
              <div v-for="(log, index) in filteredLogs" :key="index" :class="['log-item', `log-${log.type}`]">
                <span class="log-dot"></span>
                <span class="log-timestamp">{{ log.timestamp }}</span>
                <span class="log-message">{{ log.message }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="activeView === 'settings'" class="view settings-view">
        <div class="page-header">
          <div>
            <h1 class="page-title">设置</h1>
            <p class="page-desc">配置客户端行为</p>
          </div>
        </div>

        <div class="card settings-card">
          <div class="settings-group">
            <div class="setting-row">
              <div class="setting-info">
                <div class="setting-label">主题</div>
                <div class="setting-desc">选择界面外观风格</div>
              </div>
              <select v-model="themeMode" class="select">
                <option value="auto">跟随系统</option>
                <option value="light">亮色</option>
                <option value="dark">暗色</option>
              </select>
            </div>

            <div class="setting-row">
              <div class="setting-info">
                <div class="setting-label">开机启动</div>
                <div class="setting-desc">系统启动时自动运行客户端</div>
              </div>
              <span class="switch">
                <input type="checkbox" v-model="startupEnabled" />
                <span class="switch-thumb"></span>
              </span>
            </div>

            <div class="setting-row">
              <div class="setting-info">
                <div class="setting-label">记住客户端状态</div>
                <div class="setting-desc">下次启动时自动恢复已连接客户端</div>
              </div>
              <span class="switch">
                <input type="checkbox" v-model="rememberClientState" />
                <span class="switch-thumb"></span>
              </span>
            </div>

            <div class="setting-row">
              <div class="setting-info">
                <div class="setting-label">日志目录</div>
                <div class="setting-desc">日志文件的保存位置</div>
              </div>
              <div class="logdir-field">
                <input v-model="logDir" type="text" class="input logdir-input" readonly />
                <button class="btn btn-outline" @click="selectLogDirectory">浏览...</button>
              </div>
            </div>

            <div class="setting-row">
              <div class="setting-info">
                <div class="setting-label">版本</div>
                <div class="setting-desc">
                  <template v-if="updating">正在下载更新...</template>
                  <template v-else-if="checkingUpdate">正在检查更新...</template>
                  <template v-else-if="updateInfo && updateInfo.updateAvailable">
                    发现新版本 v{{ updateInfo.latestVersion }}（当前 v{{ updateInfo.currentVersion }}）
                  </template>
                  <template v-else-if="updateInfo">已是最新版本 v{{ updateInfo.currentVersion }}</template>
                  <template v-else>v{{ appVersion || 'dev' }} · 启动时自动检查更新</template>
                </div>
              </div>
              <div class="update-buttons">
                <button v-if="updateInfo && updateInfo.updateAvailable" class="btn btn-default btn-sm" :disabled="updating" @click="downloadUpdate">
                  {{ updating ? '更新中...' : '立即更新' }}
                </button>
                <button class="btn btn-outline btn-sm" :disabled="checkingUpdate || updating" @click="checkForUpdate(false)">
                  {{ checkingUpdate ? '检查中...' : '检查更新' }}
                </button>
              </div>
            </div>
          </div>
        </div>

        <div class="settings-actions">
          <div class="settings-buttons">
            <button class="btn btn-outline" @click="resetSettings">重置</button>
            <button class="btn btn-default" @click="saveSettings">保存设置</button>
          </div>
        </div>
      </div>

      <div v-if="message" :class="['toast', `toast-${message.type}`]">
        <span class="toast-dot"></span>
        {{ message.text }}
      </div>

      <!-- 手工添加客户端对话框 -->
      <div v-if="showManualDialog" class="modal-overlay" @click.self="closeManualAddDialog">
        <div class="modal-dialog">
          <div class="modal-header">
            <h3 class="modal-title">手工添加客户端</h3>
            <button class="btn btn-ghost btn-icon" @click="closeManualAddDialog">
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <line x1="18" y1="6" x2="6" y2="18"></line>
                <line x1="6" y1="6" x2="18" y2="18"></line>
              </svg>
            </button>
          </div>
          <div class="modal-body">
            <div class="form-group">
              <label class="form-label">名称</label>
              <input v-model="manualForm.name" type="text" class="input" placeholder="例如: test" />
            </div>
            <div class="form-group">
              <label class="form-label">连接地址 <span class="required">*</span></label>
              <input v-model="manualForm.addr" type="text" class="input" placeholder="例如: 127.0.0.1:8024" />
            </div>
            <div class="form-group">
              <label class="form-label">密钥 <span class="required">*</span></label>
              <input v-model="manualForm.key" type="text" class="input" placeholder="例如: 6237ed8d52" />
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
            <button class="btn btn-outline" @click="closeManualAddDialog">取消</button>
            <button class="btn btn-default" @click="submitManualAdd">确定</button>
          </div>
        </div>
      </div>

      <!-- 确认对话框 -->
      <div v-if="confirmState.show" class="modal-overlay" @click.self="confirmCancel">
        <div class="modal-dialog modal-dialog-sm">
          <div class="modal-header">
            <h3 class="modal-title">确认</h3>
            <button class="btn btn-ghost btn-icon" @click="confirmCancel">
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <line x1="18" y1="6" x2="6" y2="18"></line>
                <line x1="6" y1="6" x2="18" y2="18"></line>
              </svg>
            </button>
          </div>
          <div class="modal-body">
            <div class="confirm-text">{{ confirmState.text }}</div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-outline" @click="confirmCancel">取消</button>
            <button class="btn btn-default" @click="confirmOk">确定</button>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<script>
import { ref, onMounted, computed, watch, nextTick } from 'vue'
// mygo generate 生成的类型化客户端
import { App as AppAPI } from './mygo'
import { isMyGo } from 'mygo-runtime'

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

    // Update / upgrade check
    const updateInfo = ref(null)
    const checkingUpdate = ref(false)
    const updating = ref(false)
    const updateBannerDismissed = ref(false)

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

    // 从生成的客户端获取 API（使用 let 以便在浏览器中可替换为 mock）
    let GetShortcuts = AppAPI.getShortcuts
    let AddShortcut = AppAPI.addShortcut
    let AddShortcutFromBase64 = AppAPI.addShortcutFromBase64
    let RemoveShortcut = AppAPI.removeShortcut
    let ToggleClient = AppAPI.toggleClient
    let GetConnectionLogs = AppAPI.getConnectionLogs
    let ClearConnectionLogs = AppAPI.clearConnectionLogs

    // 在普通浏览器里运行时没有 mygo 运行时，提供简单 mock 方便调试 UI
    let GetGuiSettings = AppAPI.getGuiSettings
    let SaveGuiSettings = AppAPI.saveGuiSettings
    let GetClientStates = AppAPI.getClientStates
    let SaveClientStates = AppAPI.saveClientStates
    let SelectDirectory = AppAPI.selectDirectory
    let GetDefaultLogDir = AppAPI.getDefaultLogDir
    let GetAppVersion = AppAPI.getAppVersion
    let CheckForUpdate = AppAPI.checkForUpdate
    let DownloadAndInstallUpdate = AppAPI.downloadAndInstallUpdate
    let RestartApp = AppAPI.restartApp

    if (!isMyGo()) {
      console.warn('MyGo runtime not available — using mock implementations for browser debugging')
      GetShortcuts = async () => {
        return [
          { name: 'MyServer', addr: '127.0.0.1:8024', key: 'alefa114df', tls: false, running: false },
        ]
      }
      AddShortcut = async (sc) => {
        console.log('mock AddShortcut', sc)
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
      CheckForUpdate = async () => { console.log('mock CheckForUpdate'); return { currentVersion: 'dev', latestVersion: 'dev', updateAvailable: false, releaseNotes: '', publishedAt: '', downloadUrl: '', assetName: '' } }
      DownloadAndInstallUpdate = async () => { console.log('mock DownloadAndInstallUpdate') }
      RestartApp = async () => { console.log('mock RestartApp') }
    }

    const initBackend = async () => {
      try {
        console.log('MyGo backend loaded successfully')
        await loadClients()
      } catch (error) {
        console.error('Failed to initialize backend:', error)
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
      
      // Handle mygo CallError structure
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

        // 调用 AddShortcut API，直接传递 ShortClient 对象
        await AddShortcut(shortClient)

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

    const checkForUpdate = async (silent = true) => {
      if (checkingUpdate.value) return null
      checkingUpdate.value = true
      try {
        const info = await CheckForUpdate()
        updateInfo.value = info || null
        // 手动检查（非静默）时给出明确反馈
        if (!silent) {
          if (info && info.updateAvailable) {
            showMessage(`发现新版本 v${info.latestVersion}`, 'success')
          } else {
            showMessage('当前已是最新版本', 'success')
          }
        }
        return info || null
      } catch (error) {
        console.error('检查更新失败:', error)
        if (!silent) {
          showMessage('检查更新失败: ' + extractErrorMessage(error), 'error')
        }
        return null
      } finally {
        checkingUpdate.value = false
      }
    }

    const downloadUpdate = async () => {
      if (!updateInfo.value || !updateInfo.value.updateAvailable) {
        showMessage('当前没有可用的更新', 'info')
        return
      }
      const confirmed = await confirmDialog(
        `确定要下载并安装新版本 v${updateInfo.value.latestVersion} 吗？\n更新完成后将自动重启程序生效。`
      )
      if (!confirmed) return

      updating.value = true
      try {
        await DownloadAndInstallUpdate()
        showMessage('更新成功，正在重启程序...', 'success')
        setTimeout(async () => {
          try {
            await RestartApp()
          } catch (e) {
            console.error('重启失败:', e)
          }
        }, 800)
      } catch (error) {
        console.error('更新失败:', error)
        showMessage('更新失败: ' + extractErrorMessage(error), 'error')
      } finally {
        updating.value = false
      }
    }

    const getStatusLabel = (status) => {
      switch (status) {
        case 'connected':
          return '已连接'
        case 'connecting':
          return '连接中'
        case 'stopped':
        default:
          return '已停止'
      }
    }

    const statusBadgeClass = (status) => {
      switch (status) {
        case 'connected':
          return 'badge-success'
        case 'connecting':
          return 'badge-warning'
        default:
          return 'badge-muted'
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

      // 初始化后端
      initBackend()

      // 加载版本号（如果后端已绑定）
      await loadAppVersion()

      // 启动时自动检测升级（后台静默检查）
      checkForUpdate(true)

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
      // update
      updateInfo,
      checkingUpdate,
      updating,
      updateBannerDismissed,
      checkForUpdate,
      downloadUpdate,
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
      statusBadgeClass,
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
/* ============ shadcn-ui 风格设计令牌 ============ */
:root {
  /* 暗色主题（默认）— zinc 色调 */
  --background: hsl(240 10% 3.9%);
  --foreground: hsl(0 0% 98%);
  --card: hsl(240 10% 3.9%);
  --card-foreground: hsl(0 0% 98%);
  --popover: hsl(240 10% 3.9%);
  --popover-foreground: hsl(0 0% 98%);
  --primary: hsl(0 0% 98%);
  --primary-foreground: hsl(240 5.9% 10%);
  --secondary: hsl(240 3.7% 15.9%);
  --secondary-foreground: hsl(0 0% 98%);
  --muted: hsl(240 3.7% 15.9%);
  --muted-foreground: hsl(240 5% 64.9%);
  --accent: hsl(240 3.7% 15.9%);
  --accent-foreground: hsl(0 0% 98%);
  --destructive: hsl(0 72.2% 50.6%);
  --destructive-foreground: hsl(0 0% 98%);
  --success: hsl(142.1 70.6% 45.3%);
  --warning: hsl(38 92% 50%);
  --success-soft: hsla(142.1 70.6% 45.3% / 0.14);
  --warning-soft: hsla(38 92% 50% / 0.14);
  --destructive-soft: hsla(0 72.2% 50.6% / 0.14);
  --border: hsl(240 3.7% 15.9%);
  --input: hsl(240 3.7% 15.9%);
  --ring: hsl(240 4.9% 83.9%);
  --radius: 0.5rem;
  --shadow-sm: 0 1px 2px 0 rgb(0 0 0 / 0.25);
  --shadow-md: 0 4px 12px -2px rgb(0 0 0 / 0.35);
  --shadow-lg: 0 16px 40px -8px rgb(0 0 0 / 0.5);
  --font-mono: ui-monospace, 'SF Mono', 'Cascadia Code', 'Monaco', 'Courier New', monospace;
}

/* 亮色主题（显式） */
[data-theme="light"] {
  --background: hsl(0 0% 100%);
  --foreground: hsl(240 10% 3.9%);
  --card: hsl(0 0% 100%);
  --card-foreground: hsl(240 10% 3.9%);
  --popover: hsl(0 0% 100%);
  --popover-foreground: hsl(240 10% 3.9%);
  --primary: hsl(240 5.9% 10%);
  --primary-foreground: hsl(0 0% 98%);
  --secondary: hsl(240 4.8% 95.9%);
  --secondary-foreground: hsl(240 5.9% 10%);
  --muted: hsl(240 4.8% 95.9%);
  --muted-foreground: hsl(240 3.8% 46.1%);
  --accent: hsl(240 4.8% 95.9%);
  --accent-foreground: hsl(240 5.9% 10%);
  --destructive: hsl(0 84.2% 60.2%);
  --destructive-foreground: hsl(0 0% 98%);
  --success: hsl(142.1 76.2% 36.3%);
  --warning: hsl(24.6 95% 53.1%);
  --success-soft: hsla(142.1 76.2% 36.3% / 0.1);
  --warning-soft: hsla(24.6 95% 53.1% / 0.1);
  --destructive-soft: hsla(0 84.2% 60.2% / 0.1);
  --border: hsl(240 5.9% 90%);
  --input: hsl(240 5.9% 90%);
  --ring: hsl(240 5.9% 10%);
  --shadow-sm: 0 1px 2px 0 rgb(0 0 0 / 0.05);
  --shadow-md: 0 4px 12px -2px rgb(0 0 0 / 0.1);
  --shadow-lg: 0 16px 40px -8px rgb(0 0 0 / 0.2);
}

/* 亮色主题（跟随系统） */
@media (prefers-color-scheme: light) {
  :root:not([data-theme]) {
    --background: hsl(0 0% 100%);
    --foreground: hsl(240 10% 3.9%);
    --card: hsl(0 0% 100%);
    --card-foreground: hsl(240 10% 3.9%);
    --popover: hsl(0 0% 100%);
    --popover-foreground: hsl(240 10% 3.9%);
    --primary: hsl(240 5.9% 10%);
    --primary-foreground: hsl(0 0% 98%);
    --secondary: hsl(240 4.8% 95.9%);
    --secondary-foreground: hsl(240 5.9% 10%);
    --muted: hsl(240 4.8% 95.9%);
    --muted-foreground: hsl(240 3.8% 46.1%);
    --accent: hsl(240 4.8% 95.9%);
    --accent-foreground: hsl(240 5.9% 10%);
    --destructive: hsl(0 84.2% 60.2%);
    --destructive-foreground: hsl(0 0% 98%);
    --success: hsl(142.1 76.2% 36.3%);
    --warning: hsl(24.6 95% 53.1%);
    --success-soft: hsla(142.1 76.2% 36.3% / 0.1);
    --warning-soft: hsla(24.6 95% 53.1% / 0.1);
    --destructive-soft: hsla(0 84.2% 60.2% / 0.1);
    --border: hsl(240 5.9% 90%);
    --input: hsl(240 5.9% 90%);
    --ring: hsl(240 5.9% 10%);
    --shadow-sm: 0 1px 2px 0 rgb(0 0 0 / 0.05);
    --shadow-md: 0 4px 12px -2px rgb(0 0 0 / 0.1);
    --shadow-lg: 0 16px 40px -8px rgb(0 0 0 / 0.2);
  }
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html,
body {
  height: 100%;
  overflow: hidden;
}

body {
  font-family: ui-sans-serif, system-ui, -apple-system, 'Segoe UI', Roboto, 'PingFang SC',
    'Hiragino Sans GB', 'Microsoft YaHei', sans-serif;
  font-size: 14px;
  line-height: 1.5;
  -webkit-font-smoothing: antialiased;
}

.app {
  display: flex;
  height: 100vh;
  background: var(--background);
  color: var(--foreground);
}

/* ============ Sidebar ============ */
.sidebar {
  width: 180px;
  flex-shrink: 0;
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  padding: 16px 10px;
  gap: 8px;
}

.sidebar-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px 18px;
}

.brand-mark {
  width: 32px;
  height: 32px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--primary);
  color: var(--primary-foreground);
  box-shadow: var(--shadow-sm);
}

.brand-text {
  display: flex;
  flex-direction: column;
  line-height: 1.25;
}

.brand-title {
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.brand-sub {
  font-size: 11px;
  color: var(--muted-foreground);
}

.sidebar-nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  height: 36px;
  padding: 0 12px;
  border: none;
  border-radius: var(--radius);
  background: transparent;
  color: var(--muted-foreground);
  font-size: 13.5px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}

.nav-item svg {
  flex-shrink: 0;
  opacity: 0.85;
}

.nav-item:hover {
  background: var(--accent);
  color: var(--accent-foreground);
}

.nav-item.active {
  background: var(--secondary);
  color: var(--secondary-foreground);
  font-weight: 600;
}

.nav-item.active svg {
  opacity: 1;
}

.sidebar-footer {
  margin-top: auto;
  padding: 8px 10px 4px;
}

.version-tag {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  border: 1px solid var(--border);
  border-radius: 9999px;
  font-size: 11px;
  color: var(--muted-foreground);
  font-family: var(--font-mono);
}

/* ============ Main ============ */
.main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  position: relative;
}

.view {
  flex: 1;
  overflow-y: auto;
  padding: 28px 32px 40px;
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 20px;
}

.page-title {
  font-size: 22px;
  font-weight: 700;
  letter-spacing: -0.02em;
  margin-bottom: 4px;
}

.page-desc {
  font-size: 13.5px;
  color: var(--muted-foreground);
}

/* ============ Buttons ============ */
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  height: 36px;
  padding: 0 16px;
  border: 1px solid transparent;
  border-radius: var(--radius);
  font-size: 13.5px;
  font-weight: 500;
  font-family: inherit;
  white-space: nowrap;
  cursor: pointer;
  user-select: none;
  transition: background 0.15s ease, color 0.15s ease, border-color 0.15s ease, opacity 0.15s ease;
}

.btn:active {
  transform: translateY(0.5px);
}

.btn:disabled {
  opacity: 0.5;
  pointer-events: none;
}

.btn-default {
  background: var(--primary);
  color: var(--primary-foreground);
  box-shadow: var(--shadow-sm);
}

.btn-default:hover {
  opacity: 0.9;
}

.btn-outline {
  background: transparent;
  border-color: var(--border);
  color: var(--foreground);
}

.btn-outline:hover {
  background: var(--accent);
  color: var(--accent-foreground);
}

.btn-ghost {
  background: transparent;
  color: var(--muted-foreground);
}

.btn-ghost:hover {
  background: var(--accent);
  color: var(--accent-foreground);
}

.btn-icon {
  width: 32px;
  height: 32px;
  padding: 0;
  flex-shrink: 0;
}

.btn-icon svg {
  flex-shrink: 0;
}

.btn-sm {
  height: 30px;
  padding: 0 12px;
  font-size: 12.5px;
}

/* ============ Update Banner ============ */
.update-banner {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  margin-bottom: 20px;
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: calc(var(--radius) + 2px);
  box-shadow: var(--shadow-sm);
}

.update-banner-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--success-soft);
  color: var(--success);
  flex-shrink: 0;
}

.update-banner-body {
  flex: 1;
  min-width: 0;
}

.update-banner-title {
  font-size: 14px;
  font-weight: 600;
}

.update-banner-desc {
  font-size: 12.5px;
  color: var(--muted-foreground);
  margin-top: 2px;
}

.update-banner-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

/* ============ Input / Select ============ */
.input,
.select {
  height: 36px;
  padding: 0 12px;
  border: 1px solid var(--input);
  border-radius: var(--radius);
  background: transparent;
  color: var(--foreground);
  font-size: 13.5px;
  font-family: inherit;
  transition: box-shadow 0.15s ease, border-color 0.15s ease;
}

.input::placeholder {
  color: var(--muted-foreground);
}

.input:focus-visible,
.select:focus-visible {
  outline: none;
  border-color: var(--ring);
  box-shadow: 0 0 0 1px var(--ring);
}

.select {
  cursor: pointer;
  min-width: 0;
}

.select option {
  background: var(--popover);
  color: var(--popover-foreground);
}

/* ============ Card ============ */
.card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: calc(var(--radius) + 2px);
  box-shadow: var(--shadow-sm);
}

/* ============ Toolbar ============ */
.toolbar {
  display: flex;
  gap: 10px;
  margin-bottom: 24px;
}

.toolbar-input {
  flex: 1;
  min-width: 0;
}

/* ============ Clients Grid ============ */
.clients-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}

.empty-state {
  grid-column: 1 / -1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 72px 24px;
  border: 1px dashed var(--border);
  border-radius: calc(var(--radius) + 2px);
  color: var(--muted-foreground);
  text-align: center;
}

.empty-state svg {
  opacity: 0.5;
  margin-bottom: 4px;
}

.empty-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--foreground);
}

.empty-desc {
  font-size: 13px;
  color: var(--muted-foreground);
}

.client-card {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.client-card:hover {
  border-color: var(--ring);
  box-shadow: var(--shadow-md);
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--border);
}

.card-title-group {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.card-title {
  font-size: 14.5px;
  font-weight: 600;
  letter-spacing: -0.01em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-body {
  flex: 1;
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.info-row {
  display: flex;
  align-items: baseline;
  gap: 10px;
  font-size: 13px;
}

.label {
  color: var(--muted-foreground);
  min-width: 42px;
  flex-shrink: 0;
}

.value {
  color: var(--foreground);
  word-break: break-all;
  flex: 1;
}

.value.mono {
  font-family: var(--font-mono);
  font-size: 12.5px;
  background: var(--muted);
  padding: 2px 8px;
  border-radius: 6px;
  line-height: 1.7;
}

.error-row {
  margin-top: 2px;
}

.error-text {
  color: var(--destructive);
}

.card-footer {
  padding: 12px 16px;
  border-top: 1px solid var(--border);
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

.status-error {
  flex: 1;
  margin-right: 12px;
  font-size: 12px;
  color: var(--destructive);
  word-break: break-all;
}

/* ============ Badge ============ */
.badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 22px;
  padding: 0 10px;
  border: 1px solid transparent;
  border-radius: 9999px;
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
}

.badge-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}

.badge-success {
  background: var(--success-soft);
  color: var(--success);
}

.badge-warning {
  background: var(--warning-soft);
  color: var(--warning);
}

.badge-muted {
  background: var(--muted);
  color: var(--muted-foreground);
}

/* ============ Switch ============ */
.switch {
  position: relative;
  display: inline-block;
  width: 40px;
  height: 22px;
  border-radius: 9999px;
  background: var(--input);
  border: 1px solid var(--border);
  cursor: pointer;
  flex-shrink: 0;
  transition: background 0.2s ease, border-color 0.2s ease;
}

.switch input {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  margin: 0;
  opacity: 0;
  cursor: pointer;
}

.switch-thumb {
  position: absolute;
  top: 50%;
  left: 2px;
  width: 16px;
  height: 16px;
  border-radius: 9999px;
  background: var(--background);
  box-shadow: 0 1px 2px rgb(0 0 0 / 0.3);
  transform: translateY(-50%);
  transition: left 0.2s ease, background 0.2s ease;
  pointer-events: none;
}

.switch input:checked + .switch-thumb {
  left: 20px;
  background: var(--primary-foreground);
}

.switch:has(input:checked) {
  background: var(--primary);
  border-color: var(--primary);
}

/* ============ Logs View ============ */
.logs-view {
  display: flex;
  flex-direction: column;
}

.logs-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 420px;
  overflow: hidden;
}

.logs-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
}

.logs-select {
  flex: 1;
  max-width: 360px;
}

.logs-actions {
  margin-left: auto;
  display: flex;
  gap: 8px;
}

.logs-container {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  background: var(--background);
}

.log-content {
  flex: 1;
  padding: 14px 16px;
  font-family: var(--font-mono);
  font-size: 12.5px;
  overflow-y: auto;
  word-break: break-all;
  white-space: pre-wrap;
}

.empty-logs {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--muted-foreground);
  font-style: italic;
}

.log-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 3px 0;
  border-radius: 4px;
}

.log-item:hover {
  background: var(--accent);
}

.log-dot {
  flex-shrink: 0;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  margin-top: 6px;
  background: var(--muted-foreground);
}

.log-timestamp {
  color: var(--muted-foreground);
  flex-shrink: 0;
  font-weight: 500;
}

.log-message {
  color: var(--foreground);
  flex: 1;
}

.log-success .log-dot {
  background: var(--success);
}

.log-warning .log-dot {
  background: var(--warning);
}

.log-error .log-dot {
  background: var(--destructive);
}

.log-success .log-timestamp {
  color: var(--success);
}

.log-warning .log-timestamp {
  color: var(--warning);
}

.log-error .log-timestamp {
  color: var(--destructive);
}

.log-success .log-message {
  color: var(--success);
}

.log-warning .log-message {
  color: var(--warning);
}

.log-error .log-message {
  color: var(--destructive);
}

/* ============ Settings View ============ */
.settings-view {}

.settings-card {
  padding: 6px 0;
}

.settings-group {
  padding: 8px 0;
}

.setting-row {
  display: flex;
  align-items: center;
  gap: 24px;
  padding: 14px 24px;
  border-top: 1px solid var(--border);
}

.setting-row:first-of-type {
  border-top: none;
}

.setting-info {
  flex: 1;
  min-width: 0;
}

.setting-label {
  font-size: 14px;
  font-weight: 500;
}

.setting-desc {
  font-size: 12.5px;
  color: var(--muted-foreground);
  margin-top: 2px;
}

.logdir-field {
  display: flex;
  gap: 8px;
  min-width: 0;
}

.logdir-input {
  width: 320px;
}

.update-buttons {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.settings-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  margin-top: 20px;
  padding: 0 4px;
}

.settings-buttons {
  display: flex;
  gap: 10px;
}

/* ============ Toast ============ */
.toast {
  position: fixed;
  bottom: 24px;
  right: 24px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  border-radius: var(--radius);
  background: var(--popover);
  border: 1px solid var(--border);
  box-shadow: var(--shadow-lg);
  font-size: 13.5px;
  z-index: 3000;
  animation: toast-in 0.2s ease;
}

.toast-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.toast-success .toast-dot {
  background: var(--success);
}

.toast-error .toast-dot {
  background: var(--destructive);
}

.toast-info .toast-dot {
  background: var(--ring);
}

@keyframes toast-in {
  from {
    opacity: 0;
    transform: translateY(12px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* ============ Modal ============ */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: hsl(240 10% 3.9% / 0.6);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
  animation: fade-in 0.15s ease;
}

@keyframes fade-in {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.modal-dialog {
  width: 90%;
  max-width: 440px;
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: calc(var(--radius) + 4px);
  box-shadow: var(--shadow-lg);
  animation: modal-in 0.18s ease;
}

.modal-dialog-sm {
  max-width: 360px;
}

@keyframes modal-in {
  from {
    opacity: 0;
    transform: translateY(12px) scale(0.97);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 18px 20px 14px;
}

.modal-title {
  font-size: 16px;
  font-weight: 600;
  letter-spacing: -0.01em;
}

.modal-body {
  padding: 4px 20px 20px;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 14px 20px 18px;
}

.confirm-text {
  font-size: 14px;
  color: var(--foreground);
  white-space: pre-wrap;
  line-height: 1.6;
}

/* ============ Form ============ */
.form-group {
  margin-bottom: 16px;
}

.form-group:last-child {
  margin-bottom: 0;
}

.form-label {
  display: block;
  margin-bottom: 6px;
  font-size: 13px;
  font-weight: 500;
  color: var(--foreground);
}

.form-group .input {
  width: 100%;
}

.required {
  color: var(--destructive);
  margin-left: 2px;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  user-select: none;
  font-size: 13.5px;
  color: var(--foreground);
}

.checkbox-label input[type="checkbox"] {
  width: 16px;
  height: 16px;
  accent-color: var(--primary);
  cursor: pointer;
}

.form-error {
  margin-top: 14px;
  padding: 10px 12px;
  background: var(--destructive-soft);
  border: 1px solid var(--destructive);
  border-radius: var(--radius);
  color: var(--destructive);
  font-size: 13px;
  line-height: 1.5;
}

/* ============ Scrollbar ============ */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: transparent;
}

::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: var(--muted-foreground);
}
</style>
