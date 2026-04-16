package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/cmd/root"
)

func New() *cobra.Command {
	c := &cobra.Command{
		Short:   "nextunnel",
		Version: "0.2.0",
		Args:    cobra.ExactArgs(0),
		Run:     func(cmd *cobra.Command, _ []string) { _ = cmd.Help() },
	}
	c.AddCommand(root.NewServer())
	c.AddCommand(root.NewClient())
	return c
}
