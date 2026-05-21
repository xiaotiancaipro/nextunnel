package configs

type Database struct {
	Host     string `toml:"host"`
	Port     int64  `toml:"port"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	Database string `toml:"db"`
}
