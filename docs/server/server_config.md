# 服务端配置文件

配置文件路径：`<nps 可执行文件目录>/conf/nps.conf`


> 首次启动时若 `conf/nps.conf` 不存在会**自动生成默认配置**，其中 `web_username`（默认 `admin`）、`web_password`、`auth_key`、`auth_crypt_key` **均为随机值**并打印到终端，请在启动日志或配置文件中查看。

## 通用配置

| 名称 | 含义 | 默认值 |
| --- | --- | --- |
| bridge_type | 客户端与服务端连接方式 `tcp` 或 `kcp` | `tcp` |
| bridge_ip | 桥接监听 IP | `0.0.0.0` |
| bridge_port | 服务端客户端通信端口 | `8024` |
| tls_enable | 是否开启 TLS。开启后客户端可通过 `-tls_enable=true` 连接 `tls_bridge_port` | `true` |
| tls_bridge_port | TLS 桥接端口，默认 `8025`。**与 `bridge_port` 并存**：客户端可分别用 `bridge_port`（明文）或 `tls_bridge_port`（TLS）接入 | `8025` |
| disconnect_timeout | 客户端连接超时，单位为 5s，默认 `60`（即 5 分钟） | `60` |
| flow_store_interval | 流量数据持久化间隔，单位分钟；忽略表示不持久化| `1` |
| log_level | 日志级别 0~7 | `6` |
| log_path | 日志文件路径 | `nps.log` |
| ip_limit | 是否限制 IP 访问，`true` / `false` / 忽略 | - |


## HTTP(S) 代理

| 名称 | 含义 | 默认值 |
| --- | --- | --- |
| http_proxy_ip | http(s) 代理监听 IP，默认 `0.0.0.0` | `0.0.0.0` |
| http_proxy_port | 域名代理 http 监听端口（默认 `80`），留空关闭 | `80` |
| https_proxy_port | 域名代理 https 监听端口（默认 `443`），留空关闭。详见 [使用 https](/server/nps_extend.html#使用-https) | `443` |
| show_http_proxy_port | 域名隧道访问地址是否拼接非 80/443 端口，`true` / `false` | `true` |
| http_add_origin_header | 是否在 http(s) 请求头中追加 `X-Forwarded-For` / `X-Real-IP`，用于内网获取真实 IP | `true` |
| http_cache | 是否开启静态文件缓存 | `false` |
| http_cache_length | 缓存条目数上限，`0` 表示不限制 | `100` |


## Web 管理后台

| 名称 | 含义 | 默认值 |
| --- | --- | --- |
| web_host | web 管理使用的二级域名，端口复用时区分用 | `a.o.com` |
| web_username | web 后台用户名（默认 `admin` | `admin` |
| web_password | web 后台密码 | 首次启动随机生成 |
| web_port | web 管理端口，留空关闭 web | `8081` |
| web_ip | web 管理监听 IP | `0.0.0.0` |
| web_base_url | web 管理子路径，例如 `/nps`，用于反代到子路径时使用 | （空） |
| web_open_ssl | web 管理是否启用 https | `false` |
| web_cert_file | web 管理 https 证书路径 | `conf/server.pem` |
| web_key_file | web 管理 https 私钥路径 | `conf/server.key` |
| open_captcha | 登录是否开启验证码校验 | `false` |
| allow_user_login | 是否允许多用户登录，开启后用户名 `user`，密码为客户端的验证密钥 | `true` |
| allow_user_register | 是否允许从登录页注册账号 | `false` |
| allow_user_change_username | 多用户登录后是否允许修改用户名 | `true` |

## API 鉴权

| 名称 | 含义 | 默认值 |
| --- | --- | --- |
| auth_key | web API 鉴权密钥，详见 [Web API](/nps/extend/api.html) | 首次启动随机生成 |
| auth_crypt_key | `auth/getauthkey` 接口的 AES 加密密钥，**必须 16 位** | 首次启动随机生成 |

## P2P

| 名称 | 含义 | 默认值 |
| --- | --- | --- |
| p2p_ip | 服务端 IP，使用 p2p 模式必填 | - |
| p2p_port | p2p 模式开启的 UDP 起始端口。若设为 `6000`，请额外开放 `6001`、`6002` | - |

## 限制开关

> 这些开关默认关闭。开启后才会在 web 管理中暴露对应字段。

| 名称 | 含义 | 默认值 |
| --- | --- | --- |
| allow_flow_limit | 是否启用客户端流量限制 | `true` |
| allow_rate_limit | 是否启用客户端带宽限制 | `true` |
| allow_tunnel_num_limit | 是否启用客户端最大隧道数限制 | `true` |
| allow_connection_num_limit | 是否启用客户端最大连接数限制 | `true` |
| allow_multi_ip | 是否允许每个隧道监听不同的服务端 IP | `true` |
| allow_local_proxy | 是否允许把隧道目标转发到 nps 所在服务器本地 | `false` |
| allow_ports | 限制可开放的隧道端口范围，例如 `9001-9009,10001,11000-12000`，留空不限制 | （空） |
| system_info_display | 是否在 web 上展示服务器系统监控信息图表 | `true` |

## debug / pprof

| 名称 | 含义 | 默认值 |
| --- | --- | --- |
| pprof_ip | debug pprof 监听 IP | - |
| pprof_port | debug pprof 监听端口 | - |
