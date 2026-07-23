# nextunnel-client

`nextunnel-client` 运行在内网侧。它通过 mTLS 主动连接 `nextunnel-server`，使用已注册的客户端 ID 与匹配的客户端证书登录，提交本地代理配置，并把服务端公网端口收到的流量桥接到本地服务。

## 职责

- 使用 TLS 1.2+ 和客户端证书连接服务端。
- 使用 `[client].id` 登录；证书指纹必须属于该客户端。
- 根据 `[[proxies]]` 注册 TCP 代理。
- 当服务端远程端口收到连接时，按需打开工作连接。
- 将每条工作连接转发到 `local_ip:local_port`。
- 断线后自动重连，退避间隔从 2 秒增长到 30 秒；会话建立成功后重置为 2 秒。
- 控制通道每 30 秒发送一次心跳；读空闲超时为 90 秒。

```mermaid
flowchart LR
    User[用户] -->|TCP| Remote[服务端 remote_port]
    Remote --> Server[nextunnel-server]
    Server <-->|mTLS 控制/工作通道| Client[nextunnel-client]
    Client --> Local[local_ip:local_port]
```

## 环境要求

| 依赖 | 说明 |
| --- | --- |
| Go 1.26+ | 仅本地编译时需要。 |
| 客户端 ID | 在服务端通过 Web 控制台或 `nextunnel-server client create` 创建；配置里可用 **name 或 UUID**。 |
| mTLS 文件 | 从服务端生成或下载的 `ca.crt`、`client.crt`、`client.key`。 |

## 快速开始

```bash
# 1. 准备服务端生成的证书。
mkdir -p certs
cp /path/to/client-certs/{ca.crt,client.crt,client.key} certs/

# 2. 复制并编辑客户端配置。
cp nextunnel-client.example.toml nextunnel-client.toml

# 3. 编译并启动客户端。
make build-client
./bin/nextunnel-client-$(cat VERSION) --config nextunnel-client.toml
```

启动后，客户端会加载配置、初始化 mTLS、连接 `[server].host:[server].port`、使用 `[client].id` 登录、提交 `[[proxies]]`，然后进入控制循环。

## 配置说明

完整示例见 [`../../nextunnel-client.example.toml`](../../nextunnel-client.example.toml)。

```toml
[server]
host = "your-server.example.com"
port = 25930

[client]
id = "macbook"

[cert]
ca_file = "certs/ca.crt"
cert_file = "certs/client.crt"
key_file = "certs/client.key"

[[proxies]]
name = "ssh"
type = "tcp"
local_ip = "127.0.0.1"
local_port = 22
remote_port = 5000
```

| 配置段 | 字段 | 说明 |
| --- | --- | --- |
| `[server]` | `host` / `port` | 服务端控制通道地址。 |
| `[client]` | `id` | 已注册的客户端 name 或 UUID，不能为空，且必须与客户端证书匹配。 |
| `[cert]` | `ca_file` / `cert_file` / `key_file` | mTLS 所需 CA 和客户端证书文件；`cert_file` 与 `key_file` 均不能为空。 |
| `[logs]` | `file` / `level` / `maxSize` / `maxBackups` / `maxAge` | 日志输出与保留策略。`level` 仅允许 `info` / `warn` / `error`；`maxSize` 空值按 `100MB`；`maxBackups` / `maxAge` 为 `0` 时表示不按该维度清理。 |
| `[[proxies]]` | `name` | 代理名称，服务端创建工作连接时会引用。 |
| `[[proxies]]` | `type` | 代理类型；当前仅服务端接受 `tcp`。 |
| `[[proxies]]` | `local_ip` / `local_port` | 客户端主机或容器可访问的本地服务地址。 |
| `[[proxies]]` | `remote_port` | 服务端公网侧监听端口。 |

服务端还会校验：`name` / `local_ip` 非空，端口落在 `1–65535`，同一客户端下 `name` 与 `remote_port` 不重复，且 `remote_port` 落在分配的端口范围内（若已配置范围）。

## 代理示例：SSH

```toml
[[proxies]]
name = "ssh"
type = "tcp"
local_ip = "127.0.0.1"
local_port = 22
remote_port = 5000
```

客户端连接成功后，用户可通过 `<服务端主机>:5000` 访问本地 SSH 服务。

如果服务端为该客户端分配了端口范围，`remote_port` 必须落在该范围内。

## CLI 参考

```bash
nextunnel-client [--config <path>]
```

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--config`, `-c` | `nextunnel-client.toml` | 配置文件路径。显式指定时优先生效；未指定时回退到 `NEXTUNNEL_CLIENT_CONFIG`，再否则用默认路径。 |
| `-h`, `--help` | - | 显示帮助。 |
| `-v`, `--version` | - | 显示版本。 |

客户端以前台方式运行；按 `Ctrl+C` 或发送 `SIGTERM` 可优雅退出（约 5 秒超时关闭连接）。

## Docker

客户端 Compose 文件位于 `docker/client`，并使用 host 网络模式，便于 `local_ip` 访问宿主机上的服务。镜像内含 `tzdata`，可通过容器 `TZ` 影响日志时区。

```bash
cd docker/client

# 先准备这些文件：
# volumes/nextunnel/config/nextunnel-client.toml
# volumes/nextunnel/certs/ca.crt
# volumes/nextunnel/certs/client.crt
# volumes/nextunnel/certs/client.key
#
# Docker 下请将证书路径设为 /etc/nextunnel/certs 下的文件，
# 并将 [logs].file 设为 /var/log/nextunnel/nextunnel-client.log。

docker compose up -d
```

客户端容器挂载路径：

| 宿主机路径 | 容器路径 |
| --- | --- |
| `docker/client/volumes/nextunnel/config/nextunnel-client.toml` | `/etc/nextunnel/nextunnel-client.toml` |
| `docker/client/volumes/nextunnel/certs/` | `/etc/nextunnel/certs/` |
| `docker/client/volumes/nextunnel/logs/` | `/var/log/nextunnel/` |

容器默认启动命令：

```bash
nextunnel-client --config /etc/nextunnel/nextunnel-client.toml
```
