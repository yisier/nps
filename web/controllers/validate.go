package controllers

import "strings"

// validateClientWebPair 校验客户端 Web 登录用户名和密码必须成对设置。
func validateClientWebPair(webUserName, webPassword string) string {
	hasUserName := strings.TrimSpace(webUserName) != ""
	hasPassword := strings.TrimSpace(webPassword) != ""
	if hasUserName != hasPassword {
		return "web username and password must be set together"
	}
	return ""
}

// validateTunnelInput 按隧道模式校验必填项。
func validateTunnelInput(mode, target, password, localPath string) string {
	switch mode {
	case "tcp", "udp":
		if strings.TrimSpace(target) == "" {
			return "target is required"
		}
	case "secret", "p2p":
		if strings.TrimSpace(target) == "" {
			return "target is required"
		}
		if strings.TrimSpace(password) == "" {
			return "password is required"
		}
	case "file":
		if strings.TrimSpace(localPath) == "" {
			return "local path is required"
		}
	case "socks5", "httpProxy":
	default:
		return "invalid tunnel type"
	}
	return ""
}

// validateHostInput 校验域名解析必填项。
func validateHostInput(host, target string) string {
	if strings.TrimSpace(host) == "" {
		return "host is required"
	}
	if strings.TrimSpace(target) == "" {
		return "target is required"
	}
	return ""
}
