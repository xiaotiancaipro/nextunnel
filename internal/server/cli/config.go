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
	return sharedcli.LoadConfig(
		cmd,
		sharedcli.ConfigSpec{
			DefaultPath: ServerDefaultConfigPath,
			EnvVar:      ServerEnvConfigPath,
		},
		configs.NewConfigs,
		"Failed to load config",
	)
}
