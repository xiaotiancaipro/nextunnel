# Nextunnel Documentation

Nextunnel keeps the project overview in the root README files and places component guides here.

## English

- [Server guide](./en/server.md)
- [Client guide](./en/client.md)

## 简体中文

- [服务端指南](./zh/server.md)
- [客户端指南](./zh/client.md)

## Recommended Reading Order

1. Read the project overview: [English](../README.md) / [简体中文](../README_zh.md).
2. Deploy and start `nextunnel-server` (control port + embedded web console).
3. Register a client, create a client certificate, and download `ca.crt`, `client.crt`, `client.key`.
4. Configure and start `nextunnel-client`.

## Build Notes

- Server builds need Go 1.26+ and Node.js/npm (`make build-server` runs `web/server` via npm, then embeds the assets).
- Client builds need Go only.
- `make build` writes versioned binaries under `bin/` using the root `VERSION` file.
