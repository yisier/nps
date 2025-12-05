package controllers

import (
	"ehang.io/nps/lib/file"
	"strings"
)

type GlobalController struct {
	BaseController
}

func (s *GlobalController) Index() {
	//if s.Ctx.Request.Method == "GET" {
	//
	//	return
	//}
	s.Data["menu"] = "global"
	s.SetInfo("global")
	s.display("global/index")

	global := file.GetDb().GetGlobal()
	if global == nil {
		return
	}
	s.Data["globalBlackIpList"] = strings.Join(global.BlackIpList, "\r\n")
	s.Data["serverUrl"] = global.ServerUrl
}

// 添加全局参数
func (s *GlobalController) Save() {
	if s.Ctx.Request.Method == "GET" {
		s.Data["menu"] = "global"
		s.SetInfo("save global")
		s.display()
	} else {

		t := &file.Glob{
			BlackIpList: RemoveRepeatedElement(strings.Split(s.getEscapeString("globalBlackIpList"), "\r\n")),
			ServerUrl:   s.getEscapeString("serverUrl")}

		if err := file.GetDb().SaveGlobal(t); err != nil {
			s.AjaxErr(err.Error())
		}
		s.AjaxOk("save success")
	}
}
