package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/cmd/server/client"
	"github.com/xiaotiancaipro/nextunnel/cmd/server/ip_filter"
	"github.com/xiaotiancaipro/nextunnel/internal/server"
	"github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
)

func New(version string) *cobra.Command {
	c := &cobra.Command{
		Use:     "nextunnel-server",
		Short:   "nextunnel-server",
		Version: version,
		Args:    cobra.ExactArgs(0),
		Run:     run,
	}
	c.PersistentFlags().String("config", cli.ServerDefaultConfigPath, "configuration file path (overrides $"+cli.ServerEnvConfigPath+")")
	c.AddCommand(client.NewCommand())
	c.AddCommand(ip_filter.NewCommand())
	return c
}

func run(cmd *cobra.Command, _ []string) {
	cfg := cli.LoadServerConfig(cmd)
	app, err := server.NewApp(cfg, cmd.Version)
	if err != nil {
		cmd.PrintErr(err)
		os.Exit(1)
	}
	cli.Run(cmd, app)
}
