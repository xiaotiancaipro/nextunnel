# nextunnel

Next-Generation Intranet Tunneling Tool

## Config Reload

Both `nextunnel server` and `nextunnel client` watch their config file and automatically reload when it changes.

- Poll interval: `2s`
- Manual trigger: send `SIGHUP`
- Reload scope: full config reload, including listener address, TLS, authentication, and proxy settings
- Server reload behavior: listener/auth/ip filter updates are applied in place; existing accepted connections keep running
- Client reload behavior: a new control session connects with the same `client_id`, takes over proxies, then the old session drains before exit
- Client config requirement: `client_id` must be stable and unique per logical client
- Safety behavior: if the new config is invalid or fails to start, the current working session/listener stays active

## Build

Use the build script to compile `nextunnel` for multiple platforms at once:

```bash
./scripts/build.sh
```

By default it builds:

- `darwin/amd64`
- `darwin/arm64`
- `linux/amd64`
- `linux/arm64`
- `windows/amd64`
- `windows/arm64`

Built binaries are written to `./bin`.

Examples:

```bash
./scripts/build.sh --targets linux/amd64,linux/arm64
./scripts/build.sh --output-dir ./dist
```
