package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ehang.io/nps/client"
	_ "github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
)

type App struct{}

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
	clientErrors      = make(map[string]string)          // 存储客户端连接错误信息
	clientConnected   = make(map[string]bool)            // 存储客户端连接状态 (true=connected)
	clientAttempted   = make(map[string]bool)            // 存储客户端是否尝试过连接
	clientLoggers     = make(map[string]*logs.BeeLogger) // 为每个客户端单独管理 logger
	statusMu          sync.Mutex                         // 状态锁
	runningMu         sync.Mutex
	loggerMu          sync.Mutex // 保护 clientLoggers 的并发访问
	disconnectTimeout = 60
	connType          = "tcp"

	// 日志缓存机制，避免频繁读取文件
	logsCacheMu   sync.RWMutex
	logsCache     = make(map[string][]ConnectionLog) // 缓存：clientId -> logs
	logsCacheTime = make(map[string]time.Time)       // 缓存时间：clientId -> time
	logsCacheTTL  = 2 * time.Second                  // 缓存有效期 2 秒
)

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	// 初始化日志系统，同时使用 store logger 和文件 logger
	// store logger 用于内存缓存，文件 logger 用于持久化
	logs.SetLogger("store")

	// 可选：同时输出到文件
	logsPath := getLogsPath()
	// Windows 路径中的反斜杠需要转义
	logFilePath := strings.ReplaceAll(filepath.Join(logsPath, "npc.log"), "\\", "\\\\")
	logs.SetLogger(logs.AdapterFile, `{"filename":"`+logFilePath+`","daily":true,"maxdays":7}`)
	logs.SetLevel(logs.LevelDebug)
}

func getLogsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	cfgDir := filepath.Join(dir, "nps")
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

// initClientLogger 为客户端初始化独立的 logger
func initClientLogger(vkey string) {
	loggerMu.Lock()
	defer loggerMu.Unlock()

	if _, exists := clientLoggers[vkey]; !exists {
		// 为每个客户端创建独立的 BeeLogger
		logger := logs.NewLogger()
		// 设置为 Debug 级别，可以看到 Trace 日志
		logger.SetLevel(logs.LevelDebug)

		// 设置独立的日志文件
		logFilePath := getClientLogFilePath(vkey)
		// Windows 路径中的反斜杠需要转义
		escapedPath := strings.ReplaceAll(logFilePath, "\\", "\\\\")
		logger.SetLogger(logs.AdapterFile, `{"filename":"`+escapedPath+`","daily":true,"maxdays":7}`)

		clientLoggers[vkey] = logger
	}
}

// getClientLogger 获取客户端的 logger
func getClientLogger(id string) *logs.BeeLogger {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	return clientLoggers[id]
}

// closeClientLogger 关闭客户端的 logger
func closeClientLogger(id string) {
	loggerMu.Lock()
	defer loggerMu.Unlock()

	if logger, exists := clientLoggers[id]; exists {
		if logger != nil {
			logger.Close()
		}
		delete(clientLoggers, id)
	}
}

func (a *App) shutdown(ctx context.Context) {}

// 持久化文件结构（向后兼容旧的仅数组格式）
type GuiSettings struct {
	StartupEnabled      bool   `json:"startupEnabled"`
	RememberClientState bool   `json:"rememberClientState"`
	LogDir              string `json:"logDir"`
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
	cfgDir := filepath.Join(dir, "nps")
	_ = os.MkdirAll(cfgDir, 0o755)
	return filepath.Join(cfgDir, "npc_shortcuts.json")
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
		return GuiSettings{StartupEnabled: true, RememberClientState: true, LogDir: getLogsPath()}, nil
	}
	// 合并默认值
	s := store.Settings
	if s.LogDir == "" {
		s.LogDir = getLogsPath()
	}
	// 默认都为 true
	if !s.StartupEnabled && !s.RememberClientState && s.LogDir == "" {
		s.StartupEnabled = true
		s.RememberClientState = true
	}
	return s, nil
}

func (a *App) SaveGuiSettings(s GuiSettings) error {
	shortcutsMu.Lock()
	defer shortcutsMu.Unlock()
	store, _ := loadPersistentStore()
	store.Settings = s
	// 如果 LogDir 为空，填充默认
	if store.Settings.LogDir == "" {
		store.Settings.LogDir = getLogsPath()
	}
	savePersistentStoreLocked(store)
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

func (a *App) AddShortcut(arg string) error {
	// accept a JSON string from frontend
	var sc ShortClient
	if err := json.Unmarshal([]byte(arg), &sc); err != nil {
		return err
	}
	addShortcut(sc)
	return nil
}

func (a *App) AddShortcutFromBase64(s string) error {
	if s == "" {
		return errors.New("empty input")
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return err
	}
	payload := string(b)
	if !strings.HasPrefix(payload, "nps:") {
		return errors.New("invalid shortcut format")
	}
	payload = payload[len("nps:"):]
	parts := strings.Split(payload, "|")
	if len(parts) != 4 {
		return errors.New("invalid shortcut payload")
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
	return res, nil
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

func (a *App) TestConnection(input string) (bool, error) {
	if input == "" {
		return false, errors.New("输入密钥不能为空")
	}
	s := input
	// use environment NPC_SERVER_ADDR if set, fallback to localhost
	server := os.Getenv("NPC_SERVER_ADDR")
	if server == "" {
		server = "127.0.0.1:8024"
	}

	// Check if shortcut already exists
	shortcutsMu.Lock()
	for _, existing := range shortcuts {
		if existing.Addr == server && existing.Key == s {
			shortcutsMu.Unlock()
			return false, errors.New("this command has already been added")
		}
	}
	shortcutsMu.Unlock()

	// persist a shortcut for this local connection
	name := "local-" + time.Now().Format("20060102150405")
	sc := ShortClient{Name: name, Addr: server, Key: s, TLS: false}
	addShortcut(sc)

	// start npc client in goroutine, not as external process
	id := server + "|" + s
	go startNpcClient(id, server, s, false)
	return true, nil
}

func (a *App) ToggleClient(name, addr, key string, tls bool, runningState bool) error {
	id := addr + "|" + key
	logs.Info("ToggleClient called: name=%s, addr=%s, tls=%v, runningState=%v", name, addr, tls, runningState)
	runningMu.Lock()
	defer runningMu.Unlock()
	if runningState {
		// start npc client in goroutine
		if _, ok := running[id]; !ok {
			logs.Info("Starting NPC client: %s", id)
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
			logs.Info("Client already running: %s", id)
		}
	} else {
		// stop the client
		if cancel, ok := running[id]; ok {
			logs.Info("Stopping NPC client: %s", id)
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
			logs.Info("Client not running, nothing to stop: %s", id)
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
	// 为该客户端初始化独立的日志文件
	initClientLogger(id)
	clientLogger := getClientLogger(id)

	client.SetTlsEnable(tlsEnable)
	if clientLogger != nil {
		clientLogger.Info("启动 NPC 客户端: server=%s, vkey=%s, tls=%v", server, vkey, tlsEnable)
	}

	for {
		select {
		case <-ctx.Done():
			if clientLogger != nil {
				clientLogger.Info("停止 NPC 客户端")
			}
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
			// 关闭客户端日志
			closeClientLogger(id)
			return
		default:
		}

		if clientLogger != nil {
			clientLogger.Info("连接服务器")
		}

		// 重置连接状态，准备新的连接尝试
		statusMu.Lock()
		clientConnected[id] = false
		statusMu.Unlock()

		rpcClient := client.NewRPClient(server, vkey, connType, "", nil, disconnectTimeout)

		// 设置客户端的独立 logger
		if clientLogger != nil {
			rpcClient.SetLogger(clientLogger)
		}

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
				if clientLogger != nil {
					clientLogger.Info("Context 已取消，关闭客户端")
				}
				rpcClient.Close()
			}
		}()

		rpcClient.Start()

		// 检查 context 是否已取消
		select {
		case <-ctx.Done():
			if clientLogger != nil {
				clientLogger.Info("停止 NPC 客户端")
			}
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
			// 关闭客户端日志
			closeClientLogger(id)
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

	clientLogger := getClientLogger(id)

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
				if clientLogger != nil {
					clientLogger.Error("连接服务器失败 (timeout)，将自动重连")
				}
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
					if clientLogger != nil {
						clientLogger.Info("客户端连接成功")
					}
					connected = true
				}
			} else {
				// 如果已连接但现在断开，标记为断开状态
				if connected {
					statusMu.Lock()
					clientConnected[id] = false
					clientErrors[id] = "连接已断开，正在重新连接..."
					statusMu.Unlock()
					if clientLogger != nil {
						clientLogger.Warn("客户端连接已断开")
					}
					connected = false
				}
			}
		}
	}
}

// GetConnectionLogs 获取指定客户端的连接日志（从独立日志文件读取）
func (a *App) GetConnectionLogs(clientId string) ([]ConnectionLog, error) {
	// 检查缓存是否有效
	logsCacheMu.RLock()
	if cachedLogs, exists := logsCache[clientId]; exists {
		if cacheTime, ok := logsCacheTime[clientId]; ok {
			if time.Since(cacheTime) < logsCacheTTL {
				// 缓存仍然有效
				logsCacheMu.RUnlock()
				return cachedLogs, nil
			}
		}
	}
	logsCacheMu.RUnlock()

	// 缓存过期或不存在，重新读取日志
	// 读取该客户端的独立日志文件
	clientLogFile := getClientLogFilePath(clientId)

	data, err := os.ReadFile(clientLogFile)
	if err != nil {
		// 如果文件不存在或读取失败，返回空日志
		return []ConnectionLog{}, nil
	}

	logContent := string(data)
	if logContent == "" {
		return []ConnectionLog{}, nil
	}

	var result []ConnectionLog
	lines := strings.Split(logContent, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 判断日志类型
		logType := "info"
		if strings.Contains(line, "[E]") {
			logType = "error"
		} else if strings.Contains(line, "[W]") {
			logType = "warning"
		} else if strings.Contains(line, "成功") || strings.Contains(line, "connected") || strings.Contains(line, "Success") {
			logType = "success"
		}

		// 提取时间戳（日志格式通常是：2026-01-09 11:04:56 或 11:04:56）
		var timestamp string
		timeFields := strings.Fields(line)
		if len(timeFields) >= 2 {
			// 尝试获取前两个字段作为日期和时间
			timestamp = timeFields[0] + " " + timeFields[1]
		} else if len(timeFields) >= 1 {
			timestamp = timeFields[0]
		}

		log := ConnectionLog{
			Timestamp: timestamp,
			Message:   line,
			Type:      logType,
			ClientId:  clientId,
		}

		result = append(result, log)
	}

	// 更新缓存
	logsCacheMu.Lock()
	logsCache[clientId] = result
	logsCacheTime[clientId] = time.Now()
	logsCacheMu.Unlock()

	return result, nil
}

// ClearConnectionLogs 清空指定客户端的连接日志文件
func (a *App) ClearConnectionLogs(clientId string) error {
	// 获取日志文件路径
	clientLogFile := getClientLogFilePath(clientId)

	// 清空日志文件内容（写入空字符串）
	if err := os.WriteFile(clientLogFile, []byte(""), 0o644); err != nil {
		return err
	}

	// 清空缓存
	logsCacheMu.Lock()
	delete(logsCache, clientId)
	delete(logsCacheTime, clientId)
	logsCacheMu.Unlock()

	return nil
}
