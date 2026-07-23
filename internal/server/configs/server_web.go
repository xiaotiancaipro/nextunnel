package configs

const (
	defaultServerWebHost = "127.0.0.1"
	defaultServerWebPort = 25001
)

type ServerWeb struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

func (c *Configs) CheckServerWeb() error {
	if c.ServerWeb == nil {
		c.ServerWeb = &ServerWeb{}
	}
	if c.ServerWeb.Host == "" {
		c.ServerWeb.Host = defaultServerWebHost
	}
	if c.ServerWeb.Port <= 0 {
		c.ServerWeb.Port = defaultServerWebPort
	}
	return nil
}

func (w *ServerWeb) HostOrDefault() string {
	if w == nil || w.Host == "" {
		return defaultServerWebHost
	}
	return w.Host
}

func (w *ServerWeb) PortOrDefault() int {
	if w == nil || w.Port <= 0 {
		return defaultServerWebPort
	}
	return w.Port
}
