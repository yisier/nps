package main

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestRenderScreens 用无窗口渲染生成几张界面截图，便于检查布局。
// 仅在设置 NPC_UI_SNAPSHOT 时运行。
func TestRenderScreens(t *testing.T) {
	dir := os.Getenv("NPC_UI_SNAPSHOT")
	if dir == "" {
		t.Skip("set NPC_UI_SNAPSHOT to write screenshots")
	}

	a := &App{}
	a.ui.version = "0.26.40"
	a.ui.settings = GuiSettings{
		StartupEnabled:      true,
		RememberClientState: true,
		LogDir:              `C:\Users\User\AppData\Roaming\npc\logs`,
		ThemeMode:           "auto",
	}
	a.ui.clients = []ShortClient{
		{Name: "家庭服务器", Addr: "127.0.0.1:8024", Key: "6237ed8d52", Status: "connected", Running: true},
		{Name: "公司测试", Addr: "hn1.example.com:8024", Key: "dd8dc30956", TLS: true, Status: "connecting", Error: "连接服务器失败，正在重新连接..."},
		{Name: "备用节点", Addr: "10.0.0.2:8024", Key: "abc12345", Status: "stopped"},
	}
	a.ui.logs = []ConnectionLog{
		{Timestamp: "2026/11/19 14:24:54", Message: "[I] [control.go:111] the version of client is 0.26.40, the core version of client is 0.26.0,tls enable is false", Type: "info"},
		{Timestamp: "2026/11/19 14:24:54", Message: "[I] [client.go:104] Successful connection with server 127.0.0.1:8024", Type: "success"},
		{Timestamp: "2026/11/19 14:24:55", Message: "[D] [control.go:430] try send test packet to target 127.0.0.1:8024", Type: "info"},
		{Timestamp: "2026/11/19 14:24:55", Message: "[E] [client.go:88] The connection server failed and will be reconnected in five seconds, error dial tcp: connection refused", Type: "error"},
		{Timestamp: "2026/11/19 14:24:56", Message: "[W] [mux.go:252] mux: read session unpack from connection err EOF", Type: "warning"},
	}
	a.ui.updateInfo = &UpdateInfo{
		CurrentVersion: "0.26.40", LatestVersion: "0.26.41", UpdateAvailable: true,
		ReleaseNotes: "修复了若干连接问题。",
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	save := func(name string) {
		img := ui.Render(a.view, 1000, 600, 2)
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
	}

	a.ui.page = 0
	save("clients")

	// 暗色主题下的客户端页，检查配色对比。
	img := ui.Render(func(c *ui.Context) {
		ui.Themed(c, ui.DarkTheme()).Fill().Children(func() {
			a.view(c)
		})
	}, 1000, 600, 2)
	if f, err := os.Create(filepath.Join(dir, "clients-dark.png")); err == nil {
		_ = png.Encode(f, img)
		_ = f.Close()
	}

	a.ui.page = 1
	a.ui.logFilter = "家庭服务器-6237ed8d52"
	a.ui.logFilterID = "127.0.0.1:8024|6237ed8d52"
	save("logs")
	a.ui.logFilter = ""
	a.ui.logFilterID = ""

	a.ui.page = 2
	save("settings")

	a.ui.page = 0
	a.ui.manualOpen = true
	save("manual")
	a.ui.manualOpen = false

	a.ui.editingIndex = 0
	cl := a.ui.clients[0]
	a.ui.manualName, a.ui.manualAddr, a.ui.manualKey = cl.Name, cl.Addr, cl.Key
	a.ui.manualTLS = cl.TLS
	a.ui.manualOpen = true
	save("edit")
	a.ui.manualOpen = false
	a.ui.editingIndex = -1

	a.ui.confirmOpen = true
	a.ui.confirmText = fmt.Sprintf("确定要删除客户端 %q 吗？", "家庭服务器")
	save("confirm")
}
