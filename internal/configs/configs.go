package configs

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Configs struct {
	Logs   *Logs   `toml:"logs"`
	Tls    *Tls    `toml:"tls"`
	Server *Server `toml:"server"`
}

func NewConfigs(file string) (*Configs, error) {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil, err
	}
	var configs Configs
	if _, err := toml.DecodeFile(file, &configs); err != nil {
		return nil, err
	}
	return &configs, nil
}
