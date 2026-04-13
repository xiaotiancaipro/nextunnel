package configs

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

type ServerConfigs struct {
	BindPort int                   `toml:"bind_port"`
	Token    string                `toml:"token"`
	TLS      ServerTLSConfigs      `toml:"tls"`
	IPFilter ServerIPFilterConfigs `toml:"ip_filter"`
}

type ServerTLSConfigs struct {
	Enabled  bool   `toml:"enabled"`
	CAFile   string `toml:"ca_file"`
	CertFile string `toml:"cert_file"`
	KeyFile  string `toml:"key_file"`
}

type ServerIPFilterConfigs struct {
	Allow []string `toml:"allow"`
	Deny  []string `toml:"deny"`
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
	if err := validateIPList("ip_filter.allow", c.IPFilter.Allow); err != nil {
		return err
	}
	if err := validateIPList("ip_filter.deny", c.IPFilter.Deny); err != nil {
		return err
	}
	return nil
}

func validateIPList(field string, ips []string) error {
	for i, raw := range ips {
		ip := strings.TrimSpace(raw)
		if ip == "" {
			return fmt.Errorf("%s[%d] cannot be empty", field, i)
		}
		if net.ParseIP(ip) == nil {
			return fmt.Errorf("invalid %s[%d]: %s", field, i, raw)
		}
	}
	return nil
}
