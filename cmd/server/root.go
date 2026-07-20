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
	config := cli.LoadServerConfig(cmd)
	app := new(server.App)
	if err := app.Init(config); err != nil {
		sharedcli.ExitOnErr(cmd, err)
	}

	app, err := server.NewApp(config, cmd.Version)
	sharedcli.ExitOnErr(cmd, err)
	sharedcli.RunApp(cmd, app)
}
