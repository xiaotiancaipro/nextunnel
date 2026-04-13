package configs

import (
	"os"

	"github.com/BurntSushi/toml"
)

type ServerConfigs struct {
	BindPort int              `toml:"bind_port"`
	Token    string           `toml:"token"`
	TLS      ServerTLSConfigs `toml:"tls"`
}

type ServerTLSConfigs struct {
	Enabled  bool   `toml:"enabled"`
	CertFile string `toml:"cert_file"`
	KeyFile  string `toml:"key_file"`
}

func NewServer(file string) (*ServerConfigs, error) {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil, err
	}
	var configs ServerConfigs
	if _, err := toml.DecodeFile(file, &configs); err != nil {
		return nil, err
	}
	return &configs, nil
}
