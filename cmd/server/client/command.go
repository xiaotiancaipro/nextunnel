package client

import (
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/cmd/server/client/cert"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "client",
		Short: "client tools",
	}
	cmd.AddCommand(NewCreateCommand())
	cmd.AddCommand(cert.NewCommand())
	return cmd
}
