package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
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

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context)  {}
func (a *App) shutdown(ctx context.Context) {}

func getStoragePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	cfgDir := filepath.Join(dir, "nps")
	_ = os.MkdirAll(cfgDir, 0o755)
	return filepath.Join(cfgDir, "npc_shortcuts.json")
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
	if data, err := json.MarshalIndent(shortcuts, "", "  "); err == nil {
		_ = os.WriteFile(p, data, 0o644)
	}
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
	client.SetTlsEnable(tlsEnable)
	logs.Info("启动 NPC 客户端: server=%s, vkey=%s, tls=%v", server, vkey, tlsEnable)

	for {
		select {
		case <-ctx.Done():
			logs.Info("停止 NPC 客户端: %s", id)
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

		logs.Info("连接服务器: %s", id)

		// 重置连接状态，准备新的连接尝试
		statusMu.Lock()
		clientConnected[id] = false
		statusMu.Unlock()

		rpcClient := client.NewRPClient(server, vkey, connType, "", nil, disconnectTimeout)

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
				logs.Info("Context 已取消，关闭客户端: %s", id)
				rpcClient.Close()
			}
		}()

		rpcClient.Start()

		// 检查 context 是否已取消
		select {
		case <-ctx.Done():
			logs.Info("停止 NPC 客户端: %s", id)
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
				logs.Error("连接服务器失败 (timeout): %s，将自动重连", id)
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
					logs.Info("客户端连接成功: %s", id)
					connected = true
				}
			} else {
				// 如果已连接但现在断开，标记为断开状态
				if connected {
					statusMu.Lock()
					clientConnected[id] = false
					clientErrors[id] = "连接已断开，正在重新连接..."
					statusMu.Unlock()
					logs.Warn("客户端连接已断开: %s", id)
					connected = false
				}
			}
		}
	}
}

// startNpcProcess finds the npc executable and starts it with server/vkey args.

// findNpcBinary 已弃用 - npc 现在作为程序内置组件运行
// (无需查找外部 npc 可执行文件)
func findNpcBinary() string {
	return ""
}

// getStartupError returns structured error info for debugging
func getStartupError(addr, key string, err error) error {
	if err == nil {
		return errors.New("npc 启动失败: 未知错误")
	}
	return errors.New("npc 启动失败: " + err.Error())
}
