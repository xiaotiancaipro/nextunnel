package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/cmd/root/client"
	"github.com/xiaotiancaipro/nextunnel/cmd/root/server"
)

func New() *cobra.Command {
	c := &cobra.Command{
		Short:   "nextunnel",
		Version: "0.1.0",
		Args:    cobra.ExactArgs(0),
		Run:     func(cmd *cobra.Command, _ []string) { _ = cmd.Help() },
	}
	c.AddCommand(server.New())
	c.AddCommand(client.NewClient())
	return c
}
