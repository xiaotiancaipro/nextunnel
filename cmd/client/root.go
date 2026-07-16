package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/client"
	shared "github.com/xiaotiancaipro/nextunnel/internal/shared/cli"
)

func New(version string) *cobra.Command {
	c := &cobra.Command{
		Use:     "nextunnel-client",
		Short:   "nextunnel-client",
		Version: version,
		Args:    cobra.ExactArgs(0),
		Run:     run,
	}
	c.Flags().StringP("config", "c", shared.ClientDefaultConfigPath, "Configuration File Path")
	return c
}

func run(cmd *cobra.Command, _ []string) {
	configs := shared.LoadClientConfig(cmd)
	app, err := client.NewApp(configs)
	if err != nil {
		cmd.PrintErr(err)
		os.Exit(1)
	}
	shared.Run(cmd, app)
}
