package configs

type Web struct {
	Enabled bool `toml:"enabled"`
	Port    int  `toml:"port"`
}

func (w *Web) PortOrDefault() int {
	if w == nil || w.Port <= 0 {
		return 25001
	}
	return w.Port
}

func (w *Web) IsEnabled() bool {
	return w != nil && w.Enabled
}
