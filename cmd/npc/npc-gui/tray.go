package main

import (
	_ "embed"
	"runtime"

	"github.com/egoist/mygo"
)

//go:embed resources/tray.png
var trayIcon []byte

// startTray 初始化系统托盘。
func (a *App) startTray() {
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Label: "显示", Click: func(*mygo.MenuItem, *mygo.Window) {
			a.showMainWindow()
		}},
		{Label: "退出", Click: func(*mygo.MenuItem, *mygo.Window) {
			setQuitting()
			mygo.App.Quit()
		}},
	})

	opts := mygo.TrayOptions{
		ID:      "main",
		Icon:    trayIcon,
		ToolTip: "NPS 客户端",
	}
	if runtime.GOOS == "linux" {
		// Linux 的 AppIndicator 通过菜单响应点击。
		opts.Menu = menu
	}

	tray, err := mygo.NewTray(opts)
	if err != nil {
		return
	}

	if runtime.GOOS == "linux" {
		tray.OnClick(func() {
			a.showMainWindow()
		})
	} else {
		// 保持原有交互：左键单击显示窗口，右键弹出菜单。
		tray.OnClick(func() {
			a.showMainWindow()
		})
		tray.OnRightClick(func() {
			tray.SetMenu(menu)
			tray.PopUpMenu()
			tray.SetMenu(nil)
		})
	}

	a.tray = tray
}

// showMainWindow 显示主窗口
func (a *App) showMainWindow() {
	if isQuitting() || a.window == nil {
		return
	}
	a.window.Show()
	a.window.Focus()
}
