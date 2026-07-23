package configs

const defaultServerPort = 25930

type Server struct {
	Port int `toml:"port"`
}

func (c *Configs) CheckServer() error {
	if c.Server == nil {
		c.Server = &Server{}
	}
	if c.Server.Port <= 0 {
		c.Server.Port = defaultServerPort
	}
	return nil
}
