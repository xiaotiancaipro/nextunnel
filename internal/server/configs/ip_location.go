package configs

import "fmt"

type IPLocation struct {
	APIKey string `toml:"api_key"`
}

func (c *Configs) CheckIPLocation() error {
	if c.IPLocation == nil {
		return fmt.Errorf("[ip_location] is required")
	}
	if c.IPLocation.APIKey == "" {
		return fmt.Errorf("[ip_location.api_key] is required")
	}
	return nil
}
