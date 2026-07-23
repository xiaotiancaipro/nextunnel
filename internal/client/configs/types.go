package configs

import (
	sharedconfigs "github.com/xiaotiancaipro/nextunnel/internal/shared/configs"
)

type Configs struct {
	Server  *Server             `toml:"server"`
	Client  *Client             `toml:"client"`
	Cert    *Cert               `toml:"cert"`
	Logs    *sharedconfigs.Logs `toml:"logs"`
	Proxies []Proxy             `toml:"proxies"`
}
