package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/astaxie/beego"
)

func TestMigrateGlobalFromAppConfig(t *testing.T) {
	db := newTestDb(t)
	_ = beego.AppConfig.Set("allow_user_login", "true")
	_ = beego.AppConfig.Set("allow_ports", "9001-9009")
	_ = beego.AppConfig.Set("show_http_proxy_port", "false")
	_ = beego.AppConfig.Set("open_captcha", "true")

	db.MigrateGlobalFromAppConfig()

	settings := db.GetGlobalSettings()
	if !settings.AllowUserLogin {
		t.Fatal("allow_user_login should be migrated as true")
	}
	if settings.AllowPorts != "9001-9009" {
		t.Fatalf("unexpected allow_ports: %q", settings.AllowPorts)
	}
	if settings.ShowHttpProxyPort {
		t.Fatal("show_http_proxy_port should be migrated as false")
	}
	if !settings.OpenCaptcha {
		t.Fatal("open_captcha should be migrated as true")
	}
	if db.GetGlobal() == nil {
		t.Fatal("global settings should be created")
	}
	if _, err := os.Stat(filepath.Join(db.JsonDb.RunPath, "conf", "global.json")); err != nil {
		t.Fatalf("global.json should be written: %v", err)
	}

	// 已经通过 Web 保存过的值，后续启动不能再被 nps.conf 覆盖。
	webValue := false
	db.GetGlobal().AllowUserLogin = &webValue
	db.MigrateGlobalFromAppConfig()
	if db.GetGlobalSettings().AllowUserLogin {
		t.Fatal("migration must not overwrite an existing global value")
	}
}
