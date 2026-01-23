//go:build !darwin

package main

import (
	_ "image/png"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/energye/systray"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// QuitTray 退出托盘程序
func QuitTray() {
	systray.Quit()
}

// startTray 初始化系统托盘
func (a *App) startTray() {
	// 锁定 OS 线程，确保 Windows 消息循环稳定运行
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	log.Println("systray: start")

	// 使用 sync.WaitGroup 确保资源正确释放
	var wg sync.WaitGroup
	wg.Add(1)

	systray.Run(
		func() {
			// onReady 回调 - 在主线程执行
			log.Println("systray: onReady start")
			defer wg.Done()

			// 增加短暂延迟确保系统托盘完全初始化
			time.Sleep(200 * time.Millisecond)

			// 设置托盘图标 - 增加错误处理
			var iconSet bool
			if runtime.GOOS == "windows" {
				if len(trayIconICO) > 0 {
					log.Println("systray: using embedded ICO icon")
					systray.SetIcon(trayIconICO)
					iconSet = true
				}
			} else {
				log.Println("systray: using PNG icon for non-windows")
				if len(trayIcon) > 0 {
					systray.SetIcon(trayIcon)
					iconSet = true
				}
			}

			if !iconSet {
				log.Println("systray: no icon provided, using default")
			}

			systray.SetTitle("NPS 客户端")
			systray.SetTooltip("NPS 客户端")

			// 创建菜单项 - 确保在主线程创建
			showItem := systray.AddMenuItem("显示", "点击显示")
			quitItem := systray.AddMenuItem("退出", "点击退出")

			log.Println("systray: menu items added")

			systray.SetOnClick(func(menu systray.IMenu) {
				_ = menu
				log.Println("[Tray] Left click")
				if a.ctx != nil && !isQuitting() {
					wailsRuntime.EventsEmit(a.ctx, "tray-show")
					go func() {
						wailsRuntime.Show(a.ctx)
					}()
				}
			})

			showItem.Click(func() {
				log.Println("[Menu] Show item clicked")
				if a.ctx != nil && !isQuitting() {
					// 确保在主线程执行 UI 操作
					wailsRuntime.EventsEmit(a.ctx, "tray-show")
					// 直接调用 Show 可能跨线程，改用异步调用
					go func() {
						wailsRuntime.Show(a.ctx)
					}()
				}
			})

			quitItem.Click(func() {
				log.Println("[Menu] Quit item clicked")
				if !isQuitting() {
					setQuitting()

					// 先关闭菜单
					systray.Quit()

					// 退出应用
					if a.ctx != nil {
						wailsRuntime.Quit(a.ctx)
					} else {
						os.Exit(0)
					}
				}
			})
		},
		func() {
			// onExit 回调
			log.Println("systray: onExit")
			setQuitting()
			wg.Wait()
			log.Println("systray: exited cleanly")
		},
	)
}
