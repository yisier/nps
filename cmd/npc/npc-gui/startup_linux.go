//go:build linux

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/astaxie/beego/logs"
)

const (
	appName       = "nps-client"
	desktopFormat = `[Desktop Entry]
Type=Application
Name=NPS Client
Comment=NPS Client Auto Start
Exec=%s
Terminal=false
X-GNOME-Autostart-enabled=true
`
)

// getAutostartPath 获取自动启动目录路径
func getAutostartPath() (string, error) {
	// 优先使用 XDG_CONFIG_HOME
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configHome = filepath.Join(homeDir, ".config")
	}

	autostartDir := filepath.Join(configHome, "autostart")
	// 确保目录存在
	if err := os.MkdirAll(autostartDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(autostartDir, appName+".desktop"), nil
}

// enableStartupImpl Linux 平台的开机启动实现（使用 .desktop 文件）
func enableStartupImpl() error {
	// 获取可执行文件路径
	exePath, err := getExecutablePath()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %v", err)
	}

	// 获取 .desktop 文件路径
	desktopPath, err := getAutostartPath()
	if err != nil {
		return fmt.Errorf("获取 autostart 路径失败: %v", err)
	}

	// 生成 .desktop 文件内容
	desktopContent := fmt.Sprintf(desktopFormat, exePath)

	// 写入 .desktop 文件
	if err := os.WriteFile(desktopPath, []byte(desktopContent), 0644); err != nil {
		return fmt.Errorf("写入 desktop 文件失败: %v", err)
	}

	logs.Info("成功添加开机启动项: %s -> %s", appName, desktopPath)
	return nil
}

// disableStartupImpl Linux 平台的禁用开机启动实现（删除 .desktop 文件）
func disableStartupImpl() error {
	desktopPath, err := getAutostartPath()
	if err != nil {
		return fmt.Errorf("获取 autostart 路径失败: %v", err)
	}

	// 删除 .desktop 文件
	if err := os.Remove(desktopPath); err != nil {
		if os.IsNotExist(err) {
			logs.Info("开机启动项不存在，无需删除")
			return nil
		}
		return fmt.Errorf("删除 desktop 文件失败: %v", err)
	}

	logs.Info("成功删除开机启动项: %s", desktopPath)
	return nil
}

// isStartupEnabledImpl Linux 平台检查是否已启用开机启动
func isStartupEnabledImpl() bool {
	desktopPath, err := getAutostartPath()
	if err != nil {
		return false
	}

	// 检查 .desktop 文件是否存在
	_, err = os.Stat(desktopPath)
	return err == nil
}
