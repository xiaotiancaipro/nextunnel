package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli/client"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli/ip_filter"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli/utils"

	"github.com/xiaotiancaipro/nextunnel/internal/server"
)

func New(version string) *cobra.Command {
	c := &cobra.Command{
		Use:     "nextunnel-server",
		Short:   "nextunnel-server",
		Version: version,
		Args:    cobra.ExactArgs(0),
		Run:     run,
	}
	c.PersistentFlags().String("config", utils.ServerDefaultConfigPath, "configuration file path (overrides $"+utils.ServerEnvConfigPath+")")
	c.AddCommand(client.NewCommand())
	c.AddCommand(ip_filter.NewCommand())
	return c
}

func run(cmd *cobra.Command, _ []string) {
	cfg := utils.LoadServerConfig(cmd)
	app, err := server.NewApp(cfg, cmd.Version)
	if err != nil {
		cmd.PrintErr(err)
		os.Exit(1)
	}
	utils.Run(cmd, app)
}
