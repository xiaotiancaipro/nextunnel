package configs

type Server struct {
	Addr        string   `toml:"addr"`
	Port        int      `toml:"port"`
	Token       string   `toml:"token"`
	IpBlacklist []string `toml:"ip_blacklist"`
}
