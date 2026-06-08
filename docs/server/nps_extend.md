# 增强功能

## 使用 https

> nps 会根据域名是否上传了证书自动选择处理方式：

| 是否在 web 上传证书 | 行为 | 真实 IP |
| --- | --- | --- |
| 是 | nps 直接接管 SSL 终结，再转发到内网 | ✅ 可拿到 |
| 否 | nps 透传 TLS（端口转发），由内网服务器自行解密 | ❌ 拿不到 |

![image](/image/new/https.png)

## 自动 HTTPS

在 web 域名编辑页勾选「自动 HTTPS」后，nps 会把对应域名的 http 请求 301 重定向到 https。
重定向会按 `https_proxy_port` 配置的端口跳转（例如 `https_proxy_port=8443` 时会跳到 `https://<host>:8443/...`），方便非标准端口部署。

## HTTPS 证书来源

web 域名编辑页的「证书」/「私钥」字段同时支持：

- **直接粘贴证书 / 私钥内容**
- **填写服务器上的文件路径**（例如 `/etc/letsencrypt/live/proxy.com/fullchain.pem`）

nps 会自动识别是文件路径还是内容。文件路径方式便于配合 acme.sh / certbot 自动续期，无需每次手工拷贝。



## TLS 桥接加密

为防止客户端到服务端的桥接流量被防火墙识别和拦截，可启用 TLS：

```ini
# nps.conf
tls_enable=true
tls_bridge_port=8025      # v0.26.18 引入，与 bridge_port 并存
```

客户端任选一种方式接入：

```shell
npc.exe -server=1.1.1.1:8024 -vkey=xxx                       # 明文桥接（bridge_port）
npc.exe -server=1.1.1.1:8025 -vkey=xxx -tls_enable=true      # TLS 桥接（tls_bridge_port）
```

> 客户端忽略证书校验，请勿暴露在不可信网络里。

## 全局参数

> 未来会在此页面继续扩展更多功能，欢迎提需求。

web 后台 → 全局参数 页面（对应 `conf/global.json`）：

| 字段 | 含义 |
| --- | --- |
| 全局 IP 黑名单 | 一行一个 IPv4，不支持范围匹配。命中后所有 TCP/HTTP/HTTPS/UDP 入口请求都会被直接关闭。用于防止扫描和暴力破解 |
| 服务地址 |用于在 web 上生成 [客户端快捷启动命令]、隧道访问 URL 时拼接的服务端地址。留空则使用浏览器当前访问地址 |


## 客户端黑名单
在客户端编辑页可填写多个 IP，命中后该客户端下所有隧道都会拒绝服务，用于针对单个客户端阻断。

## 客户端白名单 + IP 授权

在客户端编辑页：

1. 直接添加 IP 到白名单
2. 或填写「IP 授权密码」。当外网 IP 不在白名单时，会被重定向到 IP 授权页面，输入正确密码即可把该 IP 自动加入白名单。

![img](/image/new/ip.png)


##  Proxy Protocol

在 web 上 TCP 隧道编辑页开启「Proxy Protocol」开关，nps 会以 PROXY v1/v2 协议向后端转发原始来源 IP。后端服务（nginx、HAProxy 等）只需正常解析 PROXY 协议即可拿到真实客户端 IP。

![img](/image/new/protocol.png)


## WEB登录验证码

`nps.conf` 中：

```ini
open_captcha=true
```

开启后 web 登录页会要求图形验证码。结合「web 管理保护」（连续 10 次失败封禁 1 分钟，见 [说明](/extend/description.html#web管理保护)），可以有效缓解暴力破解。

## 关闭代理

如需关闭 http 代理，将 `http_proxy_port` 留空；如需关闭 https 代理，将 `https_proxy_port` 留空。

## 流量数据持久化
服务端支持将流量数据持久化，默认关闭。在 `nps.conf` 中设置 `flow_store_interval`（单位分钟）即可开启。

**注意：** nps 不会持久化通过公钥连接的客户端。

## 系统信息显示
nps 服务端支持在 web 上显示和统计服务器相关信息，但默认部分统计图表是关闭的。如需开启请在 `nps.conf` 中设置 `system_info_display=true`。


## 关闭 web 管理
将 `nps.conf` 中的 `web_port` 设置为空或删除。

## 服务端多用户登录
将 `allow_user_login=true`，登录用户名 `user`，密码为对应客户端的验证密钥。登录后可进入客户端编辑修改 web 登录用户名密码。默认关闭。

## 用户注册功能
将 `allow_user_register=true` 后登录页会出现注册入口。

## 监听指定 IP

nps 支持每个隧道监听不同的服务端 IP，`nps.conf` 中设置 `allow_multi_ip=true` 后，可在 web 中控制，或 npc 配置文件中指定：

```ini
server_ip=xxx
```

## 客户端到期时间

在创建 / 修改客户端时可填写「到期时间」（可留空表示永不过期）。到期后该客户端会被**自动暂停**，所有隧道停止服务，直到管理员手工延长或清空到期时间。
支持格式：`2006-01-02 15:04:05` / `2006-01-02 15:04` / `2006-01-02T15:04:05` / `2006-01-02T15:04` / `2006-01-02`。

## 首次启动随机凭据

为避免默认密码被恶意扫描，**首次启动时** `web_username`（默认 `admin`）、`web_password`、`auth_key`、`auth_crypt_key` 全部随机生成，并直接打印到终端：

```text
[INFO] generated random web_password: <xxxxxxxx>
[INFO] generated random auth_key: <xxxxxxxx>
[INFO] generated random auth_crypt_key: <xxxxxxxxxxxxxxxx>
```

请在第一次启动时妥善记录。后续可在 `nps.conf` 中手工修改并执行 `nps reload`。



## 与 nginx 配合

有时需要在云服务器上跑 nginx 做静态缓存或多服务复用 80/443。把 `http_proxy_port` 设为非 80 端口（例如 8010），由 nginx 反代：

```
server {
    listen 80;
    server_name *.proxy.com;
    location / {
        proxy_set_header Host  $http_host;
        proxy_pass http://127.0.0.1:8010;
    }
}
```

如需 https 也由 nginx 终结，将 `https_proxy_port` 留空关闭，由 nginx 监听 443：

```
server {
    listen 443;
    server_name *.proxy.com;
    ssl on;
    ssl_certificate  certificate.crt;
    ssl_certificate_key private.key;
    ssl_session_timeout 5m;
    ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE:ECDH:AES:HIGH:!NULL:!aNULL:!MD5:!ADH:!RC4;
    ssl_protocols TLSv1 TLSv1.1 TLSv1.2;
    ssl_prefer_server_ciphers on;
    location / {
        proxy_set_header Host  $http_host;
        proxy_pass http://127.0.0.1:8020;
    }
}
```

## web 管理使用 https
在 `nps.conf` 中设置 `web_open_ssl=true`，并配置 `web_cert_file` 与 `web_key_file`。

## web 使用 Caddy 代理

实现子路径访问 nps，假设想通过 `http://caddy_ip:caddy_port/nps` 访问后台：

```nginx
caddy_ip:caddy_port/nps {
  ##server_ip 为 nps 服务器IP
  ##web_port 为 nps 后台端口
  proxy / http://server_ip:web_port/nps {
	transparent
  }
}
```

`nps.conf` 中设置：
```
web_base_url=/nps
```