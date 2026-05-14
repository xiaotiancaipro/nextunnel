<div align="center">

<h1 style="border-bottom: none"><b>Nextunnel</b></h1>

**下一代内网穿透工具（Next-Generation Intranet Tunnel）**

反向隧道 · 出站即连 · 传输层<strong>双向 TLS（mTLS）</strong>原生内建 · Go 单体二进制

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-see%20submodules-blue)](./LICENSE)

<br>

[**快速开始**](#环境要求) · [**对比**](#与其他开源项目的对比) · [**未来特性**](#未来特性)

</div>

## 什么是 Nextunnel

**Nextunnel** 是一套专注于 **出站反向隧道** 的组件：**服务端（nextunnel-server）** 暴露在公网的端口接收连接；
**客户端（nextunnel-client）** 从内网出站建立 **TLS 1.2+** 控制连接，注册多条 **TCP** 转发规则。**控制面和数据面均采用 TLS**
；服务端 **RequireAndVerifyClientCert**，仅以 **同一 CA** 下发的客户端证书作为接入凭证——这是相对许多「单 token + 可选
TLS」方案的差异点。

> 我们不会自称在「协议种类」「生态插件」「Web 控制台成熟度」上已经超越所有前辈；NEXT 一代体现在安全默认值、链路模型与演进空间

## 核心理念与特性

1. mTLS 即接入控制：接入不仅依赖「知道地址与端口」，还依赖 **有效客户端证书**；服务端可签发 `client.crt` / `client.key`，与 *
   *`ca.crt` 信任链**
   绑定，更接近设备 / workload 入网模型。
2. 面向自动化的运维体验：服务端 `tls.dir` **四件套全无**时可 **一键生成 CA + 服务端证书**；支持 **`--generate-certs`**
   为边缘节点批量签发客户端证书（存在则拒绝覆盖）。
3. 健壮性：控制连接断开后客户端 **2s～30s 指数退避** 自动重连；服务端 **`ip_blacklist`** 可作粗粒度来源过滤。

## 与其他开源项目的对比

| 能力                             | **Nextunnel** | **frp**（[fatedier/frp](https://github.com/fatedier/frp)） | **nps**（[ehang-io/nps](https://github.com/ehang-io/nps)） |
|--------------------------------|:-------------:|:--------------------------------------------------------:|:--------------------------------------------------------:|
| **TCP 反向穿透**                   |       ✅       |                            ✅                             |                            ✅                             |
| **UDP 穿透**                     |      🔜       |                            ✅                             |                            ✅                             |
| **HTTP / HTTPS 路由**            |      🔜       |                            ✅                             |                            ✅                             |
| **控制与数据链路默认全程 TLS**            |       ✅       |                            △                             |                            △                             |
| **接入侧默认 mTLS（校验客户端证书）**        |       ✅       |                            ❌                             |                            ❌                             |
| **内置 CA、一键 bootstrap、签发客户端证书** |       ✅       |                            ❌                             |                            ❌                             |
| **多用户登录体系**                    |      🔜       |                            △                             |                            ✅                             |
| **用量统计**                       |      🔜       |                            △                             |                            ✅                             |
| **Web 管理 / 控制页面**              |      🔜       |                            △                             |                            ✅                             |
| **证书策略增强（吊销窗口）**               |      🔜       |                            △                             |                            △                             |

**✅** 已具备 **❌** 当前不具备 **△** 视配置或生态、非开箱默认 **🔜** 未来支持

## 环境要求

- Go **1.26+**
- 服务端与客户端均依赖 **互为信任的证书**：CA、服务端证书、**客户端证书**

## 从源码构建

```bash
# 服务端
cd nextunnel-server
go build -o nextunnel-server .

# 客户端
cd nextunnel-client
go build -o nexttunnel-client .
```

## TLS 与证书

### 服务端证书目录

服务端配置中的 `[tls] dir` 指向一个目录，其中应包含（或自动生成）：

| 文件                          | 说明                 |
|-----------------------------|--------------------|
| `ca.crt` / `ca.key`         | CA 根证书与私钥          |
| `server.crt` / `server.key` | 服务端证书与私钥（由该 CA 签发） |

若上述四个文件 **均不存在**，首次启动服务端时会在该目录 **自动生成一整套 CA + 服务端证书**（SAN 会结合 `[server] addr`
生成，便于本机或与域名/IP 匹配的校验）。

若目录里 **只有部分文件**，启动会报错，需凑齐四套或清空后由程序生成。

服务端监听地址在代码中为 **本机所有接口上的 `port`**（即 `ListenTCP` 使用 `:port`）；`server.addr` 主要用于证书 SAN
等与主机名相关的场景，并不等于「绑定地址」。

### 签发客户端证书

服务端支持在已有 CA（即 `tls` 目录完整）的前提下，签发供客户端使用的证书：

```bash
cd nextunnel-server
./nextunnel-server --config nextunnel-server.toml --generate-certs /path/to/client-certs-dir
```

会在目标目录写入 `client.crt` 与 `client.key`。若同名文件已存在则失败，避免误覆盖。

将 **`ca.crt`** 与 **`client.crt` / `client.key`** 复制到客户端可访问的路径，并在客户端配置的 `[tls]` 中填入对应文件路径。

## 配置说明

服务端与客户端均使用 **TOML** 配置文件，可用 `-c` / `--config` 指定路径；默认分别为 `nextunnel-server.toml` 与
`nextunnel-client.toml`。

仓库内示例：

- `nextunnel-server/nextunnel-server.example.toml`
- `nextunnel-client/nextunnel-client.example.toml`

## 运行

```bash
# 服务端
./nextunnel-server --config nextunnel-server.toml

# 客户端
./nextunnel-client --config nextunnel-client.toml
```

## 使用 Docker Compose

各子项目自带 `docker-compose.yaml`，采用 `network_mode: host`，并将配置、证书、日志挂到宿主目录：

- 服务端：`nextunnel-server/volumes/` 下的 `config`、`certs`、`logs`
- 客户端：`nextunnel-client/volumes/` 下同样结构

容器内默认命令分别为：

- `nextunnel-server`：`--config config/nextunnel-server.toml`
- `nextunnel-client`：`--config config/nextunnel-client.toml`

请把宿主上的 TOML 与证书放进对应 `volumes/config` / `volumes/certs`。

## 未来特性

1. **代理类型**：UDP、HTTP/HTTPS 等。
2. **认证与密钥**：加强签发证书逻辑；支持证书有效/吊销时间与校验策略等。
3. **多用户**：多用户登录与统计 / 租户维度的用量与控制。
4. **WEB 页面**：服务端与客户端的 Web 控制台。

## 安全提示

- 认证依赖 **客户端证书**：请妥善保管 `ca.key`、各客户端 `client.key`。
- 若暴露 CA 签发能力或未限制 `remote_port`，风险较高——建议配合防火墙、`ip_blacklist` 及对公网映射端口的收口策略使用。

## 贡献

欢迎在对应子仓库提交 Issue / PR，**功能广度**与**安全默认策略**一起讨论。
