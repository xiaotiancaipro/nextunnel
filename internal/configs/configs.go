package configs

import (
	"fmt"
	"os"

	"github.com/xiaotiancaipro/nextunnel/internal/utils/timezone"
)

type ConfigsServer struct {
	Server     *Server     `toml:"server"`
	Cert       *CertServer `toml:"cert"`
	Database   *Database   `toml:"database"`
	IPLocation *IPLocation `toml:"ip_location"`
	Logs       *Logs       `toml:"logs"`
	Timezone   *Timezone   `toml:"timezone"`
	Web        *Web        `toml:"web"`
}

type ConfigsClient struct {
	Server   *Server     `toml:"server"`
	Client   *Client     `toml:"client"`
	Cert     *CertClient `toml:"cert"`
	Logs     *Logs       `toml:"logs"`
	Timezone *Timezone   `toml:"timezone"`
	Proxies  []Proxy     `toml:"proxies"`
}

func NewConfigsServer(file string) (*ConfigsServer, error) {

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil, err
	}

	var configs ConfigsServer
	if _, err := toml.DecodeFile(file, &configs); err != nil {
		return nil, err
	}

	if err := timezone.Init(configs.Timezone.NameOrDefault()); err != nil {
		return nil, fmt.Errorf("invalid timezone config: %w", err)
	}

	return &configs, nil

}

func NewConfigsClient(file string) (*ConfigsClient, error) {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil, err
	}
	var configs ConfigsClient
	if _, err := toml.DecodeFile(file, &configs); err != nil {
		return nil, err
	}
	if err := timezone.Init(configs.Timezone.NameOrDefault()); err != nil {
		return nil, fmt.Errorf("invalid timezone config: %w", err)
	}
	return &configs, nil
}
