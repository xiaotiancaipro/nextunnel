package configs

import "fmt"

type Cert struct {
	CaFile   string `toml:"ca_file"`
	CertFile string `toml:"cert_file"`
	KeyFile  string `toml:"key_file"`
}

func (c *Configs) CheckCert() error {
	if c.Cert == nil {
		return fmt.Errorf("[cert] is required")
	}
	if c.Cert.CaFile == "" {
		return fmt.Errorf("[cert.ca_file] is required")
	}
	if c.Cert.CertFile == "" {
		return fmt.Errorf("[cert.cert_file] is required")
	}
	if c.Cert.KeyFile == "" {
		return fmt.Errorf("[cert.key_file] is required")
	}
	return nil
}
