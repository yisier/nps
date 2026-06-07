# 启动

## 服务端

### 方式一：交互菜单（v0.26.25+ 推荐）

> v0.26.34 起菜单欢迎语会显示当前版本号；执行更新时会先比对版本，已是最新则直接退出。

下载并解压服务端压缩包，进入目录后执行：

```shell
./nps -server            # linux / darwin
nps.exe -server          # windows
```

> linux/darwin 安装/卸载系统服务时需 `sudo`，windows 需以管理员身份运行 cmd。

会进入一个交互菜单，按提示输入即可完成以下任意操作：

- 启动 / 调试运行
- 安装为系统服务（开机自启 + 后台守护）
- 卸载系统服务
- 启动 / 停止 / 重启 已安装的服务
- 更新到最新版本

![img](/image/new/server.png)

### 方式二：传统命令行（兼容旧版）

- 安装：`sudo ./nps install` 或 `nps.exe install`
- 启动：`sudo nps start` 或 `nps.exe start`
- 停止 / 重启：`stop` / `restart`
- 卸载：`uninstall`
- 重载部分配置：`reload`

```
安装后 windows 配置文件位于 C:\Program Files\nps；
linux 与 darwin 位于 /etc/nps。
```

如果发现没有启动成功，可以使用 `nps(.exe) stop` 停止，然后直接运行 `./nps`（不带 install）排错。
日志文件：windows 在当前运行目录，linux/darwin 在 `/var/log/nps.log`。

### 指定配置目录 `-conf_path`（v0.26.15+）

默认 nps 在**可执行文件所在目录**寻找 `conf/`。如需指定其他位置（典型场景：在多个目录跑多份 nps），可使用：

```shell
# 直接启动
./nps -conf_path=/app/nps             # linux/darwin
nps.exe -conf_path=D:\test\nps        # windows

# 安装为服务时也支持
./nps install -conf_path=/app/nps
nps.exe install -conf_path=D:\test\nps

# 已安装服务后启动：参数已写入服务定义，直接 start 即可
nps start
nps.exe start
```

### 首次启动

- v0.26.33 起，若 `conf` 目录或 `nps.conf` 不存在会**自动创建并写入默认配置**，方便 Docker 部署。
- 首次启动时的 `web_username`（默认 `admin`）、`web_password`、`auth_key`、`auth_crypt_key` **均为随机生成**，并打印到终端。请第一时间从终端日志中复制保存，否则只能改配置文件后重启。
- 访问 `服务端 IP:web 端口`（默认 `8080`，仓库默认配置为 `8081`），用上述账号登录。
- 创建客户端，进入下一步。

## 客户端

### 方式一：交互菜单（v0.26.21+ 推荐）

> v0.26.34 起菜单欢迎语会显示当前版本号，并新增 `[5] 更新客户端` 选项可一键替换 `npc` 二进制。

直接双击 `npc`（或 `npc.exe`），无需任何命令行参数，按菜单输入：

- **启动 / 安装服务**：粘贴 web 管理界面 → 客户端列表的【快捷启动命令】或【TLS 快捷启动命令】（一段 base64）
- **停止 / 卸载 / 重启服务**：只需输入隧道密钥 `vkey`

安装服务时会以 `nps-client-<vkey>` 命名注册，支持在同一台机器跑多个 npc 实例。

![img](/image/new/cmd.png)

### 方式二：命令行（自动化 / 容器场景）

在 web 管理界面 → 客户端列表，点击客户端前的 `+` 复制启动命令，例如：

```shell
./npc -server=1.1.1.1:8024 -vkey=客户端的密钥             # linux/darwin
npc.exe -server=1.1.1.1:8024 -vkey=客户端的密钥           # windows
npc.exe -server=1.1.1.1:8025 -vkey=客户端的密钥 -tls_enable=true   # TLS 模式
```

- v0.26.19 起支持多个隧道 ID 用英文逗号拼接：`-vkey=ytkpyr0er676m0r7,iwnbjfbvygvzyzzt`
- 若使用 `powershell` 运行，**请用引号把 ip:port 包起来**

需要注册系统服务请看 [注册到系统服务](/client/use.html#注册到系统服务)。

### GUI 客户端（v0.26.29+，仅 Windows）

基于 Wails 的桌面 GUI，需要 [WebView2](https://developer.microsoft.com/microsoft-edge/webview2/) 运行时。
两种添加客户端方式：

1. 粘贴快捷启动命令
2. 手动填写 `server`、`vkey` 等参数

详细使用见 [GUI 客户端 README](https://github.com/yisier/nps/blob/master/cmd/npc/npc-gui/README.md)。

![img](/image/new/gui.png)

## 版本检查
- 对服务端和客户端均可使用参数 `-version` 打印版本：
- `nps -version` 或 `./nps -version`
- `npc -version` 或 `./npc -version`

## 配置
- 客户端连接后，在 web 管理界面配置对应穿透服务即可。
- 进一步参考 [使用示例](/extend/example.html)。
