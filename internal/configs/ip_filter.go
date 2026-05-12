package configs

type IpFilter struct {
	Allow []string `toml:"allow"`
	Deny  []string `toml:"deny"`
}
