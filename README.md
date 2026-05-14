<div align="center">

<h1 style="border-bottom: none"><b>Nextunnel</b></h1>

**Next-generation intranet tunnel**

Reverse tunnel · outbound-first · transport-layer mTLS by default · single Go binaries

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-Apache%20License%20Version%202.0-blue)](./LICENSE)

<a href="./README.md"><img alt="README in English" src="https://img.shields.io/badge/English-d9d9d9"></a>
<a href="./README_zh.md"><img alt="简体中文文件" src="https://img.shields.io/badge/简体中文-d9d9d9"></a>

[**Quick start**](#requirements) · [**Comparison**](#comparison) · [**Roadmap**](#roadmap)

</div>

## What is Nextunnel

**Nextunnel** is a focused reverse outbound tunnel stack: **nextunnel-server** listens on a port reachable from the
public network;
**nextunnel-client** dials out from the intranet over TLS 1.2+, registers multiple TCP forwarding rules. Both
control and data paths run over TLS;
the server uses RequireAndVerifyClientCert, so admission is gated by client certificates issued under the same CA
—unlike many "single shared token + optional TLS" setups.

## Core ideas and features

1. **mTLS as admission control**: connecting requires more than knowing host/port—you need a valid client certificate.
   The server can issue `client.crt` / `client.key` tied to the `ca.crt` chain, closer to device/workload
   onboarding.
2. **Automation-friendly ops**: when `tls.dir` has none of the four PEM/key files, the server can bootstrap CA +
   server certs in one shot; `--generate-certs` batch-issues client certs for edges (refuses to overwrite existing
   files).
3. **Resilience**: after control-channel loss the client auto-reconnects with 2s–30s exponential backoff; `ip_blacklist`
   gives coarse client IP filtering on the server.

## Comparison

| Capability                                            | **Nextunnel** | **frp** ([fatedier/frp](https://github.com/fatedier/frp)) | **nps** ([ehang-io/nps](https://github.com/ehang-io/nps)) |
|:------------------------------------------------------|:-------------:|:---------------------------------------------------------:|:---------------------------------------------------------:|
| **TCP reverse tunnel**                                |       ✅       |                             ✅                             |                             ✅                             |
| **UDP tunnel**                                        |      🔜       |                             ✅                             |                             ✅                             |
| **HTTP / HTTPS routing**                              |      🔜       |                             ✅                             |                             ✅                             |
| **TLS by default on control + data paths**            |       ✅       |                             △                             |                             △                             |
| **Default mTLS (verify client certs)**                |       ✅       |                             ❌                             |                             ❌                             |
| **Built-in CA bootstrap + issue client certs**        |       ✅       |                             ❌                             |                             ❌                             |
| **Multi-user login model**                            |      🔜       |                             △                             |                             ✅                             |
| **Usage / traffic accounting**                        |      🔜       |                             △                             |                             ✅                             |
| **Web admin / control UI**                            |      🔜       |                             △                             |                             ✅                             |
| **Stronger cert policies (revocation windows, etc.)** |      🔜       |                             △                             |                             △                             |

✅ supported

❌ not in that shape by default

△ depends on config/ecosystem, not the default happy path

🔜 planned for Nextunnel

## Requirements

- Go 1.26+
- Mutually trusted material on both sides: CA, server certificate, client certificate

## Build from source

```bash
# Server
cd nextunnel-server
go build -o nextunnel-server .

# Client
cd nextunnel-client
go build -o nextunnel-client .
```

## TLS and certificates

### Server certificate directory

`[tls] dir` in the server config points at a folder that should contain (or will auto-generate):

| Files                       | Role                                           |
|-----------------------------|------------------------------------------------|
| `ca.crt` / `ca.key`         | CA certificate and private key                 |
| `server.crt` / `server.key` | Server certificate and key (signed by that CA) |

If all four files are missing, the server creates a full CA + server chain on first start (SANs incorporate
`[server] addr` for localhost / hostname / IP alignment).

If only some of the four exist, startup fails—you must supply all four or clear the folder for bootstrap.

Listening today is `ListenTCP` on `:<port>` (all interfaces). `server.addr` mainly feeds certificate SAN
semantics; it is not the bind address.

### Issue client certificates

With a complete CA under `tls.dir`, the server can mint client credentials:

```bash
cd nextunnel-server
./nextunnel-server --config nextunnel-server.toml --generate-certs /path/to/client-certs-dir
```

This writes `client.crt` and `client.key`. Existing names cause an error to avoid silent overwrite.

Copy `ca.crt` plus `client.crt` / `client.key` somewhere the client can read and reference them under `[tls]` in
its config.

## Configuration

Both sides use TOML, selectable via `-c` / `--config`. Defaults are `nextunnel-server.toml` and
`nextunnel-client.toml`.

Examples in-repo:

- `nextunnel-server/nextunnel-server.example.toml`
- `nextunnel-client/nextunnel-client.example.toml`

## Run

```bash
# Server
./nextunnel-server --config nextunnel-server.toml

# Client
./nextunnel-client --config nextunnel-client.toml
```

## Docker Compose

Each subproject ships `docker-compose.yaml` using `network_mode: host`, mounting config, certs, and logs:

- Server: `nextunnel-server/volumes/{config,certs,logs}`
- Client: `nextunnel-client/volumes/{config,certs,logs}`

Default container commands:

- `nextunnel-server --config config/nextunnel-server.toml`
- `nextunnel-client --config config/nextunnel-client.toml`

Place host-side TOML and PEM material under `volumes/config` and `volumes/certs`.

## Roadmap

- **Proxy types**: UDP, HTTP/HTTPS, and related modes.
- **Auth & keys**: richer issuance workflows; validity windows, revocation-aware checks, and policy knobs.
- **Multi-tenant users**: login model plus usage/statistics per tenant.
- **Web consoles**: lightweight management UI for server and client.

## Security notes

- Trust rests on client certificates—protect `ca.key` and each `client.key`.
- There is no second-factor token layered on top today; exposing issuing powers or wide `remote_port` ranges is
  risky—pair with firewall rules, `ip_blacklist`, and tight public exposure policy.

## Contributing

Issues and PRs are welcome in the respective sub-repositories—especially around feature breadth versus secure defaults.
