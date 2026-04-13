package configs

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

type ClientConfigs struct {
	ServerAddr string           `toml:"server_addr"`
	ServerPort int              `toml:"server_port"`
	Token      string           `toml:"token"`
	TLS        ClientTLSConfigs `toml:"tls"`
	Proxies    []ProxyConfig    `toml:"proxies"`
}

type ClientTLSConfigs struct {
	Enabled            bool   `toml:"enabled"`
	ServerName         string `toml:"server_name"`
	CAFile             string `toml:"ca_file"`
	CertFile           string `toml:"cert_file"`
	KeyFile            string `toml:"key_file"`
	InsecureSkipVerify bool   `toml:"insecure_skip_verify"`
}

type ProxyConfig struct {
	Name       string `toml:"name"`
	Type       string `toml:"type"`        // currently only "tcp" is supported
	RemotePort int    `toml:"remote_port"` // port exposed by the server
	LocalIP    string `toml:"local_ip"`    // local service IP
	LocalPort  int    `toml:"local_port"`  // local service port
}

func NewClient(file string) (*ClientConfigs, error) {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil, err
	}
	var configs ClientConfigs
	if _, err := toml.DecodeFile(file, &configs); err != nil {
		return nil, err
	}
	if err := configs.Validate(); err != nil {
		return nil, err
	}
	return &configs, nil
}

func (c *ClientConfigs) Validate() error {
	if strings.TrimSpace(c.ServerAddr) == "" {
		return fmt.Errorf("server_addr cannot be empty")
	}
	if c.ServerPort <= 0 || c.ServerPort > 65535 {
		return fmt.Errorf("invalid server_port: %d", c.ServerPort)
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
	}
	names := make(map[string]struct{}, len(c.Proxies))
	for i, proxy := range c.Proxies {
		if strings.TrimSpace(proxy.Name) == "" {
			return fmt.Errorf("proxies[%d].name cannot be empty", i)
		}
		if proxy.Type != "tcp" {
			return fmt.Errorf("proxies[%d].type must be tcp", i)
		}
		if proxy.LocalPort <= 0 || proxy.LocalPort > 65535 {
			return fmt.Errorf("invalid proxies[%d].local_port: %d", i, proxy.LocalPort)
		}
		if proxy.RemotePort <= 0 || proxy.RemotePort > 65535 {
			return fmt.Errorf("invalid proxies[%d].remote_port: %d", i, proxy.RemotePort)
		}
		if _, exists := names[proxy.Name]; exists {
			return fmt.Errorf("duplicate proxy name: %s", proxy.Name)
		}
		names[proxy.Name] = struct{}{}
	}
	return nil
}
