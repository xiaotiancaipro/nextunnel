package configs

type CertServer struct {
	Host string `toml:"host"`
	Dir  string `toml:"dir"`
}

type CertClient struct {
	CaFile   string `toml:"ca_file"`
	CertFile string `toml:"cert_file"`
	KeyFile  string `toml:"key_file"`
}
