<div align="center">

<h1 style="border-bottom: none"><b>nextunnel-server</b></h1>

**接受客户端连接，管理代理与 IP 访问控制**

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-Apache%20License%20Version%202.0-blue)](./LICENSE)

<a href="./README.md"><img alt="README in English" src="https://img.shields.io/badge/English-d9d9d9"></a>
<a href="./README_zh.md"><img alt="简体中文文件" src="https://img.shields.io/badge/简体中文-d9d9d9"></a>

</div>

## 快速开始

```bash
# 复制示例配置
cp nextunnel-server.example.toml nextunnel-server.toml

# 构建
go build -o nextunnel-server .

# 启动服务（默认读取 nextunnel-server.toml）
nextunnel-server
```

## CLI 用法

```bash
nextunnel-server [flags]
```

未指定任务类参数时，程序以前台方式启动服务端。

### 参数一览

| 参数                 | 默认值                     | 说明                      |
|--------------------|-------------------------|-------------------------|
| `--config`         | `nextunnel-server.toml` | 配置文件路径（支持相对/绝对路径）       |
| `--generate-certs` | —                       | 在指定目录生成客户端 TLS 证书，完成后退出 |
| `--ip-allow`       | —                       | 将 IP 加入白名单（写入数据库），完成后退出 |
| `--ip-block`       | —                       | 将 IP 加入黑名单（写入数据库），完成后退出 |
| `-h`, `--help`     | —                       | 显示帮助信息                  |
| `-v`, `--version`  | —                       | 显示版本号                   |

### 启动服务

```bash
nextunnel-server

# 指定配置文件
nextunnel-server --config /path/to/nextunnel-server.toml
```

启动时会：

1. 加载 TOML 配置文件
2. 初始化日志与数据库连接
3. 监听 `[server]` 中配置的地址与端口
4. 自动确保 TLS 证书目录中存在 CA 与服务端证书

### 生成客户端证书

```bash
nextunnel-server --generate-certs ./client-certs
```

- 读取配置中 `[tls].dir` 目录下的 CA 证书（`ca.crt` / `ca.key`）；若 CA 或服务端证书不存在，会自动生成
- 在 `--generate-certs` 指定目录下输出 `client.crt` 与 `client.key`
- 目标目录中若已存在同名文件，命令会报错并退出
- 证书有效期为 1 年

### IP 白名单 / 黑名单

```bash
# 允许某 IP 访问
nextunnel-server --ip-allow 203.0.113.10

# 禁止某 IP 访问
nextunnel-server --ip-block 203.0.113.10
```

- 支持 IPv4 / IPv6，会自动规范化 IP 格式
- 需要数据库可用（通过 `[database]` 连接 PostgreSQL）
- 若 IP 记录已存在则更新状态，否则新建记录
- 白名单对应 `status = 1`，黑名单对应 `status = 0`

## 配置文件

参考 [`nextunnel-server.example.toml`](nextunnel-server.example.toml)：

```toml
[server]
addr = "127.0.0.1"
port = 25930

[logs]
file = "logs/nextunnel-server.log"
level = "info"

[tls]
dir = "certs"

[database]
host = "127.0.0.1"
port = 5432
username = "postgres"
password = "nextunnel"
db = "nextunnel"

[geoip]
db_path = "geoip/GeoLite2-City.mmdb"
```

| 配置段          | 字段                                               | 说明                                      |
|--------------|--------------------------------------------------|-----------------------------------------|
| `[server]`   | `addr`                                           | 监听地址                                    |
|              | `port`                                           | 监听端口                                    |
| `[logs]`     | `file`                                           | 日志文件路径                                  |
|              | `level`                                          | 日志级别                                    |
| `[tls]`      | `dir`                                            | TLS 证书目录（CA、服务端及客户端证书生成均依赖此目录）          |
| `[database]` | `host` / `port` / `username` / `password` / `db` | PostgreSQL 连接信息                         |
| `[geoip]`    | `db_path`                                        | MaxMind GeoLite2-City 数据库路径；留空则跳过 GeoIP |

### GeoIP 归属地

1. 在 [MaxMind GeoLite2](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data) 注册并下载 `GeoLite2-City.mmdb`
2. 将文件放到配置中 `[geoip].db_path` 指定路径
3. 首次连接某 IP 时会查询 GeoIP 并将 `country` / `region` / `city` 写入 `ip_address` 表，后续直接读库
4. 连接日志示例：`User connection arrived: proxy=web, ip=203.0.113.10, region=CN/广东/深圳`
