# Web API 鉴权

在 `nps.conf` 中配置 `auth_key` 即可启用 API 鉴权（首次启动自动生成）。

## 鉴权方式

每个请求需附带两个参数：

| 参数 | 说明 |
| --- | --- |
| `auth_key` | `md5(配置文件中的 auth_key + 当前时间戳)` |
| `timestamp` | 当前 unix 时间戳（秒） |

时间戳有效范围为 **20 秒**，每次请求须重新生成。

## 获取服务端时间戳

由于客户端与服务端时间可能不一致，可先获取服务端时间：

```
GET /auth/gettime/
```

返回：
```json
{"time": 1717654321}
```

> 此接口无需鉴权。

## 获取服务端 auth_key（加密）

```
GET /auth/getauthkey/
```

返回经 AES-CBC 加密后的 `auth_key`（hex 编码）。

> 此接口无需鉴权。需确保 `nps.conf` 中 `auth_crypt_key` 为 **16 位**。

解密参数：
- 算法：AES-128-CBC
- 密钥：`auth_crypt_key`（16 字节）
- IV：与密钥相同
- 填充：PKCS5Padding
- 密文编码：hex

---

## 接入示例

::: tabs

@tab curl

```bash
# 1. 获取服务端时间戳
ts=$(curl -s http://127.0.0.1:8080/auth/gettime/ | sed 's/.*"time":\([0-9]*\).*/\1/')

# 2. 计算签名（Linux）
sign=$(echo -n "your_auth_key${ts}" | md5sum | awk '{print $1}')
# 或 macOS:
# sign=$(echo -n "your_auth_key${ts}" | md5)

# 3. 调用接口
curl -s -X POST "http://127.0.0.1:8080/client/list/" \
  -d "auth_key=${sign}&timestamp=${ts}&search=&order=asc&offset=0&limit=10"
```

@tab Python

```python
import hashlib, requests

host = "http://127.0.0.1:8080"
auth_key = "your_auth_key"

ts = requests.get(f"{host}/auth/gettime/").json()["time"]
sign = hashlib.md5(f"{auth_key}{ts}".encode()).hexdigest()

r = requests.post(f"{host}/client/list/", data={
    "auth_key": sign, "timestamp": ts,
    "search": "", "order": "asc", "offset": 0, "limit": 10
})
print(r.json())
```

@tab JavaScript

```javascript
const crypto = require("crypto");

const host = "http://127.0.0.1:8080";
const authKey = "your_auth_key";

(async () => {
  const ts = (await (await fetch(`${host}/auth/gettime/`)).json()).time;
  const sign = crypto.createHash("md5").update(`${authKey}${ts}`).digest("hex");

  const r = await fetch(`${host}/client/list/`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({ auth_key: sign, timestamp: ts, search: "", order: "asc", offset: 0, limit: 10 }).toString(),
  });
  console.log(await r.json());
})();
```

:::

---

## 详细接口清单

- [Web API 接口文档](webapi.html)
