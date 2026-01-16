//go:build windows

package main

import (
	"fmt"

	"github.com/astaxie/beego/logs"
	"golang.org/x/sys/windows/registry"
)

const appName = "NPS客户端"

// enableStartupImpl Windows 平台的开机启动实现（使用注册表）
func enableStartupImpl() error {
	// 获取可执行文件路径
	exePath, err := getExecutablePath()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %v", err)
	}

	// 打开注册表键
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("打开注册表失败: %v", err)
	}
	defer key.Close()

	// 设置注册表值（应用名称 -> 可执行文件路径）
	err = key.SetStringValue(appName, exePath)
	if err != nil {
		return fmt.Errorf("设置注册表值失败: %v", err)
	}

	logs.Info("成功添加开机启动项: %s -> %s", appName, exePath)
	return nil
}

// disableStartupImpl Windows 平台的禁用开机启动实现（删除注册表项）
func disableStartupImpl() error {
	// 打开注册表键
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("打开注册表失败: %v", err)
	}
	defer key.Close()

	// 删除注册表值
	err = key.DeleteValue(appName)
	if err != nil {
		// 如果键不存在，不算错误
		if err == registry.ErrNotExist {
			logs.Info("开机启动项不存在，无需删除")
			return nil
		}
		return fmt.Errorf("删除注册表值失败: %v", err)
	}

	logs.Info("成功删除开机启动项: %s", appName)
	return nil
}

// isStartupEnabledImpl Windows 平台检查是否已启用开机启动
func isStartupEnabledImpl() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	_, _, err = key.GetStringValue(appName)
	return err == nil
}
