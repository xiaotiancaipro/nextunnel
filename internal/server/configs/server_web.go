package configs

type ServerWeb struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

func (w *ServerWeb) PortOrDefault() int {
	if w == nil || w.Port <= 0 {
		return 25001
	}
	return w.Port
}
