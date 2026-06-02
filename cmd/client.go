package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel-server/cmd/args"
	"github.com/xiaotiancaipro/nextunnel-server/cmd/utils"
)

type client struct{}

func (c *client) new() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "client",
		Short: "client tools",
	}
	cmd.AddCommand(c.newGenerateCerts())
	return cmd
}

func (c *client) newGenerateCerts() *cobra.Command {
	return &cobra.Command{
		Use:   "generate-certs [output-dir]",
		Short: "generate client TLS certificates",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, posArgs []string) {
			cfg := utils.LoadConfig(cmd)
			utils.ExitOnErr(cmd, args.GenerateCerts(cmd, cfg, posArgs[0]))
		},
	}
}
