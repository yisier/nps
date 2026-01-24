# NPS 客户端 GUI (Wails 版本)

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


## 配置存储

连接配置自动保存在以下位置：
- Windows: `%APPDATA%\npc\npc_data.json`
- Linux: `~/.config/npc/npc_data.json`
- macOS: `~/Library/Application Support/npc/npc_data.json`


