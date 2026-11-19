package main

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"ehang.io/nps/lib/version"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// uiState 是原生界面在主线程上读写的状态。
type uiState struct {
	page int // 0 客户端，1 日志，2 设置

	clients []ShortClient
	command string

	settings      GuiSettings
	settingsReady bool
	themeIndex    int

	clientList ui.ListState

	// 日志
	logs        []ConnectionLog
	logCounts   map[string]int
	logFilter   string
	logFilterID string
	logList     ui.ListState

	// 手工添加
	manualOpen bool
	manualName string
	manualAddr string
	manualKey  string
	manualTLS  bool
	manualErr  string

	// 删除确认
	confirmOpen  bool
	confirmIndex int
	confirmText  string

	// 更新
	updateInfo     *UpdateInfo
	checkingUpdate bool
	updating       bool

	// 一次性提示，下一次绘制时转成 toast
	notice     string
	noticeType string

	version string
}

var themeModes = []string{"auto", "light", "dark"}

// view 是窗口的视图函数：每一帧都从状态重新构建界面。
func (a *App) view(c *ui.Context) {
	t := c.Theme()

	ui.Column(c).Fill().Padding(16).Gap(12).Children(func() {
		// 顶部：标题、版本、更新提示和页签
		ui.Row(c).Gap(10).AlignItems(ui.Baseline).Children(func() {
			ui.Text(c, "NPS 客户端").FontSize(18).Bold()
			ui.Text(c, "v"+a.ui.version).FontSize(12).TextColor(t.TextMuted)
			if info := a.ui.updateInfo; info != nil && info.UpdateAvailable {
				if ui.Button(c, "发现新版本 v"+info.LatestVersion).Clicked() {
					a.ui.page = 2
				}
			}
			ui.Spacer(c)
			if ui.Tabs(c, &a.ui.page, "客户端", "日志", "设置").Changed() {
				a.onPageChanged()
			}
		})

		switch a.ui.page {
		case 0:
			a.clientsView(c)
		case 1:
			a.logsView(c)
		case 2:
			a.settingsView(c)
		}
	})

	a.manualDialog(c)
	a.confirmDelete(c)

	// 后台操作的结果在下一次绘制时提示。
	if a.ui.notice != "" {
		c.AddToast(ui.Toast{Title: a.ui.notice, Type: a.ui.noticeType})
		a.ui.notice, a.ui.noticeType = "", ""
	}
}

// ---------------------------------------------------------------- 设置

func (a *App) loadUISettings() {
	a.ui.version = version.VERSION

	s, err := a.GetGuiSettings()
	if err != nil {
		s = GuiSettings{
			StartupEnabled:      true,
			RememberClientState: true,
			LogDir:              a.GetDefaultLogDir(),
			ThemeMode:           "auto",
		}
	}
	if s.LogDir == "" {
		s.LogDir = a.GetDefaultLogDir()
	}
	if !slices.Contains(themeModes, s.ThemeMode) {
		s.ThemeMode = "auto"
	}
	a.ui.settings = s
	a.ui.themeIndex = slices.Index(themeModes, s.ThemeMode)
	a.ui.settingsReady = true
}

func (a *App) applyThemeMode() {
	switch a.ui.settings.ThemeMode {
	case "light":
		mygo.Theme.SetSource(mygo.ThemeLight)
	case "dark":
		mygo.Theme.SetSource(mygo.ThemeDark)
	default:
		mygo.Theme.SetSource(mygo.ThemeSystem)
	}
}

func (a *App) saveSettings() {
	s := a.ui.settings
	if s.LogDir == "" {
		s.LogDir = a.GetDefaultLogDir()
	}
	a.ui.settings = s
	if err := a.SaveGuiSettings(s); err != nil {
		a.notify("保存设置失败: "+err.Error(), "error")
	}
}

// ---------------------------------------------------------------- 客户端

func (a *App) refreshClients() {
	list, err := a.GetShortcuts()
	if err != nil {
		return
	}
	a.ui.clients = list
}

func (a *App) refreshFromBackend() {
	a.ui.clients = snapshotShortcuts()
	if a.ui.page == 1 {
		a.refreshLogs()
	}
}

func (a *App) refreshLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if isQuitting() || a.window == nil || a.window.IsDestroyed() {
			return
		}
		// 窗口隐藏时不必每秒重绘；显示时 OnShow 会刷新一次。
		if !a.window.IsVisible() {
			continue
		}
		a.window.Update(a.refreshFromBackend)
	}
}

func (a *App) restoreClientStates() {
	if !a.ui.settings.RememberClientState {
		return
	}
	states, err := a.GetClientStates()
	if err != nil || len(states) == 0 {
		return
	}
	for _, cl := range a.ui.clients {
		if states[clientID(cl)] == "connected" {
			_ = a.ToggleClient(cl.Name, cl.Addr, cl.Key, cl.TLS, true)
		}
	}
}

func (a *App) persistClientStates() {
	if !a.ui.settings.RememberClientState {
		return
	}
	states := make(map[string]string, len(a.ui.clients))
	for _, cl := range a.ui.clients {
		states[clientID(cl)] = cl.Status
	}
	_ = a.SaveClientStates(states)
}

func (a *App) clientsView(c *ui.Context) {
	t := c.Theme()

	ui.Row(c).Gap(8).Children(func() {
		input := ui.TextInput(c, &a.ui.command).
			Placeholder("粘贴 Base64 快捷启动命令").
			Label("快捷启动命令")
		if input.Submitted() {
			a.addFromCommand()
		}
		input.Grow(1)
		ui.Button(c, "连接").Disabled(strings.TrimSpace(a.ui.command) == "").OnClick(a.addFromCommand)
		ui.Button(c, "手工添加").OnClick(func() {
			a.ui.manualName = ""
			a.ui.manualAddr = ""
			a.ui.manualKey = ""
			a.ui.manualTLS = false
			a.ui.manualErr = ""
			a.ui.manualOpen = true
		})
	})

	if len(a.ui.clients) == 0 {
		ui.Column(c).Fill().Center().Gap(6).Children(func() {
			ui.Text(c, "暂无客户端").FontSize(16).TextColor(t.TextMuted)
			ui.Text(c, "粘贴 Base64 快捷启动命令并点击连接，或手工添加").FontSize(12).TextColor(t.TextMuted)
		})
		return
	}

	a.ui.clientList.Key = func(i int) any { return clientID(a.ui.clients[i]) }
	a.ui.clientList.Label = func(i int) string { return clientName(a.ui.clients[i]) }
	ui.List(c, &a.ui.clientList, len(a.ui.clients), func(i int) {
		a.clientRow(c, i)
	}).Grow(1).Gap(6)
}

func (a *App) clientRow(c *ui.Context, i int) {
	t := c.Theme()
	cl := a.ui.clients[i]

	statusText, statusColor := statusOf(t, cl.Status)

	c.Key(clientID(cl))
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Padding(12, 14).
		Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
		ui.Column(c).Grow(1).Gap(4).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Baseline).Children(func() {
				ui.Text(c, clientName(cl)).Bold()
				ui.Text(c, "● "+statusText).FontSize(12).TextColor(statusColor)
			})
			ui.Text(c, fmt.Sprintf("%s  ·  %s  ·  TLS %s", cl.Addr, cl.Key, tlsLabel(cl.TLS))).
				FontSize(12).TextColor(t.TextMuted)
			if cl.Error != "" && cl.Status != "stopped" {
				ui.Text(c, cl.Error).FontSize(12).TextColor(t.Warning)
			}
		})

		ui.Button(c, "日志").OnClick(func() {
			a.showLogsFor(cl)
		})
		if cl.Status == "stopped" {
			ui.PrimaryButton(c, "启动").OnClick(func() {
				a.toggleClient(cl, true)
			})
		} else {
			ui.Button(c, "停止").OnClick(func() {
				a.toggleClient(cl, false)
			})
		}
		ui.Button(c, "删除").OnClick(func() {
			a.ui.confirmIndex = i
			a.ui.confirmText = fmt.Sprintf("确定要删除客户端 %q 吗？", clientName(cl))
			a.ui.confirmOpen = true
		})
	})
}

func (a *App) addFromCommand() {
	cmd := strings.TrimSpace(a.ui.command)
	if cmd == "" {
		return
	}
	if err := a.AddShortcutFromBase64(cmd); err != nil {
		a.notify(err.Error(), "error")
		return
	}
	a.ui.command = ""
	a.ui.clients = snapshotShortcuts()
	a.notify("连接已添加", "success")
}

func (a *App) submitManualAdd() {
	name := strings.TrimSpace(a.ui.manualName)
	addr := strings.TrimSpace(a.ui.manualAddr)
	key := strings.TrimSpace(a.ui.manualKey)

	switch {
	case addr == "":
		a.ui.manualErr = "请输入连接地址"
		return
	case key == "":
		a.ui.manualErr = "请输入密钥"
		return
	}

	id := addr + "|" + key
	for _, cl := range a.ui.clients {
		if clientID(cl) == id {
			a.ui.manualErr = "该客户端已存在，不能重复添加"
			return
		}
	}

	if err := a.AddShortcut(ShortClient{Name: name, Addr: addr, Key: key, TLS: a.ui.manualTLS}); err != nil {
		a.ui.manualErr = "添加失败: " + err.Error()
		return
	}
	a.ui.manualOpen = false
	a.ui.clients = snapshotShortcuts()
	a.notify("客户端已添加", "success")
}

func (a *App) toggleClient(cl ShortClient, run bool) {
	if err := a.ToggleClient(cl.Name, cl.Addr, cl.Key, cl.TLS, run); err != nil {
		a.notify(err.Error(), "error")
		return
	}
	a.ui.clients = snapshotShortcuts()
	if run {
		a.notify("已启动", "success")
	} else {
		a.notify("已停止", "success")
	}
	// 稍后刷新一次状态并记住连接状态。
	go func() {
		time.Sleep(600 * time.Millisecond)
		if a.window == nil {
			return
		}
		a.window.Update(func() {
			a.refreshFromBackend()
			a.persistClientStates()
		})
	}()
}

func (a *App) removeClient(i int) {
	if i < 0 || i >= len(a.ui.clients) {
		return
	}
	cl := a.ui.clients[i]
	if err := a.RemoveShortcut(cl.Name, cl.Addr, cl.Key); err != nil {
		a.notify("删除失败: "+err.Error(), "error")
		return
	}
	delete(a.ui.logCounts, clientID(cl))
	a.ui.clients = snapshotShortcuts()
	a.notify("已删除", "success")
}

// ---------------------------------------------------------------- 日志

func (a *App) showLogsFor(cl ShortClient) {
	a.ui.page = 1
	a.ui.logFilter = fmt.Sprintf("%s (%s)", clientName(cl), cl.Addr)
	a.ui.logFilterID = clientID(cl)
	a.resetLogs()
	a.refreshLogs()
}

func (a *App) onPageChanged() {
	if a.ui.page == 1 {
		a.ui.logFilter = "全部客户端"
		a.ui.logFilterID = ""
		a.resetLogs()
		a.refreshLogs()
	}
}

func (a *App) resetLogs() {
	a.ui.logs = nil
	a.ui.logCounts = map[string]int{}
	a.ui.logList = ui.ListState{}
}

func (a *App) logTargets() []string {
	if a.ui.logFilterID != "" {
		return []string{a.ui.logFilterID}
	}
	ids := make([]string, 0, len(a.ui.clients))
	for _, cl := range a.ui.clients {
		ids = append(ids, clientID(cl))
	}
	return ids
}

func (a *App) refreshLogs() {
	if a.ui.logCounts == nil {
		a.ui.logCounts = map[string]int{}
	}
	for _, id := range a.logTargets() {
		snap := getClientLogStore(id).snapshot()
		seen := a.ui.logCounts[id]
		if len(snap) < seen {
			// 日志被清空或文件轮转，移除该客户端的旧日志。
			a.ui.logs = slices.DeleteFunc(a.ui.logs, func(l ConnectionLog) bool {
				return l.ClientId == id
			})
			seen = 0
		}
		if len(snap) > seen {
			a.ui.logs = append(a.ui.logs, snap[seen:]...)
		}
		a.ui.logCounts[id] = len(snap)
	}
	if len(a.ui.logs) > 10000 {
		a.ui.logs = a.ui.logs[len(a.ui.logs)-10000:]
	}
}

func (a *App) logsView(c *ui.Context) {
	t := c.Theme()

	names, ids := a.logOptions()
	if !slices.Contains(names, a.ui.logFilter) {
		a.ui.logFilter = names[0]
		a.ui.logFilterID = ""
		a.resetLogs()
		a.refreshLogs()
	}

	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "连接日志").FontSize(16).Bold()
		ui.Spacer(c)
		if ui.Select(c, &a.ui.logFilter, names).Label("客户端").Width(280).Changed() {
			idx := slices.Index(names, a.ui.logFilter)
			if idx >= 0 {
				a.ui.logFilterID = ids[idx]
			}
			a.resetLogs()
			a.refreshLogs()
		}
		if !a.ui.logList.AtEnd() {
			ui.Button(c, "回到底部").OnClick(func() {
				a.ui.logList.ScrollToEnd()
			})
		}
		ui.Button(c, "清空日志").OnClick(a.clearLogs)
	})

	if len(a.ui.logs) == 0 {
		ui.Column(c).Fill().Center().Children(func() {
			ui.Text(c, "暂无日志记录").TextColor(t.TextMuted)
		})
		return
	}

	a.ui.logList.FollowEnd = true
	ui.List(c, &a.ui.logList, len(a.ui.logs), func(i int) {
		a.logRow(c, a.ui.logs[i])
	}).Grow(1)
}

func (a *App) logOptions() (names, ids []string) {
	names = []string{"全部客户端"}
	ids = []string{""}
	for _, cl := range a.ui.clients {
		names = append(names, fmt.Sprintf("%s (%s)", clientName(cl), cl.Addr))
		ids = append(ids, clientID(cl))
	}
	return names, ids
}

func (a *App) logRow(c *ui.Context, l ConnectionLog) {
	t := c.Theme()
	color := t.Text
	switch l.Type {
	case "error":
		color = t.Danger
	case "warning":
		color = t.Warning
	case "success":
		color = t.Success
	}
	ui.Row(c).FillWidth().Gap(10).PaddingY(2).AlignItems(ui.Start).Children(func() {
		ui.Text(c, l.Timestamp).FontSize(11).TextColor(t.TextMuted)
		ui.Text(c, l.Message).FontSize(12).TextColor(color).Grow(1)
		if a.ui.logFilterID == "" {
			ui.Text(c, a.clientLabel(l.ClientId)).FontSize(11).TextColor(t.TextMuted)
		}
	})
}

func (a *App) clientLabel(id string) string {
	for _, cl := range a.ui.clients {
		if clientID(cl) == id {
			return clientName(cl)
		}
	}
	return id
}

func (a *App) clearLogs() {
	for _, id := range a.logTargets() {
		_ = a.ClearConnectionLogs(id)
	}
	a.resetLogs()
	a.notify("日志已清空", "success")
}

// ---------------------------------------------------------------- 设置页

func (a *App) settingsView(c *ui.Context) {
	t := c.Theme()

	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).MaxWidth(720).Gap(14).Padding(0, 4).Children(func() {
			ui.Text(c, "常规").FontSize(13).Bold().TextColor(t.TextMuted)

			a.settingRow(c, "开机启动", "系统启动时自动运行客户端", func() {
				if ui.Switch(c, &a.ui.settings.StartupEnabled).Label("开机启动").Changed() {
					a.saveSettings()
				}
			})

			a.settingRow(c, "记住客户端状态", "下次启动时自动恢复已连接的客户端", func() {
				if ui.Switch(c, &a.ui.settings.RememberClientState).Label("记住客户端状态").Changed() {
					a.saveSettings()
				}
			})

			ui.Divider(c)

			ui.Text(c, "外观").FontSize(13).Bold().TextColor(t.TextMuted)
			a.settingRow(c, "主题", "选择界面外观风格", func() {
				if ui.Segmented(c, &a.ui.themeIndex, "跟随系统", "亮色", "暗色").Label("主题").Changed() {
					if a.ui.themeIndex >= 0 && a.ui.themeIndex < len(themeModes) {
						a.ui.settings.ThemeMode = themeModes[a.ui.themeIndex]
						a.applyThemeMode()
						a.saveSettings()
					}
				}
			})

			ui.Divider(c)

			ui.Text(c, "日志").FontSize(13).Bold().TextColor(t.TextMuted)
			a.settingRow(c, "日志目录", "日志文件的保存位置", func() {
				ui.Row(c).Gap(8).Width(420).Children(func() {
					ui.TextInput(c, &a.ui.settings.LogDir).ReadOnly(true).Grow(1)
					ui.Button(c, "浏览...").OnClick(a.selectLogDir)
				})
			})

			ui.Divider(c)

			ui.Text(c, "版本").FontSize(13).Bold().TextColor(t.TextMuted)
			ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Column(c).Grow(1).Gap(3).Children(func() {
					ui.Text(c, "v"+a.ui.version)
					ui.Text(c, a.updateDescription()).FontSize(12).TextColor(t.TextMuted)
				})
				if info := a.ui.updateInfo; info != nil && info.UpdateAvailable {
					label := "立即更新"
					if a.ui.updating {
						label = "更新中..."
					}
					ui.PrimaryButton(c, label).Disabled(a.ui.updating).OnClick(a.startUpdate)
				}
				checkLabel := "检查更新"
				if a.ui.checkingUpdate {
					checkLabel = "检查中..."
				}
				ui.Button(c, checkLabel).Disabled(a.ui.checkingUpdate || a.ui.updating).OnClick(func() {
					a.checkUpdate(false)
				})
			})

			if info := a.ui.updateInfo; info != nil && info.ReleaseNotes != "" {
				ui.Column(c).Gap(4).Padding(10).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
					ui.Text(c, "更新说明").FontSize(12).Bold()
					ui.Text(c, info.ReleaseNotes).FontSize(12).TextColor(t.TextMuted)
				})
			}

			ui.Row(c).Justify(ui.End).Children(func() {
				ui.Button(c, "重置为默认值").OnClick(func() {
					a.ui.settings = GuiSettings{
						StartupEnabled:      true,
						RememberClientState: true,
						LogDir:              a.GetDefaultLogDir(),
						ThemeMode:           "auto",
					}
					a.ui.themeIndex = 0
					a.applyThemeMode()
					a.saveSettings()
					a.notify("已重置为默认值", "success")
				})
			})
		})
	})
}

func (a *App) settingRow(c *ui.Context, title, desc string, control func()) {
	t := c.Theme()
	ui.Row(c).FillWidth().Gap(16).AlignItems(ui.Center).PaddingY(6).Children(func() {
		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Text(c, title)
			ui.Text(c, desc).FontSize(12).TextColor(t.TextMuted)
		})
		control()
	})
}

func (a *App) updateDescription() string {
	switch {
	case a.ui.updating:
		return "正在下载更新..."
	case a.ui.checkingUpdate:
		return "正在检查更新..."
	case a.ui.updateInfo != nil && a.ui.updateInfo.UpdateAvailable:
		return fmt.Sprintf("发现新版本 v%s（当前 v%s）",
			a.ui.updateInfo.LatestVersion, a.ui.updateInfo.CurrentVersion)
	case a.ui.updateInfo != nil:
		return "已是最新版本 v" + a.ui.updateInfo.CurrentVersion
	default:
		return "启动时自动检查更新"
	}
}

func (a *App) selectLogDir() {
	go func() {
		dir, err := a.SelectDirectory()
		if a.window == nil {
			return
		}
		a.window.Update(func() {
			if err != nil {
				a.notify("选择目录失败: "+err.Error(), "error")
				return
			}
			if dir == "" {
				return
			}
			a.ui.settings.LogDir = dir
			a.saveSettings()
			a.notify("日志目录已更新", "success")
		})
	}()
}

func (a *App) checkUpdate(silent bool) {
	if a.ui.checkingUpdate || a.ui.updating {
		return
	}
	a.ui.checkingUpdate = true
	go func() {
		info, err := a.CheckForUpdate()
		if a.window == nil {
			return
		}
		a.window.Update(func() {
			a.ui.checkingUpdate = false
			if err != nil {
				if !silent {
					a.notify("检查更新失败: "+err.Error(), "error")
				}
				return
			}
			a.ui.updateInfo = info
			if silent {
				return
			}
			if info != nil && info.UpdateAvailable {
				a.notify("发现新版本 v"+info.LatestVersion, "success")
			} else {
				a.notify("当前已是最新版本", "success")
			}
		})
	}()
}

func (a *App) startUpdate() {
	if a.ui.updating {
		return
	}
	info := a.ui.updateInfo
	if info == nil || !info.UpdateAvailable {
		a.notify("当前没有可用的更新", "info")
		return
	}
	a.ui.updating = true
	go func() {
		err := a.DownloadAndInstallUpdate()
		if a.window == nil {
			return
		}
		a.window.Update(func() {
			a.ui.updating = false
			if err != nil {
				a.notify("更新失败: "+err.Error(), "error")
			}
		})
		if err != nil {
			return
		}
		time.Sleep(800 * time.Millisecond)
		_ = a.RestartApp()
	}()
}

// ---------------------------------------------------------------- 对话框与提示

func (a *App) manualDialog(c *ui.Context) {
	t := c.Theme()
	ui.Modal(c, &a.ui.manualOpen, func() {
		ui.Column(c).Width(420).Gap(12).Children(func() {
			ui.Text(c, "手工添加客户端").FontSize(16).Bold()

			ui.Field(c, "名称", func() {
				ui.TextInput(c, &a.ui.manualName).Placeholder("例如: test").AutoFocus()
			})
			ui.Field(c, "连接地址", func() {
				ui.TextInput(c, &a.ui.manualAddr).Placeholder("例如: 127.0.0.1:8024")
			})
			ui.Field(c, "密钥", func() {
				ui.TextInput(c, &a.ui.manualKey).Placeholder("例如: 6237ed8d52")
			})
			ui.Checkbox(c, &a.ui.manualTLS, "启用 TLS")

			if a.ui.manualErr != "" {
				ui.Text(c, a.ui.manualErr).FontSize(12).TextColor(t.Danger)
			}

			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				ui.Button(c, "取消").OnClick(func() {
					a.ui.manualOpen = false
				})
				ui.PrimaryButton(c, "确定").OnClick(a.submitManualAdd)
			})
		})
	})
}

func (a *App) confirmDelete(c *ui.Context) {
	switch ui.AlertDialog(c, &a.ui.confirmOpen, "删除客户端", a.ui.confirmText, "取消", "删除") {
	case 1:
		a.removeClient(a.ui.confirmIndex)
	}
}

func (a *App) notify(text, kind string) {
	a.ui.notice = text
	a.ui.noticeType = kind
}

// ---------------------------------------------------------------- 小工具

func clientID(cl ShortClient) string {
	return cl.Addr + "|" + cl.Key
}

func clientName(cl ShortClient) string {
	if strings.TrimSpace(cl.Name) != "" {
		return cl.Name
	}
	return cl.Addr
}

func tlsLabel(on bool) string {
	if on {
		return "已启用"
	}
	return "未启用"
}

func statusOf(t *ui.Theme, status string) (string, ui.Color) {
	switch status {
	case "connected":
		return "已连接", t.Success
	case "connecting":
		return "连接中", t.Warning
	default:
		return "已停止", t.TextMuted
	}
}
