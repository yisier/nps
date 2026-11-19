package main

import (
	"log"

	"ehang.io/nps/lib/version"
	"github.com/egoist/mygo"
)

func main() {
	mygo.App.SetVersion(version.VERSION)

	svc := &App{}

	// 标记退出状态：真实的退出（托盘“退出”、系统关机、更新重启）都走
	// OnBeforeQuit，窗口的关闭钩子据此区分“关闭窗口”与“退出程序”。
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) {
		setQuitting()
	})
	mygo.App.OnQuitCanceled(func() {
		clearQuitting()
	})
	mygo.App.OnQuit(func() {
		closeAllLogStores()
	})

	// 注册监听器后，最后一个窗口关闭不会退出应用（托盘常驻）。
	mygo.App.OnWindowAllClosed(func() {})

	// macOS 点击 Dock 图标时重新显示窗口。
	mygo.App.OnActivate(func(hasVisibleWindows bool) {
		if !hasVisibleWindows {
			svc.showMainWindow()
		}
	})

	mygo.App.WhenReady(svc.startup)

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
