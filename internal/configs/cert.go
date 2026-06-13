package configs

type Cert struct {
	Host string `toml:"host"`
	Dir  string `toml:"dir"`
}
