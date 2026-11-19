package main

import "sync"

// 退出状态由公共文件维护，避免平台实现重复定义。
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

// clearQuitting 清除退出状态（退出被取消时调用）
func clearQuitting() {
	quitMutex.Lock()
	defer quitMutex.Unlock()
	quitting = false
}

// isQuitting 检查是否正在退出
func isQuitting() bool {
	quitMutex.Lock()
	defer quitMutex.Unlock()
	return quitting
}
