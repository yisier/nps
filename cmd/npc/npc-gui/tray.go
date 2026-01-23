package main

import (
	_ "image/png"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"fyne.io/systray"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// 定义退出标志
var (
	quitting  bool
	quitMutex sync.Mutex
)

// setQuitting 设置退出状态
func setQuitting() {
	quitMutex.Lock()
	defer quitMutex.Unlock()
	quitting = true
}

// isQuitting 检查是否正在退出
func isQuitting() bool {
	quitMutex.Lock()
	defer quitMutex.Unlock()
	return quitting
}

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

			// 菜单项点击处理 - 使用独立的 goroutine 且优化退出逻辑
			go func() {
				log.Println("systray: menu listener started")

				// 使用带缓冲的通道确保退出信号不丢失
				exitChan := make(chan struct{}, 1)

				// 监听退出信号
				go func() {
					for {
						if isQuitting() {
							exitChan <- struct{}{}
							return
						}
						time.Sleep(50 * time.Millisecond)
					}
				}()

				// 主事件循环
				for {
					select {
					case <-exitChan:
						log.Println("systray: exit signal received")
						return

					case <-showItem.ClickedCh:
						log.Println("[Menu] Show item clicked")
						if a.ctx != nil && !isQuitting() {
							// 确保在主线程执行 UI 操作
							wailsRuntime.EventsEmit(a.ctx, "tray-show")
							// 直接调用 Show 可能跨线程，改用异步调用
							go func() {
								wailsRuntime.Show(a.ctx)
							}()
						}

					case <-quitItem.ClickedCh:
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
						return
					}
				}
			}()
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
