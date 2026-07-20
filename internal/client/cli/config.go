package cli

import (
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/client/configs"
	sharedcli "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
)

const ClientDefaultConfigPath = "nextunnel-client.toml"

func LoadClientConfig(cmd *cobra.Command) *configs.Configs {
	spec := sharedcli.ConfigSpec{DefaultPath: ClientDefaultConfigPath}
	return sharedcli.LoadConfig(cmd, spec, configs.Configs{})
}
