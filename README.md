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

| Flag               | Default                 | Description                                             |
|--------------------|-------------------------|---------------------------------------------------------|
| `--config`         | `nextunnel-server.toml` | Path to the configuration file                          |
| `--generate-certs` | —                       | Generate client TLS certificates in the given directory |
| `--ip-allow`       | —                       | Add an IP to the allow list                             |
| `--ip-block`       | —                       | Add an IP to the block list                             |
| `--country-allow`  | —                       | Add a country to the allow list                         |
| `--country-block`  | —                       | Add a country to the block list                         |
| `--region-allow`   | —                       | Add a region/state to the allow list                    |
| `--region-block`   | —                       | Add a region/state to the block list                    |
| `--city-allow`     | —                       | Add a city to the allow list                            |
| `--city-block`     | —                       | Add a city to the block list                            |
| `--block-all`      | `false`                 | Block all connections                                   |
| `--allow-all`      | `false`                 | Allow all connections                                   |
| `--block-local`    | `false`                 | Block local network connections                         |
| `--allow-local`    | `false`                 | Allow local network connections                         |
| `-h`, `--help`     | —                       | Show help                                               |
| `-v`, `--version`  | —                       | Show version                                            |

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

### Access Control Rules

```bash
# Allow/Block an IP
nextunnel-server --ip-allow 203.0.113.10
nextunnel-server --ip-block 203.0.113.10

# Allow/block a country
nextunnel-server --country-allow China
nextunnel-server --country-block China

# Allow/block a region/state
nextunnel-server --region-allow Guangdong
nextunnel-server --region-block Guangdong

# Allow/block a city
nextunnel-server --city-allow Shenzhen
nextunnel-server --city-block Shenzhen

# Block/allow all connections
nextunnel-server --block-all
nextunnel-server --allow-all

# Block/allow local network connections
nextunnel-server --block-local
nextunnel-server --allow-local
```

- Supports IPv4 and IPv6; IP addresses are normalized automatically
- **Category** has two values: `ALL` (all connections) and `LOCAL` (local network: private, loopback, link-local)
- `Category=ALL` with `Status=0` and no other conditions blocks every connection
- `Category=LOCAL` with `Status=0` and no other conditions blocks only local network connections
- Geo rules must match GeoIP lookup results (use the same names shown in connection logs, e.g.
  `China/Guangdong/Shenzhen`
  maps to country=China, region=Guangdong, city=Shenzhen)
- Requires a working database connection (PostgreSQL via `[database]`)
- Updates the existing rule if one with the same dimension already exists, otherwise creates a new one
- Allow list maps to `status = 1`; block list maps to `status = 0`
- Rule priority: 1) Allow beats Block at the same specificity; 2) IP > City > Region > Country > Category global rules

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
