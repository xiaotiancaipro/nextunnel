package configs

type IPLocation struct {
	Type         string   `toml:"type"`
	APIKey       string   `toml:"api_key"`
	GeoIPDbPath  string   `toml:"geoip_db_path"`
	GeoIPLocales []string `toml:"geoip_locales"`
}
