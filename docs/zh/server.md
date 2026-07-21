# nextunnel-server

`nextunnel-server` 是 Nextunnel 的公网侧组件。它接收客户端的 mTLS 连接，监听公网 TCP 代理端口，执行访问规则，并把被允许的流量转发到对应客户端后的内网服务。同一二进制还内嵌 Web 控制台与 HTTP 管理 API。

## 职责

- 使用 TLS 1.2+，并通过 `RequireAndVerifyClientCert` 校验客户端证书。
- 将每次客户端登录绑定到该客户端 ID 已登记的证书指纹。
- 在 PostgreSQL 中保存客户端、证书、代理、访问规则和访问日志。
- 根据客户端提交的代理配置监听远程端口。
- 可选地通过小甜菜科技 IP API 查询归属地，并用于国家、省/区域、城市规则。
- 在 `[server_web]` 上提供内嵌管理 UI 与 `/api` 接口。

```mermaid
flowchart LR
    User[用户] -->|TCP| Proxy[服务端代理端口]
    Proxy --> Server[nextunnel-server]
    Server <-->|mTLS 控制/工作通道| Client[nextunnel-client]
    Client --> Target[内网目标]
    Server --> PG[(PostgreSQL)]
    Server --> IPLoc[小甜菜科技 IP API]
    Admin[管理员] -->|HTTP| Web[server_web UI / API]
    Web --> Server
```

## 环境要求

| 依赖 | 说明 |
| --- | --- |
| Go 1.26+ | 本地编译时需要。 |
| Node.js / npm | 构建内嵌 Web UI（`web/server`）时需要。 |
| PostgreSQL | 保存客户端、证书、代理、访问规则和访问日志。 |
| IP 归属地 API Key | 可选配置 `[ip_location].api_key`；未配置时地理规则没有归属地数据。 |

## 快速开始

```bash
# 1. 准备 PostgreSQL，或用 Docker Compose 只启动 PostgreSQL。
cd docker/server
cp example.env .env
docker compose -f docker-compose.middleware.yaml up -d
cd ../..

# 2. 复制并编辑服务端配置。
cp nextunnel-server.example.toml nextunnel-server.toml

# 3. 编译并启动服务端（会自动执行 npm 构建）。
make build-server
./bin/nextunnel-server-$(cat VERSION) --config nextunnel-server.toml
```

启动后，服务端会加载配置、连接 PostgreSQL、执行迁移、初始化 IP 归属地客户端、监听 `0.0.0.0:<server.port>`、在 `[server_web].host:<port>` 启动 Web，并确保 `[cert].dir` 下存在 `ca.crt`、`ca.key`、`server.crt`、`server.key`。

使用示例默认配置时，可打开 `http://127.0.0.1:25001` 访问控制台。

## 接入客户端

常见流程是：创建客户端记录、创建证书、下载证书对、复制 `ca.crt`，然后配置 `nextunnel-client`。可通过 Web 控制台或 CLI 完成。

```bash
# 创建客户端。省略端口范围表示允许使用任意 remote_port。
nextunnel-server --config nextunnel-server.toml client create --port-start 5000 --port-end 5005 macbook

# 创建客户端证书。未指定 --expires-at 时，应用会将其视为长期有效证书。
nextunnel-server --config nextunnel-server.toml client cert create macbook
nextunnel-server --config nextunnel-server.toml client cert list macbook

# 用证书 ID 下载 client.crt/client.key。
nextunnel-server --config nextunnel-server.toml client cert download --dir ./client-certs macbook <cert-id>

# 同时从服务端证书目录复制 CA 证书。
cp certs/ca.crt ./client-certs/
```

然后配置客户端：

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

客户端连接后，服务端会把它的 `[[proxies]]` 同步到 PostgreSQL。客户端在线时代理标记为在线，断开后标记为离线。如果客户端配置了端口范围，每个 `remote_port` 都必须在该范围内。

登录时还要求所出示的客户端证书指纹属于所声明的 `[client].id`。即使证书由本 CA 签发，也不能冒用其他客户端 ID。

## CLI 参考

```bash
nextunnel-server [--config <path>]
nextunnel-server client create [--port-start <n>] [--port-end <n>] <name>
nextunnel-server client cert create [--expires-at <RFC3339>] <name>
nextunnel-server client cert list <name>
nextunnel-server client cert download [--dir <output-dir>] <name> <cert-id>
nextunnel-server client cert delete <name> <cert-id>
nextunnel-server ip-filter list
nextunnel-server ip-filter add [--allow | --block] [--ip | --country | --region | --city | --all | --local | --remote] [value]
nextunnel-server ip-filter delete [--allow | --block] [--ip | --country | --region | --city | --all | --local | --remote] [value]
```

全局参数：

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--config` | `nextunnel-server.toml` | 配置文件路径；未指定时加载器可回退到 `NEXTUNNEL_SERVER_CONFIG`。 |
| `-h`, `--help` | - | 显示帮助。 |
| `-v`, `--version` | - | 显示版本。 |

## 访问规则

规则存储在 PostgreSQL 中，服务端运行时即时生效，无需重启。

```bash
nextunnel-server ip-filter add --allow --ip 203.0.113.10
nextunnel-server ip-filter add --block --city Shenzhen
nextunnel-server ip-filter add --allow --region Guangdong
nextunnel-server ip-filter add --block --country China
nextunnel-server ip-filter add --block --all
nextunnel-server ip-filter add --allow --local
nextunnel-server ip-filter add --block --remote
```

| 项目 | 说明 |
| --- | --- |
| 匹配字段 | IP、国家、省/区域、城市、全部流量、本地流量、远程流量。 |
| 默认策略 | 没有规则匹配时允许连接。 |
| 同级规则 | 精确度相同时，允许规则优先于阻断规则。 |
| 优先级 | IP > 城市 > 省/区域 > 国家 > 本地/远程 > 全部。 |
| 地域名称 | 国家、省/区域和城市值必须与当前 IP API 返回结果一致。 |
| API Key | 未配置 `[ip_location].api_key` 时归属地查询为空，地理规则基本不会命中。 |

## 配置说明

完整示例见 [`../../nextunnel-server.example.toml`](../../nextunnel-server.example.toml)。

| 配置段 | 字段 | 说明 |
| --- | --- | --- |
| `[server]` | `port` | 公网控制/监听端口；隧道监听绑定所有网卡。 |
| `[server_web]` | `host` / `port` | 管理 UI 与 HTTP API 监听地址。默认端口 `25001`。示例默认为 `127.0.0.1`。 |
| `[cert]` | `host` | 自动生成证书时写入 SAN 的主机名或 IP。 |
| `[cert]` | `dir` | 证书目录，用于 CA、服务端证书和生成的客户端证书。 |
| `[database]` | `host` / `port` / `username` / `password` / `db` / `sslmode` | PostgreSQL 连接配置。 |
| `[ip_location]` | `api_key` | 可选，IP 归属地查询 API Key。 |
| `[logs]` | `file` / `level` / `maxSize` / `maxBackups` / `maxAge` | 日志输出与保留策略。 |
| `[timezone]` | `location` | IANA 时区；未配置时默认 `Asia/Shanghai`。 |

## Docker

服务端 Compose 文件位于 `docker/server`。服务端容器使用 host 网络，控制口、Web 口与代理口均由 TOML 配置决定。

```bash
cd docker/server
cp example.env .env

# 先编辑 volumes/nextunnel/config/nextunnel-server.toml。
# Docker 下请将 [cert].dir 设为 "/etc/nextunnel/certs"，
# 并将 [logs].file 设为 "/var/log/nextunnel/nextunnel-server.log"。
docker compose up -d

# 或只启动 PostgreSQL。
docker compose -f docker-compose.middleware.yaml up -d
```

服务端容器挂载路径：

| 宿主机路径 | 容器路径 |
| --- | --- |
| `docker/server/volumes/nextunnel/config/nextunnel-server.toml` | `/etc/nextunnel/nextunnel-server.toml` |
| `docker/server/volumes/nextunnel/certs/` | `/etc/nextunnel/certs/` |
| `docker/server/volumes/nextunnel/logs/` | `/var/log/nextunnel/` |

## Web 控制台与 HTTP API

管理面随服务端一并启动，提供：

- 内嵌 React 控制台（SPA，缺失路径回退 `index.html`）
- `/api` 管理接口

HTTP 层没有内置认证。请将 `[server_web].host` 绑定到回环或私网地址，或放在防火墙 / 带认证的反向代理之后。示例配置绑定 `127.0.0.1:25001`。

| 接口 | 作用 |
| --- | --- |
| `GET /api/clients` / `POST /api/clients` / `DELETE /api/clients/{name}` | 管理客户端记录。 |
| `GET /api/clients/{name}/sharedcerts` / `POST /api/clients/{name}/sharedcerts` | 查看和创建客户端证书。 |
| `GET /api/clients/{name}/sharedcerts/{id}/download` | 下载客户端证书 zip。 |
| `DELETE /api/clients/{name}/sharedcerts/{id}` | 删除客户端证书。 |
| `GET /api/ca` | 下载 `ca.crt`。 |
| `GET /api/ip-filters` / `POST /api/ip-filters` / `DELETE /api/ip-filters` | 管理访问规则。 |
