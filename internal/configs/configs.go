package configs

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils/timezone"
)

type Configs struct {
	Server     *Server     `toml:"server"`
	Cert       *Cert       `toml:"cert"`
	Database   *Database   `toml:"database"`
	IPLocation *IPLocation `toml:"ip_location"`
	Logs       *Logs       `toml:"logs"`
	Timezone   *Timezone   `toml:"timezone"`
}

func NewConfigs(file string) (*Configs, error) {

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil, err
	}

	var configs Configs
	if _, err := toml.DecodeFile(file, &configs); err != nil {
		return nil, err
	}

	if err := timezone.Init(configs.Timezone.NameOrDefault()); err != nil {
		return nil, fmt.Errorf("invalid timezone config: %w", err)
	}

	return &configs, nil

}
