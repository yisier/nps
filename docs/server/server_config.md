# 服务端配置文件
配置文件路径：

- 二进制直接运行：`<nps 可执行文件目录>/conf/nps.conf`
- `nps install` 之后：`/etc/nps/conf/nps.conf`（linux/darwin）或 `C:\Program Files\nps\conf\nps.conf`（windows）
- 自定义：`-conf_path=` 参数指定的目录下的 `conf/nps.conf`

> v0.26.33 起，首次启动时若 `conf/nps.conf` 不存在会**自动生成默认配置**，其中 `web_username`（默认 `admin`）、`web_password`、`auth_key`、`auth_crypt_key` **均为随机值**并打印到终端，请第一时间复制保存。

## 通用 / 桥接 / 日志

| 名称 | 含义 |
| --- | --- |
| bridge_type | 客户端与服务端连接方式 `tcp` 或 `kcp` |
| bridge_port | 服务端客户端通信端口（默认 `8024`） |
| bridge_ip | 桥接监听 IP，默认 `0.0.0.0` |
| public_vkey | 客户端以配置文件模式启动时使用的公钥，设为空表示关闭配置文件连接模式 |
| disconnect_timeout | 客户端连接超时，单位为 5s，默认 `60`（即 5 分钟） |
| flow_store_interval | 流量数据持久化间隔，单位分钟；忽略表示不持久化 |
| log_level | 日志级别 0~7 |
| log_path | 日志文件路径 |
| ip_limit | 是否限制 IP 访问，`true` / `false` / 忽略 |

## TLS（v0.26.17+）

| 名称 | 含义 |
| --- | --- |
| tls_enable | 是否开启 TLS。开启后客户端可通过 `-tls_enable=true` 连接 `tls_bridge_port` |
| tls_bridge_port | TLS 桥接端口，默认 `8025`。**与 `bridge_port` 并存**：客户端可分别用 `bridge_port`（明文）或 `tls_bridge_port`（TLS）接入 |

## HTTP / HTTPS 代理

| 名称 | 含义 |
| --- | --- |
| http_proxy_ip | http(s) 代理监听 IP，默认 `0.0.0.0` |
| http_proxy_port | 域名代理 http 监听端口（默认 `80`），留空关闭 |
| https_proxy_port | 域名代理 https 监听端口（默认 `443`），留空关闭。详见 [使用 https](/server/nps_extend.html#使用-https) |
| show_http_proxy_port | 域名隧道访问地址是否拼接非 80/443 端口，`true` / `false` |
| http_add_origin_header | 是否在 http(s) 请求头中追加 `X-Forwarded-For` / `X-Real-IP`，用于内网获取真实 IP |
| http_cache | 是否开启静态文件缓存 |
| http_cache_length | 缓存条目数上限，`0` 表示不限制 |

> v0.26.19 起移除参数 `https_just_proxy`。逻辑改为：**域名条目上传了证书 → 由 nps 接管 SSL（可拿到真实 IP）；否则 → 端口转发由内网服务器自行处理 SSL（拿不到真实 IP）**。
>
> v0.26.33 起一并清理失效配置项：`appname`、`runmode`、`https_default_cert_file`、`https_default_key_file`，不再读取。

## Web 管理后台

| 名称 | 含义 |
| --- | --- |
| web_host | web 管理使用的二级域名，端口复用时区分用 |
| web_username | web 后台用户名（默认 `admin`，首次启动自动生成时仍为 `admin`） |
| web_password | web 后台密码（**v0.26.33 起首次启动随机生成**） |
| web_port | web 管理端口（默认 `8080`），留空关闭 web |
| web_ip | web 管理监听 IP |
| web_base_url | web 管理子路径，例如 `/nps`，用于反代到子路径时使用 |
| web_open_ssl | web 管理是否启用 https |
| web_cert_file | web 管理 https 证书路径 |
| web_key_file | web 管理 https 私钥路径 |
| open_captcha | 登录是否开启验证码校验（v0.26.27 之前提供，默认 `false`） |
| allow_user_login | 是否允许多用户登录，开启后用户名 `user`，密码为客户端的验证密钥 |
| allow_user_register | 是否允许从登录页注册账号 |
| allow_user_change_username | 多用户登录后是否允许修改用户名 |

## API 鉴权

| 名称 | 含义 |
| --- | --- |
| auth_key | web API 鉴权密钥（**v0.26.33 起首次启动随机生成**），详见 [Web API](/extend/api.html) |
| auth_crypt_key | `auth/getauthkey` 接口的 AES 加密密钥，**必须 16 位**（v0.26.33 起首次启动随机生成 16 位） |

## P2P

| 名称 | 含义 |
| --- | --- |
| p2p_ip | 服务端 IP，使用 p2p 模式必填 |
| p2p_port | p2p 模式开启的 UDP 起始端口。若设为 `6000`，请额外开放 `6001`、`6002` |

## 限制开关

> 这些开关默认关闭。开启后才会在 web 管理中暴露对应字段。

| 名称 | 含义 |
| --- | --- |
| allow_flow_limit | 是否启用客户端流量限制 |
| allow_rate_limit | 是否启用客户端带宽限制 |
| allow_tunnel_num_limit | 是否启用客户端最大隧道数限制 |
| allow_connection_num_limit | 是否启用客户端最大连接数限制 |
| allow_multi_ip | 是否允许每个隧道监听不同的服务端 IP |
| allow_local_proxy | 是否允许把隧道目标转发到 nps 所在服务器本地 |
| allow_ports | 限制可开放的隧道端口范围，例如 `9001-9009,10001,11000-12000`，留空不限制 |
| system_info_display | 是否在 web 上展示服务器系统监控信息图表 |

## debug / pprof

| 名称 | 含义 |
| --- | --- |
| pprof_ip | debug pprof 监听 IP |
| pprof_port | debug pprof 监听端口 |
