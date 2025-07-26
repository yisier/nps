package controllers

import (
	"github.com/astaxie/beego"
	"strings"
	"io/ioutil"
)

type PublicController struct {
	beego.Controller
}

func (this *PublicController) Downloads() {
	this.Data["server_addr"] = beego.AppConfig.String("bridge_ip") + ":" + beego.AppConfig.String("bridge_port")
	this.Data["menu"] = "downloads" // 设置当前菜单项
	this.Data["isAdmin"] = false   // 设置isAdmin变量以避免模板错误
	this.Data["web_base_url"] = beego.AppConfig.String("web_base_url") // 添加web_base_url确保静态资源正确加载
	
	// 读取版本信息
	versionData, err := ioutil.ReadFile("VERSION")
	if err != nil {
		this.Data["version"] = "unknown"
	} else {
		this.Data["version"] = strings.TrimSpace(string(versionData))
	}
	
	this.Layout = "public/layout.html" // 使用公共布局
	this.TplName = "public/downloads_content.html" // 使用内容模板
}