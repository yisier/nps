## 程序安装

从 [releases](https://github.com/yisier/nps/releases) 下载对应系统版本，服务端 (`nps`) 和客户端 (`npc`) 是单独的压缩包。

::: tabs

@tab 服务端
> 首次启动时 `nps` 会在可执行文件同级目录自动生成 `conf/nps.conf`，并随机生成 `web_password`、`auth_key`、`auth_crypt_key`，web 管理界面的访问密码请在conf/nps.conf文件中或启动日志中查看。  


`./npc(.exe) -server`    

![img.png](/image/new/server.png)



@tab 客户端
> 客户端可直接双击运行，通过交互菜单完成安装、卸载、启动等操作。

`./npc(.exe)`


![image](/image/new/cmd.png)

@tab 客户端(GUI)
> 通过Web 后台的【快捷启动命令】或 手动填写 `-server`、`-vkey` 等参数配置客户端  

![img](/image/new/gui.png)


:::

## Docker 安装

::: tabs
@tab 服务端

先拉取镜像：

```shell
docker pull yisier1/nps
```

启动容器，请把 `<本机conf目录>` 替换为实际路径（例如 `/root/nps/conf`）：

```shell
docker run -d \
  --restart=always \
  --name nps \
  --net=host \
  -v <本机conf目录>:/conf \
  -v /etc/localtime:/etc/localtime:ro \
  yisier1/nps
```

@tab 客户端

先拉取镜像：

```shell
docker pull yisier1/npc
```

启动容器，请把 `<>` 内的值替换为实际配置：

```shell
docker run -d \
  --restart=always \
  --name <自定义名称> \
  --net=host \
  yisier1/npc \
  -server=<服务器IP:端口> \
  -vkey=<密钥>
```
::: 


镜像主页：[NPS](https://hub.docker.com/r/yisier1/nps) ｜ [NPC](https://hub.docker.com/r/yisier1/npc)。

## 宝塔面板（一键部署）

> 推荐宝塔版本 9.2.0+，前往 [宝塔面板官网](https://www.bt.cn/new/download.html) 安装。

![宝塔面板](/image/bt/bt1.jpg)

### 安装宝塔面板

1. 前往 [宝塔面板官网](https://www.bt.cn/new/download.html)，选择正式版脚本安装。
2. 登录面板，点击左侧 **Docker** 进入 Docker 管理。
3. 如提示未安装 Docker / Docker Compose，可根据上方引导安装。

### 安装 NPS 服务端

1. 在宝塔面板 Docker 菜单进入 **应用商城**，搜索 `NPS`，找到 **NPS 服务端** 并点击安装。
2. 进入安装目录，修改 `conf/nps.conf` 即可（修改后重启容器）。参数说明详见 [配置文件](/server/server_config.html)。

   ![配置截图](/image/bt/bt2.png)

> **注意**：NPS 默认占用的端口为 `80`、`443`，如被占用请修改 `nps.conf` 中的 `http_proxy_port` 和 `https_proxy_port`。Web 管理默认端口 `8081`，启动后访问 `http://<IP>:8081`。

### 安装 NPS 客户端

1. 在宝塔面板 Docker 菜单进入 **应用商城**，搜索 `NPS`，找到 **NPS 客户端** 并点击安装。

2. 客户端支持两种配置方式：

   - **无配置文件模式（推荐）**：输入 `服务地址`、`连接密钥` 即可启动。如需多个客户端，安装多个即可（名称不要重复）。
   - **配置文件模式**：`服务地址`、`连接密钥` 留空直接安装，然后在安装目录下找到 `conf/npc.conf` 修改配置并重启容器。详见 [配置文件说明](/client/use.html#配置文件模式)。

   ![客户端配置](/image/bt/bt3.png)

> **推荐使用无配置文件模式**，所有数据在服务端保存和配置，客户端只做连接转发。配置文件模式对新手不友好，容易出错。



## 源码安装

需要 Go 1.22 及以上（仓库 `go.mod` 中 `toolchain` 锁定为 `go1.24.9`）。

```shell
git clone https://github.com/yisier/nps.git
cd nps
go build -o nps cmd/nps/nps.go
go build -o npc cmd/npc/npc.go
```