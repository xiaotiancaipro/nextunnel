<div align="center">

<h1 style="border-bottom: none"><b>nextunnel-server</b></h1>

**Accepts client connections, manages proxies, and controls IP access**

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-Apache%20License%20Version%202.0-blue)](./LICENSE)

<a href="./README.md"><img alt="README in English" src="https://img.shields.io/badge/English-d9d9d9"></a>
<a href="./README_zh.md"><img alt="简体中文文件" src="https://img.shields.io/badge/简体中文-d9d9d9"></a>

</div>

## Quick Start

```bash
# Copy the example config
cp nextunnel-server.example.toml nextunnel-server.toml

# Build
go build -o nextunnel-server .

# Start the server (reads nextunnel-server.toml by default)
nextunnel-server
```

## CLI Usage

```bash
nextunnel-server [flags]
```

When no task flags are provided, the program starts the server in the foreground.

### Flags

| Flag               | Default                 | Description                                                        |
|--------------------|-------------------------|--------------------------------------------------------------------|
| `--config`         | `nextunnel-server.toml` | Path to the configuration file (relative or absolute)              |
| `--generate-certs` | —                       | Generate client TLS certificates in the given directory, then exit |
| `--ip-allow`       | —                       | Add an IP to the allow list (persisted in `rules_ip`), then exit   |
| `--ip-block`       | —                       | Add an IP to the block list (persisted in `rules_ip`), then exit   |
| `-h`, `--help`     | —                       | Show help                                                          |
| `-v`, `--version`  | —                       | Show version                                                       |

### Start the Server

```bash
nextunnel-server

# Use a custom config file
nextunnel-server --config /path/to/nextunnel-server.toml
```

On startup, the server will:

1. Load the TOML configuration file
2. Initialize logging and the database connection
3. Listen on the address and port configured under `[server]`
4. Ensure CA and server TLS certificates exist in the configured certificate directory

### Generate Client Certificates

```bash
nextunnel-server --generate-certs ./client-certs
```

- Reads the CA certificate from `[tls].dir` (`ca.crt` / `ca.key`); if the CA or server certificate is missing, it is
  generated automatically
- Writes `client.crt` and `client.key` to the directory specified by `--generate-certs`
- Exits with an error if either file already exists in the target directory
- Certificates are valid for 1 year

### IP Allow / Block Lists

```bash
# Allow an IP
nextunnel-server --ip-allow 203.0.113.10

# Block an IP
nextunnel-server --ip-block 203.0.113.10
```

- Supports IPv4 and IPv6; IP addresses are normalized automatically
- Requires a working database connection (PostgreSQL via `[database]`)
- Updates the existing record if the IP is already present, otherwise creates a new one
- Allow list maps to `status = 1`; block list maps to `status = 0`

## Configuration

See [`nextunnel-server.example.toml`](nextunnel-server.example.toml):

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

| Section      | Field                                            | Description                                                                        |
|--------------|--------------------------------------------------|------------------------------------------------------------------------------------|
| `[server]`   | `addr`                                           | Listen address                                                                     |
|              | `port`                                           | Listen port                                                                        |
| `[logs]`     | `file`                                           | Log file path                                                                      |
|              | `level`                                          | Log level                                                                          |
| `[tls]`      | `dir`                                            | TLS certificate directory (used for CA, server, and client certificate generation) |
| `[database]` | `host` / `port` / `username` / `password` / `db` | PostgreSQL connection settings                                                     |
| `[geoip]`    | `db_path`                                        | Path to MaxMind GeoLite2-City database; leave empty to disable GeoIP               |

### GeoIP region lookup

1. Register at [MaxMind GeoLite2](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data) and download
   `GeoLite2-City.mmdb`
2. Place the file at the path configured in `[geoip].db_path`
3. On each connection, GeoIP is queried and the IP with `country` / `region` / `city` is stored in `logs_access`; IP
   restriction rules are stored in `rules_ip`
4. Log example: `User connection arrived: proxy=web, ip=203.0.113.10, region=CN/Guangdong/Shenzhen`
