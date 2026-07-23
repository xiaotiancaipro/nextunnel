package configs

import "fmt"

type Database struct {
	Host     string `toml:"host"`
	Port     int64  `toml:"port"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	Database string `toml:"db"`
	SSLMode  string `toml:"sslmode"`
}

func (c *Configs) CheckDatabase() error {
	if c.Database == nil {
		return fmt.Errorf("[database] is required")
	}
	if c.Database.Host == "" {
		return fmt.Errorf("[database.host] is required")
	}
	if c.Database.Port <= 0 {
		return fmt.Errorf("[database.port] is required")
	}
	if c.Database.Username == "" {
		return fmt.Errorf("[database.username] is required")
	}
	if c.Database.Password == "" {
		return fmt.Errorf("[database.password] is required")
	}
	if c.Database.Database == "" {
		return fmt.Errorf("[database.db] is required")
	}
	if c.Database.SSLMode == "" {
		return fmt.Errorf("[database.sslmode] is required")
	}
	return nil
}

func (d *Database) SSLModeOrDefault() string {
	if d == nil || d.SSLMode == "" {
		return "disable"
	}
	return d.SSLMode
}
