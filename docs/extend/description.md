# 说明
## 获取用户真实ip
如需使用需要在`nps.conf`中设置`http_add_origin_header=true`

在域名代理模式中，可以通过request请求 header 中的 X-Forwarded-For 和 X-Real-IP 来获取用户真实 IP。

**本代理前会在每一个http(s)请求中添加了这两个 header。**

## 热更新支持
对于绝大多数配置，在web管理中的修改将实时使用，无需重启客户端或者服务端

## 客户端地址显示
在web管理中将显示客户端的连接地址

## 流量统计
可统计显示每个代理使用的流量，由于压缩和加密等原因，会和实际环境中的略有差异

## 当前客户端带宽
可统计每个客户端当前的带宽，可能和实际有一定差异，仅供参考。

## 客户端与服务端版本对比
为了程序正常运行，客户端与服务端的核心版本必须一致，否则将导致客户端无法成功连接致服务端。

## Linux系统限制
默认情况下linux对连接数量有限制，对于性能好的机器完全可以调整内核参数以处理更多的连接。
`tcp_max_syn_backlog` `somaxconn`
酌情调整参数，增强网络性能

## web管理保护
当一个 ip 连续登陆失败次数超过 10 次，将在一分钟内禁止该 ip 再次尝试。

可在 `nps.conf` 设置 `open_captcha=true` 进一步开启图形验证码（2022-10-27+）。

## 首次启动随机凭据（v0.26.33+）

为避免默认密码被恶意扫描，**v0.26.33 起首次启动时** `web_username`（默认 `admin`）、`web_password`、`auth_key`、`auth_crypt_key` 全部随机生成，并直接打印到终端：

```text
[INFO] generated random web_password: <xxxxxxxx>
[INFO] generated random auth_key: <xxxxxxxx>
[INFO] generated random auth_crypt_key: <xxxxxxxxxxxxxxxx>
```

请在第一次启动时妥善记录。后续可在 `nps.conf` 中手工修改并执行 `nps reload`。

## 客户端到期时间（v0.26.34+）

在创建 / 修改客户端时可填写「到期时间」（可留空表示永不过期）。到期后该客户端会被**自动暂停**，所有隧道停止服务，直到管理员手工延长或清空到期时间。
支持格式：`2006-01-02 15:04:05` / `2006-01-02 15:04` / `2006-01-02T15:04:05` / `2006-01-02T15:04` / `2006-01-02`。
