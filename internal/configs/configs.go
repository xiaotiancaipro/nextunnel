package configs

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/xiaotiancaipro/nextunnel-server/internal/utils/timezone"
)

type Configs struct {
	Server   *Server   `toml:"server"`
	Logs     *Logs     `toml:"logs"`
	Tls      *Tls      `toml:"tls"`
	Database *Database `toml:"database"`
	GeoIP    *GeoIP    `toml:"geoip"`
	Timezone *Timezone `toml:"timezone"`
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
