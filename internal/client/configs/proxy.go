package configs

type Proxy struct {
	Name       string `toml:"name"`
	Type       string `toml:"type"`
	LocalIP    string `toml:"local_ip"`
	LocalPort  int    `toml:"local_port"`
	RemotePort int    `toml:"remote_port"`
}
