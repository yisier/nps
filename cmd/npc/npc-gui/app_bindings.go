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
	Running bool   `json:"running"`
}

var (
	shortcuts   []ShortClient
	shortcutsMu sync.Mutex

	// 改为用 context 管理内置客户端，而不是外部进程
	running           = make(map[string]context.CancelFunc)
	clients           = make(map[string]*client.TRPClient)
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

func (a *App) AddShortcut(arg interface{}) error {
	// accept a map or struct from frontend
	b, err := json.Marshal(arg)
	if err != nil {
		return err
	}
	var sc ShortClient
	if err := json.Unmarshal(b, &sc); err != nil {
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
	defer runningMu.Unlock()
	for i, sc := range shortcuts {
		key := sc.Addr + "|" + sc.Key
		sc.Running = false
		if _, ok := running[key]; ok {
			sc.Running = true
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

func (a *App) TestConnection(input interface{}) (bool, error) {
	s, ok := input.(string)
	if !ok || s == "" {
		return false, errors.New("输入密钥不能为空")
	}
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
		rpcClient := client.NewRPClient(server, vkey, connType, "", nil, disconnectTimeout)

		// 将客户端保存到全局 map，以便在需要时关闭
		runningMu.Lock()
		clients[id] = rpcClient
		runningMu.Unlock()

		// 在后台监听 context 取消事件，以便立即关闭连接
		ctxDone := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				logs.Info("Context 已取消，关闭客户端: %s", id)
				rpcClient.Close()
				close(ctxDone)
			}
		}()

		rpcClient.Start()

		logs.Info("客户端已关闭: %s，将在 5 秒后重新连接", id)

		// 检查 context 是否已取消，以及是否需要重连
		select {
		case <-ctx.Done():
			logs.Info("停止 NPC 客户端: %s", id)
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

// startNpcProcess 已弃用，保留供后向兼容性
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
