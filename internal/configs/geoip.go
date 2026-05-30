package configs

type GeoIP struct {
	DbPath  string   `toml:"db_path"`
	Locales []string `toml:"locales"`
}
