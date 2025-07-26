package controllers

import (
	"github.com/astaxie/beego"
)

type PublicController struct {
	beego.Controller
}

func (this *PublicController) Downloads() {
	this.Data["server_addr"] = beego.AppConfig.String("bridge_ip") + ":" + beego.AppConfig.String("bridge_port")
	this.TplName = "public/downloads.html"
}
