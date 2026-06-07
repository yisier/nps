# 基本使用

> **强烈推荐使用无配置文件模式启动客户端**：所有数据应在服务端保存和配置，客户端只做连接转发。
> v0.26.27 起客户端默认发布包不再附带 `conf/npc.conf`；v0.26.21 起客户端可直接双击运行，通过交互菜单完成全部操作。

## 无配置文件模式

此模式的各种配置都在服务端 web 管理中完成，客户端无需任何文件，只需一条命令或一段【快捷启动命令】。

### 方式 A：交互菜单（v0.26.21+ 推荐）

直接双击 `npc`/`npc.exe`，无任何参数，按提示输入即可：

| 菜单项 | 输入内容 |
| --- | --- |
| 启动客户端 | 【快捷启动命令】（base64）或【TLS 快捷启动命令】|
| 安装为系统服务 | 【快捷启动命令】或【TLS 快捷启动命令】|
| 卸载 / 启动 / 停止已安装服务 | 隧道密钥 `vkey` |
| 更新客户端（v0.26.34+） | — |

【快捷启动命令】在 web 后台 → 客户端列表 → 客户端前的小图标里复制，本质是 `服务端地址 + vkey` 的 base64 编码。TLS 模式有独立的【TLS 快捷启动命令】（v0.26.25+）。

服务以 `nps-client-<vkey>` 命名注册，可在同一台机器并存多个客户端实例。

![image](/image/new/cmd.png)

### 方式 B：直接命令行

```shell
./npc -server=ip:port -vkey=web界面中显示的密钥                            # 标准
./npc -server=ip:8025 -vkey=xxx -tls_enable=true                            # TLS 桥接
./npc -server=ip:8024 -vkey=ytkpyr0er676m0r7,iwnbjfbvygvzyzzt              # 同时拉起多个隧道 ID（v0.26.19+，逗号拼接）
```

> 注意：v0.26.21 起 vkey 由 16 位缩短至 10 位（uuid 前 10 位），都是合法的。

## GUI 客户端（v0.26.29+，Windows）

基于 Wails 开发的桌面 GUI，需要安装 [WebView2](https://developer.microsoft.com/microsoft-edge/webview2/) 运行时。

支持两种添加客户端方式：

1. 粘贴 web 后台的【快捷启动命令】
2. 手动填写 `-server`、`-vkey` 等参数

详细说明请参考 [GUI 客户端 README](https://github.com/yisier/nps/blob/master/cmd/npc/npc-gui/README.md)。

![img](/image/new/gui.png)

## 注册到系统服务（开机启动、守护进程）

### 方式 A：通过 npc 交互菜单（推荐）

```shell
./npc          # linux/darwin（需 sudo）
npc.exe        # windows（管理员 cmd）
```

按菜单选「安装」并粘贴【快捷启动命令】即可，服务名为 `nps-client-<vkey>`。

### 方式 B：传统 install / start 命令

对于 linux、darwin：

- 注册：`sudo ./npc install 其他参数（例如 -server=xx -vkey=xx 或 -config=xxx）`
- 启动：`sudo npc start`
- 停止：`sudo npc stop`
- 更换启动参数需先 `./npc uninstall` 再重新注册

对于 windows，使用管理员身份运行 cmd：

- 注册：`npc.exe install 其他参数（-server=xx -vkey=xx 或 -config=xxx）`
- 启动 / 停止：`npc.exe start` / `npc.exe stop`
- 退出时自动重启请按下图配置

![image](https://raw.githubusercontent.com/yisier/nps/master/docs/windows_client_service_configuration.png)

### 系统服务日志（v0.26.22+）

注册系统服务后日志按 **每个 vkey 一个文件** 保存，命名为 `npc-<vkey>.log`：

- windows：与 `npc.exe` 同级目录
- linux / darwin：`/var/log/`

老式 `npc install` 注册的，日志路径与旧版一致（windows 同级目录、linux `/var/log/npc.log`）。

## 客户端更新

### 方式 A：交互菜单（v0.26.34+ 推荐）

```shell
./npc          # 选择菜单 [5] 更新客户端
```

自动下载并替换 `npc` 二进制。

### 方式 B：传统命令

```shell
# 先停止
sudo npc stop                  # linux/darwin
npc.exe stop                   # windows

# 更新
sudo npc-update update         # linux/darwin
npc-update.exe update          # windows

# 重启
sudo npc start
npc.exe start
```

如果无法成功更新，可以直接下载 releases 压缩包覆盖原有的 `npc` 二进制。

## 配置文件模式

> ⚠️ **配置文件模式对小白不友好，容易出错，已不推荐使用。**
> v0.26.27 起新版客户端默认不附带 `conf/npc.conf`；本节保留是为了兼容老用户。

此模式使用 nps 的公钥或客户端私钥验证，各种配置在客户端完成，同时服务端 web 也可以进行管理。

```shell
 ./npc -config=npc配置文件路径
```

### 配置文件说明
[示例配置文件](https://github.com/yisier/nps/tree/master/conf/npc.conf)

#### 全局配置
```ini
[common]
server_addr=1.1.1.1:8024
conn_type=tcp
vkey=123
username=111
password=222
rate_limit=10000
flow_limit=100
remark=test
max_conn=10
tls_enable=true
#pprof_addr=0.0.0.0:9999
```

项 | 含义
---|---
server_addr | 服务端 ip / 域名:port
conn_type | 与服务端通信模式（tcp 或 kcp）
vkey | 服务端配置文件中的密钥（非 web）
username | socks5 或 http(s) Basic 认证用户名（可忽略）
password | socks5 或 http(s) Basic 认证密码（可忽略）
rate_limit | 速度限制，可忽略
flow_limit | 流量限制，可忽略
remark | 客户端备注，可忽略
max_conn | 最大连接数，可忽略
tls_enable | 是否使用 TLS 桥接（v0.26.17+），需要服务端同时开启
pprof_addr | debug pprof ip:port

> v0.26.27 已移除客户端的 `compress`（snappy 压缩）和 `crypt`（AES 加密）配置项，"利大于弊"。如需链路保护请改用 [TLS 桥接](/server/nps_extend.html#tls-桥接加密v02617)。

#### 域名代理

```ini
[common]
server_addr=1.1.1.1:8024
vkey=123
[web1]
host=a.proxy.com
target_addr=127.0.0.1:8080,127.0.0.1:8082
host_change=www.proxy.com
header_set_proxy=nps
```
项 | 含义
---|---
web1 | 备注
host | 域名（http \| https 都可解析）
target_addr | 内网目标，负载均衡时多个目标，逗号隔开
host_change | 请求 host 修改
header_xxx | 请求 header 修改或添加，`header_proxy` 表示添加 header `proxy:nps`

#### tcp 隧道模式

```ini
[common]
server_addr=1.1.1.1:8024
vkey=123
[tcp]
mode=tcp
target_addr=127.0.0.1:8080
server_port=9001
```
项 | 含义
---|---
mode | tcp
server_port | 在服务端的代理端口
target_addr | 内网目标

#### udp 隧道模式

```ini
[common]
server_addr=1.1.1.1:8024
vkey=123
[udp]
mode=udp
target_addr=127.0.0.1:8080
server_port=9002
```
项 | 含义
---|---
mode | udp
server_port | 在服务端的代理端口
target_addr | 内网目标

#### http 代理模式

```ini
[common]
server_addr=1.1.1.1:8024
vkey=123
[http]
mode=httpProxy
server_port=9003
```
项 | 含义
---|---
mode | httpProxy
server_port | 在服务端的代理端口

#### socks5 代理模式

```ini
[common]
server_addr=1.1.1.1:8024
vkey=123
[socks5]
mode=socks5
server_port=9004
multi_account=multi_account.conf
```
项 | 含义
---|---
mode | socks5
server_port | 在服务端的代理端口
multi_account | socks5 多账号配置文件（可选），配置后 `basic_username` / `basic_password` 无法通过认证

#### 私密代理模式

```ini
[common]
server_addr=1.1.1.1:8024
vkey=123
[secret_ssh]
mode=secret
password=ssh2
target_addr=10.1.50.2:22
```
项 | 含义
---|---
mode | secret
password | 唯一密钥
target_addr | 内网目标

#### p2p 代理模式

```ini
[common]
server_addr=1.1.1.1:8024
vkey=123
[p2p_ssh]
mode=p2p
password=ssh2
target_addr=10.1.50.2:22
```
项 | 含义
---|---
mode | p2p
password | 唯一密钥
target_addr | 内网目标

#### 文件访问模式
利用 nps 提供一个公网可访问的本地文件服务，**此模式仅客户端配置文件模式方可启动**。

```ini
[common]
server_addr=1.1.1.1:8024
vkey=123
[file]
mode=file
server_port=9100
local_path=/tmp/
strip_pre=/web/
```

项 | 含义
---|---
mode | file
server_port | 服务端开启的端口
local_path | 本地文件目录
strip_pre | 前缀

对于 `strip_pre`，访问公网 `ip:9100/web/` 相当于访问 `/tmp/` 目录。

#### 断线重连
```ini
[common]
auto_reconnection=true
```
