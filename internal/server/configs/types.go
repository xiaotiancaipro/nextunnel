package configs

import (
	sharedconfigs "github.com/xiaotiancaipro/nextunnel/internal/shared/configs"
)

type Configs struct {
	Server     *Server                 `toml:"server"`
	Cert       *Cert                   `toml:"cert"`
	Database   *Database               `toml:"database"`
	IPLocation *IPLocation             `toml:"ip_location"`
	Logs       *sharedconfigs.Logs     `toml:"logs"`
	Timezone   *sharedconfigs.Timezone `toml:"timezone"`
	Web        *Web                    `toml:"web"`
}
