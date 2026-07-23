package configs

import (
	sharedconfigs "github.com/xiaotiancaipro/nextunnel/internal/shared/configs"
)

type Configs struct {
	Server     *Server             `toml:"server"`
	ServerWeb  *ServerWeb          `toml:"server_web"`
	Cert       *Cert               `toml:"cert"`
	Database   *Database           `toml:"database"`
	IPLocation *IPLocation         `toml:"ip_location"`
	Logs       *sharedconfigs.Logs `toml:"logs"`
}
