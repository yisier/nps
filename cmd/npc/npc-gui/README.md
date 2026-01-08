# NPS 客户端 GUI (Wails 版本)

一个现代化、可开箱即用的 NPS 客户端 GUI，使用 Wails 框架（Go + Vue3）开发。

## 功能特性

- ✨ 现代化的深色主题 UI
- 🔗 支持快捷命令（Base64 编码）
- 🚀 双击打开即可运行
- 💾 自动保存连接配置
- 🔄 实时连接状态管理
- 📝 连接日志和设置面板（开发中）

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

## 安装与运行

### 前置要求
- Go 1.21+
- Node.js 16+
- Yarn

### 开发模式

```bash
cd npc-gui

# 安装依赖
yarn install

# 运行开发服务器
wails dev
```

### 构建

```bash
# 构建 Windows 可执行文件
wails build -platform windows/amd64

# 构建带 NSIS 安装程序
wails build -platform windows/amd64 -nsis
```

输出文件将在 `build/bin` 目录中。

## 项目结构

```
npc-gui/
├── frontend/              # Vue3 前端
│   ├── src/
│   │   ├── App.vue       # 主应用组件
│   │   ├── main.js       # 入口文件
│   │   └── assets/
│   ├── index.html        # HTML 模板
│   ├── package.json
│   └── vite.config.js    # Vite 配置
├── main.go               # Wails 主入口
├── app.go                # 应用逻辑
├── wails.json            # Wails 配置
└── Makefile             # 构建脚本
```

## 使用说明

1. **添加连接**
   - 复制 Base64 快捷命令到输入框，点击"连接"按钮
   - 或输入原始的连接密钥

2. **管理连接**
   - 使用切换开关启动/停止连接
   - 点击"✕"删除连接

3. **菜单**
   - 📋 连接日志：查看连接日志
   - ⚙️ 设置：应用设置（开发中）

## 配置存储

连接配置自动保存在以下位置：
- Windows: `%APPDATA%\nps\npc_shortcuts.json`
- Linux: `~/.config/nps/npc_shortcuts.json`
- macOS: `~/Library/Application Support/nps/npc_shortcuts.json`

## 开发相关

### 后端 API

主要的 Go 方法通过 Wails 绑定到前端：

- `GetShortcuts()` - 获取所有保存的连接
- `AddShortcutFromBase64(encoded string)` - 从 Base64 添加快捷命令
- `RemoveShortcut(name, addr, key string)` - 删除连接
- `ToggleClient(name, addr, key string, tls bool, running bool)` - 启动/停止连接
- `TestConnection(key string)` - 测试连接

### 前端技术栈

- Vue 3 Composition API
- Vite 4
- 原生 CSS（深色主题）

## 故障排查

### Wails 命令未找到

如果 `wails` 命令未找到，尝试：

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 依赖问题

如果编译失败，尝试：

```bash
go mod tidy
cd frontend && npm install
```

## 许可证

MIT

## 贡献

欢迎提交 Issue 和 Pull Request！
