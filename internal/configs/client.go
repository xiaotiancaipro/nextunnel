package configs

import (
	"os"

	"github.com/BurntSushi/toml"
)

type ClientConfigs struct {
	ServerAddr string        `toml:"server_addr"`
	ServerPort int           `toml:"server_port"`
	Token      string        `toml:"token"`
	Proxies    []ProxyConfig `toml:"proxies"`
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
	return &configs, nil
}
