<div align="center">

<h1 style="border-bottom: none"><b>Nextunnel</b></h1>

**下一代内网穿透工具**

反向隧道 · 出站即连 · 传输层 mTLS 原生内建 · Go 单体二进制

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-Apache%20License%20Version%202.0-blue)](./LICENSE)

<a href="./README.md"><img alt="README in English" src="https://img.shields.io/badge/English-d9d9d9"></a>
<a href="./README_zh.md"><img alt="简体中文文件" src="https://img.shields.io/badge/简体中文-d9d9d9"></a>

**[nextunnel-server](https://github.com/xiaotiancaipro/nextunnel-server)** ·
**[nextunnel-client](https://github.com/xiaotiancaipro/nextunnel-client)**

</div>

## 什么是 Nextunnel

**Nextunnel** 是一套专注于 出站反向隧道 的组件：**服务端（nextunnel-server）** 暴露在公网的端口接收连接；
**客户端（nextunnel-client）** 从内网出站建立 TLS 1.2+ 控制连接，注册多条 TCP 转发规则。控制面和数据面均采用 TLS
；服务端 RequireAndVerifyClientCert，仅以同一 CA 下发的客户端证书作为接入凭证——这是相对许多「单 token + 可选
TLS」方案的差异点。

## 核心理念与特性

1. mTLS 即接入控制：接入不仅依赖「知道地址与端口」，还依赖有效客户端证书；服务端可签发 `client.crt` / `client.key`，与
   `ca.crt` 信任链 绑定，更接近设备 / workload 入网模型。
2. 面向自动化的运维体验：服务端 `tls.dir` 四件套全无时可一键生成 CA + 服务端证书；支持 `--generate-certs`
   为边缘节点批量签发客户端证书（存在则拒绝覆盖）。
3. 健壮性：控制连接断开后客户端 2s～30s 指数退避自动重连；服务端 `ip_blacklist` 可作粗粒度来源过滤。

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

✅ 已具备

❌ 当前不具备

△ 视配置或生态、非开箱默认

🔜 未来支持

## 未来特性

1. **代理类型**：UDP、HTTP/HTTPS 等。
2. **认证与密钥**：加强签发证书逻辑；支持证书有效/吊销时间与校验策略等。
3. **多用户**：多用户登录与统计 / 租户维度的用量与控制。
4. **WEB 页面**：服务端与客户端的 Web 控制台。

## 贡献

欢迎在对应子仓库提交 Issue / PR，功能广度与安全默认策略一起讨论。
