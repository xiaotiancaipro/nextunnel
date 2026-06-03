<div align="center">

<h1 style="border-bottom: none"><b>Nextunnel</b></h1>

**Next-generation intranet tunnel**

Reverse tunnel · outbound-first · transport-layer mTLS by default · single Go binaries

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-Apache%20License%20Version%202.0-blue)](./LICENSE)

<a href="./README.md"><img alt="README in English" src="https://img.shields.io/badge/English-d9d9d9"></a>
<a href="./README_zh.md"><img alt="简体中文文件" src="https://img.shields.io/badge/简体中文-d9d9d9"></a>

**[nextunnel-server](https://github.com/xiaotiancaipro/nextunnel-server)** ·
**[nextunnel-client](https://github.com/xiaotiancaipro/nextunnel-client)**

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

## Roadmap

- **Proxy types**: UDP, HTTP/HTTPS, and related modes.
- **Auth & keys**: richer issuance workflows; validity windows, revocation-aware checks, and policy knobs.
- **Multi-tenant users**: login model plus usage/statistics per tenant.
- **Web consoles**: lightweight management UI for server and client.

## Contributing

Issues and PRs are welcome in the respective sub-repositories—especially around feature breadth versus secure defaults.
