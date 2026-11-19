# NPS 客户端 GUI (MyGo 版本)

基于 [MyGo](https://github.com/egoist/mygo) 构建的 NPS 客户端图形界面（纯 Go，无 cgo）。

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
- Node.js 22.6+ 与 Yarn（构建/调试 Vue 前端）
- 运行 Windows 版本需要 WebView2 Runtime（Windows 11 已内置）

MyGo CLI 通过 `go.mod` 的 `tool` 指令锁定版本，无需单独安装：

```powershell
go tool mygo version
```

## 启动 / 调试

```powershell
cd cmd/npc/npc-gui

# 启动开发模式：生成 TypeScript 客户端、运行 Vite dev server、
# 编译并启动应用；修改 Go 代码后自动重启
go tool mygo dev
```

开发应用会加载 `http://127.0.0.1:34115`（见 `mygo.json` 的 `devUrl`），
前端仍支持 Vite 热更新。开发构建自带 Web 检查器（右键 → 检查）。

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

只要一份安装包/压缩包时：

```powershell
go tool mygo build -skip-installer -skip-dmg
```

### 跨平台构建

MyGo 不使用 cgo，任意平台都可以交叉编译：

```powershell
$env:GOFLAGS = "-tags=npcgui"
go tool mygo build -platform darwin/universal,windows/amd64,linux/amd64
```

### 生成前端绑定

Go 侧绑定方法签名变更后，重新生成类型化客户端：

```powershell
go tool mygo generate
```

生成结果位于 `frontend/src/mygo.ts`（已提交）。`mygo dev` 与 `mygo build`
会自动重新生成。

## 项目结构

```
cmd/npc/npc-gui/
├── mygo.json             # 应用名称、版本、前端构建与打包配置
├── main.go               # 应用入口（绑定服务、窗口生命周期、托盘）
├── app_bindings.go       # App 服务（前端可调用的方法）与客户端逻辑
├── tray.go               # 系统托盘
├── update.go             # GitHub Release 检查与热更新
├── logger.go             # 应用日志 / 客户端日志
├── resources/            # 应用图标与托盘图标
└── frontend/
    ├── src/App.vue       # 主界面
    ├── src/mygo.ts       # go tool mygo generate 生成（已提交）
    └── dist/             # 前端构建产物（由 mygo build 嵌入二进制）
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
