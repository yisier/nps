# NPS

![Version](https://img.shields.io/badge/version-v0.26.39-blue)
![License](https://img.shields.io/badge/license-GPL--3.0-green)
![Go](https://img.shields.io/badge/Go-%3E%3D1.22-00ADD8?logo=go)

> 一款轻量级、高性能、功能强大的内网穿透代理服务器，基于 [ehang-io/nps](https://github.com/ehang-io/nps) 0.26.10 二次开发 —— 修复大量 bug，持续更新中。  
> 💬 聊天灌水 QQ 群：**619833483**（热心群主可提供免费远程协助）

### 📖 在线文档
<small>[完整文档](https://yisier.github.io/nps) · [更新日志](https://yisier.github.io/nps/changelog/) · [程序安装](https://yisier.github.io/nps/install/#%E7%A8%8B%E5%BA%8F%E5%AE%89%E8%A3%85) · [宝塔面板](https://yisier.github.io/nps/install/#%E5%AE%9D%E5%A1%94%E9%9D%A2%E6%9D%BF-%E4%B8%80%E9%94%AE%E9%83%A8%E7%BD%B2) · [Docker安装](https://yisier.github.io/nps/install/#docker-%E5%AE%89%E8%A3%85)</small>

---
## 💰 赞助 / 推荐
#### 【腾讯云】[2核2G4M 服务器新客99元/年起](https://cloud.tencent.com/act/cps/redirect?redirect=6544&cps_key=de079542f18563fceb5c9b687dad2943&from=console)

####  【NATNPS 云穿透】 — [无需自备服务器，注册即用，免费 3M 带宽 / 2 条隧道](https://natnps.com/register?utm_from=MQ==)


---
## 🚀 快速开始

### 下载

从 [releases](https://github.com/yisier/nps/releases) 下载对应平台版本。服务端 `nps` 和客户端 `npc` 是独立的压缩包。

### 服务端

```shell
./nps -server    # 交互菜单：安装/卸载/启动/停止/更新
```

首次启动自动生成 `conf/nps.conf`，随机生成 `web_password`、`auth_key`、`auth_crypt_key`，请从启动日志或 `nps.conf` 获取。

![img](https://raw.githubusercontent.com/yisier/nps/master/docs/.vuepress/public/image/new/server.png)

### 客户端

```shell
./npc            # 直接双击运行，按提示输入即可
```

在 Web 后台复制【快捷启动命令】，客户端粘贴即可注册系统服务、启停或卸载。


![image](https://raw.githubusercontent.com/yisier/nps/master/docs/.vuepress/public/image/new/cmd.png)


#### GUI

![image](https://yisier.github.io/nps/image/new/gui.png)


#### 命令行直接启动

```shell
npc -server=ip:8024 -vkey=xxx                           # 标准
npc -server=ip:8025 -vkey=xxx -tls_enable=true           # TLS 桥接
npc -server=ip:8024 -vkey=vkey1,vkey2                    # 多隧道
```


---

## ☕ 请作者喝杯咖啡
> 如果这个项目帮到了你，可以请作者喝杯咖啡 ☕ 感谢支持！

![微信/支付宝扫码赞赏](docs/.vuepress/public/image/payme.png)


---

## 📈 Star History

[![Star History Chart](https://star-history.dera.page/svg?repos=yisier/nps&type=Date)](https://star-history.dera.page/#yisier/nps&Date)

---

## 📃 License

GPL-3.0 License — 基于 [ehang-io/nps](https://github.com/ehang-io/nps) 0.26.10 二次开发。
