package client

import (
	"github.com/spf13/cobra"
	"github.com/xiaotiancaipro/nextunnel/internal/server/cli/utils"
)

var (
	portStart int
	portEnd   int
)

func NewCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "create a new client access record",
		Args:  cobra.ExactArgs(1),
		Run:   createRun,
	}
	cmd.Flags().IntVar(&portStart, "port-start", 0, "inclusive start of allocated remote port range")
	cmd.Flags().IntVar(&portEnd, "port-end", 0, "inclusive end of allocated remote port range")
	return cmd
}

func createRun(cmd *cobra.Command, args []string) {
	cfg := utils.LoadServerConfig(cmd)
	utils.ExitOnErr(cmd, utils.CreateClient(cmd, cfg, args[0], portStart, portEnd))
}
