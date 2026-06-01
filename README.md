<div align="center">

<h1 style="border-bottom: none"><b>nextunnel-server</b></h1>

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-Apache%20License%20Version%202.0-blue)](./LICENSE)

<a href="./README.md"><img alt="README in English" src="https://img.shields.io/badge/English-d9d9d9"></a>
<a href="./README_zh.md"><img alt="简体中文文件" src="https://img.shields.io/badge/简体中文-d9d9d9"></a>

</div>

## Overview

`nextunnel-server` is the server-side component of the nextunnel reverse-tunnel system. It:

- Accepts mutual TLS (mTLS) connections from nextunnel clients
- Applies proxy configurations submitted by clients
- Enforces IP / geo / network-category access control rules stored in PostgreSQL
- Records every inbound user connection (IP, geo, category, allow/deny decision) in PostgreSQL

## Requirements

- Go 1.26+ (for local builds)
- PostgreSQL
- MaxMind [GeoLite2-City](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data) database (`GeoLite2-City.mmdb`)

## Quick Start

```bash
# Download GeoLite2-City.mmdb and place it at geoip/GeoLite2-City.mmdb

# Copy the example config
cp nextunnel-server.example.toml nextunnel-server.toml

# Build
go build -o nextunnel-server .

# Start the server (reads nextunnel-server.toml by default)
nextunnel-server
```

Cross-platform release binaries can be built with:

```bash
./script/build.sh
```

## Docker

The `docker/` directory provides Compose stacks for production and middleware-only deployments.

```bash
cd docker
cp example.env .env
# Edit .env if needed

# Start PostgreSQL + nextunnel-server
docker compose up -d

# Or start PostgreSQL only
docker compose -f docker-compose.middleware.yaml up -d
```

Default volume layout:

| Host path                         | Container path                | Purpose          |
|-----------------------------------|-------------------------------|------------------|
| `docker/volumes/nextunnel/config` | `/usr/local/nextunnel/config` | Configuration    |
| `docker/volumes/nextunnel/certs`  | `/usr/local/nextunnel/certs`  | TLS certificates |
| `docker/volumes/nextunnel/geoip`  | `/usr/local/nextunnel/geoip`  | GeoIP database   |
| `docker/volumes/nextunnel/logs`   | `/usr/local/nextunnel/logs`   | Log files        |

The server container uses `network_mode: host` so client proxy ports bind directly on the host.

## CLI Usage

```bash
nextunnel-server [flags]
```

When no task flags are provided, the program starts the server in the foreground. Press `Ctrl+C` or send `SIGTERM` for
graceful shutdown.

### Flags

| Flag                        | Default                 | Description                                             |
|-----------------------------|-------------------------|---------------------------------------------------------|
| `--config`                  | `nextunnel-server.toml` | Path to the configuration file                          |
| `--generate-certs`          | —                       | Generate client TLS certificates in the given directory |
| `--ip-filter-allow-ip`      | —                       | Add an IP to the allow list                             |
| `--ip-filter-block-ip`      | —                       | Add an IP to the block list                             |
| `--ip-filter-allow-country` | —                       | Add a country to the allow list                         |
| `--ip-filter-block-country` | —                       | Add a country to the block list                         |
| `--ip-filter-allow-region`  | —                       | Add a region/state to the allow list                    |
| `--ip-filter-block-region`  | —                       | Add a region/state to the block list                    |
| `--ip-filter-allow-city`    | —                       | Add a city to the allow list                            |
| `--ip-filter-block-city`    | —                       | Add a city to the block list                            |
| `--ip-filter-block-all`     | `false`                 | Block all connections                                   |
| `--ip-filter-allow-all`     | `false`                 | Allow all connections                                   |
| `--ip-filter-block-local`   | `false`                 | Block local network connections                         |
| `--ip-filter-allow-local`   | `false`                 | Allow local network connections                         |
| `--ip-filter-block-remote`  | `false`                 | Block remote (non-local) network connections            |
| `--ip-filter-allow-remote`  | `false`                 | Allow remote (non-local) network connections            |
| `-h`, `--help`              | —                       | Show help                                               |
| `-v`, `--version`           | —                       | Show version                                            |

### Start the Server

```bash
nextunnel-server

# Use a custom config file
nextunnel-server --config /path/to/nextunnel-server.toml
```

On startup, the server will:

1. Load the TOML configuration file
2. Initialize logging and the PostgreSQL connection (with auto-migration)
3. Load the GeoIP database
4. Listen on `0.0.0.0:<port>` (all interfaces)
5. Ensure CA and server TLS certificates exist in the configured certificate directory

> `[server].host` is used for TLS certificate SAN generation, not for the listen address.

### Generate Client Certificates

```bash
nextunnel-server --generate-certs ./client-certs
```

- Reads the CA certificate from `[tls].dir` (`ca.crt` / `ca.key`); if the CA or server certificate is missing, it is
  generated automatically
- Writes `client.crt` and `client.key` to the directory specified by `--generate-certs`
- Exits with an error if either file already exists in the target directory
- Client certificates are valid for 1 year; CA certificates for 10 years

### Access Control Rules

```bash
# Allow/block an IP
nextunnel-server --ip-filter-allow-ip 203.0.113.10
nextunnel-server --ip-filter-block-ip 203.0.113.10

# Allow/block a country
nextunnel-server --ip-filter-allow-country China
nextunnel-server --ip-filter-block-country China

# Allow/block a region/state
nextunnel-server --ip-filter-allow-region Guangdong
nextunnel-server --ip-filter-block-region Guangdong

# Allow/block a city
nextunnel-server --ip-filter-allow-city Shenzhen
nextunnel-server --ip-filter-block-city Shenzhen

# Block/allow all connections
nextunnel-server --ip-filter-block-all
nextunnel-server --ip-filter-allow-all

# Block/allow local network connections
nextunnel-server --ip-filter-block-local
nextunnel-server --ip-filter-allow-local

# Block/allow remote (non-local) network connections
nextunnel-server --ip-filter-block-remote
nextunnel-server --ip-filter-allow-remote
```

- Supports IPv4 and IPv6; IP addresses are normalized automatically
- Geo rules must match GeoIP lookup results under the configured `[geoip].locales` (use the same names shown in
  connection logs, e.g. `China/Guangdong/Shenzhen` maps to country=China, region=Guangdong, city=Shenzhen)
- Requires a working database connection (PostgreSQL via `[database]`)
- Updates the existing rule if one with the same dimension already exists, otherwise creates a new one
- Allow list maps to `status = 1`; block list maps to `status = 0`
- When no rule matches, the connection is **allowed** by default
- Rule priority: 1) Allow beats Block at the same specificity; 2) IP > City > Region > Country > Category global rules

## Configuration

See [`nextunnel-server.example.toml`](nextunnel-server.example.toml):

| Section      | Field                                            | Description                                                                                                                       |
|--------------|--------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------|
| `[server]`   | `host`                                           | Hostname or IP for TLS certificate SAN (not the listen address)                                                                   |
|              | `port`                                           | Listen port (binds to all interfaces)                                                                                             |
| `[logs]`     | `file`                                           | Log file path (daily rotation with size-based segments)                                                                           |
|              | `level`                                          | Log level (`debug`, `info`, `warn`, `error`)                                                                                      |
|              | `maxSize`                                        | Max size per log segment (e.g. `100MB`, `1GB`; bare number = MB)                                                                  |
|              | `maxBackups`                                     | Max number of daily log files to retain                                                                                           |
|              | `maxAge`                                         | Max age of log files in days                                                                                                      |
| `[tls]`      | `dir`                                            | TLS certificate directory (CA, server, and client cert generation)                                                                |
| `[database]` | `host` / `port` / `username` / `password` / `db` | PostgreSQL connection settings                                                                                                    |
|              | `sslmode`                                        | libpq SSL mode (`disable`, `require`, `verify-ca`, `verify-full`); defaults to `disable`                                          |
| `[geoip]`    | `db_path`                                        | Path to MaxMind GeoLite2-City database (required)                                                                                 |
|              | `locales`                                        | Ordered locale codes for GeoIP name lookup (e.g. `["zh-CN", "en"]`); geo access rules must use names resolved under these locales |
