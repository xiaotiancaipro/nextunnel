package cli

import (
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/client/configs"
	shared "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
)

const ClientDefaultConfigPath = "nextunnel-client.toml"

func LoadClientConfig(cmd *cobra.Command) *configs.Configs {
	return shared.LoadConfig(
		cmd,
		shared.ConfigSpec{DefaultPath: ClientDefaultConfigPath},
		configs.NewConfigs,
		"Failed to load client config",
	)
}
