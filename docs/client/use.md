## 直接运行

直接双击 `npc`/`npc.exe`，无任何参数，按提示输入即可：

| 菜单项 | 输入内容 |
| --- | --- |
| 注册系统服务 | 【快捷启动命令】或【TLS 快捷启动命令】|
| 卸载系统服务 |  隧道密钥 `vkey` |
| 启动 |  隧道密钥 `vkey` |
| 停止 |  隧道密钥 `vkey` |
| 更新客户端 | — |

【快捷启动命令】在 web 后台 → 客户端列表 → 客户端前的小图标里复制，本质是 `服务端地址 + vkey` 的 base64 编码。TLS 模式有独立的【TLS 快捷启动命令】。

服务以 `nps-client-<vkey>` 命名注册，可在同一台机器并存多个客户端实例。

![image](/image/new/cmd.png)

## 直接命令行

```shell
./npc -server=ip:port -vkey=web界面中显示的密钥         # 标准
./npc -server=ip:8025 -vkey=xxx -tls_enable=true       # TLS 桥接
./npc -server=ip:8024 -vkey=ytkpyr0er676m0r7,iwnbjfbvygvzyzzt  # 同时拉起多个隧道 ID（逗号拼接）
```


## GUI 客户端

> 基于 Wails 开发的桌面 GUI，体积只有 5M，免安装，解压即用。

支持两种添加客户端方式：

1. 粘贴 web 后台的【快捷启动命令】
2. 手动填写 `-server`、`-vkey` 等参数

详细说明请参考 [GUI 客户端 README](https://github.com/yisier/nps/blob/master/cmd/npc/npc-gui/README.md)。

![img](/image/new/gui.png)


## 注册到系统服务
> 实现开机启动、守护进程

```shell
./npc          # linux/darwin（需 sudo）
npc.exe        # windows（管理员 cmd）
```

按菜单选「注册系统服务」并粘贴【快捷启动命令】或【TLS 快捷启动命令】即可，服务名为 `nps-client-<vkey>`。

> Windows 如果需要当客户端退出时自动重启客户端，请按照如图所示配置

![img](/image/windows_client_service_configuration.png)


## 系统服务日志
注册系统服务后日志按 **每个 vkey 一个文件** 保存，命名为 `npc-<vkey>.log`：

- windows：与 `npc.exe` 同级目录
- linux / darwin：`/var/log/`


## 客户端更新
> 运行客户端，会显示当前客户端版本号。
```shell
./npc          # 选择菜单 [5] 更新客户端
```

自动下载并替换 `npc` 二进制。

如果无法成功更新，可以直接下载 releases 压缩包覆盖原有的 `npc` 二进制。
