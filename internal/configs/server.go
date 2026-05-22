package configs

type Server struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}
