# nextunnel

Next-Generation Intranet Tunneling Tool

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
