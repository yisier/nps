# NPS 客户端 GUI (MyGo 原生 UI 版本)

基于 [MyGo](https://github.com/egoist/mygo) 构建的 NPS 客户端图形界面。
界面由 Go 代码用 `ui` 包直接绘制，不启动 WebView，也不需要 Node.js/Yarn：

- 纯 Go、无 cgo，任意平台可交叉编译
- 不需要 WebView2（Windows）或 WebKitGTK（Linux）
- 启动更快、内存占用更低，单个二进制约 22 MB

## 快捷命令格式

快捷命令使用 Base64 编码，解码后的格式为：
```
nps:name|addr|key|tls
```

示例：
```
nps:MyServer|127.0.0.1:8024|mykey123|false
```

编码后的 Base64：
```
bnBzOk15U2VydmVyfDEyNy4wLjAuMTo4MDI0fG15a2V5MTIzfGZhbHNl
```

## 前置要求

- Go 1.27.3+（MyGo 的要求，`go` 命令会按 `go.mod` 自动切换工具链）

MyGo CLI 通过 `go.mod` 的 `tool` 指令锁定版本，无需单独安装：

```powershell
go tool mygo version
```

## 启动 / 调试

```powershell
cd cmd/npc/npc-gui

# 编译并启动应用；修改 Go 代码后自动重新编译、重启
go tool mygo dev
```

## 构建

```powershell
cd cmd/npc/npc-gui

# Windows PowerShell
$env:GOFLAGS = "-tags=npcgui"
go tool mygo build

# macOS / Linux
GOFLAGS="-tags=npcgui" go tool mygo build
```

产物输出到 `bin/<os>-<arch>/`：

- Windows：`npc-gui.exe`，以及 `npc-gui-setup-<version>-<arch>.exe` 安装器
- macOS：`npc-gui.app` 与 `.dmg`
- Linux：`npc-gui` 可执行文件、desktop entry、deb/rpm 等

只要可执行文件时：

```powershell
go tool mygo build -skip-installer -skip-dmg
```

### 跨平台构建

MyGo 不使用 cgo，任意平台都可以交叉编译：

```powershell
$env:GOFLAGS = "-tags=npcgui"
go tool mygo build -platform darwin/universal,windows/amd64,linux/amd64
```

## 减小体积

`mygo build` 默认已经带 `-trimpath -ldflags "-s -w"`，Go 层面的常规优化不需要再做。
当前 `npc-gui.exe` 约 22 MB，大致构成：

- Go 运行时、类型信息与 pclntab：约 15 MB（Go 程序必需，无法去掉）
- mygo 框架（原生 UI、文本排版）：约 3 MB（最小 mygo 原生应用约 10.5 MB）
- NPS 客户端依赖（TLS / GM TLS、KCP + FEC、beego 日志等）：其余

还可以做的：

1. **不链接 WebView 支持**（Windows，本程序只用原生 UI）：

   ```powershell
   $env:GOFLAGS = "-tags=npcgui,mygo_nowebview"
   go tool mygo build
   ```

   约省 0.1 MB，意义不大，但可以彻底不依赖 WebView2 代码。

2. **UPX 压缩**（发布流程已启用，Windows 产物自动压缩）：

   本地手动压：

   ```powershell
   go tool mygo build -skip-installer
   upx --best --lzma bin\windows-amd64\npc-gui.exe
   ```

   22 MB → 约 **6.7 MB**（默认压缩约 8.1 MB）。`release.yml` 的 Windows job 会在打包 zip
   前自动下载 UPX 5.3.0 并执行 `--best --lzma`。需要注意：

   - 部分杀毒软件会对 UPX 壳报毒或直接拦截
   - 启动时多一次解压，约慢几百毫秒
   - UPX 会破坏已有签名；如果以后要签名，应在压缩之后再签
   - 本来就没有符号表，崩溃栈定位不会更差

3. **发布包已经会压缩**：22 MB 的 exe 打成 zip 约 9 MB。

4. 想进一步缩小，只能裁剪 NPS 客户端依赖（KCP/FEC、GM TLS、beego 日志，
   以及 `server/proxy` → `server/tool` → `gopsutil` 这条链），需要改动
   `ehang.io/nps/client` 的导入结构，改动大且容易影响功能，一般不建议。

`mygo.json` 里已通过 `resourcesExclude` 把图标 PNG 排除在安装目录之外，
安装目录只保留 `npc-gui.exe`。

## 项目结构

```
cmd/npc/npc-gui/
├── mygo.json             # 应用名称、版本、打包配置
├── main.go               # 应用入口（窗口生命周期、托盘、退出状态）
├── ui.go                 # 原生界面（视图、状态刷新、对话框、操作）
├── app_bindings.go       # App 状态与客户端管理逻辑
├── tray.go               # 系统托盘
├── update.go             # GitHub Release 检查与热更新
├── logger.go             # 应用日志 / 客户端日志
└── resources/            # 应用图标与托盘图标
```

## 构建标签

- `npcgui`：GUI 客户端构建必须传入（排除 `server/proxy/tcp.go` 中的服务端隧道实现，
  见 `server/proxy/tcp_npcgui.go` 占位实现）。MyGo CLI 会把 `GOFLAGS` 里的
  `-tags=...` 传给底层 `go build`。

## 配置存储

连接配置自动保存在以下位置：
- Windows: `%APPDATA%\npc\npc_data.json`
- Linux: `~/.config/npc/npc_data.json`
- macOS: `~/Library/Application Support/npc/npc_data.json`

客户端日志默认保存在 `<配置目录>/npc/logs/`，可在设置中修改日志目录。
日志页显示的是整个客户端进程的运行日志，格式与 npc 一致：

```
2026/11/19 16:16:30 [I] [client.go:104] Successful connection with server 127.0.0.1:8024
```

同时会写入：
- `logs/npc-client.log`：上述文本格式的完整运行日志（与 npc 控制台/文件日志一致）
- `logs/npc-client-<vkey>.log`：每个客户端单独的 JSON Lines 日志

日志页支持按客户端过滤（按地址/密钥匹配，npc 全局日志里不带地址的行只在"全部日志"里显示），
日志文本可以选中复制，也可以点"复制全部"复制当前筛选结果。
