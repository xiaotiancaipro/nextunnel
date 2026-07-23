package configs

import "fmt"

type Client struct {
	Id string `toml:"id"`
}

func (c *Configs) CheckClient() error {
	if c.Client == nil {
		return fmt.Errorf("[client] is required")
	}
	if c.Client.Id == "" {
		return fmt.Errorf("[client.id] is required")
	}
	return nil
}
