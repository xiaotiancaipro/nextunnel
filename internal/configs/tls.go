package configs

type Tls struct {
	ServerName string `toml:"server_name"`
	CaFile     string `toml:"ca_file"`
	CertFile   string `toml:"cert_file"`
	KeyFile    string `toml:"key_file"`
}
