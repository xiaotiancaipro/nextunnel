package configs

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Configs struct {
	Server   *Server   `toml:"server"`
	Logs     *Logs     `toml:"logs"`
	Tls      *Tls      `toml:"tls"`
	Database *Database `toml:"database"`
	GeoIP    *GeoIP    `toml:"geoip"`
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
