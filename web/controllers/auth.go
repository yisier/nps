package controllers

import (
	"encoding/hex"
	"github.com/astaxie/beego/logs"
	"html"
	"time"

	"ehang.io/nps/lib/file"

	"ehang.io/nps/lib/crypt"
	"github.com/astaxie/beego"
)

type AuthController struct {
	beego.Controller
}

func (s *AuthController) GetAuthKey() {
	m := make(map[string]interface{})
	defer func() {
		s.Data["json"] = m
		s.ServeJSON()
	}()
	if cryptKey := beego.AppConfig.String("auth_crypt_key"); len(cryptKey) != 16 {
		m["status"] = 0
		return
	} else {
		b, err := crypt.AesEncrypt([]byte(beego.AppConfig.String("auth_key")), []byte(cryptKey))
		if err != nil {
			m["status"] = 0
			return
		}
		m["status"] = 1
		m["crypt_auth_key"] = hex.EncodeToString(b)
		m["crypt_type"] = "aes cbc"
		return
	}
}

func (s *AuthController) GetTime() {
	m := make(map[string]interface{})
	m["time"] = time.Now().Unix()
	s.Data["json"] = m
	s.ServeJSON()
}

func (s *AuthController) IpWhiteAuth() {
	s.Ctx.ResponseWriter.Header().Set("Access-Control-Allow-Origin", "*")
	s.Ctx.ResponseWriter.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	s.Ctx.ResponseWriter.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	vkey := s.getEscapeString("vkey")
	ip := s.getEscapeString("ip")
	password := s.getEscapeString("pass")

	if vkey == "" || password == "" {
		s.Data["json"] = map[string]interface{}{"success": false, "message": "参数错误"}
		s.ServeJSON()
		return
	}

	// 如果未提供 ip，则使用请求中的客户端 IP（支持代理头）
	if ip == "" {
		ip = s.Ctx.Input.IP()
		ip = html.EscapeString(ip)
	}

	c, err := file.GetDb().GetClientByVkey(vkey)
	if err != nil {
		s.Data["json"] = map[string]interface{}{"success": false, "message": "客户端密钥错误"}
		s.ServeJSON()
		logs.Error("客户端IP白名单认证失败,客户端密钥错误:vkey [%s] ip [%s] password [%s]", vkey, ip, password)
		return
	}

	if c.IpWhitePass != password {
		s.Data["json"] = map[string]interface{}{"success": false, "message": "授权密码错误"}
		s.ServeJSON()
		logs.Error("客户端IP白名单认证失败,授权密码错误:vkey [%s] ip [%s] password [%s]", vkey, ip, password)
		return
	}

	ipExists := false
	for _, existingIp := range c.IpWhiteList {
		if existingIp == ip {
			ipExists = true
			break
		}
	}

	if !ipExists {
		c.IpWhiteList = append(c.IpWhiteList, ip)
		file.GetDb().UpdateClient(c)
	}

	s.Data["json"] = map[string]interface{}{"success": true, "message": "授权成功"}
	s.ServeJSON()

	logs.Info("客户端IP白名单认证授权成功:vkey [%s] ip [%s] password [%s]", vkey, ip, password)

}

func (s *AuthController) getEscapeString(key string) string {
	return html.EscapeString(s.GetString(key))
}
