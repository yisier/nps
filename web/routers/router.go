package routers

import (
	"ehang.io/nps/web/controllers"
	"github.com/astaxie/beego"
)

func Init() {
	beego.SetStaticPath("/static", "web/static")
	web_base_url := beego.AppConfig.String("web_base_url")
	if len(web_base_url) > 0 {
		ns := beego.NewNamespace(web_base_url,
			beego.NSRouter("/", &controllers.IndexController{}, "*:Index"),
			beego.NSAutoRouter(&controllers.IndexController{}),
			beego.NSAutoRouter(&controllers.LoginController{}),
			beego.NSAutoRouter(&controllers.ClientController{}),
			beego.NSAutoRouter(&controllers.AuthController{}),
			beego.NSAutoRouter(&controllers.GlobalController{}),
			beego.NSRouter("/downloads", &controllers.PublicController{}, "*:Downloads"),
		)
		beego.AddNamespace(ns)
	} else {
		beego.Router("/", &controllers.IndexController{}, "*:Index")
		beego.AutoRouter(&controllers.IndexController{})
		beego.AutoRouter(&controllers.LoginController{})
		beego.AutoRouter(&controllers.ClientController{})
		beego.AutoRouter(&controllers.AuthController{})
		beego.AutoRouter(&controllers.GlobalController{})
		beego.Router("/downloads", &controllers.PublicController{}, "*:Downloads")

	}
}
