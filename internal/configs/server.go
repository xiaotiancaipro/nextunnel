package configs

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

type ServerConfigs struct {
	BindPort int              `toml:"bind_port"`
	Token    string           `toml:"token"`
	TLS      ServerTLSConfigs `toml:"tls"`
}

type ServerTLSConfigs struct {
	Enabled  bool   `toml:"enabled"`
	CAFile   string `toml:"ca_file"`
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
	if err := configs.Validate(); err != nil {
		return nil, err
	}
	return &configs, nil
}

func (c *ServerConfigs) Validate() error {
	if c.BindPort <= 0 || c.BindPort > 65535 {
		return fmt.Errorf("invalid bind_port: %d", c.BindPort)
	}
	if strings.TrimSpace(c.Token) == "" {
		return fmt.Errorf("token cannot be empty")
	}
	if c.TLS.Enabled {
		if strings.TrimSpace(c.TLS.CertFile) == "" {
			return fmt.Errorf("tls.cert_file is required when tls is enabled")
		}
		if strings.TrimSpace(c.TLS.KeyFile) == "" {
			return fmt.Errorf("tls.key_file is required when tls is enabled")
		}
		if strings.TrimSpace(c.TLS.CAFile) == "" {
			return fmt.Errorf("tls.ca_file is required when tls is enabled")
		}
	}
	return nil
}
