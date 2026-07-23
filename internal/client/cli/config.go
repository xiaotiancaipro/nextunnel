package cli

import (
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/client/configs"
	sharedcli "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
)

const (
	ClientDefaultConfigPath = "nextunnel-client.toml"
	ClientEnvConfigPath     = "NEXTUNNEL_CLIENT_CONFIG"
)

func LoadClientConfig(cmd *cobra.Command) *configs.Configs {
	spec := sharedcli.ConfigSpec{
		DefaultPath: ClientDefaultConfigPath,
		EnvVar:      ClientEnvConfigPath,
	}
	c := sharedcli.LoadConfig(cmd, spec, configs.Configs{})
	sharedcli.ExitOnErr(cmd, c.CheckCert())
	sharedcli.ExitOnErr(cmd, c.CheckClient())
	sharedcli.ExitOnErr(cmd, c.CheckLogs())
	sharedcli.ExitOnErr(cmd, c.CheckServer())
	return c
}
