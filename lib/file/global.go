package file

import (
	"github.com/astaxie/beego"
)

// GlobalSettings 是 Glob 的只读快照：把尚未迁移的指针字段解析成带默认值的普通值，
// 方便在请求/连接处理中直接读取，保证 Web 保存后立即生效。
type GlobalSettings struct {
	AllowPorts              string
	IpLimit                 bool
	ShowHttpProxyPort       bool
	AllowFlowLimit          bool
	AllowRateLimit          bool
	AllowConnectionNumLimit bool
	AllowTunnelNumLimit     bool
	AllowMultiIp            bool
	AllowLocalProxy         bool
	AllowUserLogin          bool
	AllowUserRegister       bool
	AllowUserChangeUsername bool
	OpenCaptcha             bool
	SystemInfoDisplay       bool
	HttpAddOriginHeader     bool
}

func boolPtrValue(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}

func stringPtrValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// GetGlobalSettings 返回全局参数的只读快照，未初始化的字段使用历史默认值。
func (s *DbUtils) GetGlobalSettings() GlobalSettings {
	settings := GlobalSettings{
		// 与原来的 DefaultBool / 默认 nps.conf 保持一致
		ShowHttpProxyPort:       true,
		AllowFlowLimit:          true,
		AllowRateLimit:          true,
		AllowConnectionNumLimit: true,
		AllowTunnelNumLimit:     true,
		AllowMultiIp:            true,
		AllowUserLogin:          true,
		AllowUserChangeUsername: true,
		SystemInfoDisplay:       true,
		HttpAddOriginHeader:     true,
	}
	global := s.GetGlobal()
	if global == nil {
		return settings
	}
	settings.AllowPorts = stringPtrValue(global.AllowPorts)
	settings.IpLimit = boolPtrValue(global.IpLimit)
	settings.ShowHttpProxyPort = boolPtrValue(global.ShowHttpProxyPort)
	settings.AllowFlowLimit = boolPtrValue(global.AllowFlowLimit)
	settings.AllowRateLimit = boolPtrValue(global.AllowRateLimit)
	settings.AllowConnectionNumLimit = boolPtrValue(global.AllowConnectionNumLimit)
	settings.AllowTunnelNumLimit = boolPtrValue(global.AllowTunnelNumLimit)
	settings.AllowMultiIp = boolPtrValue(global.AllowMultiIp)
	settings.AllowLocalProxy = boolPtrValue(global.AllowLocalProxy)
	settings.AllowUserLogin = boolPtrValue(global.AllowUserLogin)
	settings.AllowUserRegister = boolPtrValue(global.AllowUserRegister)
	settings.AllowUserChangeUsername = boolPtrValue(global.AllowUserChangeUsername)
	settings.OpenCaptcha = boolPtrValue(global.OpenCaptcha)
	settings.SystemInfoDisplay = boolPtrValue(global.SystemInfoDisplay)
	settings.HttpAddOriginHeader = boolPtrValue(global.HttpAddOriginHeader)
	return settings
}

func appConfigBool(key string, defaultValue bool) bool {
	value, err := beego.AppConfig.Bool(key)
	if err != nil {
		return defaultValue
	}
	return value
}

func boolPtr(value bool) *bool {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

// MigrateGlobalFromAppConfig 在启动时把 nps.conf 中可热更新的参数补齐到 global.json。
// 只填充仍未初始化的字段：已经通过 Web 页面保存过的值不会被 nps.conf 覆盖。
func (s *DbUtils) MigrateGlobalFromAppConfig() {
	global := s.GetGlobal()
	if global == nil {
		global = new(Glob)
	}
	if global.BlackIpList == nil {
		global.BlackIpList = []string{}
	}
	changed := false

	if global.AllowPorts == nil {
		global.AllowPorts = stringPtr(beego.AppConfig.String("allow_ports"))
		changed = true
	}
	if global.IpLimit == nil {
		global.IpLimit = boolPtr(appConfigBool("ip_limit", false))
		changed = true
	}
	if global.ShowHttpProxyPort == nil {
		global.ShowHttpProxyPort = boolPtr(beego.AppConfig.DefaultBool("show_http_proxy_port", true))
		changed = true
	}
	if global.AllowFlowLimit == nil {
		global.AllowFlowLimit = boolPtr(appConfigBool("allow_flow_limit", true))
		changed = true
	}
	if global.AllowRateLimit == nil {
		global.AllowRateLimit = boolPtr(appConfigBool("allow_rate_limit", true))
		changed = true
	}
	if global.AllowConnectionNumLimit == nil {
		global.AllowConnectionNumLimit = boolPtr(appConfigBool("allow_connection_num_limit", true))
		changed = true
	}
	if global.AllowTunnelNumLimit == nil {
		global.AllowTunnelNumLimit = boolPtr(appConfigBool("allow_tunnel_num_limit", true))
		changed = true
	}
	if global.AllowMultiIp == nil {
		global.AllowMultiIp = boolPtr(appConfigBool("allow_multi_ip", true))
		changed = true
	}
	if global.AllowLocalProxy == nil {
		global.AllowLocalProxy = boolPtr(appConfigBool("allow_local_proxy", false))
		changed = true
	}
	if global.AllowUserLogin == nil {
		global.AllowUserLogin = boolPtr(appConfigBool("allow_user_login", true))
		changed = true
	}
	if global.AllowUserRegister == nil {
		global.AllowUserRegister = boolPtr(appConfigBool("allow_user_register", false))
		changed = true
	}
	if global.AllowUserChangeUsername == nil {
		global.AllowUserChangeUsername = boolPtr(appConfigBool("allow_user_change_username", true))
		changed = true
	}
	if global.OpenCaptcha == nil {
		global.OpenCaptcha = boolPtr(appConfigBool("open_captcha", false))
		changed = true
	}
	if global.SystemInfoDisplay == nil {
		global.SystemInfoDisplay = boolPtr(appConfigBool("system_info_display", true))
		changed = true
	}
	if global.HttpAddOriginHeader == nil {
		global.HttpAddOriginHeader = boolPtr(appConfigBool("http_add_origin_header", true))
		changed = true
	}

	if changed {
		_ = s.SaveGlobal(global)
	}
}
