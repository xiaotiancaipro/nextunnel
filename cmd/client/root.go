package main

import (
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/client"
	"github.com/xiaotiancaipro/nextunnel/internal/client/cli"
	sharedcli "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
)

func New(version string) *cobra.Command {
	c := &cobra.Command{
		Use:     "nextunnel-client",
		Short:   "nextunnel-client",
		Version: version,
		Args:    cobra.ExactArgs(0),
		Run:     run,
	}
	c.Flags().StringP("config", "c", cli.ClientDefaultConfigPath, "Configuration File Path")
	return c
}

func run(cmd *cobra.Command, _ []string) {
	configs := cli.LoadClientConfig(cmd)
	app, err := client.NewApp(configs)
	sharedcli.ExitOnErr(cmd, err)
	sharedcli.RunApp(cmd, app)
}
