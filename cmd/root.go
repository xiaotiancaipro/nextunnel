package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel-client/cmd/args"
	"github.com/xiaotiancaipro/nextunnel-client/internal/utils"
)

type root struct{}

func New() *cobra.Command {
	c := &cobra.Command{
		Short:   "nextunnel-client",
		Version: "v0.0.1",
		Args:    cobra.ExactArgs(0),
		Run:     new(root).run,
	}
	c.Flags().StringP("config", "c", "nextunnel-client.toml", "Configuration File Path")
	return c
}

func (c *root) run(cmd *cobra.Command, _ []string) {

	configs := new(args.Config).New(cmd)

	logger, err := utils.NewLogger(configs.Logs)
	if err != nil {
		cmd.PrintErrf("Failed to initialize logging: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Starting nextunnel-client")

}
