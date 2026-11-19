package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ehang.io/nps/client"
	"ehang.io/nps/lib/version"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type App struct {
	window *mygo.Window
	tray   *mygo.Tray

	// ui 只在主线程（视图回调与 WhenReady）读写。
	ui uiState
}

// ShortClient 与前端结构对应
type ShortClient struct {
	Name    string `json:"name"`
	Addr    string `json:"addr"`
	Key     string `json:"key"`
	TLS     bool   `json:"tls"`
	Running bool   `json:"running"` // 兼容旧版本，实际用Status
	Error   string `json:"error"`   // 连接错误信息
	Status  string `json:"status"`  // 连接状态: "stopped", "connecting", "connected"
}

// ConnectionLog 连接日志项
type ConnectionLog struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
	Type      string `json:"type"` // "info", "error", "warning", "success"
	ClientId  string `json:"clientId"`
}

var (
	shortcuts   []ShortClient
	shortcutsMu sync.Mutex

	// 改为用 context 管理内置客户端，而不是外部进程
	running           = make(map[string]context.CancelFunc)
	clients           = make(map[string]*client.TRPClient)
	clientErrors      = make(map[string]string) // 存储客户端连接错误信息
	clientConnected   = make(map[string]bool)   // 存储客户端连接状态 (true=connected)
	clientAttempted   = make(map[string]bool)   // 存储客户端是否尝试过连接
	statusMu          sync.Mutex                // 状态锁
	runningMu         sync.Mutex
	disconnectTimeout = 60
	connType          = "tcp"
)

// startup 在应用就绪后创建主窗口并初始化托盘（由 mygo.App.WhenReady 调用）。
func (a *App) startup() {
	initAppLogger()

	// 同步开机启动状态：检查配置与实际系统设置是否一致
	go syncStartupState()

	a.loadUISettings()
	a.ui.editingIndex = -1
	a.ui.icon, _ = ui.DecodeBitmap(windowIcon)
	a.applyThemeMode()
	a.refreshClients()
	a.restoreClientStates()

	a.window = mygo.NewWindow(mygo.WindowOptions{
		Title:          "NPS 客户端",
		Width:          1000,
		Height:         600,
		MinWidth:       1000,
		MinHeight:      600,
		TitleBarStyle:  mygo.TitleBarHidden,
		TitleBarHeight: 40,
		Content:        ui.View(a.view),
	})

	// 点击关闭按钮时隐藏窗口而不是退出（托盘应用常驻）
	a.window.OnClose(func(e *mygo.CloseEvent) {
		if isQuitting() {
			return
		}
		e.PreventDefault()
		a.window.Hide()
	})

	// 标题栏图标 + 标题栏深浅色跟随应用主题。
	_ = a.window.SetIcon(windowIcon)
	a.syncTitleBar()
	mygo.Theme.OnUpdated(func() {
		if a.window == nil || a.window.IsDestroyed() {
			return
		}
		a.syncTitleBar()
		a.window.Invalidate()
	})

	a.startTray()

	// 窗口重新显示时立即刷新一次状态。
	a.window.OnShow(func() {
		a.window.Update(a.refreshFromBackend)
	})

	// 启动时后台静默检查更新。
	a.checkUpdate(true)
	// 定时刷新客户端状态与日志。
	go a.refreshLoop()
}

// syncStartupState 让配置文件里的开机启动设置与系统实际状态保持一致。
func syncStartupState() {
	store, err := loadPersistentStore()
	if err != nil {
		return
	}
	actualEnabled := mygo.App.OpenAtLogin()
	if store.Settings.StartupEnabled == actualEnabled {
		return
	}
	if err := mygo.App.SetOpenAtLogin(store.Settings.StartupEnabled); err != nil {
		slog.Error("同步开机启动失败", "enabled", store.Settings.StartupEnabled, "err", err)
	}
}

func getLogsPath() string {
	// 尝试从配置中读取用户设置的日志目录
	store, err := loadPersistentStore()
	if err == nil && store.Settings.LogDir != "" {
		// 确保目录存在
		_ = os.MkdirAll(store.Settings.LogDir, 0o755)
		return store.Settings.LogDir
	}

	// 如果没有配置或读取失败，使用默认路径
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	cfgDir := filepath.Join(dir, "npc")
	_ = os.MkdirAll(cfgDir, 0o755)
	logsDir := filepath.Join(cfgDir, "logs")
	_ = os.MkdirAll(logsDir, 0o755)
	return logsDir
}

// getClientLogFilePath 获取客户端的独立日志文件路径
func getClientLogFilePath(vkey string) string {
	logsDir := getLogsPath()
	// 如果传入的是包含 addr|vkey 的 id，则提取最后一段作为 vkey
	if strings.Contains(vkey, "|") {
		parts := strings.Split(vkey, "|")
		vkey = parts[len(parts)-1]
	}
	// 进一步替换可能在 vkey 中出现的不适合作为文件名的字符
	vkey = strings.ReplaceAll(vkey, ":", "-")
	vkey = strings.ReplaceAll(vkey, "\\", "-")
	vkey = strings.ReplaceAll(vkey, "/", "-")
	return filepath.Join(logsDir, fmt.Sprintf("npc-client-%s.log", vkey))
}

// 持久化文件结构（向后兼容旧的仅数组格式）
type GuiSettings struct {
	StartupEnabled      bool   `json:"startupEnabled"`
	RememberClientState bool   `json:"rememberClientState"`
	LogDir              string `json:"logDir"`
	ThemeMode           string `json:"themeMode"` // "auto", "light", "dark"
}

type PersistentStore struct {
	Shortcuts    []ShortClient     `json:"shortcuts"`
	Settings     GuiSettings       `json:"settings,omitempty"`
	ClientStates map[string]string `json:"clientStates,omitempty"`
}

func getStoragePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	cfgDir := filepath.Join(dir, "npc")
	_ = os.MkdirAll(cfgDir, 0o755)
	return filepath.Join(cfgDir, "npc_data.json")
}

// 读取持久化 store，向后兼容：如果是数组则解析为 shortcuts
func loadPersistentStore() (PersistentStore, error) {
	p := getStoragePath()
	b, err := os.ReadFile(p)
	if err != nil {
		return PersistentStore{}, err
	}

	// 尝试解析为对象结构
	var store PersistentStore
	if err := json.Unmarshal(b, &store); err == nil {
		// 如果文件是对象但没有 shortcuts 字段，ensure empty slice
		if store.Shortcuts == nil {
			store.Shortcuts = []ShortClient{}
		}
		return store, nil
	}

	// 兼容旧格式：直接是 ShortClient 数组
	var arr []ShortClient
	if err := json.Unmarshal(b, &arr); err == nil {
		return PersistentStore{Shortcuts: arr}, nil
	}

	return PersistentStore{}, errors.New("invalid storage format")
}

// 保存整个 store，保持 settings 与 clientStates
func savePersistentStoreLocked(store PersistentStore) {
	p := getStoragePath()
	if data, err := json.MarshalIndent(store, "", "  "); err == nil {
		_ = os.WriteFile(p, data, 0o644)
	}
}

func loadShortcuts() {
	shortcutsMu.Lock()
	defer shortcutsMu.Unlock()
	p := getStoragePath()
	b, err := os.ReadFile(p)
	if err != nil {
		shortcuts = []ShortClient{}
		return
	}

	// 先尝试解析为对象格式
	var store PersistentStore
	if err := json.Unmarshal(b, &store); err == nil {
		shortcuts = store.Shortcuts
		if shortcuts == nil {
			shortcuts = []ShortClient{}
		}
		return
	}

	// 兼容旧格式：直接是 ShortClient 数组
	var s []ShortClient
	if err := json.Unmarshal(b, &s); err != nil {
		shortcuts = []ShortClient{}
		return
	}
	shortcuts = s
}

func saveShortcuts() {
	shortcutsMu.Lock()
	defer shortcutsMu.Unlock()
	saveShortcutsLocked()
}

func saveShortcutsLocked() {
	p := getStoragePath()

	var store PersistentStore
	// 尝试读取已有 store 以保留 settings/clientStates
	if b, err := os.ReadFile(p); err == nil {
		_ = json.Unmarshal(b, &store) // ignore errors, we'll overwrite shortcuts
	}
	store.Shortcuts = shortcuts
	savePersistentStoreLocked(store)
}

// 以下为对 settings 与 clientStates 的访问方法（供前端调用）
func (a *App) GetGuiSettings() (GuiSettings, error) {
	store, err := loadPersistentStore()
	if err != nil {
		// 返回默认值
		return GuiSettings{StartupEnabled: true, RememberClientState: true, LogDir: getLogsPath(), ThemeMode: "auto"}, nil
	}
	// 合并默认值
	s := store.Settings
	// 如果 LogDir 为空，使用默认路径
	if s.LogDir == "" {
		s.LogDir = getLogsPath()
	}
	// 如果 ThemeMode 为空，使用默认值 "auto"
	if s.ThemeMode == "" {
		s.ThemeMode = "auto"
	}
	// 检测是否为首次使用（配置为空），如果是则使用默认值 true
	// 注意：这里无法区分用户主动设置为 false 还是从未设置过，所以采用保守策略
	// 只有当配置文件存在但 Settings 字段完全为空时才认为是首次使用
	isFirstTime := !s.StartupEnabled && !s.RememberClientState
	if isFirstTime {
		// 首次使用，设置为默认值 true
		s.StartupEnabled = true
		s.RememberClientState = true
	}
	return s, nil
}

func (a *App) SaveGuiSettings(s GuiSettings) error {
	shortcutsMu.Lock()
	defer shortcutsMu.Unlock()
	store, _ := loadPersistentStore()

	// 保存之前检查开机启动设置是否有变化
	oldStartupEnabled := store.Settings.StartupEnabled

	store.Settings = s
	// 如果 LogDir 为空，填充默认
	if store.Settings.LogDir == "" {
		store.Settings.LogDir = getLogsPath()
	}
	savePersistentStoreLocked(store)

	// 如果开机启动设置发生变化，应用到系统
	if oldStartupEnabled != s.StartupEnabled {
		if err := mygo.App.SetOpenAtLogin(s.StartupEnabled); err != nil {
			slog.Error("设置开机启动失败", "enabled", s.StartupEnabled, "err", err)
			return fmt.Errorf("设置开机启动失败: %v", err)
		}
		slog.Info("开机启动设置已应用", "enabled", s.StartupEnabled)
	}

	return nil
}

func (a *App) GetClientStates() (map[string]string, error) {
	store, err := loadPersistentStore()
	if err != nil {
		return map[string]string{}, nil
	}
	if store.ClientStates == nil {
		return map[string]string{}, nil
	}
	return store.ClientStates, nil
}

func (a *App) SaveClientStates(m map[string]string) error {
	shortcutsMu.Lock()
	defer shortcutsMu.Unlock()
	store, _ := loadPersistentStore()
	store.ClientStates = m
	savePersistentStoreLocked(store)
	return nil
}

// GetDefaultLogDir 获取默认日志目录路径
func (a *App) GetDefaultLogDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	cfgDir := filepath.Join(dir, "npc")
	logsDir := filepath.Join(cfgDir, "logs")
	return logsDir
}

// SelectDirectory 打开目录选择对话框
func (a *App) SelectDirectory() (string, error) {
	slog.Info("SelectDirectory 被调用")

	// 获取当前设置的日志目录作为默认目录
	defaultDir := ""
	store, err := loadPersistentStore()
	if err == nil && store.Settings.LogDir != "" {
		defaultDir = store.Settings.LogDir
	}

	slog.Info("默认目录", "dir", defaultDir)

	// 打开目录选择对话框
	selected, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:      a.window,
		Title:       "选择日志目录",
		DefaultPath: defaultDir,
		Directory:   true,
	})
	if err != nil {
		slog.Error("打开目录选择对话框失败", "err", err)
		return "", err
	}
	if len(selected) == 0 {
		return "", nil
	}
	selectedDir := selected[0]

	slog.Info("用户选择的目录", "dir", selectedDir)
	return selectedDir, nil
}

// GetAppVersion 返回来自 lib/version 的版本号
func (a *App) GetAppVersion() string {
	return version.VERSION
}

func addShortcut(sc ShortClient) {
	shortcutsMu.Lock()
	shortcuts = append(shortcuts, sc)
	shortcutsMu.Unlock()
	saveShortcuts()
}

func findShortcutIndex(name, addr, key string) int {
	shortcutsMu.Lock()
	defer shortcutsMu.Unlock()
	for i, it := range shortcuts {
		if it.Name == name && it.Addr == addr && it.Key == key {
			return i
		}
	}
	return -1
}

func (a *App) AddShortcut(sc ShortClient) error {
	addShortcut(sc)
	return nil
}

// UpdateShortcut 修改已有客户端的配置；如果它正在运行，会用新配置自动重启。
func (a *App) UpdateShortcut(oldName, oldAddr, oldKey string, updated ShortClient) error {
	oldID := oldAddr + "|" + oldKey

	shortcutsMu.Lock()
	idx := -1
	for i, it := range shortcuts {
		if it.Name == oldName && it.Addr == oldAddr && it.Key == oldKey {
			idx = i
			continue
		}
		if it.Addr == updated.Addr && it.Key == updated.Key {
			shortcutsMu.Unlock()
			return errors.New("该客户端已存在，不能重复添加")
		}
	}
	if idx == -1 {
		shortcutsMu.Unlock()
		return errors.New("客户端不存在")
	}
	shortcutsMu.Unlock()

	runningMu.Lock()
	_, wasRunning := running[oldID]
	runningMu.Unlock()

	if wasRunning {
		_ = a.ToggleClient(oldName, oldAddr, oldKey, false, false)
	}

	shortcutsMu.Lock()
	idx = -1
	for i, it := range shortcuts {
		if it.Name == oldName && it.Addr == oldAddr && it.Key == oldKey {
			idx = i
			break
		}
	}
	if idx >= 0 {
		shortcuts[idx] = updated
		saveShortcutsLocked()
	}
	shortcutsMu.Unlock()

	if idx < 0 {
		return errors.New("客户端不存在")
	}

	if wasRunning {
		return a.ToggleClient(updated.Name, updated.Addr, updated.Key, updated.TLS, true)
	}
	return nil
}

func (a *App) AddShortcutFromBase64(s string) error {
	if s == "" {
		return errors.New("无效的启动命令")
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return err
	}
	payload := string(b)
	if !strings.HasPrefix(payload, "nps:") {
		return errors.New("无效的启动命令")
	}
	payload = payload[len("nps:"):]
	parts := strings.Split(payload, "|")
	if len(parts) != 4 {
		return errors.New("无效的启动命令")
	}
	tls := false
	if parts[3] == "true" {
		tls = true
	}
	sc := ShortClient{Name: parts[0], Addr: parts[1], Key: parts[2], TLS: tls}

	// Check if shortcut already exists
	shortcutsMu.Lock()
	for _, existing := range shortcuts {
		if existing.Addr == sc.Addr && existing.Key == sc.Key {
			shortcutsMu.Unlock()
			return errors.New("该客户端已被添加")
		}
	}
	shortcutsMu.Unlock()

	addShortcut(sc)
	return nil
}

func (a *App) GetShortcuts() ([]ShortClient, error) {
	loadShortcuts()
	return snapshotShortcuts(), nil
}

// snapshotShortcuts 返回内存中的客户端列表及其运行状态，不重新读取配置文件。
func snapshotShortcuts() []ShortClient {
	shortcutsMu.Lock()
	defer shortcutsMu.Unlock()
	// populate running state
	res := make([]ShortClient, len(shortcuts))
	runningMu.Lock()
	statusMu.Lock()
	defer runningMu.Unlock()
	defer statusMu.Unlock()

	for i, sc := range shortcuts {
		key := sc.Addr + "|" + sc.Key
		sc.Running = false
		sc.Status = "stopped"
		sc.Error = ""

		// 检查客户端是否在 running map 中
		if _, ok := running[key]; ok {
			// 客户端正在运行（或重连中）
			if isConnected, ok := clientConnected[key]; ok && isConnected {
				// 连接成功
				sc.Status = "connected"
				sc.Running = true
			} else if _, attempted := clientAttempted[key]; attempted {
				// 尝试过连接但失败，显示为"连接中"（正在重连）
				sc.Status = "connecting"
				if errMsg, ok := clientErrors[key]; ok && errMsg != "" {
					sc.Error = errMsg
				}
			} else {
				// 刚启动，还未尝试连接
				sc.Status = "connecting"
			}
		}
		res[i] = sc
	}
	return res
}

func (a *App) RemoveShortcut(name, addr, key string) error {
	// stop if running
	id := addr + "|" + key
	runningMu.Lock()
	if cancel, ok := running[id]; ok {
		cancel()
		delete(running, id)
		// 也要关闭客户端
		if rpcClient, ok := clients[id]; ok {
			rpcClient.Close()
			delete(clients, id)
		}
	}
	statusMu.Lock()
	delete(clientErrors, id)
	delete(clientConnected, id)
	delete(clientAttempted, id)
	statusMu.Unlock()
	runningMu.Unlock()

	// remove from slice
	shortcutsMu.Lock()
	defer shortcutsMu.Unlock()
	idx := -1
	for i, it := range shortcuts {
		if it.Name == name && it.Addr == addr && it.Key == key {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil
	}
	shortcuts = append(shortcuts[:idx], shortcuts[idx+1:]...)
	saveShortcutsLocked()
	return nil
}

func (a *App) ToggleClient(name, addr, key string, tls bool, runningState bool) error {
	id := addr + "|" + key
	slog.Info("ToggleClient called", "name", name, "addr", addr, "tls", tls, "runningState", runningState)
	runningMu.Lock()
	defer runningMu.Unlock()
	if runningState {
		// start npc client in goroutine
		if _, ok := running[id]; !ok {
			slog.Info("Starting NPC client", "id", id)
			ctx, cancel := context.WithCancel(context.Background())
			running[id] = cancel
			// 清除之前的状态
			statusMu.Lock()
			delete(clientErrors, id)
			clientConnected[id] = false
			delete(clientAttempted, id)
			statusMu.Unlock()
			go startNpcClientWithContext(ctx, id, addr, key, tls)
		} else {
			slog.Info("Client already running", "id", id)
		}
	} else {
		// stop the client
		if cancel, ok := running[id]; ok {
			slog.Info("Stopping NPC client", "id", id)
			cancel()
			delete(running, id)
			// 清除状态
			statusMu.Lock()
			delete(clientErrors, id)
			delete(clientConnected, id)
			delete(clientAttempted, id)
			statusMu.Unlock()
			// 也要关闭客户端
			if rpcClient, ok := clients[id]; ok {
				rpcClient.Close()
				delete(clients, id)
			}
		} else {
			slog.Info("Client not running, nothing to stop", "id", id)
		}
	}
	return nil
}

// startNpcClient 在 goroutine 中启动 npc 客户端（内置，不是外部进程）
func startNpcClient(id, server, vkey string, tlsEnable bool) {
	ctx, cancel := context.WithCancel(context.Background())
	runningMu.Lock()
	running[id] = cancel
	runningMu.Unlock()

	startNpcClientWithContext(ctx, id, server, vkey, tlsEnable)
}

// startNpcClientWithContext 在给定的 context 中运行 npc 客户端
func startNpcClientWithContext(ctx context.Context, id, server, vkey string, tlsEnable bool) {
	// 为该客户端准备日志存储（内存缓冲 + 独立日志文件）
	store := getClientLogStore(id)
	clientLogger := &trpcLogger{store: store}

	client.SetTlsEnable(tlsEnable)
	clientLogger.Info("the version of client is %s, the core version of client is %s,tls enable is %t", version.VERSION, version.GetVersion(), tlsEnable)
	clientLogger.Info("start vkey:%s", vkey)

	for {
		select {
		case <-ctx.Done():
			statusMu.Lock()
			delete(clientConnected, id)
			delete(clientAttempted, id)
			delete(clientErrors, id)
			statusMu.Unlock()
			runningMu.Lock()
			if rpcClient, ok := clients[id]; ok {
				rpcClient.Close()
				delete(clients, id)
			}
			runningMu.Unlock()
			return
		default:
		}

		// 重置连接状态，准备新的连接尝试
		statusMu.Lock()
		clientConnected[id] = false
		statusMu.Unlock()

		rpcClient := client.NewRPClient(server, vkey, connType, "", nil, disconnectTimeout)

		// 设置客户端的独立 logger
		rpcClient.SetLogger(clientLogger)

		// 将客户端保存到全局 map
		runningMu.Lock()
		clients[id] = rpcClient
		runningMu.Unlock()

		// 启动连接监听器（每次重连都启动）
		go monitorFirstConnection(ctx, id, rpcClient)

		// 在后台监听 context 取消事件
		go func() {
			select {
			case <-ctx.Done():
				rpcClient.Close()
			}
		}()

		rpcClient.Start()

		// 检查 context 是否已取消
		select {
		case <-ctx.Done():
			statusMu.Lock()
			delete(clientConnected, id)
			delete(clientAttempted, id)
			delete(clientErrors, id)
			statusMu.Unlock()
			runningMu.Lock()
			if rpcClient, ok := clients[id]; ok {
				rpcClient.Close()
				delete(clients, id)
			}
			runningMu.Unlock()
			return
		case <-time.After(5 * time.Second):
			// 继续重新连接
		}
	}
}

// monitorFirstConnection 监听连接的结果，持续检查连接状态
func monitorFirstConnection(ctx context.Context, id string, rpcClient *client.TRPClient) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	initialTimeout := time.After(5 * time.Second)
	connected := false

	for {
		select {
		case <-ctx.Done():
			return
		case <-initialTimeout:
			if !connected {
				statusMu.Lock()
				clientAttempted[id] = true // 标记为已尝试
				clientConnected[id] = false
				clientErrors[id] = "连接服务器失败，正在重新连接..."
				statusMu.Unlock()
			}
			// 即使超时，也继续监听连接状态变化
		case <-ticker.C:
			// 检查连接是否成功
			isNowConnected := rpcClient.IsConnected()
			if isNowConnected {
				if !connected {
					statusMu.Lock()
					clientAttempted[id] = true
					clientConnected[id] = true
					delete(clientErrors, id)
					statusMu.Unlock()
					connected = true
				}
			} else {
				// 如果已连接但现在断开，标记为断开状态
				if connected {
					statusMu.Lock()
					clientConnected[id] = false
					clientErrors[id] = "连接已断开，正在重新连接..."
					statusMu.Unlock()
					connected = false
				}
			}
		}
	}
}

// GetConnectionLogs 获取指定客户端的连接日志（直接读内存缓冲，无需解析文件）
func (a *App) GetConnectionLogs(clientId string) ([]ConnectionLog, error) {
	return getClientLogStore(clientId).snapshot(), nil
}

// ClearConnectionLogs 清空指定客户端的连接日志（内存缓冲与日志文件）
func (a *App) ClearConnectionLogs(clientId string) error {
	return getClientLogStore(clientId).clear()
}

// getExecutablePath 获取当前可执行文件的完整路径
func getExecutablePath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	// 解析符号链接，获取真实路径
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return "", err
	}
	return exePath, nil
}
