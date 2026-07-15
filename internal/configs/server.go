package configs

type Server struct {
	Port int `toml:"port"`
}

type ServerClient struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}
