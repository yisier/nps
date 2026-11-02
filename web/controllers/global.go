package controllers

import (
	"strings"

	"ehang.io/nps/lib/file"
)

type GlobalController struct {
	BaseController
}

func (s *GlobalController) Index() {
	s.Data["menu"] = "global"
	s.SetInfo("global")

	global := file.GetDb().GetGlobal()
	settings := file.GetDb().GetGlobalSettings()
	blackIpList := ""
	serverUrl := ""
	if global != nil {
		blackIpList = strings.Join(global.BlackIpList, "\r\n")
		serverUrl = global.ServerUrl
	}

	s.Data["globalBlackIpList"] = blackIpList
	s.Data["serverUrl"] = serverUrl
	s.Data["allowPorts"] = settings.AllowPorts
	s.Data["ipLimit"] = settings.IpLimit
	s.Data["showHttpProxyPort"] = settings.ShowHttpProxyPort
	s.Data["allowFlowLimit"] = settings.AllowFlowLimit
	s.Data["allowRateLimit"] = settings.AllowRateLimit
	s.Data["allowConnectionNumLimit"] = settings.AllowConnectionNumLimit
	s.Data["allowTunnelNumLimit"] = settings.AllowTunnelNumLimit
	s.Data["allowMultiIp"] = settings.AllowMultiIp
	s.Data["allowLocalProxy"] = settings.AllowLocalProxy
	s.Data["allowUserLogin"] = settings.AllowUserLogin
	s.Data["allowUserRegister"] = settings.AllowUserRegister
	s.Data["allowUserChangeUsername"] = settings.AllowUserChangeUsername
	s.Data["openCaptcha"] = settings.OpenCaptcha
	s.Data["systemInfoDisplay"] = settings.SystemInfoDisplay
	s.Data["httpAddOriginHeader"] = settings.HttpAddOriginHeader

	s.display("global/index")
}

// Save 保存全局参数。global.json 会同时更新内存和文件，保存后相关配置立即生效。
func (s *GlobalController) Save() {
	if s.Ctx.Request.Method == "GET" {
		s.Data["menu"] = "global"
		s.SetInfo("save global")
		s.display()
		return
	}

	global := file.GetDb().GetGlobal()
	if global == nil {
		global = new(file.Glob)
	}
	global.BlackIpList = RemoveRepeatedElement(strings.Split(s.getEscapeString("globalBlackIpList"), "\r\n"))
	global.ServerUrl = s.getEscapeString("serverUrl")
	global.AllowPorts = globalStringPtr(strings.TrimSpace(s.getEscapeString("allowPorts")))
	global.IpLimit = globalBoolPtr(s.GetBoolNoErr("ipLimit"))
	global.ShowHttpProxyPort = globalBoolPtr(s.GetBoolNoErr("showHttpProxyPort"))
	global.AllowFlowLimit = globalBoolPtr(s.GetBoolNoErr("allowFlowLimit"))
	global.AllowRateLimit = globalBoolPtr(s.GetBoolNoErr("allowRateLimit"))
	global.AllowConnectionNumLimit = globalBoolPtr(s.GetBoolNoErr("allowConnectionNumLimit"))
	global.AllowTunnelNumLimit = globalBoolPtr(s.GetBoolNoErr("allowTunnelNumLimit"))
	global.AllowMultiIp = globalBoolPtr(s.GetBoolNoErr("allowMultiIp"))
	global.AllowLocalProxy = globalBoolPtr(s.GetBoolNoErr("allowLocalProxy"))
	global.AllowUserLogin = globalBoolPtr(s.GetBoolNoErr("allowUserLogin"))
	global.AllowUserRegister = globalBoolPtr(s.GetBoolNoErr("allowUserRegister"))
	global.AllowUserChangeUsername = globalBoolPtr(s.GetBoolNoErr("allowUserChangeUsername"))
	global.OpenCaptcha = globalBoolPtr(s.GetBoolNoErr("openCaptcha"))
	global.SystemInfoDisplay = globalBoolPtr(s.GetBoolNoErr("systemInfoDisplay"))
	global.HttpAddOriginHeader = globalBoolPtr(s.GetBoolNoErr("httpAddOriginHeader"))

	if err := file.GetDb().SaveGlobal(global); err != nil {
		s.AjaxErr(err.Error())
		return
	}
	s.AjaxOk("save success")
}

func globalBoolPtr(value bool) *bool {
	return &value
}

func globalStringPtr(value string) *string {
	return &value
}
