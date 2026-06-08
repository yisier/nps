## Web 管理

进入 web 界面：`公网IP:web 端口`（默认  `8081`）。

> **首次启动 `web_username`（默认 `admin`）、`web_password`、`auth_key`、`auth_crypt_key` 均为随机生成**，会打印到终端日志中，请第一时间复制保存。后续如需变更可直接修改 `conf/nps.conf` 后 `nps reload`。

进入 web 管理界面，有详细的说明。

## 服务端管理脚本


```shell
./nps -server         # linux/darwin（操作系统服务时需 sudo）
nps.exe -server       # windows（操作系统服务时需管理员）
```

交互菜单覆盖了安装、卸载、启停、重启、更新等全部场景。

![img.png](/image/new/server.png)

## 工作目录与配置文件位置

- nps安装系统服务后，配置文件默认在当前目录，目录结构为：
```
nps_dir/
├── conf/
│   └── nps.conf    // nps主配置文件
│   └── client.json // 客户端列表配置文件
│   └── tasks.json  // 隧道列表配置文件
│   └── hosts.json  // 域名解析列表配置文件
│   └── global.json // 全局参数配置文件
├── nps // nps主程序
```


## 更新


```shell
./nps -server   # 选择 “更新” 菜单项
```

更新只会替换 `nps` 二进制文件，对配置文件和用户数据无影响，更新完成后重新启动即可。
>如果无法成功更新，可直接下载 releases 压缩包覆盖原有的 `nps` 二进制文件。


## 数据迁移

数据迁移只需迁移 `conf` 目录，该目录包含配置文件以及用户数据（json格式文件），放到 nps(.exe) 的同级目录下，直接启动即可。

