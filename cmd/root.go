package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/cmd/client"
	"github.com/xiaotiancaipro/nextunnel/cmd/server"
)

type Root struct{}

func (r *Root) New(version string) *cobra.Command {
	c := &cobra.Command{
		Use:     "nextunnel",
		Short:   "nextunnel",
		Version: version,
		Args:    cobra.ExactArgs(0),
		Run:     func(cmd *cobra.Command, _ []string) { _ = cmd.Help() },
	}
	c.AddCommand(new(server.Root).New())
	c.AddCommand(new(client.Root).New())
	return c
}
