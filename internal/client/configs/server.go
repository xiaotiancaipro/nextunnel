package configs

import "fmt"

type Server struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

func (c *Configs) CheckServer() error {
	if c.Server == nil {
		c.Server = &Server{}
	}
	if c.Server.Host == "" {
		return fmt.Errorf("[server.host] is required")
	}
	if c.Server.Port <= 0 {
		return fmt.Errorf("[server.port] is required")
	}
	return nil
}
