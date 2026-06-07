# 增强功能

## 使用 https

> **v0.26.19 起 `https_just_proxy` 参数已被移除**，nps 会根据域名是否上传了证书自动选择处理方式：

| 是否在 web 上传证书 | 行为 | 真实 IP |
| --- | --- | --- |
| 是 | nps 直接接管 SSL 终结，再转发到内网 | ✅ 可拿到 |
| 否 | nps 透传 TLS（端口转发），由内网服务器自行解密 | ❌ 拿不到 |

![image](/image/new/https.png)

### 自动 HTTPS（v0.26.19+）

在 web 域名编辑页勾选「自动 HTTPS」后，nps 会把对应域名的 http 请求 301 重定向到 https。
v0.26.30 起重定向会按 `https_proxy_port` 配置的端口跳转（例如 `https_proxy_port=8443` 时会跳到 `https://<host>:8443/...`），方便非标准端口部署。

### HTTPS 证书来源（v0.26.22+）

web 域名编辑页的「证书」/「私钥」字段同时支持：

- **直接粘贴证书 / 私钥内容**
- **填写服务器上的文件路径**（例如 `/etc/letsencrypt/live/proxy.com/fullchain.pem`）

nps 会自动识别是文件路径还是内容。文件路径方式便于配合 acme.sh / certbot 自动续期，无需每次手工拷贝。

### 默认 HTTPS 证书

可以在 `nps.conf` 中配置一组默认证书，遇到未在 web 上单独配置的域名解析、或某些客户端发送的 `ClientHello` 不带 SNI 扩展时，自动落到默认证书。

> v0.26.33 起，`https_default_cert_file` / `https_default_key_file` 已被移除，请改用 web 上配置一条 host 为 `*`（或最具体的兜底域名）的解析来承担默认证书职责。

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

## TLS 桥接加密（v0.26.17+）

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
> v0.26.27 已移除客户端配置中的 `compress` / `crypt`（snappy 压缩与 AES 加密）。如需链路保护请使用此 TLS 桥接。

## 全局参数

web 后台 → 全局参数 页面（对应 `<runPath>/conf/global.json`）：

| 字段 | 含义 |
| --- | --- |
| 全局 IP 黑名单 | v0.26.16 引入，一行一个 IPv4，不支持范围匹配。命中后所有 TCP/HTTP/HTTPS/UDP 入口请求都会被直接关闭。用于防止扫描和暴力破解 |
| 服务地址 | v0.26.28 引入，用于在 web 上生成 [客户端快捷启动命令]、隧道访问 URL 时拼接的服务端地址。留空则使用浏览器当前访问地址 |

## 客户端 IP 黑/白名单（v0.26.27+）

### 黑名单（早期 2022-10-30 加入）
在客户端编辑页可填写多个 IP，命中后该客户端下所有隧道都会拒绝服务，用于针对单个客户端阻断。

### 白名单 + IP 授权（v0.26.27）

在客户端编辑页：

1. 直接添加 IP 到白名单
2. 或填写「IP 授权密码」。当外网 IP 不在白名单时，会被重定向到 IP 授权页面，输入正确密码即可把该 IP 自动加入白名单。

> v0.26.28 起 IP 授权页面优化为**通过穿透端口本身**提交，无需额外暴露 web 端口。

![img](/image/new/ip.png)

## HTTP/WebSocket 支持
nps 在域名代理模式下原生支持 WebSocket，无需任何额外配置（2022-10-24 引入；v0.26.30 进一步修复了 Home Assistant 等场景下的兼容性问题）。

## TCP 隧道 Proxy Protocol（v0.26.23+）

在 web 上 TCP 隧道编辑页开启「Proxy Protocol」开关，nps 会以 PROXY v1/v2 协议向后端转发原始来源 IP。后端服务（nginx、HAProxy 等）只需正常解析 PROXY 协议即可拿到真实客户端 IP。

![img](/image/new/protocol.png)

## TCP 隧道 Basic 认证（v0.26.31+）

TCP 隧道编辑页可填写 Basic 认证用户名/密码，连接 TCP 隧道时需通过 HTTP Basic 头校验后才放行。

> v0.26.33 修复了 TCP Basic 认证与「域名解析的客户端 Basic 认证」冲突的问题：TCP 隧道不再做 Basic 探测，域名解析仍沿用客户端级 Basic 认证。

## 域名解析记录开关（v0.26.31+）
web 上域名解析编辑页新增「访问日志」开关，可独立控制单条域名是否记录访问明细。

## 登录验证码（2022-10-27+，`open_captcha`）

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

## 自定义客户端连接密钥
web 上可以自定义客户端连接密钥，但必须保持全局唯一。
> v0.26.21 起 vkey 自动生成方式由 16 位改为 uuid 前 10 位，避免重复。

## 关闭公钥访问
将 `nps.conf` 中的 `public_vkey` 设置为空或删除。

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

## 代理到服务端本地
在 nps 监听 80 或 443 时，默认所有请求都转发到内网。但如果 nps 服务器本机也跑了服务，需要复用这两个端口对外提供，可类似 nginx `proxy_pass` 直接转发到本地。

**使用方式：** 在 `nps.conf` 中设置 `allow_local_proxy=true`，然后在 web 上设置想转发的隧道或域名，勾选「转发到本地」即可。
