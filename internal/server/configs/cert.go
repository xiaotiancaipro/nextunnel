package configs

import "fmt"

type Cert struct {
	Host string `toml:"host"`
	Dir  string `toml:"dir"`
}

func (c *Configs) CheckCert() error {
	if c.Cert == nil {
		return fmt.Errorf("[cert] is required")
	}
	if c.Cert.Host == "" {
		return fmt.Errorf("[cert.host] is required")
	}
	if c.Cert.Dir == "" {
		return fmt.Errorf("[cert.dir] is required")
	}
	return nil
}
