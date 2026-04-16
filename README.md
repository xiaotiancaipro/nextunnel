# nextunnel

Next-Generation Intranet Tunneling Tool

## server configuration

```toml
bind_port = 7000
token = "your-secret-token"

[tls]
enabled = false
ca_file = "./certs/ca.crt"
cert_file = "./certs/server.crt"
key_file = "./certs/server.key"

[ip_filter]
allow = []
deny = []
```

## client configuration

```toml
client_id = "edge-client-1"
server_addr = "x.x.x.x"
server_port = 7000
token = "your-secret-token"

[tls]
enabled = false
server_name = "your-server-domain"
ca_file = "./certs/ca.crt"
cert_file = "./certs/client.crt"
key_file = "./certs/client.key"
insecure_skip_verify = false

# Example 1
[[proxies]]
name = "ssh"
type = "tcp"
local_ip = "127.0.0.1"
local_port = 22
remote_port = 6000

# Example 2
[[proxies]]
name = "web"
type = "tcp"
local_ip = "127.0.0.1"
local_port = 8080
remote_port = 8000
```
