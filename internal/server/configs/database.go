package configs

type Database struct {
	Host     string `toml:"host"`
	Port     int64  `toml:"port"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	Database string `toml:"db"`
	SSLMode  string `toml:"sslmode"`
}

func (d *Database) SSLModeOrDefault() string {
	if d == nil || d.SSLMode == "" {
		return "disable"
	}
	return d.SSLMode
}
