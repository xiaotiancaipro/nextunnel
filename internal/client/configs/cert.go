package configs

type Cert struct {
	CaFile   string `toml:"ca_file"`
	CertFile string `toml:"cert_file"`
	KeyFile  string `toml:"key_file"`
}
