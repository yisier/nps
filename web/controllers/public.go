package controllers

import (
	"github.com/astaxie/beego"
)

type PublicController struct {
	beego.Controller
}

func (this *PublicController) Downloads() {
	this.Data["server_addr"] = beego.AppConfig.String("bridge_ip") + ":" + beego.AppConfig.String("bridge_port")
	this.Data["menu"] = "downloads" // 设置当前菜单项
	this.Layout = "public/layout.html" // 使用公共布局
	this.TplName = "public/downloads.html"
}