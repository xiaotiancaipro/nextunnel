package configs

type Logs struct {
	File  string `toml:"file"`
	Level string `toml:"level"`
}
