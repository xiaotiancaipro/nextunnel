<div align="center">

<h1 style="border-bottom: none"><b>nextunnel-client</b></h1>

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-Apache%20License%20Version%202.0-blue)](./LICENSE)

<a href="./README.md"><img alt="README in English" src="https://img.shields.io/badge/English-d9d9d9"></a>
<a href="./README_zh.md"><img alt="简体中文" src="https://img.shields.io/badge/简体中文-d9d9d9"></a>

</div>

## 概述

`nextunnel-client` 是 [nextunnel](https://github.com/xiaotiancaipro/nextunnel) 反向隧道系统的**客户端**组件，通过
mTLS 连接 [nextunnel-server](https://github.com/xiaotiancaipro/nextunnel-server)，向服务端注册本地代理配置，并将公网入站流量转发到客户端所在机器上的内网服务。

主要能力：

- 通过双向 TLS（mTLS）连接服务端
- 注册 TCP 代理，并在服务端申请远程监听端口
- 将每条入站用户连接桥接到本地 `local_ip:local_port` 目标
- 断线后自动重连（指数退避，2 秒 → 30 秒）
- 周期性发送心跳维持控制通道

```mermaid
flowchart LR
    User[用户] -->|TCP| Proxy[服务端代理端口]
    Proxy --> Server[nextunnel-server]
    Server <-->|mTLS 控制 + 工作通道| Client[nextunnel-client]
    Client --> Target[本地目标]
```

## 环境要求

| 依赖       | 说明                                                                                                                      |
|----------|-------------------------------------------------------------------------------------------------------------------------|
| Go 1.26+ | 仅本地编译时需要                                                                                                                |
| mTLS 证书  | 在服务端执行 `nextunnel-server client generate-certs` 生成（参见 [服务端 README](https://github.com/xiaotiancaipro/nextunnel-server)） |

## 快速开始

```bash
# 1. 在服务端主机生成客户端证书
# nextunnel-server client generate-certs ./client-certs

# 2. 复制证书与配置
mkdir -p certs
cp /path/to/client-certs/{ca.crt,client.crt,client.key} certs/
cp nextunnel-client.example.toml nextunnel-client.toml
# 编辑 nextunnel-client.toml：服务端地址、客户端 ID、代理列表、证书路径、时区

# 3. 编译并启动（默认读取 nextunnel-client.toml）
go build -o nextunnel-client .
./nextunnel-client
```

启动后程序会：加载配置 → 初始化 mTLS → 连接服务端 → 使用 `[client].id` 登录 → 向服务端提交 `[[proxies]]` →
进入控制循环（心跳 + 工作连接）。

> `[client].id` **必填**；服务端会拒绝空的客户端 ID。

### 多平台构建

```bash
./script/build.sh
```

二进制文件输出至 `bin/`，命名格式为 `nextunnel-client-<version>-<os>-<arch>[.exe]`。

## Docker 部署

`docker/` 目录提供 Compose 编排，使用 **host 网络模式**（便于 `local_ip` 访问宿主机上的服务）。

```bash
cd docker

# 将证书放入 volumes/certs/（ca.crt、client.crt、client.key）
# 编辑 volumes/config/nextunnel-client.toml（服务端地址、客户端 ID、代理列表、时区）

docker compose up -d
```

容器内挂载路径：

| 宿主机路径             | 容器路径                           |
|-------------------|--------------------------------|
| `volumes/config/` | `/usr/local/nextunnel/config/` |
| `volumes/certs/`  | `/usr/local/nextunnel/certs/`  |
| `volumes/logs/`   | `/usr/local/nextunnel/logs/`   |

默认启动命令：`nextunnel-client --config config/nextunnel-client.toml`。

## CLI 参考

```bash
nextunnel-client [--config <path>]    # 启动客户端（前台）
```

| 标志                | 默认值                     | 说明     |
|-------------------|-------------------------|--------|
| `--config`, `-c`  | `nextunnel-client.toml` | 配置文件路径 |
| `-h`, `--help`    | —                       | 显示帮助   |
| `-v`, `--version` | —                       | 显示版本   |

无子命令，以前台方式运行；按 `Ctrl+C` 或发送 `SIGTERM` 可优雅退出。

## 配置说明

完整示例见 [`nextunnel-client.example.toml`](nextunnel-client.example.toml)。

| 配置段           | 字段                                   | 说明                                    |
|---------------|--------------------------------------|---------------------------------------|
| `[server]`    | `addr` / `port`                      | nextunnel-server 控制通道地址               |
| `[client]`    | `id`                                 | 客户端标识（必填；同一时刻每个连接应唯一）                 |
| `[cert]`      | `ca_file` / `cert_file` / `key_file` | mTLS 所需的 CA 与客户端证书路径                  |
| `[logs]`      | `file`                               | 日志路径（按天轮转，超出大小自动分段）                   |
|               | `level`                              | `debug` / `info` / `warn` / `error`   |
|               | `maxSize`                            | 单段最大大小，如 `100MB`、`1GB`；纯数字默认为 MB      |
|               | `maxBackups`                         | 保留的按天日志文件数量上限                         |
|               | `maxAge`                             | 日志最大保留天数                              |
| `[timezone]`  | `location`                           | 日志展示与按天轮转的 IANA 时区，默认 `Asia/Shanghai` |
| `[[proxies]]` | `name`                               | 代理名称（服务端建立工作连接时引用）                    |
|               | `type`                               | 代理类型；当前支持 `tcp`                       |
|               | `local_ip` / `local_port`            | 本地转发目标地址与端口                           |
|               | `remote_port`                        | 服务端为该代理监听的远程端口                        |

### 示例：通过远程端口暴露 SSH

```toml
[[proxies]]
name = "ssh"
type = "tcp"
local_ip = "127.0.0.1"
local_port = 22
remote_port = 8022
```

客户端连接成功后，用户可通过 `<服务端主机>:8022` 访问本地 SSH 服务。

## 许可证

本项目采用 [Apache License 2.0](./LICENSE)。
