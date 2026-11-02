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

## HTTP(S) 代理

| 名称 | 含义 | 默认值 |
| --- | --- | --- |
| http_proxy_ip | http(s) 代理监听 IP，默认 `0.0.0.0` | `0.0.0.0` |
| http_proxy_port | 域名代理 http 监听端口（默认 `80`），留空关闭 | `80` |
| https_proxy_port | 域名代理 https 监听端口（默认 `443`），留空关闭。详见 [使用 https](/server/nps_extend.html#使用-https) | `443` |
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

## 全局参数（原限制开关）

以下配置已从 `nps.conf` 迁移到 `conf/global.json`，请在 **Web 后台 → 全局参数** 中修改，保存后立即生效：

`allow_flow_limit`、`allow_rate_limit`、`allow_tunnel_num_limit`、`allow_connection_num_limit`、`allow_multi_ip`、`allow_local_proxy`、`allow_ports`、`system_info_display`

完整字段说明见 [全局参数](/nps/server/nps_extend.html#全局参数)。

## debug / pprof

| 名称 | 含义 | 默认值 |
| --- | --- | --- |
| pprof_ip | debug pprof 监听 IP | - |
| pprof_port | debug pprof 监听端口 | - |
