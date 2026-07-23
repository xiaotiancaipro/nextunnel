package main

import (
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/client"
	"github.com/xiaotiancaipro/nextunnel/internal/client/cli"
	sharedcli "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
)

func New(version string) *cobra.Command {
	c := &cobra.Command{
		Use:          "nextunnel-client",
		Short:        "nextunnel-client",
		Version:      version,
		Args:         cobra.ExactArgs(0),
		RunE:         run,
		SilenceUsage: true,
	}
	c.Flags().StringP("config", "c", cli.ClientDefaultConfigPath, "configuration file path (overrides $"+cli.ClientEnvConfigPath+")")
	return c
}

func run(cmd *cobra.Command, _ []string) error {
	config, err := cli.LoadClientConfig(cmd)
	if err != nil {
		return err
	}
	app := client.App{Configs: config}
	if err := app.Init(); err != nil {
		return err
	}
	return sharedcli.RunApp(&app)
}
