# 使用

## web 管理

进入 web 界面：`公网IP:web 端口`（默认 `8080`，仓库默认配置 `8081`）。

> v0.26.33 起，**首次启动 `web_username`（默认 `admin`）、`web_password`、`auth_key`、`auth_crypt_key` 均为随机生成**，会打印到终端日志中，请第一时间复制保存。后续如需变更可直接修改 `conf/nps.conf` 后 `nps reload`。

进入 web 管理界面，有详细的说明。

## 服务端管理脚本（v0.26.25+ 推荐）

```shell
./nps -server         # linux/darwin（操作系统服务时需 sudo）
nps.exe -server       # windows（操作系统服务时需管理员）
```

交互菜单覆盖了启动、调试、安装、卸载、启停、重启、**更新**全部场景，v0.26.34 起菜单会显示当前版本，更新前会先比对版本号，已是最新则不再覆盖。

> 客户端有对应的 `npc` 交互菜单，见 [客户端基本使用](/client/use.html)。

## 工作目录与配置文件位置

- `nps` 会从**可执行文件所在目录**寻找 `conf/`、`web/`（v0.26.33 起 web 已打包进二进制，无需 `web/` 目录）。
- 通过 `nps install` 安装系统服务后，配置文件迁移到：
  - linux / darwin：`/etc/nps/conf/nps.conf`
  - windows：`C:\Program Files\nps\conf\nps.conf`
- 想自定义位置请使用 `-conf_path` 参数（v0.26.15+），详见 [启动 → 指定配置目录](/client/run.html#指定配置目录-conf_pathv02615)。

## 服务端配置文件重载
对于 linux、darwin
```shell
 sudo nps reload
```
对于 windows
```shell
 nps.exe reload
```
**说明：** 仅支持部分配置重载，例如 `allow_user_login`、`auth_crypt_key`、`auth_key`、`web_username`、`web_password` 等，未来将支持更多。


## 服务端停止或重启
对于 linux、darwin
```shell
 sudo nps stop|restart
```
对于 windows
```shell
 nps.exe stop|restart
```

## 服务端更新

### 方式一：交互菜单（推荐）

```shell
./nps -server   # 选择 “更新” 菜单项
```

v0.26.34 起会先比对版本号，已是最新则直接退出，避免无意义覆盖。

### 方式二：传统命令

首先 `sudo nps stop` 或 `nps.exe stop` 停止服务，然后：

```shell
sudo nps-update update     # linux
nps-update.exe update      # windows
```

更新完成后再 `nps start` / `nps.exe start` 启动即可。

如果无法成功更新，可直接下载 releases 压缩包覆盖原有的 `nps` 二进制文件。

> 注意：`nps install` 之后的 `nps` 不在原位置，请使用 `whereis nps` 查找具体目录覆盖二进制。
> 自 v0.26.33 起，web 资源已嵌入到二进制；同时静态文件 URL 会带上版本号参数（`?v=xxx`），升级后浏览器会自动刷新缓存，无需手动清缓存或拷贝 `web/` 目录。
