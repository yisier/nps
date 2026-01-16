//go:build darwin

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/astaxie/beego/logs"
)

const (
	appName     = "com.nps.client"
	plistFormat = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<false/>
</dict>
</plist>`
)

// getLaunchAgentPath 获取 LaunchAgent plist 文件路径
func getLaunchAgentPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	launchAgentsDir := filepath.Join(homeDir, "Library", "LaunchAgents")
	// 确保目录存在
	if err := os.MkdirAll(launchAgentsDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(launchAgentsDir, appName+".plist"), nil
}

// enableStartupImpl macOS 平台的开机启动实现（使用 LaunchAgents）
func enableStartupImpl() error {
	// 获取可执行文件路径
	exePath, err := getExecutablePath()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %v", err)
	}

	// 获取 plist 文件路径
	plistPath, err := getLaunchAgentPath()
	if err != nil {
		return fmt.Errorf("获取 LaunchAgent 路径失败: %v", err)
	}

	// 生成 plist 内容
	plistContent := fmt.Sprintf(plistFormat, appName, exePath)

	// 写入 plist 文件
	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("写入 plist 文件失败: %v", err)
	}

	logs.Info("成功添加开机启动项: %s -> %s", appName, plistPath)
	return nil
}

// disableStartupImpl macOS 平台的禁用开机启动实现（删除 plist 文件）
func disableStartupImpl() error {
	plistPath, err := getLaunchAgentPath()
	if err != nil {
		return fmt.Errorf("获取 LaunchAgent 路径失败: %v", err)
	}

	// 删除 plist 文件
	if err := os.Remove(plistPath); err != nil {
		if os.IsNotExist(err) {
			logs.Info("开机启动项不存在，无需删除")
			return nil
		}
		return fmt.Errorf("删除 plist 文件失败: %v", err)
	}

	logs.Info("成功删除开机启动项: %s", plistPath)
	return nil
}

// isStartupEnabledImpl macOS 平台检查是否已启用开机启动
func isStartupEnabledImpl() bool {
	plistPath, err := getLaunchAgentPath()
	if err != nil {
		return false
	}

	// 检查 plist 文件是否存在
	_, err = os.Stat(plistPath)
	return err == nil
}
