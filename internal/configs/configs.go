package configs

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/xiaotiancaipro/nextunnel-client/internal/utils/timezone"
)

type Configs struct {
	Server   *Server   `toml:"server"`
	Client   *Client   `toml:"client"`
	Cert     *Cert     `toml:"cert"`
	Logs     *Logs     `toml:"logs"`
	Timezone *Timezone `toml:"timezone"`
	Proxies  []Proxy   `toml:"proxies"`
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
