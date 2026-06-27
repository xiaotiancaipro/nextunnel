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
	cmd.AddCommand(c.newCreate())
	cmd.AddCommand(c.newGenerateCerts())
	return cmd
}

func (c *client) newCreate() *cobra.Command {
	var portStart, portEnd int
	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "create a new client access record",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, posArgs []string) {
			cfg := utils.LoadConfig(cmd)
			utils.ExitOnErr(cmd, args.CreateClient(cmd, cfg, posArgs[0], portStart, portEnd))
		},
	}
	cmd.Flags().IntVar(&portStart, "port-start", 0, "inclusive start of allocated remote port range")
	cmd.Flags().IntVar(&portEnd, "port-end", 0, "inclusive end of allocated remote port range")
	return cmd
}

func (c *client) newGenerateCerts() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "generate-certs [name]",
		Short: "generate client TLS certificates",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, posArgs []string) {
			cfg := utils.LoadConfig(cmd)
			utils.ExitOnErr(cmd, args.GenerateCerts(cmd, cfg, dir, posArgs[0]))
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "output directory for client certificates")
	return cmd
}
