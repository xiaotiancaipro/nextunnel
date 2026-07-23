package cli

import (
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/configs"
	sharedcli "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
)

const (
	ServerDefaultConfigPath = "nextunnel-server.toml"
	ServerEnvConfigPath     = "NEXTUNNEL_SERVER_CONFIG"
)

func LoadServerConfig(cmd *cobra.Command) *configs.Configs {
	spec := sharedcli.ConfigSpec{
		DefaultPath: ServerDefaultConfigPath,
		EnvVar:      ServerEnvConfigPath,
	}
	c := sharedcli.LoadConfig(cmd, spec, configs.Configs{})
	sharedcli.ExitOnErr(cmd, c.CheckCert())
	sharedcli.ExitOnErr(cmd, c.CheckDatabase())
	sharedcli.ExitOnErr(cmd, c.CheckIPLocation())
	sharedcli.ExitOnErr(cmd, c.CheckLogs())
	sharedcli.ExitOnErr(cmd, c.CheckServer())
	sharedcli.ExitOnErr(cmd, c.CheckServerWeb())
	return c
}
