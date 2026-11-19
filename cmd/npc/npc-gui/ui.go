package main

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"
	"time"

	"ehang.io/nps/lib/version"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

//go:embed resources/icon-64.png
var windowIcon []byte

// uiState 是原生界面在主线程上读写的状态。
type uiState struct {
	page int // 0 终端，1 日志，2 设置

	clients []ShortClient
	command string

	settings      GuiSettings
	settingsReady bool

	clientList ui.ListState

	// 日志（全局运行日志，与 npc 进程一致）
	logs    []ConnectionLog
	logList ui.ListState

	// 手工添加 / 编辑
	manualOpen   bool
	manualName   string
	manualAddr   string
	manualKey    string
	manualTLS    bool
	manualErr    string
	editingIndex int // -1 表示新增；>=0 表示编辑该客户端

	// 删除确认
	confirmOpen  bool
	confirmIndex int
	confirmText  string

	// 更新
	updateInfo     *UpdateInfo
	checkingUpdate bool
	updating       bool

	// 标题栏主题菜单
	themeMenuOpen bool

	// 一次性提示，下一次绘制时转成 toast
	notice     string
	noticeType string

	version string

	// 标题栏里的应用图标（解码一次）。
	icon *ui.Bitmap
}

var themeModes = []string{"auto", "light", "dark"}

// view 是窗口的视图函数：每一帧都从状态重新构建界面。
func (a *App) view(c *ui.Context) {
	t := c.Theme()
	bar := c.TitleBar()

	ui.Column(c).Fill().Children(func() {
		// 自绘标题栏：颜色完全跟随主题，而不是系统的标题栏。
		if bar.Height > 0 {
			ui.Row(c).FillWidth().Height(bar.Height).
				Padding(0, bar.Right, 0, bar.Left).
				Background(t.Background).
				BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
				// 只有左侧这块区域拖动窗口，右侧按钮才能正常点击。
				ui.Row(c).Grow(1).FillHeight().Padding(0, 10).Gap(8).
					AlignItems(ui.Center).DragWindow().Children(func() {
					if a.ui.icon != nil {
						ui.Image(c, a.ui.icon).Height(16).AspectRatio(1)
					}
					ui.Text(c, "NPS 客户端").FontSize(12).TextColor(t.Text)
				})

				ui.Row(c).Padding(0, 8, 0, 4).Gap(4).AlignItems(ui.Center).Children(func() {
					themeBtn := a.titleButton(c, themeIconFor(a.ui.settings.ThemeMode), "切换主题", false)
					if themeBtn.Clicked() {
						a.ui.themeMenuOpen = !a.ui.themeMenuOpen
					}

					githubBtn := a.titleButton(c, iconGitHub, "GitHub 项目地址", false)
					if githubBtn.Clicked() {
						go func() {
							_ = mygo.Shell.OpenExternal("https://github.com/yisier/nps")
						}()
					}

					settingsBtn := a.titleButton(c, iconGear, "设置", a.ui.page == 2)
					if settingsBtn.Clicked() {
						a.ui.page = 2
					}

					ui.Popover(c, themeBtn, &a.ui.themeMenuOpen, func() {
						ui.Column(c).Width(180).Gap(2).Children(func() {
							a.themeMenuItem(c, "auto", "跟随系统", iconMonitor)
							a.themeMenuItem(c, "light", "亮色模式", iconSun)
							a.themeMenuItem(c, "dark", "暗黑模式", iconMoon)
						})
					})
				})
			})
		}

		ui.Row(c).Grow(1).Children(func() {
			a.sidebar(c)
			ui.Column(c).Grow(1).FillHeight().Padding(24, 28).Gap(18).Children(func() {
				switch a.ui.page {
				case 0:
					a.clientsView(c)
				case 1:
					a.logsView(c)
				case 2:
					a.settingsView(c)
				}
			})
		})
	})

	a.manualDialog(c)
	a.confirmDelete(c)

	// 后台操作的结果在下一次绘制时提示。
	if a.ui.notice != "" {
		c.AddToast(ui.Toast{Title: a.ui.notice, Type: a.ui.noticeType})
		a.ui.notice, a.ui.noticeType = "", ""
	}
}

// ---------------------------------------------------------------- 侧栏与页面标题

func (a *App) sidebar(c *ui.Context) {
	t := c.Theme()
	sidebarBG := t.Background.Mix(t.Text, 0.035)

	ui.Column(c).Width(212).FillHeight().Padding(16, 14).Gap(4).
		Background(sidebarBG).BorderWidth(0, 1, 0, 0).BorderColor(t.Border).Children(func() {
		// 品牌
		ui.Row(c).Gap(10).AlignItems(ui.Center).Padding(4, 6, 20, 6).Children(func() {
			ui.Box(c).Size(38, 38).Radius(10).Background(ui.Hex("#111114")).Center().Children(func() {
				ui.Icon(c, iconLogo).Size(22, 22).TextColor(ui.Hex("#ffffff"))
			})
			ui.Column(c).Gap(1).Children(func() {
				ui.Text(c, "NPS Client").FontSize(16).Bold()
				ui.Text(c, "内网穿透客户端").FontSize(11).TextColor(t.TextMuted)
			})
		})

		a.navItem(c, 0, iconTerminal, "终端")
		a.navItem(c, 1, iconLogs, "日志")

		ui.Spacer(c)

		ui.Box(c).AlignSelf(ui.Start).Padding(4, 10).Radius(999).
			Background(t.Background).Border(1, t.Border).Children(func() {
			ui.Text(c, "v"+a.ui.version).FontSize(11).TextColor(t.TextMuted)
		})
	})
}

func (a *App) navItem(c *ui.Context, page int, icon *ui.SVG, label string) {
	t := c.Theme()

	b := ui.ButtonBase(c).FillWidth().Padding(9, 12).Radius(9).Gap(10).Justify(ui.Start).Label(label)
	if a.ui.page == page {
		b.Background(t.SurfaceHover)
	} else if b.Hovered() {
		b.Background(t.Surface)
	}
	b.Children(func() {
		ui.Icon(c, icon).Size(17, 17).TextColor(t.Text)
		ui.Text(c, label).FontSize(14)
	})
	if b.Clicked() && a.ui.page != page {
		a.ui.page = page
		a.onPageChanged()
	}
}

// titleButton 是标题栏右侧的小图标按钮。
func (a *App) titleButton(c *ui.Context, icon *ui.SVG, tip string, active bool) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Size(30, 28).Radius(6).Center().Label(tip).Tooltip(tip)
	if active {
		b.Background(t.SurfaceHover)
	} else if b.Hovered() {
		b.Background(t.Surface)
	}
	b.Children(func() {
		ui.Icon(c, icon).Size(16, 16).TextColor(t.Text)
	})
	return b
}

// themeMenuItem 是主题菜单中的一行。
func (a *App) themeMenuItem(c *ui.Context, mode, label string, icon *ui.SVG) {
	t := c.Theme()
	selected := a.ui.settings.ThemeMode == mode

	b := ui.ButtonBase(c).FillWidth().Padding(7, 9).Radius(6).Gap(10).Justify(ui.Start).Label(label)
	if b.Hovered() {
		b.Background(t.Surface)
	}
	b.Children(func() {
		ui.Icon(c, icon).Size(16, 16).TextColor(t.Text)
		ui.Text(c, label).FontSize(13).Grow(1)
		if selected {
			ui.Icon(c, iconCheck).Size(15, 15).TextColor(t.Accent)
		}
	})
	if b.Clicked() {
		a.setThemeMode(mode)
	}
}

func (a *App) setThemeMode(mode string) {
	a.ui.themeMenuOpen = false
	if a.ui.settings.ThemeMode == mode {
		return
	}
	a.ui.settings.ThemeMode = mode
	a.applyThemeMode()
	a.saveSettings()
}

func themeIconFor(mode string) *ui.SVG {
	switch mode {
	case "light":
		return iconSun
	case "dark":
		return iconMoon
	default:
		return iconMonitor
	}
}

func (a *App) pageHeader(c *ui.Context, title, desc string) {
	t := c.Theme()
	ui.Column(c).Gap(4).Children(func() {
		ui.Text(c, title).FontSize(26).Bold()
		ui.Text(c, desc).FontSize(13).TextColor(t.TextMuted)
	})
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
	a.syncTitleBar()
}

// syncTitleBar 让窗口控件（最小化/最大化/关闭）跟随应用外观。
func (a *App) syncTitleBar() {
	if a.window == nil {
		return
	}
	dark := false
	switch a.ui.settings.ThemeMode {
	case "dark":
		dark = true
	case "light":
		dark = false
	default:
		dark = mygo.Theme.IsDark()
	}
	if dark {
		a.window.SetWindowButtonsTheme(mygo.ThemeDark)
	} else {
		a.window.SetWindowButtonsTheme(mygo.ThemeLight)
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

	a.pageHeader(c, "终端", "管理你的内网穿透客户端连接")

	// 工具栏
	ui.Row(c).Gap(10).Children(func() {
		input := ui.TextInput(c, &a.ui.command).
			Placeholder("粘贴 Base64 快捷启动命令").
			Label("快捷启动命令").Height(42)
		if input.Submitted() {
			a.addFromCommand()
		}
		input.Grow(1)

		connect := ui.PrimaryButton(c, "连接").
			Disabled(strings.TrimSpace(a.ui.command) == "").
			OnClick(a.addFromCommand)
		connect.Label("连接")
		ui.Button(c, "手工添加").Height(42).OnClick(func() {
			a.ui.manualName = ""
			a.ui.manualAddr = ""
			a.ui.manualKey = ""
			a.ui.manualTLS = false
			a.ui.manualErr = ""
			a.ui.editingIndex = -1
			a.ui.manualOpen = true
		})
	})

	if len(a.ui.clients) == 0 {
		ui.Column(c).Fill().Center().Gap(10).Children(func() {
			ui.Icon(c, iconLogo).Size(42, 42).TextColor(t.TextMuted.Alpha(0.5))
			ui.Text(c, "暂无客户端").FontSize(16).TextColor(t.TextMuted)
			ui.Text(c, "粘贴 Base64 格式的快捷命令并点击连接即可添加").FontSize(12).TextColor(t.TextMuted)
		})
		return
	}

	ui.Scroll(c).Grow(1).Children(func() {
		ui.Grid(c).Columns(2).Gap(16).FillWidth().Children(func() {
			for i := range a.ui.clients {
				a.clientCard(c, i)
			}
		})
	})
}

func (a *App) clientCard(c *ui.Context, i int) {
	t := c.Theme()
	cl := a.ui.clients[i]

	c.Key(clientID(cl))
	ui.Column(c).FillWidth().Radius(12).Border(1, t.Border).Background(t.Background).Children(func() {
		// 卡片头部
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Padding(14, 16).
			BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			ui.Text(c, clientName(cl)).FontSize(16).Bold()
			a.statusPill(c, cl.Status)
			ui.Spacer(c)

			editBtn := ui.ButtonBase(c).Size(26, 26).Radius(6).Center().Label("编辑")
			if editBtn.Hovered() {
				editBtn.Background(t.Surface)
			}
			editBtn.Children(func() {
				ui.Icon(c, iconEdit).Size(14, 14).TextColor(t.TextMuted)
			})
			editBtn.Tooltip("编辑")
			if editBtn.Clicked() {
				a.openEditClient(i)
			}

			closeBtn := ui.ButtonBase(c).Size(26, 26).Radius(6).Center().Label("删除")
			if closeBtn.Hovered() {
				closeBtn.Background(t.Surface)
			}
			closeBtn.Children(func() {
				ui.Icon(c, iconClose).Size(13, 13).TextColor(t.TextMuted)
			})
			closeBtn.Tooltip("删除")
			if closeBtn.Clicked() {
				a.ui.confirmIndex = i
				a.ui.confirmText = fmt.Sprintf("确定要删除客户端 %q 吗？", clientName(cl))
				a.ui.confirmOpen = true
			}
		})

		// 卡片内容
		ui.Column(c).Padding(14, 16).Gap(8).Children(func() {
			a.infoRow(c, "地址", cl.Addr)
			a.infoRow(c, "密钥", cl.Key)
			a.infoRow(c, "TLS", tlsLabel(cl.TLS))
			if cl.Error != "" && cl.Status != "stopped" {
				ui.Text(c, cl.Error).FontSize(12).TextColor(t.Warning)
			}
		})

		// 卡片底部
		ui.Row(c).FillWidth().Padding(10, 16).AlignItems(ui.Center).Gap(10).
			BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
			ui.Spacer(c)
			running := cl.Status != "stopped"
			if ui.Switch(c, &running).Label("连接").Changed() {
				a.toggleClient(cl, running)
			}
		})
	})
}

func (a *App) infoRow(c *ui.Context, label, value string) {
	t := c.Theme()
	ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(13).TextColor(t.TextMuted).Width(42)
		ui.Box(c).Grow(1).Padding(3, 8).Radius(5).Background(t.Surface).Children(func() {
			ui.Text(c, value).FontSize(13)
		})
	})
}

func (a *App) statusPill(c *ui.Context, status string) {
	t := c.Theme()
	label, color := statusOf(t, status)
	ui.Row(c).Gap(6).AlignItems(ui.Center).Padding(3, 9).Radius(999).
		Background(t.Surface).Children(func() {
		ui.Box(c).Size(7, 7).Radius(4).Background(color)
		ui.Text(c, label).FontSize(12).TextColor(t.TextMuted)
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
	for i, cl := range a.ui.clients {
		if i == a.ui.editingIndex {
			continue
		}
		if clientID(cl) == id {
			a.ui.manualErr = "该客户端已存在，不能重复添加"
			return
		}
	}

	updated := ShortClient{Name: name, Addr: addr, Key: key, TLS: a.ui.manualTLS}

	if a.ui.editingIndex >= 0 && a.ui.editingIndex < len(a.ui.clients) {
		old := a.ui.clients[a.ui.editingIndex]
		if err := a.UpdateShortcut(old.Name, old.Addr, old.Key, updated); err != nil {
			a.ui.manualErr = "保存失败: " + err.Error()
			return
		}
		a.ui.manualOpen = false
		a.ui.editingIndex = -1
		a.ui.clients = snapshotShortcuts()
		a.notify("已保存", "success")
		return
	}

	if err := a.AddShortcut(updated); err != nil {
		a.ui.manualErr = "添加失败: " + err.Error()
		return
	}
	a.ui.manualOpen = false
	a.ui.clients = snapshotShortcuts()
	a.notify("客户端已添加", "success")
}

func (a *App) openEditClient(i int) {
	if i < 0 || i >= len(a.ui.clients) {
		return
	}
	cl := a.ui.clients[i]
	a.ui.editingIndex = i
	a.ui.manualName = cl.Name
	a.ui.manualAddr = cl.Addr
	a.ui.manualKey = cl.Key
	a.ui.manualTLS = cl.TLS
	a.ui.manualErr = ""
	a.ui.manualOpen = true
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
	a.ui.clients = snapshotShortcuts()
	a.notify("已删除", "success")
}

// ---------------------------------------------------------------- 日志

func (a *App) onPageChanged() {
	if a.ui.page == 1 {
		a.refreshLogs()
	}
}

// refreshLogs 读取全局运行日志（beego + 标准库 log + 客户端 logger）。
func (a *App) refreshLogs() {
	a.ui.logs = runLogSnapshot()
	if len(a.ui.logs) > 10000 {
		a.ui.logs = a.ui.logs[len(a.ui.logs)-10000:]
	}
}

func (a *App) logsView(c *ui.Context) {
	t := c.Theme()

	a.pageHeader(c, "日志", "与 npc 进程一致的完整运行日志")

	ui.Column(c).Grow(1).Radius(12).Border(1, t.Border).Background(t.Background).Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Padding(12, 16).
			BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			ui.Text(c, fmt.Sprintf("共 %d 条", len(a.ui.logs))).FontSize(12).TextColor(t.TextMuted)
			ui.Spacer(c)
			if !a.ui.logList.AtEnd() {
				ui.Button(c, "回到底部").OnClick(func() {
					a.ui.logList.ScrollToEnd()
				})
			}
			ui.Button(c, "清空日志").OnClick(a.clearLogs)
		})

		if len(a.ui.logs) == 0 {
			ui.Column(c).Grow(1).Center().Children(func() {
				ui.Text(c, "暂无日志记录").TextColor(t.TextMuted)
			})
			return
		}

		a.ui.logList.FollowEnd = true
		ui.List(c, &a.ui.logList, len(a.ui.logs), func(i int) {
			a.logRow(c, a.ui.logs[i])
		}).Grow(1).Padding(8, 12)
	})
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
	ui.Row(c).FillWidth().Gap(9).PaddingY(3).AlignItems(ui.Start).Children(func() {
		ui.Box(c).Size(6, 6).Radius(3).Background(color).Margin(5, 4, 0, 0)
		ui.Text(c, l.Timestamp).FontSize(11).TextColor(t.TextMuted)
		ui.Text(c, l.Message).FontSize(12).TextColor(color).Grow(1)
	})
}

func (a *App) clearLogs() {
	clearRunLogs()
	a.ui.logs = nil
	a.refreshLogs()
	a.notify("日志已清空", "success")
}

// ---------------------------------------------------------------- 设置页

func (a *App) settingsView(c *ui.Context) {
	t := c.Theme()

	a.pageHeader(c, "设置", "配置客户端行为")

	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).MaxWidth(820).Radius(12).Border(1, t.Border).
			Background(t.Background).Children(func() {
			a.settingRow(c, "开机启动", "系统启动时自动运行客户端", func() {
				if ui.Switch(c, &a.ui.settings.StartupEnabled).Label("开机启动").Changed() {
					a.saveSettings()
				}
			})
			ui.Divider(c)
			a.settingRow(c, "记住客户端状态", "下次启动时自动恢复已连接的客户端", func() {
				if ui.Switch(c, &a.ui.settings.RememberClientState).Label("记住客户端状态").Changed() {
					a.saveSettings()
				}
			})
			ui.Divider(c)
			a.settingRow(c, "日志目录", "日志文件的保存位置", func() {
				ui.Row(c).Gap(8).Width(420).Children(func() {
					ui.TextInput(c, &a.ui.settings.LogDir).ReadOnly(true).Grow(1)
					ui.Button(c, "浏览...").OnClick(a.selectLogDir)
				})
			})
			ui.Divider(c)
			a.settingRow(c, "版本", a.updateDescription(), func() {
				ui.Row(c).Gap(8).Children(func() {
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
			})
		})

		if info := a.ui.updateInfo; info != nil && info.ReleaseNotes != "" {
			ui.Column(c).MaxWidth(820).Gap(4).Padding(12, 14).Radius(12).
				Background(t.Surface).Border(1, t.Border).Children(func() {
				ui.Text(c, "更新说明").FontSize(12).Bold()
				ui.Text(c, info.ReleaseNotes).FontSize(12).TextColor(t.TextMuted)
			})
		}

		ui.Row(c).Gap(8).Justify(ui.End).Padding(12, 0).Children(func() {
			ui.Button(c, "重置").OnClick(func() {
				a.ui.settings = GuiSettings{
					StartupEnabled:      true,
					RememberClientState: true,
					LogDir:              a.GetDefaultLogDir(),
					ThemeMode:           "auto",
				}
				a.applyThemeMode()
				a.saveSettings()
				a.notify("已重置为默认值", "success")
			})
			ui.PrimaryButton(c, "保存设置").OnClick(func() {
				a.saveSettings()
				a.notify("设置已保存", "success")
			})
		})
	})
}

func (a *App) settingRow(c *ui.Context, title, desc string, control func()) {
	t := c.Theme()
	ui.Row(c).FillWidth().Gap(16).AlignItems(ui.Center).Padding(14, 16).Children(func() {
		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(14)
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
	title, confirm := "手工添加客户端", "确定"
	if a.ui.editingIndex >= 0 {
		title, confirm = "编辑客户端", "保存"
	}
	ui.Modal(c, &a.ui.manualOpen, func() {
		ui.Column(c).Width(420).Gap(12).Children(func() {
			ui.Text(c, title).FontSize(16).Bold()

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
				ui.PrimaryButton(c, confirm).OnClick(a.submitManualAdd)
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
