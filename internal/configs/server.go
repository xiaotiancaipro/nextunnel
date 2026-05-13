package configs

type Server struct {
	Addr        string   `toml:"addr"`
	Port        int      `toml:"port"`
	IpBlacklist []string `toml:"ip_blacklist"`
}
