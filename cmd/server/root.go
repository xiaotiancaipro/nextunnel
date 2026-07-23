package main

import (
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/cmd/server/client"
	"github.com/xiaotiancaipro/nextunnel/cmd/server/ip_filter"
	"github.com/xiaotiancaipro/nextunnel/internal/server"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli"
	sharedcli "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
)

func New(version string) *cobra.Command {
	c := &cobra.Command{
		Use:          "nextunnel-server",
		Short:        "nextunnel-server",
		Version:      version,
		Args:         cobra.ExactArgs(0),
		RunE:         run,
		SilenceUsage: true,
	}
	c.PersistentFlags().String("config", cli.ServerDefaultConfigPath, "configuration file path (overrides $"+cli.ServerEnvConfigPath+")")
	c.AddCommand(client.NewCommand())
	c.AddCommand(ip_filter.NewCommand())
	return c
}

func run(cmd *cobra.Command, _ []string) error {
	config, err := cli.LoadServerConfig(cmd)
	if err != nil {
		return err
	}
	app := server.App{Configs: config}
	if err := app.Init(); err != nil {
		return err
	}
	return sharedcli.RunApp(&app)
}
