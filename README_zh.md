<div align="center">

<h1 style="border-bottom: none"><b>nextunnel-server</b></h1>

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-Apache%20License%20Version%202.0-blue)](./LICENSE)

<a href="./README.md"><img alt="README in English" src="https://img.shields.io/badge/English-d9d9d9"></a>
<a href="./README_zh.md"><img alt="简体中文文件" src="https://img.shields.io/badge/简体中文-d9d9d9"></a>

</div>

## 概述

`nextunnel-server` 是 nextunnel 反向隧道系统的服务端组件，主要功能包括：

- 接受 nextunnel 客户端的双向 TLS（mTLS）连接
- 根据客户端提交的代理配置监听远程端口
- 基于 PostgreSQL 中的规则执行 IP / 地域 / 网络类别访问控制
- 将每次入站用户连接（IP、地域、网络类别、放行/拒绝结果）写入 PostgreSQL

## 环境要求

- Go 1.26+（本地编译）
- PostgreSQL
- MaxMind [GeoLite2-City](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data) 数据库（`GeoLite2-City.mmdb`）

## 快速开始

```bash
# 下载 GeoLite2-City.mmdb 并放到 geoip/GeoLite2-City.mmdb

# 复制示例配置
cp nextunnel-server.example.toml nextunnel-server.toml

# 构建
go build -o nextunnel-server .

# 启动服务（默认读取 nextunnel-server.toml）
nextunnel-server
```

多平台发布包可通过以下脚本构建：

```bash
./script/build.sh
```

## Docker 部署

`docker/` 目录提供了完整服务与仅中间件两种 Compose 编排。

```bash
cd docker
cp example.env .env
# 按需修改 .env

# 启动 PostgreSQL + nextunnel-server
docker compose up -d

# 或仅启动 PostgreSQL
docker compose -f docker-compose.middleware.yaml up -d
```

默认卷映射：

| 宿主机路径                             | 容器路径                          | 用途        |
|-----------------------------------|-------------------------------|-----------|
| `docker/volumes/nextunnel/config` | `/usr/local/nextunnel/config` | 配置文件      |
| `docker/volumes/nextunnel/certs`  | `/usr/local/nextunnel/certs`  | TLS 证书    |
| `docker/volumes/nextunnel/geoip`  | `/usr/local/nextunnel/geoip`  | GeoIP 数据库 |
| `docker/volumes/nextunnel/logs`   | `/usr/local/nextunnel/logs`   | 日志文件      |

服务端容器使用 `network_mode: host`，代理端口直接绑定在宿主机上。

## CLI 用法

```bash
nextunnel-server [flags]
```

未指定任务类参数时，程序以前台方式启动服务端。按 `Ctrl+C` 或发送 `SIGTERM` 可优雅退出。

### 参数一览

| 参数                          | 默认值                     | 说明                |
|-----------------------------|-------------------------|-------------------|
| `--config`                  | `nextunnel-server.toml` | 配置文件路径            |
| `--generate-certs`          | —                       | 在指定目录生成客户端 TLS 证书 |
| `--ip-filter-allow-ip`      | —                       | 将 IP 加入白名单        |
| `--ip-filter-block-ip`      | —                       | 将 IP 加入黑名单        |
| `--ip-filter-allow-country` | —                       | 将国家/地区加入白名单       |
| `--ip-filter-block-country` | —                       | 将国家/地区加入黑名单       |
| `--ip-filter-allow-region`  | —                       | 将省/州加入白名单         |
| `--ip-filter-block-region`  | —                       | 将省/州加入黑名单         |
| `--ip-filter-allow-city`    | —                       | 将城市加入白名单          |
| `--ip-filter-block-city`    | —                       | 将城市加入黑名单          |
| `--ip-filter-block-all`     | `false`                 | 拒绝所有连接            |
| `--ip-filter-allow-all`     | `false`                 | 允许所有连接            |
| `--ip-filter-block-local`   | `false`                 | 拒绝局域网连接           |
| `--ip-filter-allow-local`   | `false`                 | 允许局域网连接           |
| `--ip-filter-block-remote`  | `false`                 | 拒绝公网（非局域网）连接      |
| `--ip-filter-allow-remote`  | `false`                 | 允许公网（非局域网）连接      |
| `-h`, `--help`              | —                       | 显示帮助信息            |
| `-v`, `--version`           | —                       | 显示版本号             |

### 启动服务

```bash
nextunnel-server

# 指定配置文件
nextunnel-server --config /path/to/nextunnel-server.toml
```

启动时会：

1. 加载 TOML 配置文件
2. 初始化日志与 PostgreSQL 连接（含自动建表）
3. 加载 GeoIP 数据库
4. 在 `0.0.0.0:<port>` 上监听（所有网卡）
5. 自动确保 TLS 证书目录中存在 CA 与服务端证书

> `[server].host` 用于 TLS 证书 SAN 生成，并非实际监听地址。

### 生成客户端证书

```bash
nextunnel-server --generate-certs ./client-certs
```

- 读取配置中 `[tls].dir` 目录下的 CA 证书（`ca.crt` / `ca.key`）；若 CA 或服务端证书不存在，会自动生成
- 在 `--generate-certs` 指定目录下输出 `client.crt` 与 `client.key`
- 目标目录中若已存在同名文件，命令会报错并退出
- 客户端证书有效期 1 年，CA 证书有效期 10 年

### 访问控制规则

```bash
# 允许/禁止某 IP 访问
nextunnel-server --ip-filter-allow-ip 203.0.113.10
nextunnel-server --ip-filter-block-ip 203.0.113.10

# 允许/禁止某个国家/地区
nextunnel-server --ip-filter-allow-country China
nextunnel-server --ip-filter-block-country China

# 允许/禁止某个省/州
nextunnel-server --ip-filter-allow-region Guangdong
nextunnel-server --ip-filter-block-region Guangdong

# 允许/禁止某个城市
nextunnel-server --ip-filter-allow-city Shenzhen
nextunnel-server --ip-filter-block-city Shenzhen

# 拒绝/允许所有连接
nextunnel-server --ip-filter-block-all
nextunnel-server --ip-filter-allow-all

# 拒绝/允许局域网连接
nextunnel-server --ip-filter-block-local
nextunnel-server --ip-filter-allow-local

# 拒绝/允许公网（非局域网）连接
nextunnel-server --ip-filter-block-remote
nextunnel-server --ip-filter-allow-remote
```

- 支持 IPv4 / IPv6，会自动规范化 IP 格式
- 地域规则需与 `[geoip].locales` 解析出的 GeoIP 结果一致（名称需与连接日志中的 `region` 字段匹配，如
  `China/Guangdong/Shenzhen` 对应 country=China、region=Guangdong、city=Shenzhen）
- 需要数据库可用（通过 `[database]` 连接 PostgreSQL）
- 若同维度规则已存在则更新状态，否则新建记录
- 白名单对应 `status = 1`，黑名单对应 `status = 0`
- 无匹配规则时，连接**默认允许**
- 规则优先级：1) 同等精确度下 Allow > Block；2) IP > City > Region > Country > Category 全局规则

## 配置文件

参考 [`nextunnel-server.example.toml`](nextunnel-server.example.toml)：

| 配置段          | 字段                                               | 说明                                                   |
|--------------|--------------------------------------------------|------------------------------------------------------|
| `[server]`   | `host`                                           | TLS 证书 SAN 用的主机名或 IP（非监听地址）                          |
|              | `port`                                           | 监听端口（绑定所有网卡）                                         |
| `[logs]`     | `file`                                           | 日志文件路径（按天轮转，超出大小自动分段）                                |
|              | `level`                                          | 日志级别（`debug`、`info`、`warn`、`error`）                  |
|              | `maxSize`                                        | 单个日志分段最大大小（如 `100MB`、`1GB`；纯数字默认为 MB）                |
|              | `maxBackups`                                     | 保留的按天日志文件数量上限                                        |
|              | `maxAge`                                         | 日志文件最大保留天数                                           |
| `[tls]`      | `dir`                                            | TLS 证书目录（CA、服务端及客户端证书生成均依赖此目录）                       |
| `[database]` | `host` / `port` / `username` / `password` / `db` | PostgreSQL 连接信息                                      |
| `[geoip]`    | `db_path`                                        | MaxMind GeoLite2-City 数据库路径                          |
|              | `locales`                                        | GeoIP 地名解析的语言优先级（如 `["zh-CN", "en"]`）；地域访问规则须与解析结果一致 |
