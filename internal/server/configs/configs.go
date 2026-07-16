package configs

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	sharedconfigs "github.com/xiaotiancaipro/nextunnel/internal/shared/configs"
	sharedtimezone "github.com/xiaotiancaipro/nextunnel/internal/shared/timezone"
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

func NewConfigs(file string) (*Configs, error) {

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil, err
	}

	var cfg Configs
	if _, err := toml.DecodeFile(file, &cfg); err != nil {
		return nil, err
	}

	if err := sharedtimezone.Init(cfg.Timezone.NameOrDefault()); err != nil {
		return nil, fmt.Errorf("invalid timezone config: %w", err)
	}

	return &cfg, nil

}
