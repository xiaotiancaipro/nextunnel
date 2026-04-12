package configs

import (
	"os"

	"github.com/BurntSushi/toml"
)

const DefaultLocalIP = "127.0.0.1"

// ClientConfigs 客户端配置
type ClientConfigs struct {
	ServerAddr string        `toml:"server_addr"`
	ServerPort int           `toml:"server_port"`
	Token      string        `toml:"token"`
	Proxies    []ProxyConfig `toml:"proxies"`
}

// ProxyConfig 单个代理配置
type ProxyConfig struct {
	Name       string `toml:"name"`
	Type       string `toml:"type"`        // 当前只支持 "tcp"
	RemotePort int    `toml:"remote_port"` // 服务端暴露的端口
	LocalIP    string `toml:"local_ip"`    // 本地服务 IP
	LocalPort  int    `toml:"local_port"`  // 本地服务端口
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
